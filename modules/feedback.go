package modules

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"sort"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
)

type feedbackEntry struct {
	ID      uint64 `json:"id"`
	UserID  int64  `json:"user_id"`
	ChatID  int64  `json:"chat_id"`
	Name    string `json:"name"`
	Text    string `json:"text"`
	TS      int64  `json:"ts"`
	Replied bool   `json:"replied"`
}

var feedbackBucket = []byte("feedback")

func humanTime(ts int64) string {
	if ts == 0 {
		return "unknown"
	}
	diff := time.Now().Unix() - ts
	if diff < 0 {
		diff = 0
	}
	switch {
	case diff < 60:
		return fmt.Sprintf("%ds ago", diff)
	case diff < 3600:
		return fmt.Sprintf("%dm ago", diff/60)
	case diff < 86400:
		return fmt.Sprintf("%dh ago", diff/3600)
	case diff < 604800:
		return fmt.Sprintf("%dd ago", diff/86400)
	default:
		return time.Unix(ts, 0).Format("Jan 2, 2006")
	}
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func btoi(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

func FeedbackHandler(m *tg.NewMessage) error {
	text := strings.TrimSpace(m.Args())
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/feedback &lt;your message&gt;</code>")
		return nil
	}

	if len(text) < 5 {
		m.Reply("<b>Feedback too short.</b> Please provide more detail.")
		return nil
	}

	if len(text) > 2000 {
		m.Reply("<b>Feedback too long.</b> Max 2000 characters.")
		return nil
	}

	database, err := db.GetDB()
	if err != nil || database == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	name := m.Sender.FirstName
	if m.Sender.LastName != "" {
		name = name + " " + m.Sender.LastName
	}
	if name == "" {
		name = "User"
	}

	var saved feedbackEntry
	err = database.Update(func(tx *bbolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(feedbackBucket)
		if e != nil {
			return e
		}
		id, e := b.NextSequence()
		if e != nil {
			return e
		}
		saved = feedbackEntry{
			ID:      id,
			UserID:  m.SenderID(),
			ChatID:  m.ChatID(),
			Name:    name,
			Text:    text,
			TS:      time.Now().Unix(),
			Replied: false,
		}
		data, e := json.Marshal(saved)
		if e != nil {
			return e
		}
		return b.Put(itob(id), data)
	})

	if err != nil {
		m.Reply("<b>Failed to save feedback.</b>")
		return nil
	}

	if OwnerId != 0 {
		var uname string
		if m.Sender.Username != "" {
			uname = "@" + m.Sender.Username
		} else {
			uname = fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", saved.UserID, html.EscapeString(saved.Name))
		}
		ownerMsg := fmt.Sprintf("<b>New Feedback #%d</b>\n\n<b>From:</b> %s\n<b>User ID:</b> <code>%d</code>\n<b>Chat ID:</b> <code>%d</code>\n\n<b>Message:</b>\n%s\n\n<i>Reply with</i> <code>/reply %d &lt;message&gt;</code>",
			saved.ID, uname, saved.UserID, saved.ChatID, html.EscapeString(saved.Text), saved.ID)
		_, _ = m.Client.SendMessage(OwnerId, ownerMsg)
	}

	m.Reply("Thanks.")
	return nil
}

func MyFeedbackHandler(m *tg.NewMessage) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	userID := m.SenderID()
	var entries []feedbackEntry

	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(feedbackBucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var e feedbackEntry
			if err := json.Unmarshal(v, &e); err == nil {
				if e.UserID == userID {
					entries = append(entries, e)
				}
			}
			return nil
		})
	})

	if len(entries) == 0 {
		m.Reply("<b>No feedback sent yet.</b> Use <code>/feedback &lt;message&gt;</code> to send one.")
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].TS > entries[j].TS
	})

	if len(entries) > 10 {
		entries = entries[:10]
	}

	var sb strings.Builder
	sb.WriteString("<b>Your Recent Feedback</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━\n\n")

	for _, e := range entries {
		status := "Pending"
		if e.Replied {
			status = "Replied"
		}
		preview := e.Text
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("<b>#%d</b> • %s • <i>%s</i>\n%s\n\n",
			e.ID, status, humanTime(e.TS), html.EscapeString(preview)))
	}

	sb.WriteString(fmt.Sprintf("━━━━━━━━━━━━━━━━\n<b>Showing:</b> %d", len(entries)))
	m.Reply(sb.String())
	return nil
}

