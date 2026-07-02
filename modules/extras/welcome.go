package extras

import (
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	bolt "go.etcd.io/bbolt"
	modules "main/modules"
)

func formatWelcomeText(text string, user *tg.UserObj, chat *tg.Channel) string {
	if user == nil {
		return text
	}

	chatTitle := ""
	if chat != nil {
		chatTitle = chat.Title
	}

	mention := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", user.ID, user.FirstName)

	replacements := map[string]string{
		"{first}":     user.FirstName,
		"{last}":      user.LastName,
		"{fullname}":  strings.TrimSpace(user.FirstName + " " + user.LastName),
		"{username}":  "@" + user.Username,
		"{mention}":   mention,
		"{id}":        strconv.FormatInt(user.ID, 10),
		"{chatname}":  chatTitle,
		"{chattitle}": chatTitle,
	}

	result := text
	for placeholder, value := range replacements {
		if value == "@" {
			value = mention
		}
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

func SetWelcomeHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Welcome message can only be set in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to set welcome message")
		return nil
	}

	welcomeMsg := &db.WelcomeMessage{Enabled: true}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error getting reply message")
			return nil
		}

		if reply.IsMedia() {
			if reply.Photo() != nil && reply.File != nil {
				welcomeMsg.MediaType = "photo"
				welcomeMsg.FileID = reply.File.FileID
			} else if reply.Document() != nil && reply.File != nil {
				welcomeMsg.MediaType = "document"
				welcomeMsg.FileID = reply.File.FileID
			} else if reply.Video() != nil && reply.File != nil {
				welcomeMsg.MediaType = "video"
				welcomeMsg.FileID = reply.File.FileID
			} else if reply.Animation() != nil && reply.File != nil {
				welcomeMsg.MediaType = "animation"
				welcomeMsg.FileID = reply.File.FileID
			} else if reply.Sticker() != nil && reply.File != nil {
				welcomeMsg.MediaType = "sticker"
				welcomeMsg.FileID = reply.File.FileID
			}
		}

		welcomeMsg.Content = reply.RawText()
		if m.Args() != "" {
			welcomeMsg.Content = m.Args()
		}
	} else {
		welcomeMsg.Content = m.Args()
	}

	if welcomeMsg.Content == "" && welcomeMsg.FileID == "" {
		m.Reply(`<b>Set Welcome Message</b>

Usage: /setwelcome <message> or reply to a message

<b>Available variables:</b>
 - {first} - User's first name
 - {last} - User's last name
 - {fullname} - User's full name
 - {username} - User's @username
 - {mention} - Clickable mention
 - {id} - User's ID
 - {chatname} - Chat title

<b>Button Format:</b>
 - [Button Text](https://url)
 - [same:Button](https://url) - Same row
 - [Rules](rules) - Show rules button`)
		return nil
	}

	cleanContent, buttons := parseButtons(welcomeMsg.Content)
	welcomeMsg.Content = cleanContent
	if len(buttons) > 0 {
		welcomeMsg.Buttons = serializeButtons(buttons)
	}

	if err := db.SetWelcome(m.ChatID(), welcomeMsg); err != nil {
		m.Reply("Failed to save welcome message")
		return nil
	}

	mediaTag := ""
	if welcomeMsg.FileID != "" {
		mediaTag = " [with media]"
	}

	m.Reply(fmt.Sprintf("Welcome message saved%s", mediaTag))
	return nil
}

func serializeButtons(buttons [][]tg.KeyboardButton) string {
	var lines []string
	for _, row := range buttons {
		var rowParts []string
		for i, btn := range row {
			switch b := btn.(type) {
			case *tg.KeyboardButtonURL:
				if i > 0 {
					rowParts = append(rowParts, fmt.Sprintf("[same:%s](%s)", b.Text, b.URL))
				} else {
					rowParts = append(rowParts, fmt.Sprintf("[%s](%s)", b.Text, b.URL))
				}
			case *tg.KeyboardButtonCallback:
				data := string(b.Data)
				if data == "rules_show" {
					data = "rules"
				}
				if i > 0 {
					rowParts = append(rowParts, fmt.Sprintf("[same:%s](%s)", b.Text, data))
				} else {
					rowParts = append(rowParts, fmt.Sprintf("[%s](%s)", b.Text, data))
				}
			}
		}
		lines = append(lines, strings.Join(rowParts, " "))
	}
	return strings.Join(lines, "\n")
}

func SetGoodbyeHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Goodbye message can only be set in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to set goodbye message")
		return nil
	}

	goodbyeMsg := &db.WelcomeMessage{Enabled: true}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error getting reply message")
			return nil
		}

		if reply.IsMedia() {
			if reply.Photo() != nil {
				goodbyeMsg.MediaType = "photo"
				goodbyeMsg.FileID = reply.File.FileID
			} else if reply.Document() != nil {
				goodbyeMsg.MediaType = "document"
				goodbyeMsg.FileID = reply.File.FileID
			} else if reply.Video() != nil {
				goodbyeMsg.MediaType = "video"
				goodbyeMsg.FileID = reply.File.FileID
			} else if reply.Animation() != nil {
				goodbyeMsg.MediaType = "animation"
				goodbyeMsg.FileID = reply.File.FileID
			}
		}

		goodbyeMsg.Content = reply.RawText()
		if m.Args() != "" {
			goodbyeMsg.Content = m.Args()
		}
	} else {
		goodbyeMsg.Content = m.Args()
	}

	if goodbyeMsg.Content == "" && goodbyeMsg.FileID == "" {
		m.Reply(`<b>Set Goodbye Message</b>

Usage: /setgoodbye <message> or reply to a message

<b>Available variables:</b>
 - {first} - User's first name
 - {last} - User's last name
 - {fullname} - User's full name
 - {username} - User's @username
 - {mention} - Clickable mention
 - {id} - User's ID
 - {chatname} - Chat title`)
		return nil
	}

	if err := db.SetGoodbye(m.ChatID(), goodbyeMsg); err != nil {
		m.Reply("Failed to save goodbye message")
		return nil
	}

	m.Reply("Goodbye message saved")
	return nil
}

func WelcomeToggleHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Welcome settings can only be changed in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to modify welcome settings")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))

	welcomeMsg, _ := db.GetWelcome(m.ChatID())
	if welcomeMsg == nil {
		welcomeMsg = &db.WelcomeMessage{Enabled: false}
	}

	switch args {
	case "on", "yes", "enable", "1":
		welcomeMsg.Enabled = true
		db.SetWelcome(m.ChatID(), welcomeMsg)
		m.Reply("Welcome messages enabled")
	case "off", "no", "disable", "0":
		welcomeMsg.Enabled = false
		db.SetWelcome(m.ChatID(), welcomeMsg)
		m.Reply("Welcome messages disabled")
	default:
		status := "disabled"
		if welcomeMsg.Enabled {
			status = "enabled"
		}
		m.Reply(fmt.Sprintf("Welcome is currently <b>%s</b>\n\nUsage: /welcome on/off", status))
	}
	return nil
}

func GoodbyeToggleHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Goodbye settings can only be changed in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to modify goodbye settings")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))

	goodbyeMsg, _ := db.GetGoodbye(m.ChatID())
	if goodbyeMsg == nil {
		goodbyeMsg = &db.WelcomeMessage{Enabled: false}
	}

	switch args {
	case "on", "yes", "enable", "1":
		goodbyeMsg.Enabled = true
		db.SetGoodbye(m.ChatID(), goodbyeMsg)
		m.Reply("Goodbye messages enabled")
	case "off", "no", "disable", "0":
		goodbyeMsg.Enabled = false
		db.SetGoodbye(m.ChatID(), goodbyeMsg)
		m.Reply("Goodbye messages disabled")
	default:
		status := "disabled"
		if goodbyeMsg.Enabled {
			status = "enabled"
		}
		m.Reply(fmt.Sprintf("Goodbye is currently <b>%s</b>\n\nUsage: /goodbye on/off", status))
	}
	return nil
}

func ClearWelcomeHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Welcome can only be cleared in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to clear welcome")
		return nil
	}

	db.SetWelcome(m.ChatID(), &db.WelcomeMessage{})
	m.Reply("Welcome message cleared")
	return nil
}

func ClearGoodbyeHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Goodbye can only be cleared in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to clear goodbye")
		return nil
	}

	db.SetGoodbye(m.ChatID(), &db.WelcomeMessage{})
	m.Reply("Goodbye message cleared")
	return nil
}

func WelcomeHandler(p *tg.ParticipantUpdate) error {
	if !p.IsJoined() && !p.IsAdded() {
		return nil
	}
	user := p.User
	if user == nil || user.Bot {
		return nil
	}
	chatID := p.ChatID()

	if reason, matched := joinFilterMatch(chatID, user); matched {
		joinFilterBan(p.Client, chatID, user, reason)
		return nil
	}

	if getCaptchaConfig(chatID).Enabled {
		startCaptcha(p)
		return nil
	}
	sendWelcome(p.Client, chatID, user)
	return nil
}

func sendWelcome(client *tg.Client, chatID int64, user *tg.UserObj) {
	if user == nil {
		return
	}

	welcomeMsg, err := db.GetWelcome(chatID)
	if err != nil {
		return
	}

	if welcomeMsg != nil && welcomeMsg.DeletePrevious {
		if lastID, _ := db.GetLastWelcomeID(chatID); lastID > 0 {
			client.DeleteMessages(chatID, []int32{int32(lastID)})
		}
	}

	content := ""
	fileID := ""
	buttons := ""
	autoDelete := 0
	if welcomeMsg != nil {
		content = welcomeMsg.Content
		fileID = welcomeMsg.FileID
		buttons = welcomeMsg.Buttons
		autoDelete = welcomeMsg.AutoDeleteSec
	}

	if content == "" && fileID == "" {
		content = "Hey {mention}, welcome to {chatname}!"
	}

	channel, _ := client.GetChannel(chatID)
	text := formatWelcomeText(content, user, channel)

	var keyboard *tg.ReplyInlineMarkup
	if buttons != "" {
		_, btns := parseButtons(buttons)
		keyboard = buildKeyboard(btns)
	}

	var sentMsg tg.NewMessage
	if fileID != "" {
		media, err := tg.ResolveBotFileID(fileID)
		if err == nil {
			msg, err := client.SendMedia(chatID, media, &tg.MediaOptions{Caption: text, ReplyMarkup: keyboard})
			if err == nil {
				sentMsg = *msg
			}
		}
	} else if text != "" {
		opts := &tg.SendOptions{}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		msg, err := client.SendMessage(chatID, text, opts)
		if err == nil {
			sentMsg = *msg
		}
	}

	if sentMsg.ID > 0 {
		db.SetLastWelcomeID(chatID, int(sentMsg.ID))
		if autoDelete > 0 {
			go func() {
				time.Sleep(time.Duration(autoDelete) * time.Second)
				client.DeleteMessages(chatID, []int32{int32(sentMsg.ID)})
			}()
		}
	}
}

func WelcomeSettingsHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Welcome settings can only be viewed in groups")
		return nil
	}

	welcomeMsg, _ := db.GetWelcome(m.ChatID())
	goodbyeMsg, _ := db.GetGoodbye(m.ChatID())

	welcomeStatus := "not set"
	if welcomeMsg != nil && (welcomeMsg.Content != "" || welcomeMsg.FileID != "") {
		if welcomeMsg.Enabled {
			welcomeStatus = "enabled"
		} else {
			welcomeStatus = "disabled"
		}
	}

	goodbyeStatus := "not set"
	if goodbyeMsg != nil && (goodbyeMsg.Content != "" || goodbyeMsg.FileID != "") {
		if goodbyeMsg.Enabled {
			goodbyeStatus = "enabled"
		} else {
			goodbyeStatus = "disabled"
		}
	}

	m.Reply(fmt.Sprintf(`<b>Greetings Settings</b>

<b>Welcome:</b> %s
<b>Goodbye:</b> %s

<b>Commands:</b>
 /setwelcome - Set welcome message
 /setgoodbye - Set goodbye message
 /welcome on/off - Toggle welcome
 /goodbye on/off - Toggle goodbye
 /clearwelcome - Clear welcome
 /cleargoodbye - Clear goodbye`, welcomeStatus, goodbyeStatus))
	return nil
}

func CleanServiceHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("This can only be used in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))

	welcomeMsg, _ := db.GetWelcome(m.ChatID())
	if welcomeMsg == nil {
		welcomeMsg = &db.WelcomeMessage{}
	}

	switch args {
	case "on", "yes", "enable":
		welcomeMsg.DeletePrevious = true
		db.SetWelcome(m.ChatID(), welcomeMsg)
		m.Reply("Previous welcome messages will be deleted")
	case "off", "no", "disable":
		welcomeMsg.DeletePrevious = false
		db.SetWelcome(m.ChatID(), welcomeMsg)
		m.Reply("Previous welcome messages will not be deleted")
	default:
		status := "disabled"
		if welcomeMsg.DeletePrevious {
			status = "enabled"
		}
		m.Reply(fmt.Sprintf("Clean welcome: <b>%s</b>\n\nUsage: /cleanwelcome on/off", status))
	}
	return nil
}

func WelcomeAutoDeleteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("This can only be used in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission")
		return nil
	}

	args := strings.TrimSpace(m.Args())

	welcomeMsg, _ := db.GetWelcome(m.ChatID())
	if welcomeMsg == nil {
		welcomeMsg = &db.WelcomeMessage{}
	}

	if args == "off" || args == "0" {
		welcomeMsg.AutoDeleteSec = 0
		db.SetWelcome(m.ChatID(), welcomeMsg)
		m.Reply("Welcome auto-delete disabled")
		return nil
	}

	durationRegex := regexp.MustCompile(`^(\d+)([smh]?)$`)
	matches := durationRegex.FindStringSubmatch(args)

	if len(matches) == 0 {
		m.Reply(fmt.Sprintf("Current: <b>%ds</b>\n\nUsage: /wautodelete <seconds> or <number>s/m/h\nExample: /wautodelete 30s or /wautodelete 5m", welcomeMsg.AutoDeleteSec))
		return nil
	}

	value, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch unit {
	case "m":
		value *= 60
	case "h":
		value *= 3600
	}

	if value < 5 || value > 86400 {
		m.Reply("Auto-delete time must be between 5 seconds and 24 hours")
		return nil
	}

	welcomeMsg.AutoDeleteSec = value
	db.SetWelcome(m.ChatID(), welcomeMsg)
	m.Reply(fmt.Sprintf("Welcome messages will be auto-deleted after <b>%d seconds</b>", value))
	return nil
}

var defaultJoinFilterEmoji = []string{
	"🏳️‍🌈", "🏳‍🌈", "🏳️‍⚧️", "🏳‍⚧",
	"🌈",
}

type JoinFilterConfig struct {
	Enabled  bool     `json:"enabled"`
	Emoji    []string `json:"emoji"`
	Patterns []string `json:"patterns"`
}

const joinFilterBucket = "joinfilter_cfg"

func getJoinFilterConfig(chatID int64) *JoinFilterConfig {
	cfg := &JoinFilterConfig{Enabled: false, Emoji: append([]string{}, defaultJoinFilterEmoji...)}
	database, err := db.GetDB()
	if err != nil || database == nil {
		return cfg
	}
	_ = database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(joinFilterBucket))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(strconv.FormatInt(chatID, 10)))
		if data == nil {
			return nil
		}
		_ = json.Unmarshal(data, cfg)
		return nil
	})
	return cfg
}

