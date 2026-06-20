package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"sort"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
)

type respEntry struct {
	Keyword  string `json:"keyword"`
	Response string `json:"response"`
}

func respBucketName(userID int64) []byte {
	return []byte(fmt.Sprintf("resp_%d", userID))
}

func RespHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Autoresponder works in groups only.</b>")
		return nil
	}

	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b>\n" +
			" • <code>/resp set &lt;keyword&gt; &lt;response&gt;</code>\n" +
			" • <code>/resp list</code>\n" +
			" • <code>/resp del &lt;keyword&gt;</code>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	sub := strings.ToLower(strings.TrimSpace(parts[0]))
	rest := ""
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}

	switch sub {
	case "set", "add":
		return respSet(m, rest)
	case "list", "ls":
		return respList(m)
	case "del", "delete", "rm", "remove":
		return respDel(m, rest)
	default:
		m.Reply("<b>Unknown subcommand.</b> Use <code>set</code>, <code>list</code>, or <code>del</code>.")
		return nil
	}
}

func respSet(m *tg.NewMessage, rest string) error {
	if rest == "" {
		m.Reply("<b>Usage:</b> <code>/resp set &lt;keyword&gt; &lt;response&gt;</code>")
		return nil
	}

	kv := strings.SplitN(rest, " ", 2)
	if len(kv) < 2 {
		m.Reply("<b>Provide both keyword and response.</b>")
		return nil
	}

	keyword := strings.ToLower(strings.TrimSpace(kv[0]))
	response := strings.TrimSpace(kv[1])
	if keyword == "" || response == "" {
		m.Reply("<b>Keyword and response cannot be empty.</b>")
		return nil
	}

	if len(keyword) < 2 {
		m.Reply("<b>Keyword must be at least 2 characters.</b>")
		return nil
	}

	database, err := db.GetDB()
	if err != nil || database == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	bucketName := respBucketName(m.SenderID())
	err = database.Update(func(tx *bbolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(bucketName)
		if e != nil {
			return e
		}
		entry := respEntry{Keyword: keyword, Response: response}
		data, e := json.Marshal(entry)
		if e != nil {
			return e
		}
		return b.Put([]byte(keyword), data)
	})

	if err != nil {
		m.Reply("<b>Failed to save autoresponder.</b>")
		return nil
	}

	m.Reply(fmt.Sprintf("<b>Saved:</b> <code>%s</code> → <i>%s</i>",
		html.EscapeString(keyword), html.EscapeString(response)))
	return nil
}

func respList(m *tg.NewMessage) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	var entries []respEntry
	bucketName := respBucketName(m.SenderID())
	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var e respEntry
			if err := json.Unmarshal(v, &e); err == nil {
				entries = append(entries, e)
			}
			return nil
		})
	})

	if len(entries) == 0 {
		m.Reply("<b>No autoresponders set.</b> Use <code>/resp set &lt;keyword&gt; &lt;response&gt;</code>")
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Keyword < entries[j].Keyword
	})

	var sb strings.Builder
	sb.WriteString("<b>Your Autoresponders</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━\n\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf(" • <code>%s</code> → %s\n",
			html.EscapeString(e.Keyword), html.EscapeString(e.Response)))
	}
	sb.WriteString(fmt.Sprintf("\n━━━━━━━━━━━━━━━━\n<b>Total:</b> %d", len(entries)))

	m.Reply(sb.String())
	return nil
}

func respDel(m *tg.NewMessage, rest string) error {
	keyword := strings.ToLower(strings.TrimSpace(rest))
	if keyword == "" {
		m.Reply("<b>Usage:</b> <code>/resp del &lt;keyword&gt;</code>")
		return nil
	}

	database, err := db.GetDB()
	if err != nil || database == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	bucketName := respBucketName(m.SenderID())
	found := false
	_ = database.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil
		}
		if b.Get([]byte(keyword)) != nil {
			found = true
			return b.Delete([]byte(keyword))
		}
		return nil
	})

	if !found {
		m.Reply(fmt.Sprintf("<b>Not found:</b> <code>%s</code>", html.EscapeString(keyword)))
		return nil
	}

	m.Reply(fmt.Sprintf("<b>Deleted:</b> <code>%s</code>", html.EscapeString(keyword)))
	return nil
}

func RespWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() || m.Text() == "" || m.IsCommand() {
		return nil
	}

	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil
	}

	chat, err := m.GetChat()
	if err != nil || chat == nil {
		return nil
	}

	participants, _, err := m.Client.GetChatMembers(chat)
	if err != nil || len(participants) == 0 {
		return nil
	}

	msgLower := strings.ToLower(m.Text())
	senderID := m.SenderID()

	for _, p := range participants {
		user := p.User
		if user == nil {
			continue
		}
		if user.ID == senderID {
			continue
		}
		if user.Bot {
			continue
		}

		bucketName := respBucketName(user.ID)
		var match *respEntry
		_ = database.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket(bucketName)
			if b == nil {
				return nil
			}
			return b.ForEach(func(k, v []byte) error {
				kw := string(k)
				if strings.Contains(msgLower, kw) {
					var e respEntry
					if err := json.Unmarshal(v, &e); err == nil {
						match = &e
						return fmt.Errorf("stop")
					}
				}
				return nil
			})
		})

		if match != nil {
			name := user.FirstName
			if name == "" {
				name = "user"
			}
			mention := fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", user.ID, html.EscapeString(name))
			m.Reply(fmt.Sprintf("%s: %s", mention, html.EscapeString(match.Response)))
			return nil
		}
	}

	return nil
}

func registerAutoresponderHandlers() {
	c := Client
	c.On("cmd:resp", RespHandler)
	c.On("cmd:autoresp", RespHandler)
	c.On(tg.OnNewMessage, RespWatcher)
}

func init() {
	QueueHandlerRegistration(registerAutoresponderHandlers)

	Mods.AddModule("Autoresponder", `<b>Autoresponder Module</b>

Set personal keyword triggers. When someone mentions your keyword in a group while you are a member, the bot replies with your configured response and mentions you.

<b>Commands:</b>
 • <code>/resp set &lt;keyword&gt; &lt;response&gt;</code> - Set a keyword response
 • <code>/resp list</code> - List your autoresponders
 • <code>/resp del &lt;keyword&gt;</code> - Delete a keyword

<b>Notes:</b>
 • Works only in groups.
 • Per-user storage; each user manages their own triggers.
 • Triggers on substring match (case-insensitive).`)
}
