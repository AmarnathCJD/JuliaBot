package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"main/modules/db"

	"go.etcd.io/bbolt"
)

const lastSeenBucket = "last_seen"

type lastSeenEntry struct {
	UserID   int64  `json:"user_id"`
	ChatID   int64  `json:"chat_id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Time     int64  `json:"time"`
}

func humanRelative(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = -d
	}
	secs := int64(d.Seconds())
	if secs < 5 {
		return "just now"
	}
	if secs < 60 {
		return fmt.Sprintf("%d seconds ago", secs)
	}
	mins := secs / 60
	if mins < 60 {
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	hours := mins / 60
	if hours < 24 {
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := hours / 24
	if days < 7 {
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	weeks := days / 7
	if weeks < 4 {
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}
	months := days / 30
	if months < 12 {
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := days / 365
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

func lastSeenKey(chatID, userID int64) []byte {
	return []byte(fmt.Sprintf("%d:%d", chatID, userID))
}

func saveLastSeen(entry *lastSeenEntry) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db error")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(lastSeenBucket))
		if err != nil {
			return err
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return b.Put(lastSeenKey(entry.ChatID, entry.UserID), data)
	})
}

func loadLastSeen(chatID, userID int64) (*lastSeenEntry, error) {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil, fmt.Errorf("db error")
	}
	var entry *lastSeenEntry
	err = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(lastSeenBucket))
		if b == nil {
			return nil
		}
		v := b.Get(lastSeenKey(chatID, userID))
		if v == nil {
			return nil
		}
		var e lastSeenEntry
		if err := json.Unmarshal(v, &e); err != nil {
			return err
		}
		entry = &e
		return nil
	})
	return entry, err
}

func loadAnyLastSeen(userID int64) (*lastSeenEntry, error) {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil, fmt.Errorf("db error")
	}
	var best *lastSeenEntry
	suffix := ":" + strconv.FormatInt(userID, 10)
	err = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(lastSeenBucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			if !strings.HasSuffix(string(k), suffix) {
				return nil
			}
			var e lastSeenEntry
			if err := json.Unmarshal(v, &e); err != nil {
				return nil
			}
			if best == nil || e.Time > best.Time {
				ec := e
				best = &ec
			}
			return nil
		})
	})
	return best, err
}

func LastSeenTracker(m *tg.NewMessage) error {
	if m.SenderID() == 0 {
		return nil
	}
	if m.ChatID() == 0 {
		return nil
	}
	entry := &lastSeenEntry{
		UserID: m.SenderID(),
		ChatID: m.ChatID(),
		Time:   time.Now().Unix(),
	}
	if m.Sender != nil {
		entry.Name = m.Sender.FirstName
		entry.Username = m.Sender.Username
	}
	_ = saveLastSeen(entry)
	return nil
}

func AfkUsersHandler(m *tg.NewMessage) error {
	chatID := m.ChatID()
	if chatID == 0 {
		m.Reply("<b>This command works in chats only.</b>")
		return nil
	}
	if len(afkList) == 0 {
		m.Reply("<b>No one is AFK right now.</b>")
		return nil
	}

	type row struct {
		id       int64
		name     string
		username string
		since    time.Time
		reason   string
	}
	var rows []row
	for uid, a := range afkList {
		name := a.Name
		username := a.Name
		entry, _ := loadLastSeen(chatID, uid)
		if entry != nil {
			if entry.Name != "" {
				name = entry.Name
			}
			if entry.Username != "" {
				username = entry.Username
			}
		}
		if name == "" {
			name = strconv.FormatInt(uid, 10)
		}
		rows = append(rows, row{
			id:       uid,
			name:     name,
			username: username,
			since:    time.Unix(a.Time, 0),
			reason:   a.Message,
		})
	}

	if len(rows) == 0 {
		m.Reply("<b>No one is AFK right now.</b>")
		return nil
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].since.Before(rows[j].since)
	})

	var sb strings.Builder
	sb.WriteString("<b>AFK Users</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━\n")
	for _, r := range rows {
		mention := fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", r.id, html.EscapeString(r.name))
		sb.WriteString(fmt.Sprintf(" • %s — last seen %s", mention, humanRelative(r.since)))
		if strings.TrimSpace(r.reason) != "" {
			sb.WriteString(fmt.Sprintf(" (<i>%s</i>)", html.EscapeString(r.reason)))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("\n<b>Total:</b> %d", len(rows)))
	m.Reply(sb.String())
	return nil
}

func LastSeenHandler(m *tg.NewMessage) error {
	chatID := m.ChatID()
	if chatID == 0 {
		m.Reply("<b>This command works in chats only.</b>")
		return nil
	}

	var targetID int64
	var targetName string

	if m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil && r != nil {
			targetID = r.SenderID()
			if r.Sender != nil {
				targetName = r.Sender.FirstName
			}
		}
	} else if strings.TrimSpace(m.Args()) != "" {
		arg := strings.TrimSpace(strings.Split(m.Args(), " ")[0])
		argInt, convErr := strconv.ParseInt(arg, 10, 64)
		if convErr == nil {
			peer, err := m.Client.ResolvePeer(argInt)
			if err == nil {
				targetID = m.Client.GetPeerID(peer)
			}
		} else {
			peer, err := m.Client.ResolvePeer(arg)
			if err == nil {
				targetID = m.Client.GetPeerID(peer)
			}
		}
		if targetID != 0 {
			if u, _ := m.Client.GetUser(targetID); u != nil {
				targetName = u.FirstName
			}
		}
	} else {
		targetID = m.SenderID()
		if m.Sender != nil {
			targetName = m.Sender.FirstName
		}
	}

	if targetID == 0 {
		m.Reply("<b>Could not resolve user.</b>\nReply to a user or use <code>/lastseen @username</code>.")
		return nil
	}

	entry, _ := loadLastSeen(chatID, targetID)
	if entry == nil {
		fallback, _ := loadAnyLastSeen(targetID)
		if fallback == nil {
			name := targetName
			if name == "" {
				name = strconv.FormatInt(targetID, 10)
			}
			m.Reply(fmt.Sprintf("<b>%s</b> has not been seen in any tracked chat.", html.EscapeString(name)))
			return nil
		}
		name := fallback.Name
		if targetName != "" {
			name = targetName
		}
		if name == "" {
			name = strconv.FormatInt(targetID, 10)
		}
		mention := fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", targetID, html.EscapeString(name))
		m.Reply(fmt.Sprintf("%s was last seen %s (in another chat).", mention, humanRelative(time.Unix(fallback.Time, 0))))
		return nil
	}

	name := entry.Name
	if targetName != "" {
		name = targetName
	}
	if name == "" {
		name = strconv.FormatInt(targetID, 10)
	}
	mention := fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", targetID, html.EscapeString(name))

	extra := ""
	if afk, ok := afkList[targetID]; ok {
		extra = fmt.Sprintf("\n<b>Currently AFK</b> since %s.", humanRelative(time.Unix(afk.Time, 0)))
		if strings.TrimSpace(afk.Message) != "" {
			extra += fmt.Sprintf("\n<b>Reason:</b> %s", html.EscapeString(afk.Message))
		}
	}

	m.Reply(fmt.Sprintf("%s was last seen %s.%s", mention, humanRelative(time.Unix(entry.Time, 0)), extra))
	return nil
}

func registerAfkStatusHandlers() {
	c := Client
	c.On(tg.OnNewMessage, LastSeenTracker)
	c.On("cmd:afkusers", AfkUsersHandler)
	c.On("cmd:lastseen", LastSeenHandler)
	c.On("cmd:seen", LastSeenHandler)

	Mods.AddModule("AFKStatus", `<b>AFK Status Module</b>

<b>Commands:</b>
 • /afkusers - List AFK users in this chat with last-seen times
 • /lastseen [reply | @user] - Show when a user was last seen
 • /seen - Alias for /lastseen

<i>Tracks last-seen timestamps per chat automatically.</i>`)
}

func init() {
	QueueHandlerRegistration(registerAfkStatusHandlers)
}
