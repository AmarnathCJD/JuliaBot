package modules

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
)

const tmpRulesBucket = "tmprules"

type tmpRule struct {
	ID        int64  `json:"id"`
	ChatID    int64  `json:"chat_id"`
	MsgID     int32  `json:"msg_id"`
	Text      string `json:"text"`
	ExpiresAt int64  `json:"expires_at"`
	CreatedBy int64  `json:"created_by"`
}

var (
	tmpRuleTimers   = make(map[int64]*time.Timer)
	tmpRuleTimersMu sync.Mutex
)

func saveTmpRule(r *tmpRule) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db error")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(tmpRulesBucket))
		if err != nil {
			return err
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(r.ID))
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		return b.Put(key, data)
	})
}

func deleteTmpRule(id int64) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db error")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(tmpRulesBucket))
		if b == nil {
			return nil
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(id))
		return b.Delete(key)
	})
}

func getTmpRule(id int64) *tmpRule {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil
	}
	var r *tmpRule
	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(tmpRulesBucket))
		if b == nil {
			return nil
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(id))
		v := b.Get(key)
		if len(v) == 0 {
			return nil
		}
		var tmp tmpRule
		if err := json.Unmarshal(v, &tmp); err != nil {
			return nil
		}
		r = &tmp
		return nil
	})
	return r
}

func listTmpRulesForChat(chatID int64) []*tmpRule {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil
	}
	var out []*tmpRule
	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(tmpRulesBucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var tmp tmpRule
			if err := json.Unmarshal(v, &tmp); err != nil {
				return nil
			}
			if tmp.ChatID == chatID {
				out = append(out, &tmp)
			}
			return nil
		})
	})
	return out
}

func listAllTmpRules() []*tmpRule {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil
	}
	var out []*tmpRule
	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(tmpRulesBucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var tmp tmpRule
			if err := json.Unmarshal(v, &tmp); err != nil {
				return nil
			}
			out = append(out, &tmp)
			return nil
		})
	})
	return out
}

func scheduleTmpRule(r *tmpRule) {
	delay := time.Until(time.Unix(r.ExpiresAt, 0))
	if delay < 0 {
		delay = 0
	}
	tmpRuleTimersMu.Lock()
	if existing, ok := tmpRuleTimers[r.ID]; ok {
		existing.Stop()
	}
	tmpRuleTimers[r.ID] = time.AfterFunc(delay, func() {
		expireTmpRule(r.ID)
	})
	tmpRuleTimersMu.Unlock()
}

func expireTmpRule(id int64) {
	r := getTmpRule(id)
	if r == nil {
		return
	}
	if Client != nil {
		_, _ = Client.UnpinMessage(r.ChatID, r.MsgID)
		_, _ = Client.DeleteMessages(r.ChatID, []int32{r.MsgID})
	}
	_ = deleteTmpRule(id)
	tmpRuleTimersMu.Lock()
	delete(tmpRuleTimers, id)
	tmpRuleTimersMu.Unlock()
}

func TmpRuleHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>This command only works in groups.</b>")
		return nil
	}
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "pin") {
		m.Reply("<b>Permission denied.</b> You need Pin Messages permission.")
		return nil
	}

	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/tmprule &lt;duration&gt; &lt;text&gt;</code>\n<b>Example:</b> <code>/tmprule 2h No spam allowed</code>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		m.Reply("<b>Usage:</b> <code>/tmprule &lt;duration&gt; &lt;text&gt;</code>")
		return nil
	}

	duration, err := parseDuration(parts[0])
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}

	text := strings.TrimSpace(parts[1])
	if text == "" {
		m.Reply("<b>Error:</b> rule text required.")
		return nil
	}

	body := fmt.Sprintf("<b>Temporary Rule</b>\n\n%s\n\n<i>Expires in %s</i>", html.EscapeString(text), formatDuration(duration))
	sent, err := m.Reply(body)
	if err != nil || sent == nil {
		m.Reply("<b>Failed to post rule.</b>")
		return nil
	}

	if CanBot(m.Client, m.Channel, "pin") {
		_, perr := m.Client.PinMessage(m.ChatID(), sent.ID, &tg.PinOptions{Silent: true})
		if perr != nil {
			m.Reply("<b>Rule posted.</b> <i>Could not pin: missing permission.</i>")
		}
	} else {
		m.Reply("<b>Rule posted.</b> <i>I need pin permission to pin it.</i>")
	}

	id := time.Now().UnixNano()
	r := &tmpRule{
		ID:        id,
		ChatID:    m.ChatID(),
		MsgID:     sent.ID,
		Text:      text,
		ExpiresAt: time.Now().Add(duration).Unix(),
		CreatedBy: m.SenderID(),
	}

	if err := saveTmpRule(r); err != nil {
		m.Reply("<b>Warning:</b> rule saved in memory only.")
	}
	scheduleTmpRule(r)
	return nil
}

func TmpRulesListHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>This command only works in groups.</b>")
		return nil
	}

	rules := listTmpRulesForChat(m.ChatID())
	if len(rules) == 0 {
		m.Reply("<b>No active temporary rules in this chat.</b>")
		return nil
	}

	var b strings.Builder
	b.WriteString("<b>Active Temporary Rules:</b>\n\n")
	for _, r := range rules {
		remaining := time.Until(time.Unix(r.ExpiresAt, 0))
		if remaining < 0 {
			remaining = 0
		}
		preview := r.Text
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		b.WriteString(fmt.Sprintf("<b>ID:</b> <code>%d</code>\n%s\n<i>Expires in %s</i>\n\n", r.ID, html.EscapeString(preview), formatDuration(remaining)))
	}

	m.Reply(b.String())
	return nil
}

func RmTmpRuleHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>This command only works in groups.</b>")
		return nil
	}
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "pin") {
		m.Reply("<b>Permission denied.</b> You need Pin Messages permission.")
		return nil
	}

	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/rmtmprule &lt;id&gt;</code>")
		return nil
	}

	id, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		m.Reply("<b>Error:</b> invalid id.")
		return nil
	}

	r := getTmpRule(id)
	if r == nil || r.ChatID != m.ChatID() {
		m.Reply("<b>No such temporary rule in this chat.</b>")
		return nil
	}

	tmpRuleTimersMu.Lock()
	if t, ok := tmpRuleTimers[id]; ok {
		t.Stop()
		delete(tmpRuleTimers, id)
	}
	tmpRuleTimersMu.Unlock()

	if CanBot(m.Client, m.Channel, "pin") {
		_, _ = m.Client.UnpinMessage(r.ChatID, r.MsgID)
	}
	_, _ = m.Client.DeleteMessages(r.ChatID, []int32{r.MsgID})
	_ = deleteTmpRule(id)

	m.Reply("<b>Temporary rule removed.</b>")
	return nil
}

func rescheduleTmpRules() {
	rules := listAllTmpRules()
	now := time.Now().Unix()
	for _, r := range rules {
		if r.ExpiresAt <= now {
			go expireTmpRule(r.ID)
			continue
		}
		scheduleTmpRule(r)
	}
}

func registerTmpRulesHandlers() {
	c := Client
	c.On("cmd:tmprule", TmpRuleHandler)
	c.On("cmd:tmprules", TmpRulesListHandler)
	c.On("cmd:rmtmprule", RmTmpRuleHandler)

	go rescheduleTmpRules()

	Mods.AddModule("TempRules", `<b>Temporary Rules Module</b>

Pin a rule for a limited time. After expiry it is auto-unpinned and deleted.

<b>Commands:</b>
 - /tmprule &lt;duration&gt; &lt;text&gt; - Post a temporary rule (admin)
 - /tmprules - List active temporary rules
 - /rmtmprule &lt;id&gt; - Remove a temporary rule early (admin)

<b>Duration format:</b> 30s, 5m, 2h, 1d, 1w`)
}

func init() {
	QueueHandlerRegistration(registerTmpRulesHandlers)
}