func saveJoinFilterConfig(chatID int64, cfg *JoinFilterConfig) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db unavailable")
	}
	return database.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(joinFilterBucket))
		if err != nil {
			return err
		}
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return b.Put([]byte(strconv.FormatInt(chatID, 10)), data)
	})
}

func joinFilterDisplayName(user *tg.UserObj) string {
	if user == nil {
		return ""
	}
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = user.Username
	}
	return name
}

func joinFilterMatch(chatID int64, user *tg.UserObj) (string, bool) {
	cfg := getJoinFilterConfig(chatID)
	if !cfg.Enabled {
		return "", false
	}
	name := joinFilterDisplayName(user)
	if name == "" {
		return "", false
	}
	for _, e := range cfg.Emoji {
		if e != "" && strings.Contains(name, e) {
			return "name contains blocklisted emoji: " + e, true
		}
	}
	for _, p := range cfg.Patterns {
		re, err := modules.ValidateUserRegex(p)
		if err != nil {
			continue
		}
		if re.MatchString(name) {
			return "name matches blocked pattern: " + p, true
		}
	}
	return "", false
}

func joinFilterBan(client *tg.Client, chatID int64, user *tg.UserObj, reason string) {
	peer, err := client.ResolvePeer(user.ID)
	if err != nil || peer == nil {
		return
	}
	_, _ = client.EditBanned(chatID, peer, &tg.BannedOptions{Ban: true})
	_, _ = client.SendMessage(chatID, fmt.Sprintf(
		"Removed %s on join — %s.",
		html.EscapeString(joinFilterDisplayName(user)),
		html.EscapeString(reason),
	))
}

func JoinFilterCommandHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Join filter is only available in groups.")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		m.Reply("You need Ban Users permission to manage the join filter.")
		return nil
	}

	args := strings.Fields(m.Args())
	cfg := getJoinFilterConfig(m.ChatID())

	if len(args) == 0 {
		status := "off"
		if cfg.Enabled {
			status = "on"
		}
		m.Reply(fmt.Sprintf(`<b>Join Filter</b>

State: <b>%s</b>
Emoji entries: <b>%d</b>
Regex patterns: <b>%d</b>

<b>Usage:</b>
/joinfilter on|off
/joinfilter list
/joinfilter addemoji &lt;emoji&gt; — block joiners whose name contains it
/joinfilter rmemoji &lt;emoji&gt;
/joinfilter addregex &lt;pattern&gt; — case-insensitive; wrap in / /
/joinfilter rmregex &lt;pattern&gt;
/joinfilter reset — restore defaults (pride flags)

Matches auto-ban new joiners before they see captcha or welcome.`, status, len(cfg.Emoji), len(cfg.Patterns)))
		return nil
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "on", "enable", "yes", "1":
		cfg.Enabled = true
	case "off", "disable", "no", "0":
		cfg.Enabled = false
	case "list":
		var b strings.Builder
		b.WriteString("<b>Join Filter entries</b>\n\n")
		b.WriteString("<b>Emoji blocklist:</b>\n")
		if len(cfg.Emoji) == 0 {
			b.WriteString("(empty)\n")
		}
		for _, e := range cfg.Emoji {
			b.WriteString(html.EscapeString(e))
			b.WriteString("\n")
		}
		b.WriteString("\n<b>Regex patterns:</b>\n")
		if len(cfg.Patterns) == 0 {
			b.WriteString("(empty)\n")
		}
		for _, p := range cfg.Patterns {
			b.WriteString("<code>")
			b.WriteString(html.EscapeString(p))
			b.WriteString("</code>\n")
		}
		m.Reply(b.String())
		return nil
	case "addemoji":
		if len(args) < 2 {
			m.Reply("Usage: /joinfilter addemoji &lt;emoji&gt;")
			return nil
		}
		emoji := strings.Join(args[1:], "")
		for _, e := range cfg.Emoji {
			if e == emoji {
				m.Reply("Already in list.")
				return nil
			}
		}
		cfg.Emoji = append(cfg.Emoji, emoji)
	case "rmemoji":
		if len(args) < 2 {
			m.Reply("Usage: /joinfilter rmemoji &lt;emoji&gt;")
			return nil
		}
		emoji := strings.Join(args[1:], "")
		out := cfg.Emoji[:0]
		removed := false
		for _, e := range cfg.Emoji {
			if e == emoji {
				removed = true
				continue
			}
			out = append(out, e)
		}
		cfg.Emoji = out
		if !removed {
			m.Reply("Not in list.")
			return nil
		}
	case "addregex":
		if len(args) < 2 {
			m.Reply("Usage: /joinfilter addregex &lt;pattern&gt;")
			return nil
		}
		pattern := strings.Join(args[1:], " ")
		if _, err := modules.ValidateUserRegex(pattern); err != nil {
			m.Reply("Rejected: " + html.EscapeString(err.Error()))
			return nil
		}
		for _, p := range cfg.Patterns {
			if p == pattern {
				m.Reply("Already in list.")
				return nil
			}
		}
		cfg.Patterns = append(cfg.Patterns, pattern)
	case "rmregex":
		if len(args) < 2 {
			m.Reply("Usage: /joinfilter rmregex &lt;pattern&gt;")
			return nil
		}
		pattern := strings.Join(args[1:], " ")
		out := cfg.Patterns[:0]
		removed := false
		for _, p := range cfg.Patterns {
			if p == pattern {
				removed = true
				continue
			}
			out = append(out, p)
		}
		cfg.Patterns = out
		if !removed {
			m.Reply("Not in list.")
			return nil
		}
	case "reset":
		cfg.Emoji = append([]string{}, defaultJoinFilterEmoji...)
		cfg.Patterns = nil
	default:
		m.Reply("Usage: /joinfilter on|off|list|addemoji|rmemoji|addregex|rmregex|reset")
		return nil
	}

	if err := saveJoinFilterConfig(m.ChatID(), cfg); err != nil {
		m.Reply("Failed to save join filter config.")
		return nil
	}
	m.Reply("Join filter updated.")
	return nil
}

