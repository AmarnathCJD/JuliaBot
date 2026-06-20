package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
)

type autoReactRule struct {
	ID      int    `json:"id"`
	Pattern string `json:"pattern"`
	Emoji   string `json:"emoji"`
	AddedBy int64  `json:"added_by"`
}

type compiledAutoReact struct {
	re    *regexp.Regexp
	emoji string
	id    int
}

var (
	autoReactBucketName = []byte("autoreact")
	autoReactCache      = map[int64][]compiledAutoReact{}
	autoReactCacheMu    sync.RWMutex
)

func ensureAutoReactBucket(database *bbolt.DB) error {
	return database.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(autoReactBucketName)
		return err
	})
}

func autoReactChatBucket(tx *bbolt.Tx, chatID int64) (*bbolt.Bucket, error) {
	root := tx.Bucket(autoReactBucketName)
	if root == nil {
		return nil, nil
	}
	return root.CreateBucketIfNotExists([]byte(strconv.FormatInt(chatID, 10)))
}

func loadAutoReactRules(chatID int64) ([]autoReactRule, error) {
	database, err := db.GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureAutoReactBucket(database); err != nil {
		return nil, err
	}

	var rules []autoReactRule
	err = database.View(func(tx *bbolt.Tx) error {
		root := tx.Bucket(autoReactBucketName)
		if root == nil {
			return nil
		}
		chatBucket := root.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if chatBucket == nil {
			return nil
		}
		return chatBucket.ForEach(func(k, v []byte) error {
			var r autoReactRule
			if err := json.Unmarshal(v, &r); err == nil {
				rules = append(rules, r)
			}
			return nil
		})
	})
	return rules, err
}

func saveAutoReactRule(chatID int64, rule autoReactRule) error {
	database, err := db.GetDB()
	if err != nil {
		return err
	}
	if err := ensureAutoReactBucket(database); err != nil {
		return err
	}

	return database.Update(func(tx *bbolt.Tx) error {
		chatBucket, err := autoReactChatBucket(tx, chatID)
		if err != nil || chatBucket == nil {
			return err
		}
		data, err := json.Marshal(rule)
		if err != nil {
			return err
		}
		return chatBucket.Put([]byte(strconv.Itoa(rule.ID)), data)
	})
}

func deleteAutoReactRule(chatID int64, id int) (bool, error) {
	database, err := db.GetDB()
	if err != nil {
		return false, err
	}
	if err := ensureAutoReactBucket(database); err != nil {
		return false, err
	}

	deleted := false
	err = database.Update(func(tx *bbolt.Tx) error {
		root := tx.Bucket(autoReactBucketName)
		if root == nil {
			return nil
		}
		chatBucket := root.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if chatBucket == nil {
			return nil
		}
		key := []byte(strconv.Itoa(id))
		if chatBucket.Get(key) == nil {
			return nil
		}
		deleted = true
		return chatBucket.Delete(key)
	})
	return deleted, err
}

func nextAutoReactID(rules []autoReactRule) int {
	max := 0
	for _, r := range rules {
		if r.ID > max {
			max = r.ID
		}
	}
	return max + 1
}

func compileAutoReactRules(rules []autoReactRule) []compiledAutoReact {
	out := make([]compiledAutoReact, 0, len(rules))
	for _, r := range rules {
		re, err := regexp.Compile("(?i)" + r.Pattern)
		if err != nil {
			continue
		}
		out = append(out, compiledAutoReact{re: re, emoji: r.Emoji, id: r.ID})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

func getAutoReactCompiled(chatID int64) []compiledAutoReact {
	autoReactCacheMu.RLock()
	cached, ok := autoReactCache[chatID]
	autoReactCacheMu.RUnlock()
	if ok {
		return cached
	}

	rules, err := loadAutoReactRules(chatID)
	if err != nil {
		return nil
	}
	compiled := compileAutoReactRules(rules)

	autoReactCacheMu.Lock()
	autoReactCache[chatID] = compiled
	autoReactCacheMu.Unlock()
	return compiled
}

func invalidateAutoReactCache(chatID int64) {
	autoReactCacheMu.Lock()
	delete(autoReactCache, chatID)
	autoReactCacheMu.Unlock()
}

func AutoReactHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Autoreact works in groups only.</b>")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> Admins with Change Info only.")
		return nil
	}

	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b>\n" +
			" • <code>/autoreact add &lt;pattern&gt; &lt;emoji&gt;</code>\n" +
			" • <code>/autoreact list</code>\n" +
			" • <code>/autoreact rm &lt;id&gt;</code>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	sub := strings.ToLower(strings.TrimSpace(parts[0]))
	rest := ""
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}

	switch sub {
	case "add":
		return autoReactAdd(m, rest)
	case "list", "ls":
		return autoReactList(m)
	case "rm", "del", "delete", "remove":
		return autoReactRemove(m, rest)
	default:
		m.Reply("<b>Unknown subcommand.</b> Use <code>add</code>, <code>list</code>, or <code>rm</code>.")
		return nil
	}
}

