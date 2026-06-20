package modules

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"image/color"
	"main/modules/db"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	bolt "go.etcd.io/bbolt"
)

type moodEntry struct {
	Mood string `json:"mood"`
	TS   int64  `json:"ts"`
}

type moodRecord struct {
	UserID  int64       `json:"u"`
	Name    string      `json:"n"`
	Current moodEntry   `json:"c"`
	Recent  []moodEntry `json:"r"`
}

var (
	moodBucket = []byte("moods")
	moodKinds  = map[string]string{
		"happy":   "😄",
		"sad":     "😢",
		"angry":   "😠",
		"tired":   "😴",
		"excited": "🤩",
		"bored":   "🥱",
	}
	moodOrder = []string{"happy", "sad", "angry", "tired", "excited", "bored"}
)

func moodKeyBytes(userID int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(userID))
	return b
}

func moodEnsureBucket() error {
	d, err := db.GetDB()
	if err != nil {
		return err
	}
	return d.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(moodBucket)
		return e
	})
}

func moodGet(userID int64) (*moodRecord, error) {
	d, err := db.GetDB()
	if err != nil {
		return nil, err
	}
	if err := moodEnsureBucket(); err != nil {
		return nil, err
	}
	var rec *moodRecord
	err = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(moodBucket)
		if b == nil {
			return nil
		}
		raw := b.Get(moodKeyBytes(userID))
		if raw == nil {
			return nil
		}
		var r moodRecord
		if jerr := json.Unmarshal(raw, &r); jerr != nil {
			return nil
		}
		rec = &r
		return nil
	})
	return rec, err
}

func moodPut(rec *moodRecord) error {
	d, err := db.GetDB()
	if err != nil {
		return err
	}
	if err := moodEnsureBucket(); err != nil {
		return err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return d.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(moodBucket)
		if b == nil {
			return fmt.Errorf("bucket missing")
		}
		return b.Put(moodKeyBytes(rec.UserID), data)
	})
}

func moodList() ([]*moodRecord, error) {
	d, err := db.GetDB()
	if err != nil {
		return nil, err
	}
	if err := moodEnsureBucket(); err != nil {
		return nil, err
	}
	var out []*moodRecord
	err = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(moodBucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var r moodRecord
			if jerr := json.Unmarshal(v, &r); jerr != nil {
				return nil
			}
			out = append(out, &r)
			return nil
		})
	})
	return out, err
}

func moodSenderName(m *tg.NewMessage) string {
	if m.Sender == nil {
		return ""
	}
	name := strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
	if name == "" && m.Sender.Username != "" {
		name = "@" + m.Sender.Username
	}
	return name
}

func moodEmoji(name string) string {
	if e, ok := moodKinds[name]; ok {
		return e
	}
	return "❓"
}

func moodFormatAgo(ts int64) string {
	if ts <= 0 {
		return "?"
	}
	d := time.Since(time.Unix(ts, 0))
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func MoodHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(strings.ToLower(m.Args()))
	if args == "" {
		rec, _ := moodGet(m.SenderID())
		if rec == nil || rec.Current.Mood == "" {
			var keys []string
			for _, k := range moodOrder {
				keys = append(keys, fmt.Sprintf("%s %s", moodKinds[k], k))
			}
			m.Reply("<b>Usage:</b> <code>/mood &lt;mood&gt;</code>\n<b>Moods:</b> " + html.EscapeString(strings.Join(keys, ", ")))
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Your mood:</b> %s <code>%s</code>\n<i>set %s</i>",
			moodEmoji(rec.Current.Mood), html.EscapeString(rec.Current.Mood), moodFormatAgo(rec.Current.TS)))
		return nil
	}

	kind := strings.Fields(args)[0]
	if _, ok := moodKinds[kind]; !ok {
		m.Reply("<b>Unknown mood.</b> Try: happy, sad, angry, tired, excited, bored.")
		return nil
	}

	rec, _ := moodGet(m.SenderID())
	if rec == nil {
		rec = &moodRecord{UserID: m.SenderID()}
	}
	rec.Name = moodSenderName(m)
	now := time.Now().Unix()
	entry := moodEntry{Mood: kind, TS: now}
	rec.Recent = append([]moodEntry{entry}, rec.Recent...)
	if len(rec.Recent) > 5 {
		rec.Recent = rec.Recent[:5]
	}
	rec.Current = entry

	if err := moodPut(rec); err != nil {
		m.Reply("<b>Failed to save mood.</b>")
		return nil
	}
	m.Reply(fmt.Sprintf("<b>Mood set:</b> %s <code>%s</code>", moodEmoji(kind), html.EscapeString(kind)))
	return nil
}