const (
	captchaBucket   = "captcha_cfg"
	captchaTypeBtn  = "button"
	captchaTypeMath = "math"
	captchaTypeText = "text"
	captchaPrefix   = "cap:"
)

type CaptchaConfig struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
	Timeout int    `json:"timeout"`
}

type captchaState struct {
	UserID    int64
	ChatID    int64
	Type      string
	Answer    string
	MessageID int32
	Deadline  time.Time
	Cancel    chan struct{}
	UserName  string
	User      *tg.UserObj
}

var captchaStates sync.Map

func captchaStateKey(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func defaultCaptchaConfig() *CaptchaConfig {
	return &CaptchaConfig{Enabled: false, Type: captchaTypeBtn, Timeout: 60}
}

func getCaptchaConfig(chatID int64) *CaptchaConfig {
	cfg := defaultCaptchaConfig()
	database, err := db.GetDB()
	if err != nil || database == nil {
		return cfg
	}
	_ = database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(captchaBucket))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(strconv.FormatInt(chatID, 10)))
		if data == nil {
			return nil
		}
		_ = json.Unmarshal(data, cfg)
		return nil
	})
	if cfg.Type != captchaTypeBtn && cfg.Type != captchaTypeMath && cfg.Type != captchaTypeText {
		cfg.Type = captchaTypeBtn
	}
	if cfg.Timeout < 15 || cfg.Timeout > 600 {
		cfg.Timeout = 60
	}
	return cfg
}

func saveCaptchaConfig(chatID int64, cfg *CaptchaConfig) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db unavailable")
	}
	return database.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(captchaBucket))
		if err != nil {
			return err
		}
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return b.Put([]byte(strconv.FormatInt(chatID, 10)), data)
	})
}

func CaptchaEnabled(chatID int64) bool {
	return getCaptchaConfig(chatID).Enabled
}