func autoReactAdd(m *tg.NewMessage, rest string) error {
	if rest == "" {
		m.Reply("<b>Usage:</b> <code>/autoreact add &lt;pattern&gt; &lt;emoji&gt;</code>")
		return nil
	}

	idx := strings.LastIndex(rest, " ")
	if idx <= 0 {
		m.Reply("<b>Provide both pattern and emoji.</b>")
		return nil
	}

	pattern := strings.TrimSpace(rest[:idx])
	emoji := strings.TrimSpace(rest[idx+1:])
	if pattern == "" || emoji == "" {
		m.Reply("<b>Pattern and emoji cannot be empty.</b>")
		return nil
	}

	if _, err := regexp.Compile("(?i)" + pattern); err != nil {
		m.Reply("<b>Invalid regex:</b> <code>" + html.EscapeString(err.Error()) + "</code>")
		return nil
	}

	rules, err := loadAutoReactRules(m.ChatID())
	if err != nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	if len(rules) >= 50 {
		m.Reply("<b>Limit reached.</b> Max 50 rules per chat.")
		return nil
	}

	rule := autoReactRule{
		ID:      nextAutoReactID(rules),
		Pattern: pattern,
		Emoji:   emoji,
		AddedBy: m.SenderID(),
	}

	if err := saveAutoReactRule(m.ChatID(), rule); err != nil {
		m.Reply("<b>Failed to save rule.</b>")
		return nil
	}

	invalidateAutoReactCache(m.ChatID())

	m.Reply(fmt.Sprintf("<b>Added rule</b> <code>#%d</code>\nPattern: <code>%s</code>\nEmoji: %s",
		rule.ID, html.EscapeString(pattern), html.EscapeString(emoji)))
	return nil
}

func autoReactList(m *tg.NewMessage) error {
	rules, err := loadAutoReactRules(m.ChatID())
	if err != nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	if len(rules) == 0 {
		m.Reply("<b>No autoreact rules.</b> Add one with <code>/autoreact add &lt;pattern&gt; &lt;emoji&gt;</code>")
		return nil
	}

	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })

	var sb strings.Builder
	sb.WriteString("<b>Auto-Reactions</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━\n\n")
	for _, r := range rules {
		sb.WriteString(fmt.Sprintf(" • <code>#%d</code> <code>%s</code> → %s\n",
			r.ID, html.EscapeString(r.Pattern), html.EscapeString(r.Emoji)))
	}
	sb.WriteString(fmt.Sprintf("\n━━━━━━━━━━━━━━━━\n<b>Total:</b> %d", len(rules)))

	m.Reply(sb.String())
	return nil
}

func autoReactRemove(m *tg.NewMessage, rest string) error {
	if rest == "" {
		m.Reply("<b>Usage:</b> <code>/autoreact rm &lt;id&gt;</code>")
		return nil
	}

	id, err := strconv.Atoi(strings.TrimSpace(rest))
	if err != nil || id <= 0 {
		m.Reply("<b>Invalid id.</b> Use <code>/autoreact list</code> to see ids.")
		return nil
	}

	deleted, err := deleteAutoReactRule(m.ChatID(), id)
	if err != nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}
	if !deleted {
		m.Reply(fmt.Sprintf("<b>Not found:</b> rule <code>#%d</code>", id))
		return nil
	}

	invalidateAutoReactCache(m.ChatID())
	m.Reply(fmt.Sprintf("<b>Removed rule</b> <code>#%d</code>", id))
	return nil
}

func AutoReactWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() || m.Text() == "" || m.IsCommand() {
		return nil
	}

	rules := getAutoReactCompiled(m.ChatID())
	if len(rules) == 0 {
		return nil
	}

	text := m.Text()
	for _, r := range rules {
		if r.re.MatchString(text) {
			_ = m.React(r.emoji)
			return nil
		}
	}
	return nil
}

func registerAutoReactHandlers() {
	c := Client
	c.On("cmd:autoreact", AutoReactHandler)
	c.On(tg.OnNewMessage, AutoReactWatcher)
}

func init() {
	QueueHandlerRegistration(registerAutoReactHandlers)

	Mods.AddModule("AutoReact", `<b>Auto-Reactions</b>

Automatically react to messages matching a regex pattern with an emoji.

<b>Commands:</b>
 • <code>/autoreact add &lt;pattern&gt; &lt;emoji&gt;</code> - Add a rule
 • <code>/autoreact list</code> - List all rules
 • <code>/autoreact rm &lt;id&gt;</code> - Remove a rule by id

<b>Notes:</b>
Pattern is a Go regex matched case-insensitively against the message text.
Only one reaction per message is applied (first matching rule by id).
Compiled regex is cached per chat.

<b>Permission:</b> Admins with Change Info only.`)
}
