package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
)

type AFK struct {
	Name    string
	Message string
	Media   string
	Time    int64
}

var afkList = make(map[int64]AFK)
var afkMu sync.RWMutex

var randomAFKMessages = []string{
	"<b>%s</b> is AFK since <b>%s</b>.",
	"<b>%s</b> is AFK for <b>%s</b>.",
	"Mr. <b>%s</b> is AFK for <b>%s</b>.",
	"<b>%s</b> has been AFK since <b>%s</b>.",
	"<b>%s</b> stepped away and is AFK for <b>%s</b>.",
	"<b>%s</b> is currently AFK for <b>%s</b>.",
}

func AFKHandler(m *tg.NewMessage) error {
	if m.Sender == nil {
		return nil
	}
	text := m.Text()
	cmd := ""
	if fields := strings.Fields(text); len(fields) > 0 {
		cmd = fields[0]
	}
	isAfkCmd := cmd == "/afk" || cmd == "!afk" || cmd == ".afk" ||
		strings.HasPrefix(cmd, "/afk@") || strings.HasPrefix(cmd, "!afk@") || strings.HasPrefix(cmd, ".afk@")
	if isAfkCmd {
		media := ""
		if m.IsReply() {
			r, err := m.GetReplyMessage()
			if err == nil {
				if r.IsMedia() && r.File != nil {
					media = r.File.FileID
				}
			}
		}
		afkMu.Lock()
		afkList[m.Sender.ID] = AFK{
			Name:    m.Sender.Username,
			Message: m.Args(),
			Media:   media,
			Time:    time.Now().Unix(),
		}
		afkMu.Unlock()

		m.Reply("You are now AFK.")
		return nil
	} else {
		afkMu.RLock()
		afk, ok := afkList[m.SenderID()]
		afkMu.RUnlock()
		if ok {
			afkMu.Lock()
			delete(afkList, m.SenderID())
			afkMu.Unlock()
			duration := time.Since(time.Unix(afk.Time, 0)).String()
			m.Reply(fmt.Sprintf("Welcome back <b>%s</b>! You were AFK for %s.", afk.Name, duration))
		} else {
			if m.IsReply() {
				r, err := m.GetReplyMessage()
				if err == nil {
					afkMu.RLock()
					afk, ok := afkList[r.SenderID()]
					afkMu.RUnlock()
					if ok {
						duration := time.Since(time.Unix(afk.Time, 0)).String()
						msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
						if afk.Media != "" {
							var msg = fmt.Sprintf(msg, afk.Name, duration)
							if afk.Message != "" {
								msg += "\nReason: " + afk.Message
							}
							media, _ := tg.ResolveBotFileID(afk.Media)
							if IsSticker(media) {
								m.ReplyMedia(media)
								m.Respond(msg)
							} else {
								m.ReplyMedia(media, &tg.MediaOptions{
									Caption: msg,
								})
							}
						} else {
							var msg = fmt.Sprintf(msg, afk.Name, duration)
							if afk.Message != "" {
								msg += "\nReason: " + afk.Message
							}

							m.Reply(msg)
						}
					}
				}
			} else {
				if len(m.Message.Entities) > 0 {
					for _, entity := range m.Message.Entities {
						switch e := entity.(type) {
						case *tg.MessageEntityMentionName:
							afkMu.RLock()
							afk, ok := afkList[e.UserID]
							afkMu.RUnlock()
							if ok {
								duration := time.Since(time.Unix(afk.Time, 0)).String()
								msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
								if afk.Media != "" {
									var msg = fmt.Sprintf(msg, afk.Name, duration)
									if afk.Message != "" {
										msg += "\nReason: " + afk.Message
									}
									media, _ := tg.ResolveBotFileID(afk.Media)
									if IsSticker(media) {
										m.ReplyMedia(media)
										m.Respond(msg)
									} else {
										m.ReplyMedia(media, &tg.MediaOptions{
											Caption: msg,
										})
									}
								} else {
									var msg = fmt.Sprintf(msg, afk.Name, duration)
									if afk.Message != "" {
										msg += "\nReason: " + afk.Message
									}

									m.Reply(msg)
								}
							}
						case *tg.MessageEntityMention:
							offset := e.Offset
							length := e.Length

							username := m.Text()[offset : offset+length]
							afkMu.RLock()
							afkSnap := make(map[int64]AFK, len(afkList))
							for k, v := range afkList {
								afkSnap[k] = v
							}
							afkMu.RUnlock()
							for _, afk := range afkSnap {
								if afk.Name == username {
									duration := time.Since(time.Unix(afk.Time, 0)).String()
									msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
									if afk.Media != "" {
										var msg = fmt.Sprintf(msg, afk.Name, trimDecimal(duration))
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}
										media, _ := tg.ResolveBotFileID(afk.Media)
										if IsSticker(media) {
											m.ReplyMedia(media)
											m.Respond(msg)
										} else {
											m.ReplyMedia(media, &tg.MediaOptions{
												Caption: msg,
											})
										}
									} else {
										var msg = fmt.Sprintf(msg, afk.Name, trimDecimal(duration))
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}

										m.Reply(msg)
									}
								}
							}

							user, err := m.Client.ResolvePeer(username)
							if err == nil {
								peerId := m.Client.GetPeerID(user)
								afkMu.RLock()
								afk, ok := afkList[peerId]
								afkMu.RUnlock()
								if ok {
									duration := time.Since(time.Unix(afk.Time, 0)).String()
									msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
									if afk.Media != "" {
										var msg = fmt.Sprintf(msg, afk.Name, trimDecimal(duration))
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}
										media, _ := tg.ResolveBotFileID(afk.Media)
										if IsSticker(media) {
											m.ReplyMedia(media)
											m.Respond(msg)
										} else {
											m.ReplyMedia(media, &tg.MediaOptions{
												Caption: msg,
											})
										}
									} else {
										var msg = fmt.Sprintf(msg, afk.Name, trimDecimal(duration))
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}

										m.Reply(msg)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func trimDecimal(s string) string {
	if strings.Contains(s, ".") {
		return strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

func IsSticker(m tg.MessageMedia) bool {
	switch m := m.(type) {
	case *tg.MessageMediaDocument:
		attrs := m.Document.(*tg.DocumentObj).Attributes
		for _, attr := range attrs {
			if _, ok := attr.(*tg.DocumentAttributeSticker); ok {
				return true
			}
		}
	}

	return false
}

func parseSed(text string) (find, replace, flags string, ok bool) {
	if len(text) < 4 || text[0] != 's' {
		return "", "", "", false
	}
	d := text[1]
	if d != '/' && d != '\\' && d != '|' && d != '#' {
		return "", "", "", false
	}
	rest := text[2:]

	take := func(s string) (segment, remainder string, found bool) {
		var b strings.Builder
		for i := 0; i < len(s); i++ {
			if s[i] == '\\' && i+1 < len(s) && s[i+1] == d {
				b.WriteByte(d)
				i++
				continue
			}
			if s[i] == d {
				return b.String(), s[i+1:], true
			}
			b.WriteByte(s[i])
		}
		return "", s, false
	}

	var found1, found2 bool
	find, rest, found1 = take(rest)
	if !found1 {
		return "", "", "", false
	}
	replace, rest, found2 = take(rest)
	if !found2 {
		replace = rest
		rest = ""
	}

	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case 'g', 'i':
			flags += string(rest[i])
		case ' ', '\t':
			i = len(rest)
		}
	}

	if find == "" {
		return "", "", "", false
	}
	return find, replace, flags, true
}

func expandSedBackrefs(repl string) string {
	var b strings.Builder
	for i := 0; i < len(repl); i++ {
		if repl[i] == '\\' && i+1 < len(repl) {
			c := repl[i+1]
			if c == '&' || c == '0' {
				b.WriteString("${0}")
				i++
				continue
			}
			if c >= '1' && c <= '9' {
				b.WriteString("${")
				b.WriteByte(c)
				b.WriteString("}")
				i++
				continue
			}
			if c == '\\' {
				b.WriteString(`\\`)
				i++
				continue
			}
		}
		if repl[i] == '&' {
			b.WriteString("${0}")
			continue
		}
		if repl[i] == '$' {
			b.WriteString(`$$`)
			continue
		}
		b.WriteByte(repl[i])
	}
	return b.String()
}

const sedMaxOutputLen = 4096

func SedHandler(m *tg.NewMessage) error {
	find, replace, flags, ok := parseSed(m.Text())
	if !ok {
		return nil
	}
	if !m.IsReply() {
		return nil
	}
	replyMsg, err := m.GetReplyMessage()
	if err != nil {
		return nil
	}
	if me := m.Client.Me(); me != nil && replyMsg.SenderID() == me.ID {
		return nil
	}
	originalText := replyMsg.Text()
	if originalText == "" {
		return nil
	}

	global := strings.ContainsRune(flags, 'g')
	insensitive := strings.ContainsRune(flags, 'i')

	pattern := find
	if insensitive {
		pattern = "(?i)" + pattern
	}

	var newText string
	if re, rerr := regexp.Compile(pattern); rerr == nil && re.MatchString(originalText) {
		goRepl := expandSedBackrefs(replace)
		if global {
			newText = re.ReplaceAllString(originalText, goRepl)
		} else {
			loc := re.FindStringIndex(originalText)
			if loc == nil {
				return nil
			}
			newText = originalText[:loc[0]] +
				re.ReplaceAllString(originalText[loc[0]:loc[1]], goRepl) +
				originalText[loc[1]:]
		}
	} else {
		haystack := originalText
		needle := find
		if insensitive {
			if lre, err := regexp.Compile("(?i)" + regexp.QuoteMeta(find)); err == nil {
				if !lre.MatchString(haystack) {
					return nil
				}
				if global {
					newText = lre.ReplaceAllString(haystack, replace)
				} else {
					newText = lre.ReplaceAllStringFunc(haystack, func(s string) string {
						return replace
					})
					if loc := lre.FindStringIndex(haystack); loc != nil {
						newText = haystack[:loc[0]] + replace + haystack[loc[1]:]
					}
				}
			} else {
				return nil
			}
		} else {
			if !strings.Contains(haystack, needle) {
				return nil
			}
			if global {
				newText = strings.ReplaceAll(haystack, needle, replace)
			} else {
				newText = strings.Replace(haystack, needle, replace, 1)
			}
		}
	}

	if newText == originalText {
		return nil
	}
	if len(newText) > sedMaxOutputLen {
		return nil
	}

	replyID := replyMsg.ID
	if gp := replyMsg.ReplyToMsgID(); gp != 0 {
		replyID = gp
	}
	m.Client.SendMessage(m.ChatID(), newText, &tg.SendOptions{ReplyID: replyID})
	return nil
}

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

func registerAFKHandlers() {
	c := Client
	c.On(tg.OnNewMessage, AFKHandler)
	c.On(tg.OnNewMessage, SedHandler)
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
	QueueHandlerRegistration(registerAFKHandlers)
}