func moodChatMembers(chatID int64) []*moodRecord {
	chatStatsMu.Lock()
	cs := chatStatsLoad(chatID)
	userIDs := make([]int64, 0, len(cs.Users))
	for uid := range cs.Users {
		userIDs = append(userIDs, uid)
	}
	chatStatsMu.Unlock()

	var out []*moodRecord
	for _, uid := range userIDs {
		rec, _ := moodGet(uid)
		if rec == nil || rec.Current.Mood == "" {
			continue
		}
		out = append(out, rec)
	}
	return out
}

func MoodsHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		rec, _ := moodGet(m.SenderID())
		if rec == nil || len(rec.Recent) == 0 {
			m.Reply("<b>No moods saved yet.</b>\n<i>Use</i> <code>/mood happy</code>")
			return nil
		}
		var b strings.Builder
		b.WriteString("<b>Your recent moods</b>\n")
		b.WriteString("━━━━━━━━━━━━━━━━\n")
		for i, e := range rec.Recent {
			b.WriteString(fmt.Sprintf(" %d. %s <code>%s</code> — <i>%s</i>\n",
				i+1, moodEmoji(e.Mood), html.EscapeString(e.Mood), moodFormatAgo(e.TS)))
		}
		m.Reply(b.String())
		return nil
	}

	members := moodChatMembers(m.ChatID())
	if len(members) == 0 {
		m.Reply("<b>No moods tracked for this chat yet.</b>\n<i>Use</i> <code>/mood happy</code>")
		return nil
	}

	sort.Slice(members, func(i, j int) bool {
		return members[i].Current.TS > members[j].Current.TS
	})

	limit := 10
	if len(members) < limit {
		limit = len(members)
	}

	var b strings.Builder
	b.WriteString("<b>Recent Moods</b>\n")
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	for i := 0; i < limit; i++ {
		rec := members[i]
		name := html.EscapeString(rec.Name)
		if name == "" {
			name = strconv.FormatInt(rec.UserID, 10)
		}
		mention := fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", rec.UserID, name)
		b.WriteString(fmt.Sprintf("\n %s\n", mention))
		max := len(rec.Recent)
		if max > 5 {
			max = 5
		}
		if max == 0 {
			b.WriteString(fmt.Sprintf("   %s <code>%s</code> <i>(%s)</i>\n",
				moodEmoji(rec.Current.Mood), html.EscapeString(rec.Current.Mood), moodFormatAgo(rec.Current.TS)))
			continue
		}
		for j := 0; j < max; j++ {
			e := rec.Recent[j]
			b.WriteString(fmt.Sprintf("   %s <code>%s</code> <i>(%s)</i>\n",
				moodEmoji(e.Mood), html.EscapeString(e.Mood), moodFormatAgo(e.TS)))
		}
	}
	m.Reply(b.String())
	return nil
}

func moodLoadFont(dc *gg.Context, size float64) {
	name := getRandomFont()
	candidates := []string{
		"./assets/" + name,
		"assets/" + name,
		"../assets/" + name,
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "assets", name))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "assets", name),
			filepath.Join(dir, "..", "assets", name),
		)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if err := dc.LoadFontFace(p, size); err == nil {
				return
			}
		}
	}
}

func moodTruncate(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	if n <= 1 {
		return string(rs[:n])
	}
	return string(rs[:n-1]) + "…"
}

func MoodBoardHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>/moodboard only works in groups.</b>")
		return nil
	}

	members := moodChatMembers(m.ChatID())
	if len(members) == 0 {
		m.Reply("<b>No moods tracked for this chat yet.</b>\n<i>Use</i> <code>/mood happy</code>")
		return nil
	}

	sort.Slice(members, func(i, j int) bool {
		return members[i].Current.TS > members[j].Current.TS
	})

	if len(members) > 36 {
		members = members[:36]
	}

	status, _ := m.Reply("<code>building moodboard...</code>")

	cols := int(math.Ceil(math.Sqrt(float64(len(members)))))
	if cols < 1 {
		cols = 1
	}
	rows := int(math.Ceil(float64(len(members)) / float64(cols)))

	const cell = 200
	const pad = 20
	const headerH = 90
	W := cols*cell + pad*2
	H := rows*cell + pad*2 + headerH

	dc := gg.NewContext(W, H)
	dc.SetRGB(0.07, 0.07, 0.10)
	dc.Clear()

	moodLoadFont(dc, 36)
	dc.SetRGBA255(245, 245, 250, 255)
	dc.DrawStringAnchored("MOODBOARD", float64(W)/2, float64(headerH)/2-4, 0.5, 0.5)
	moodLoadFont(dc, 18)
	dc.SetRGBA255(180, 180, 200, 255)
	dc.DrawStringAnchored(fmt.Sprintf("%d members tracked", len(members)), float64(W)/2, float64(headerH)/2+28, 0.5, 0.5)

	palette := []color.RGBA{
		{0x2a, 0x2d, 0x3a, 0xff},
		{0x33, 0x29, 0x42, 0xff},
		{0x23, 0x33, 0x3d, 0xff},
		{0x3a, 0x2a, 0x2a, 0xff},
		{0x29, 0x36, 0x2c, 0xff},
		{0x35, 0x33, 0x24, 0xff},
	}

	for i, rec := range members {
		col := i % cols
		row := i / cols
		x := float64(pad + col*cell)
		y := float64(pad + headerH + row*cell)

		bg := palette[i%len(palette)]
		dc.SetRGBA255(int(bg.R), int(bg.G), int(bg.B), 255)
		dc.DrawRoundedRectangle(x+6, y+6, float64(cell)-12, float64(cell)-12, 16)
		dc.Fill()

		dc.SetRGBA255(255, 255, 255, 30)
		dc.SetLineWidth(2)
		dc.DrawRoundedRectangle(x+6, y+6, float64(cell)-12, float64(cell)-12, 16)
		dc.Stroke()

		emoji := moodEmoji(rec.Current.Mood)
		moodLoadFont(dc, 72)
		dc.SetRGBA255(255, 255, 255, 255)
		dc.DrawStringAnchored(emoji, x+float64(cell)/2, y+float64(cell)/2-18, 0.5, 0.5)

		moodLoadFont(dc, 20)
		dc.SetRGBA255(255, 255, 255, 235)
		dc.DrawStringAnchored(strings.ToUpper(rec.Current.Mood), x+float64(cell)/2, y+float64(cell)/2+38, 0.5, 0.5)

		moodLoadFont(dc, 14)
		dc.SetRGBA255(200, 200, 215, 220)
		name := rec.Name
		if name == "" {
			name = strconv.FormatInt(rec.UserID, 10)
		}
		dc.DrawStringAnchored(moodTruncate(name, 18), x+float64(cell)/2, y+float64(cell)-26, 0.5, 0.5)
	}

	out := filepath.Join(os.TempDir(), fmt.Sprintf("moodboard_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		msg := "<b>Failed to render moodboard.</b>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  fmt.Sprintf("<b>Moodboard</b> — %d members", len(members)),
		FileName: "moodboard.png",
		MimeType: "image/png",
	})
	os.Remove(out)
	if merr != nil {
		m.Reply("<b>Upload failed:</b> " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerMoodHandlers() {
	c := Client
	c.On("cmd:mood", MoodHandler)
	c.On("cmd:moods", MoodsHandler)
	c.On("cmd:moodboard", MoodBoardHandler)

	Mods.AddModule("Mood", `<b>Mood Module</b>

<b>Commands:</b>
 • /mood &lt;happy|sad|angry|tired|excited|bored&gt; - Set your current mood
 • /mood - Show your current saved mood
 • /moods - Show recent moods of chat members (last 5 each)
 • /moodboard - Render an image grid of all tracked members' current moods

<i>Moods are stored per user and visible across the chats they have spoken in.</i>`)
}

func init() {
	QueueHandlerRegistration(registerMoodHandlers)
}
