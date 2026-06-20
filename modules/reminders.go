package modules

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"main/modules/db"

	"go.etcd.io/bbolt"
)

type reminderEntry struct {
	ID         uint64 `json:"id"`
	UserID     int64  `json:"user_id"`
	ChatID     int64  `json:"chat_id"`
	Msg        string `json:"msg"`
	FireAtUnix int64  `json:"fire_at_unix"`
}

const remindersBucket = "reminders"

var reminderTimers sync.Map

func parseReminderDuration(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	var total time.Duration
	var numBuf strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			numBuf.WriteRune(r)
		case r == 'd':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("missing number before 'd'")
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			total += time.Duration(n) * 24 * time.Hour
		case r == 'h':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("missing number before 'h'")
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			total += time.Duration(n) * time.Hour
		case r == 'm':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("missing number before 'm'")
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			total += time.Duration(n) * time.Minute
		case r == 's':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("missing number before 's'")
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			total += time.Duration(n) * time.Second
		default:
			return 0, fmt.Errorf("invalid character '%c'", r)
		}
	}
	if numBuf.Len() > 0 {
		n, _ := strconv.Atoi(numBuf.String())
		total += time.Duration(n) * time.Second
	}
	if total <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}
	return total, nil
}

func formatReminderDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	var parts []string
	if days := d / (24 * time.Hour); days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
		d -= days * 24 * time.Hour
	}
	if hours := d / time.Hour; hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
		d -= hours * time.Hour
	}
	if mins := d / time.Minute; mins > 0 {
		parts = append(parts, fmt.Sprintf("%dm", mins))
		d -= mins * time.Minute
	}
	if secs := d / time.Second; secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}
	return strings.Join(parts, " ")
}

func reminderKey(id uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, id)
	return b
}

func saveReminder(entry *reminderEntry) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db error")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(remindersBucket))
		if err != nil {
			return err
		}
		if entry.ID == 0 {
			id, _ := bkt.NextSequence()
			entry.ID = id
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return bkt.Put(reminderKey(entry.ID), data)
	})
}

func deleteReminder(id uint64) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db error")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(remindersBucket))
		if bkt == nil {
			return nil
		}
		return bkt.Delete(reminderKey(id))
	})
}

func listReminders(userID int64) ([]*reminderEntry, error) {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil, fmt.Errorf("db error")
	}
	var out []*reminderEntry
	err = database.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(remindersBucket))
		if bkt == nil {
			return nil
		}
		return bkt.ForEach(func(k, v []byte) error {
			var entry reminderEntry
			if err := json.Unmarshal(v, &entry); err != nil {
				return nil
			}
			if userID == 0 || entry.UserID == userID {
				out = append(out, &entry)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].FireAtUnix < out[j].FireAtUnix
	})
	return out, nil
}

func loadAllReminders() ([]*reminderEntry, error) {
	return listReminders(0)
}

func fireReminder(entry *reminderEntry) {
	reminderTimers.Delete(entry.ID)
	_ = deleteReminder(entry.ID)
	if Client == nil {
		return
	}
	mention := fmt.Sprintf("<a href=\"tg://user?id=%d\">user</a>", entry.UserID)
	text := fmt.Sprintf("%s reminder: %s", mention, html.EscapeString(entry.Msg))
	Client.SendMessage(entry.ChatID, text, &tg.SendOptions{})
}

func scheduleReminder(entry *reminderEntry) {
	now := time.Now().Unix()
	delay := time.Duration(entry.FireAtUnix-now) * time.Second
	if delay <= 0 {
		go fireReminder(entry)
		return
	}
	t := time.AfterFunc(delay, func() {
		fireReminder(entry)
	})
	reminderTimers.Store(entry.ID, t)
}

func RemindHandler(m *tg.NewMessage) error {
	args := m.Args()
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/remind &lt;duration&gt; &lt;message&gt;</code>\n<b>Example:</b> <code>/remind 10m take a break</code>\n<i>Duration units: s, m, h, d</i>")
		return nil
	}
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		m.Reply("<b>Usage:</b> <code>/remind &lt;duration&gt; &lt;message&gt;</code>")
		return nil
	}
	duration, err := parseReminderDuration(parts[0])
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	msg := strings.TrimSpace(parts[1])
	if msg == "" {
		m.Reply("<b>Error:</b> reminder message cannot be empty")
		return nil
	}
	entry := &reminderEntry{
		UserID:     m.SenderID(),
		ChatID:     m.ChatID(),
		Msg:        msg,
		FireAtUnix: time.Now().Add(duration).Unix(),
	}
	if err := saveReminder(entry); err != nil {
		m.Reply("<b>Error saving reminder:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	scheduleReminder(entry)
	m.Reply(fmt.Sprintf("Reminder #<b>%d</b> set for <b>%s</b>", entry.ID, formatReminderDuration(duration)))
	return nil
}

func RemindersHandler(m *tg.NewMessage) error {
	entries, err := listReminders(m.SenderID())
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	if len(entries) == 0 {
		m.Reply("You have no pending reminders.")
		return nil
	}
	var sb strings.Builder
	sb.WriteString("<b>Your pending reminders:</b>\n\n")
	now := time.Now().Unix()
	for _, e := range entries {
		remaining := time.Duration(e.FireAtUnix-now) * time.Second
		sb.WriteString(fmt.Sprintf("• <b>#%d</b> in <b>%s</b> — %s\n", e.ID, formatReminderDuration(remaining), html.EscapeString(e.Msg)))
	}
	sb.WriteString("\n<i>Use</i> <code>/delremind &lt;id&gt;</code> <i>to cancel.</i>")
	m.Reply(sb.String())
	return nil
}

func DelRemindHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/delremind &lt;id&gt;</code>")
		return nil
	}
	id, err := strconv.ParseUint(args, 10, 64)
	if err != nil {
		m.Reply("<b>Error:</b> invalid id")
		return nil
	}
	entries, err := listReminders(m.SenderID())
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	var owned *reminderEntry
	for _, e := range entries {
		if e.ID == id {
			owned = e
			break
		}
	}
	if owned == nil {
		m.Reply("<b>Error:</b> reminder not found or not yours")
		return nil
	}
	if v, ok := reminderTimers.LoadAndDelete(id); ok {
		if t, ok := v.(*time.Timer); ok {
			t.Stop()
		}
	}
	if err := deleteReminder(id); err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	m.Reply(fmt.Sprintf("Reminder #<b>%d</b> cancelled.", id))
	return nil
}

func bootstrapReminders() {
	entries, err := loadAllReminders()
	if err != nil {
		return
	}
	for _, e := range entries {
		scheduleReminder(e)
	}
}

func registerRemindersHandlers() {
	c := Client
	c.On("cmd:remind", RemindHandler)
	c.On("cmd:reminders", RemindersHandler)
	c.On("cmd:delremind", DelRemindHandler)
	go bootstrapReminders()
}

func init() {
	QueueHandlerRegistration(registerRemindersHandlers)
}
