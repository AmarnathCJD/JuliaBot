package extras

import (
	"encoding/binary"
	"encoding/json"
	"maps"
	"strings"
	"sync"
	"time"

	modules "main/modules"
	"main/modules/db"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
)

const userPerfBucket = "user_perf_v1"

type userPerf struct {
	TotalMsgs   int64            `json:"total_msgs"`
	MediaMsgs   int64            `json:"media_msgs"`
	StickerMsgs int64            `json:"sticker_msgs"`
	LinkMsgs    int64            `json:"link_msgs"`
	ReplyMsgs   int64            `json:"reply_msgs"`
	CmdMsgs     int64            `json:"cmd_msgs"`
	NightMsgs   int64            `json:"night_msgs"`
	CharSum     int64            `json:"char_sum"`
	Chats       map[int64]int64  `json:"chats"`
	Commands    map[string]int64 `json:"commands"`
	HourBuckets [24]int64        `json:"hour_buckets"`
	DailyMsgs   map[string]int64 `json:"daily_msgs"`
	FirstSeen   int64            `json:"first_seen"`
	LastSeen    int64            `json:"last_seen"`
}

var (
	userPerfCache   = map[int64]*userPerf{}
	userPerfDirty   = map[int64]bool{}
	userPerfMu      sync.Mutex
	userPerfFlushed bool
	userPerfFlushMu sync.Mutex
)

func userPerfKey(userID int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(userID))
	return b
}

func userPerfNew() *userPerf {
	return &userPerf{
		Chats:     map[int64]int64{},
		Commands:  map[string]int64{},
		DailyMsgs: map[string]int64{},
	}
}

func userPerfLoadDisk(userID int64) *userPerf {
	out := userPerfNew()
	d, err := db.GetDB()
	if err != nil || d == nil {
		return out
	}
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(userPerfBucket))
		if b == nil {
			return nil
		}
		raw := b.Get(userPerfKey(userID))
		if len(raw) == 0 {
			return nil
		}
		_ = json.Unmarshal(raw, out)
		if out.Chats == nil {
			out.Chats = map[int64]int64{}
		}
		if out.Commands == nil {
			out.Commands = map[string]int64{}
		}
		if out.DailyMsgs == nil {
			out.DailyMsgs = map[string]int64{}
		}
		return nil
	})
	return out
}

func UserPerfGet(userID int64) *userPerf {
	if userID == 0 {
		return userPerfNew()
	}
	userPerfMu.Lock()
	defer userPerfMu.Unlock()
	if e, ok := userPerfCache[userID]; ok {
		cp := *e
		cp.Chats = mapCopyInt64(e.Chats)
		cp.Commands = mapCopyStrInt64(e.Commands)
		return &cp
	}
	e := userPerfLoadDisk(userID)
	userPerfCache[userID] = e
	cp := *e
	cp.Chats = mapCopyInt64(e.Chats)
	cp.Commands = mapCopyStrInt64(e.Commands)
	cp.DailyMsgs = mapCopyStrInt64(e.DailyMsgs)
	return &cp
}

func mapCopyInt64(m map[int64]int64) map[int64]int64 {
	out := make(map[int64]int64, len(m))
	maps.Copy(out, m)
	return out
}

func mapCopyStrInt64(m map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func userPerfFlushAll() {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return
	}
	userPerfMu.Lock()
	pending := make(map[int64]*userPerf, len(userPerfDirty))
	for uid := range userPerfDirty {
		if e, ok := userPerfCache[uid]; ok {
			cp := *e
			cp.Chats = mapCopyInt64(e.Chats)
			cp.Commands = mapCopyStrInt64(e.Commands)
			cp.DailyMsgs = mapCopyStrInt64(e.DailyMsgs)
			pending[uid] = &cp
		}
	}
	userPerfDirty = map[int64]bool{}
	userPerfMu.Unlock()

	_ = d.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(userPerfBucket))
		if err != nil {
			return err
		}
		for uid, e := range pending {
			raw, jerr := json.Marshal(e)
			if jerr != nil {
				continue
			}
			_ = b.Put(userPerfKey(uid), raw)
		}
		return nil
	})
}

func userPerfStartFlusher() {
	userPerfFlushMu.Lock()
	if userPerfFlushed {
		userPerfFlushMu.Unlock()
		return
	}
	userPerfFlushed = true
	userPerfFlushMu.Unlock()
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			userPerfFlushAll()
		}
	}()
}

func userPerfHasURL(text string) bool {
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	return strings.Contains(lower, "http://") || strings.Contains(lower, "https://") || strings.Contains(lower, "t.me/") || strings.Contains(lower, "www.")
}

func UserPerfTracker(m *tg.NewMessage) error {
	if m == nil || m.Message == nil {
		return nil
	}
	if m.IsService() {
		return nil
	}
	senderID := m.SenderID()
	if senderID == 0 {
		return nil
	}
	if m.Sender != nil && m.Sender.Bot {
		return nil
	}
	chatID := m.ChatID()
	now := time.Now()
	hour := now.Hour()
	isNight := hour < 6 || hour >= 22

	text := m.Text()
	isCmd := m.IsCommand()
	cmdName := ""
	if isCmd {
		fields := strings.Fields(text)
		if len(fields) > 0 {
			head := strings.ToLower(fields[0])
			if len(head) > 1 && (head[0] == '/' || head[0] == '.' || head[0] == '!' || head[0] == '?' || head[0] == '-') {
				head = head[1:]
			}
			if at := strings.Index(head, "@"); at > 0 {
				head = head[:at]
			}
			if head != "" && len(head) <= 32 {
				cmdName = head
			}
		}
	}

	isSticker := m.Sticker() != nil
	isMedia := m.Media() != nil
	hasLink := userPerfHasURL(text)
	isReply := m.IsReply()
	charLen := int64(len([]rune(text)))

	userPerfMu.Lock()
	e, ok := userPerfCache[senderID]
	if !ok {
		e = userPerfLoadDisk(senderID)
		userPerfCache[senderID] = e
	}
	e.TotalMsgs++
	if isMedia {
		e.MediaMsgs++
	}
	if isSticker {
		e.StickerMsgs++
	}
	if hasLink {
		e.LinkMsgs++
	}
	if isReply {
		e.ReplyMsgs++
	}
	if isCmd {
		e.CmdMsgs++
	}
	if isNight {
		e.NightMsgs++
	}
	e.CharSum += charLen
	if chatID != 0 {
		e.Chats[chatID]++
	}
	if cmdName != "" {
		if len(e.Commands) < 128 || e.Commands[cmdName] > 0 {
			e.Commands[cmdName]++
		}
	}
	e.HourBuckets[hour]++
	if e.DailyMsgs == nil {
		e.DailyMsgs = map[string]int64{}
	}
	today := now.Format("2006-01-02")
	e.DailyMsgs[today]++
	if len(e.DailyMsgs) > 40 {
		cutoff := now.AddDate(0, 0, -30).Format("2006-01-02")
		for k := range e.DailyMsgs {
			if k < cutoff {
				delete(e.DailyMsgs, k)
			}
		}
	}
	if e.FirstSeen == 0 {
		e.FirstSeen = now.Unix()
	}
	e.LastSeen = now.Unix()
	userPerfDirty[senderID] = true
	userPerfMu.Unlock()

	userPerfStartFlusher()
	return nil
}

func registerUserPerfHandlers() {
	c := modules.Client
	c.On(tg.OnNewMessage, UserPerfTracker)
}

func init() {
	modules.QueueHandlerRegistration(registerUserPerfHandlers)
}