func captchaLoadFont(dc *gg.Context, size float64) {
	name := modules.GetRandomFont()
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

func captchaDrawNoise(dc *gg.Context, w, h int) {
	for range 1200 {
		x := rand.Float64() * float64(w)
		y := rand.Float64() * float64(h)
		dc.SetRGBA(rand.Float64(), rand.Float64(), rand.Float64(), 0.35)
		dc.DrawPoint(x, y, 1.2)
		dc.Fill()
	}
	for range 8 {
		dc.SetRGBA(rand.Float64(), rand.Float64(), rand.Float64(), 0.45)
		dc.SetLineWidth(1 + rand.Float64()*2)
		dc.DrawLine(
			rand.Float64()*float64(w), rand.Float64()*float64(h),
			rand.Float64()*float64(w), rand.Float64()*float64(h),
		)
		dc.Stroke()
	}
}

func captchaRenderText(text string, w, h int, fontSize float64) (string, error) {
	dc := gg.NewContext(w, h)
	dc.SetRGB(0.95, 0.95, 0.95)
	dc.Clear()

	captchaDrawNoise(dc, w, h)

	captchaLoadFont(dc, fontSize)
	chars := []rune(text)
	if len(chars) == 0 {
		return "", fmt.Errorf("empty text")
	}
	cellW := float64(w) / float64(len(chars)+1)
	for i, ch := range chars {
		dc.Push()
		x := cellW*float64(i+1) + (rand.Float64()-0.5)*6
		y := float64(h)/2 + (rand.Float64()-0.5)*10
		angle := (rand.Float64() - 0.5) * 0.6
		dc.RotateAbout(angle, x, y)
		dc.SetRGB(rand.Float64()*0.4, rand.Float64()*0.4, rand.Float64()*0.4)
		dc.DrawStringAnchored(string(ch), x, y, 0.5, 0.5)
		dc.Pop()
	}

	for range 4 {
		dc.SetRGBA(rand.Float64()*0.6, rand.Float64()*0.6, rand.Float64()*0.6, 0.4)
		dc.SetLineWidth(1.2)
		dc.DrawLine(
			rand.Float64()*float64(w), rand.Float64()*float64(h),
			rand.Float64()*float64(w), rand.Float64()*float64(h),
		)
		dc.Stroke()
	}

	out := filepath.Join(os.TempDir(), fmt.Sprintf("captcha_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func captchaRandomString(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

func captchaGenMath() (string, string) {
	a := rand.Intn(20) + 1
	b := rand.Intn(20) + 1
	switch rand.Intn(3) {
	case 0:
		return fmt.Sprintf("%d + %d = ?", a, b), strconv.Itoa(a + b)
	case 1:
		if a < b {
			a, b = b, a
		}
		return fmt.Sprintf("%d - %d = ?", a, b), strconv.Itoa(a - b)
	default:
		x := rand.Intn(9) + 2
		y := rand.Intn(9) + 2
		return fmt.Sprintf("%d x %d = ?", x, y), strconv.Itoa(x * y)
	}
}

func captchaRestrictUser(c *tg.Client, chatID int64, userID int64, restrict bool) {
	user, err := c.ResolvePeer(userID)
	if err != nil || user == nil {
		return
	}
	if restrict {
		_, _ = c.EditBanned(chatID, user, &tg.BannedOptions{Mute: true})
	} else {
		_, _ = c.EditBanned(chatID, user, &tg.BannedOptions{Unmute: true})
	}
}

func captchaKickUser(c *tg.Client, chatID int64, userID int64) {
	user, err := c.ResolvePeer(userID)
	if err != nil || user == nil {
		return
	}
	_, _ = c.EditBanned(chatID, user, &tg.BannedOptions{Ban: true})
	_, _ = c.EditBanned(chatID, user, &tg.BannedOptions{Unban: true})
}

func captchaCleanup(chatID, userID int64, c *tg.Client, msgID int32) {
	key := captchaStateKey(chatID, userID)
	captchaStates.Delete(key)
	if msgID > 0 {
		_, _ = c.DeleteMessages(chatID, []int32{msgID})
	}
}

func captchaScheduleTimeout(c *tg.Client, st *captchaState) {
	go func() {
		select {
		case <-time.After(time.Until(st.Deadline)):
			if _, ok := captchaStates.Load(captchaStateKey(st.ChatID, st.UserID)); !ok {
				return
			}
			captchaKickUser(c, st.ChatID, st.UserID)
			captchaCleanup(st.ChatID, st.UserID, c, st.MessageID)
			_, _ = c.SendMessage(st.ChatID, fmt.Sprintf("%s failed the captcha and was removed.", html.EscapeString(st.UserName)))
		case <-st.Cancel:
			return
		}
	}()
}

func captchaPassed(client *tg.Client, st *captchaState) {
	captchaRestrictUser(client, st.ChatID, st.UserID, false)
	close(st.Cancel)
	captchaCleanup(st.ChatID, st.UserID, client, st.MessageID)
	if st.User != nil {
		sendWelcome(client, st.ChatID, st.User)
	}
}

func startCaptcha(p *tg.ParticipantUpdate) {
	chatID := p.ChatID()
	cfg := getCaptchaConfig(chatID)
	userID := p.User.ID
	key := captchaStateKey(chatID, userID)
	if _, exists := captchaStates.Load(key); exists {
		return
	}

	captchaRestrictUser(p.Client, chatID, userID, true)

	userName := p.User.FirstName
	if userName == "" {
		userName = "user"
	}
	mention := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", userID, html.EscapeString(userName))
	deadline := time.Now().Add(time.Duration(cfg.Timeout) * time.Second)

	st := &captchaState{
		UserID:   userID,
		ChatID:   chatID,
		Type:     cfg.Type,
		Deadline: deadline,
		Cancel:   make(chan struct{}),
		UserName: userName,
		User:     p.User,
	}

	switch cfg.Type {
	case captchaTypeBtn:
		st.Answer = "ok"
		b := tg.Button
		kb := tg.NewKeyboard().AddRow(
			b.Data("I'm not a bot", fmt.Sprintf("%sbtn:%d:%d", captchaPrefix, chatID, userID)),
		).Build()
		msg, err := p.Client.SendMessage(chatID, fmt.Sprintf(
			"Welcome %s! Please click the button below within %d seconds to verify you are human.",
			mention, cfg.Timeout,
		), &tg.SendOptions{ReplyMarkup: kb})
		if err == nil && msg != nil {
			st.MessageID = msg.ID
		}

	case captchaTypeMath:
		question, answer := captchaGenMath()
		st.Answer = answer
		img, err := captchaRenderText(question, 360, 140, 36)
		if err != nil {
			captchaRestrictUser(p.Client, chatID, userID, false)
			return
		}
		defer os.Remove(img)

		correct, _ := strconv.Atoi(answer)
		opts := []int{correct}
		for len(opts) < 4 {
			delta := rand.Intn(11) - 5
			if delta == 0 {
				delta = 1
			}
			candidate := correct + delta
			if candidate < 0 {
				candidate = correct + rand.Intn(5) + 1
			}
			dup := false
			for _, o := range opts {
				if o == candidate {
					dup = true
					break
				}
			}
			if !dup {
				opts = append(opts, candidate)
			}
		}
		rand.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })

		b := tg.Button
		kb := tg.NewKeyboard()
		row := []tg.KeyboardButton{}
		for i, o := range opts {
			label := strconv.Itoa(o)
			data := fmt.Sprintf("%smath:%d:%d:%s", captchaPrefix, chatID, userID, label)
			row = append(row, b.Data(label, data))
			if (i+1)%2 == 0 {
				kb.AddRow(row...)
				row = []tg.KeyboardButton{}
			}
		}
		if len(row) > 0 {
			kb.AddRow(row...)
		}

		msg, err := p.Client.SendMedia(chatID, img, &tg.MediaOptions{
			Caption: fmt.Sprintf(
				"Welcome %s! Solve the captcha within %d seconds. Pick the correct answer below.",
				mention, cfg.Timeout,
			),
			FileName:    "captcha.png",
			MimeType:    "image/png",
			ReplyMarkup: kb.Build(),
		})
		if err == nil && msg != nil {
			st.MessageID = msg.ID
		}

	case captchaTypeText:
		text := captchaRandomString(5)
		st.Answer = text
		img, err := captchaRenderText(text, 320, 120, 44)
		if err != nil {
			captchaRestrictUser(p.Client, chatID, userID, false)
			return
		}
		defer os.Remove(img)

		msg, err := p.Client.SendMedia(chatID, img, &tg.MediaOptions{
			Caption: fmt.Sprintf(
				"Welcome %s! Reply to this message with the 5 characters shown within %d seconds (case-insensitive).",
				mention, cfg.Timeout,
			),
			FileName: "captcha.png",
			MimeType: "image/png",
		})
		if err == nil && msg != nil {
			st.MessageID = msg.ID
		}
	}

	captchaStates.Store(key, st)
	captchaScheduleTimeout(p.Client, st)
}

func CaptchaCallbackHandler(c *tg.CallbackQuery) error {
	data := c.DataString()
	if !strings.HasPrefix(data, captchaPrefix) {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(data, captchaPrefix), ":")
	if len(parts) < 3 {
		return nil
	}
	kind := parts[0]
	chatID, _ := strconv.ParseInt(parts[1], 10, 64)
	userID, _ := strconv.ParseInt(parts[2], 10, 64)

	if c.SenderID != userID {
		c.Answer("This captcha is not for you.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	key := captchaStateKey(chatID, userID)
	raw, ok := captchaStates.Load(key)
	if !ok {
		c.Answer("Captcha expired.", &tg.CallbackOptions{Alert: true})
		return nil
	}
	st := raw.(*captchaState)

	switch kind {
	case "btn":
		captchaPassed(c.Client, st)
		c.Answer("Verified!", &tg.CallbackOptions{Alert: false})
	case "math":
		if len(parts) < 4 {
			return nil
		}
		choice := parts[3]
		if choice == st.Answer {
			captchaPassed(c.Client, st)
			c.Answer("Verified!", &tg.CallbackOptions{Alert: false})
		} else {
			captchaKickUser(c.Client, chatID, userID)
			close(st.Cancel)
			captchaCleanup(chatID, userID, c.Client, st.MessageID)
			c.Answer("Wrong answer. Removed.", &tg.CallbackOptions{Alert: true})
			_, _ = c.Client.SendMessage(chatID, fmt.Sprintf("%s failed the captcha and was removed.", html.EscapeString(st.UserName)))
		}
	}
	return nil
}

func CaptchaMessageWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	senderID := m.SenderID()
	if senderID == 0 {
		return nil
	}
	key := captchaStateKey(m.ChatID(), senderID)
	raw, ok := captchaStates.Load(key)
	if !ok {
		return nil
	}
	st := raw.(*captchaState)
	if st.Type != captchaTypeText {
		return nil
	}
	text := strings.ToUpper(strings.TrimSpace(m.Text()))
	if text == "" {
		return nil
	}
	if text == strings.ToUpper(st.Answer) {
		captchaPassed(m.Client, st)
		_, _ = m.Client.DeleteMessages(m.ChatID(), []int32{int32(m.ID)})
		reply, _ := m.Client.SendMessage(m.ChatID(), fmt.Sprintf("%s verified!", html.EscapeString(st.UserName)))
		if reply != nil {
			go func(id int32) {
				time.Sleep(10 * time.Second)
				_, _ = m.Client.DeleteMessages(m.ChatID(), []int32{id})
			}(reply.ID)
		}
	} else {
		captchaKickUser(m.Client, m.ChatID(), senderID)
		close(st.Cancel)
		captchaCleanup(m.ChatID(), senderID, m.Client, st.MessageID)
		_, _ = m.Client.DeleteMessages(m.ChatID(), []int32{int32(m.ID)})
		_, _ = m.Client.SendMessage(m.ChatID(), fmt.Sprintf("%s failed the captcha and was removed.", html.EscapeString(st.UserName)))
	}
	return nil
}

func CaptchaCommandHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Captcha is only available in groups")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to manage captcha")
		return nil
	}

	args := strings.Fields(m.Args())
	cfg := getCaptchaConfig(m.ChatID())

	if len(args) == 0 {
		status := "off"
		if cfg.Enabled {
			status = "on"
		}
		m.Reply(fmt.Sprintf(`<b>Captcha Settings</b>

Status: <b>%s</b>
Type: <b>%s</b>
Timeout: <b>%d seconds</b>

<b>Usage:</b>
/captcha on|off|status
/captcha type &lt;button|math|text&gt;
/captcha timeout &lt;seconds&gt;`, status, html.EscapeString(cfg.Type), cfg.Timeout))
		return nil
	}

	switch strings.ToLower(args[0]) {
	case "on", "enable", "yes", "1":
		cfg.Enabled = true
		if err := saveCaptchaConfig(m.ChatID(), cfg); err != nil {
			m.Reply("Failed to enable captcha")
			return nil
		}
		m.Reply(fmt.Sprintf("Captcha <b>enabled</b>\nType: <b>%s</b>\nTimeout: <b>%d s</b>", html.EscapeString(cfg.Type), cfg.Timeout))
	case "off", "disable", "no", "0":
		cfg.Enabled = false
		if err := saveCaptchaConfig(m.ChatID(), cfg); err != nil {
			m.Reply("Failed to disable captcha")
			return nil
		}
		m.Reply("Captcha <b>disabled</b>")
	case "status":
		status := "off"
		if cfg.Enabled {
			status = "on"
		}
		m.Reply(fmt.Sprintf(`<b>Captcha Status</b>

State: <b>%s</b>
Type: <b>%s</b>
Timeout: <b>%d seconds</b>`, status, html.EscapeString(cfg.Type), cfg.Timeout))
	case "type":
		if len(args) < 2 {
			m.Reply("Usage: /captcha type &lt;button|math|text&gt;")
			return nil
		}
		t := strings.ToLower(args[1])
		if t == "image" || t == "img" {
			t = captchaTypeText
		}
		if t != captchaTypeBtn && t != captchaTypeMath && t != captchaTypeText {
			m.Reply("Invalid type. Options: button, math, text")
			return nil
		}
		cfg.Type = t
		if err := saveCaptchaConfig(m.ChatID(), cfg); err != nil {
			m.Reply("Failed to update captcha type")
			return nil
		}
		m.Reply(fmt.Sprintf("Captcha type set to <b>%s</b>", html.EscapeString(t)))
	case "timeout":
		if len(args) < 2 {
			m.Reply("Usage: /captcha timeout &lt;seconds&gt;")
			return nil
		}
		secs, err := strconv.Atoi(args[1])
		if err != nil || secs < 15 || secs > 600 {
			m.Reply("Invalid timeout. Must be between 15 and 600 seconds")
			return nil
		}
		cfg.Timeout = secs
		if err := saveCaptchaConfig(m.ChatID(), cfg); err != nil {
			m.Reply("Failed to update captcha timeout")
			return nil
		}
		m.Reply(fmt.Sprintf("Captcha timeout set to <b>%d seconds</b>", secs))
	default:
		m.Reply("Usage: /captcha on|off|status | type &lt;button|math|text&gt; | timeout &lt;seconds&gt;")
	}
	return nil
}