func AllFeedbackHandler(m *tg.NewMessage) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	var entries []feedbackEntry
	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(feedbackBucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var e feedbackEntry
			if err := json.Unmarshal(v, &e); err == nil {
				entries = append(entries, e)
			}
			return nil
		})
	})

	if len(entries) == 0 {
		m.Reply("<b>No feedback received yet.</b>")
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].TS > entries[j].TS
	})

	total := len(entries)
	if len(entries) > 50 {
		entries = entries[:50]
	}

	var sb strings.Builder
	sb.WriteString("<b>All Feedback</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━\n\n")

	pendingCount := 0
	for _, e := range entries {
		if !e.Replied {
			pendingCount++
		}
		status := "Pending"
		if e.Replied {
			status = "Replied"
		}
		preview := e.Text
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		sb.WriteString(fmt.Sprintf("<b>#%d</b> • %s • <i>%s</i>\n<b>From:</b> %s (<code>%d</code>)\n%s\n\n",
			e.ID, status, humanTime(e.TS),
			html.EscapeString(e.Name), e.UserID,
			html.EscapeString(preview)))
	}

	sb.WriteString(fmt.Sprintf("━━━━━━━━━━━━━━━━\n<b>Showing:</b> %d / %d • <b>Pending:</b> %d",
		len(entries), total, pendingCount))
	m.Reply(sb.String())
	return nil
}

func ReplyFeedbackHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/reply &lt;id&gt; &lt;message&gt;</code>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		m.Reply("<b>Usage:</b> <code>/reply &lt;id&gt; &lt;message&gt;</code>")
		return nil
	}

	idStr := strings.TrimSpace(parts[0])
	replyMsg := strings.TrimSpace(parts[1])

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		m.Reply("<b>Invalid ID.</b> Must be a number.")
		return nil
	}

	if replyMsg == "" {
		m.Reply("<b>Reply message cannot be empty.</b>")
		return nil
	}

	database, err := db.GetDB()
	if err != nil || database == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	var entry feedbackEntry
	found := false

	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(feedbackBucket)
		if b == nil {
			return nil
		}
		data := b.Get(itob(id))
		if data == nil {
			return nil
		}
		if err := json.Unmarshal(data, &entry); err == nil {
			found = true
		}
		return nil
	})

	if !found {
		m.Reply(fmt.Sprintf("<b>Feedback not found:</b> <code>#%d</code>", id))
		return nil
	}

	dmText := fmt.Sprintf("<b>Reply to your feedback #%d</b>\n\n<b>Your message:</b>\n<i>%s</i>\n\n<b>Reply:</b>\n%s",
		entry.ID, html.EscapeString(entry.Text), html.EscapeString(replyMsg))

	_, sendErr := m.Client.SendMessage(entry.UserID, dmText)
	if sendErr != nil {
		m.Reply(fmt.Sprintf("<b>Failed to DM user.</b>\n<code>%s</code>", html.EscapeString(sendErr.Error())))
		return nil
	}

	_ = database.Update(func(tx *bbolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(feedbackBucket)
		if e != nil {
			return e
		}
		entry.Replied = true
		data, e := json.Marshal(entry)
		if e != nil {
			return e
		}
		return b.Put(itob(entry.ID), data)
	})

	m.Reply(fmt.Sprintf("<b>Reply sent to</b> %s <b>for feedback</b> <code>#%d</code>",
		html.EscapeString(entry.Name), entry.ID))
	return nil
}

func registerFeedbackHandlers() {
	c := Client
	c.On("cmd:feedback", FeedbackHandler)
	c.On("cmd:myfeedback", MyFeedbackHandler)
	c.On("cmd:allfeedback", AllFeedbackHandler, tg.CustomFilter(FilterOwner))
	c.On("cmd:reply", ReplyFeedbackHandler, tg.CustomFilter(FilterOwner))
}

func init() {
	QueueHandlerRegistration(registerFeedbackHandlers)

	Mods.AddModule("Feedback", `<b>Feedback Module</b>

<b>Commands:</b>
 • /feedback &lt;text&gt; - Send feedback to the bot owner
 • /myfeedback - View your last 10 feedback messages
 • /allfeedback - View last 50 feedback (owner only)
 • /reply &lt;id&gt; &lt;message&gt; - DM the original sender (owner only)

<b>Notes:</b>
Your feedback is delivered directly to the bot owner. The owner can reply to you privately via /reply.`)
}