func registerWelcomeHandlers() {
	c := modules.Client
	c.On("cmd:setwelcome", SetWelcomeHandler)
	c.On("cmd:setgoodbye", SetGoodbyeHandler)
	c.On("cmd:goodbye", GoodbyeToggleHandler)
	c.On("cmd:clearwelcome", ClearWelcomeHandler)
	c.On("cmd:cleargoodbye", ClearGoodbyeHandler)
	c.On("cmd:cleanwelcome", CleanServiceHandler)
	c.On("cmd:wautodelete", WelcomeAutoDeleteHandler)
	c.On("cmd:welcomesettings", WelcomeSettingsHandler)
	c.On("cmd:greetings", WelcomeSettingsHandler)
	c.On("cmd:greet", WelcomeToggleHandler)
	c.On("cmd:welcome", WelcomeToggleHandler)
	c.On("cmd:captcha", CaptchaCommandHandler)
	c.On("callback:"+captchaPrefix, CaptchaCallbackHandler)
	c.On("cmd:joinfilter", JoinFilterCommandHandler)
	c.On(tg.OnParticipant, WelcomeHandler)
	c.On(tg.OnNewMessage, CaptchaMessageWatcher)
}

func init() {
	modules.QueueHandlerRegistration(registerWelcomeHandlers)

	modules.Mods.AddModule("Welcome", `<b>Greetings + Captcha Module</b>

Welcomes new users, greets returners, and gates joins behind a captcha challenge when enabled.

<b>Welcome Commands:</b>
 - /setwelcome &lt;text&gt; - Set welcome message
 - /welcome on/off - Toggle welcome messages
 - /clearwelcome - Clear welcome message
 - /cleanwelcome on/off - Delete previous welcome
 - /wautodelete &lt;time&gt; - Auto-delete welcome

<b>Goodbye Commands:</b>
 - /setgoodbye &lt;text&gt; - Set goodbye message
 - /goodbye on/off - Toggle goodbye messages
 - /cleargoodbye - Clear goodbye message

<b>Captcha Commands:</b>
 - /captcha on|off|status - Toggle or view captcha
 - /captcha type &lt;button|math|text&gt; - Challenge style
 - /captcha timeout &lt;seconds&gt; - Verification window (15-600)

<b>Captcha types:</b>
 button - Click a single inline button
 math   - Solve arithmetic via 4 inline buttons (grainy image question)
 text   - Reply with the 5 characters shown in a noisy image

When captcha is on, the welcome message fires only AFTER the new member
verifies. Failures and timeouts get removed automatically.

<b>Variables:</b>
 {first}, {last}, {fullname}, {username}
 {mention}, {id}, {chatname}

<b>Button Format:</b>
 [Button](https://url)
 [same:Button](https://url) - Same row
 [Rules](rules) - Show rules`)
}
