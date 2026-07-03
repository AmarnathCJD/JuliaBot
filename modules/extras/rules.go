package extras

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
	"html"
	modules "main/modules"
	"main/modules/db"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func parseButtons(content string) (string, [][]tg.KeyboardButton) {
	btnRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	matches := btnRegex.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return content, nil
	}

	cleanContent := content
	var rows [][]tg.KeyboardButton
	var currentRow []tg.KeyboardButton

	for _, match := range matches {
		fullMatch := match[0]
		label := match[1]
		action := match[2]

		cleanContent = strings.Replace(cleanContent, fullMatch, "", 1)

		sameRow := strings.HasPrefix(label, "same:")
		if sameRow {
			label = strings.TrimPrefix(label, "same:")
		}

		var btn tg.KeyboardButton
		if action == "rules" {
			btn = tg.Button.Data(label, "rules_show")
		} else if strings.HasPrefix(action, "http://") || strings.HasPrefix(action, "https://") {
			btn = tg.Button.URL(label, action)
		} else {
			btn = tg.Button.Data(label, "btn_"+action)
		}

		if sameRow && len(currentRow) > 0 {
			currentRow = append(currentRow, btn)
		} else {
			if len(currentRow) > 0 {
				rows = append(rows, currentRow)
			}
			currentRow = []tg.KeyboardButton{btn}
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	cleanContent = strings.TrimSpace(cleanContent)
	cleanContent = regexp.MustCompile(`\n{3,}`).ReplaceAllString(cleanContent, "\n\n")

	return cleanContent, rows
}

func buildKeyboard(rows [][]tg.KeyboardButton) *tg.ReplyInlineMarkup {
	if len(rows) == 0 {
		return nil
	}

	kb := tg.NewKeyboard()
	for _, row := range rows {
		kb.AddRow(row...)
	}
	return kb.Build()
}

func SetRulesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Rules can only be set in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to set rules")
		return nil
	}

	rules := &db.Rules{}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error getting reply message")
			return nil
		}

		if reply.IsMedia() {
			if reply.Photo() != nil {
				rules.MediaType = "photo"
				rules.FileID = reply.File.FileID
			} else if reply.Document() != nil {
				rules.MediaType = "document"
				rules.FileID = reply.File.FileID
			} else if reply.Video() != nil {
				rules.MediaType = "video"
				rules.FileID = reply.File.FileID
			} else if reply.Animation() != nil {
				rules.MediaType = "animation"
				rules.FileID = reply.File.FileID
			}
		}

		rules.Content = reply.RawText()
		if m.Args() != "" {
			rules.Content = m.Args()
		}
	} else {
		rules.Content = m.Args()
	}

	if rules.Content == "" && rules.FileID == "" {
		m.Reply("Usage: /setrules <rules text> or reply to a message with /setrules\n\n<b>Button Format:</b>\n[Button Name](https://url)\n[same:Button 2](https://url2) - same row\n[Rules](rules) - shows rules popup")
		return nil
	}

	cleanContent, buttons := parseButtons(rules.Content)
	rules.Content = cleanContent
	if len(buttons) > 0 {
		var btnStr []string
		for _, row := range buttons {
			var rowStr []string
			for _, btn := range row {
				switch b := btn.(type) {
				case *tg.KeyboardButtonURL:
					rowStr = append(rowStr, fmt.Sprintf("[%s](%s)", b.Text, b.URL))
				case *tg.KeyboardButtonCallback:
					if string(b.Data) == "rules_show" {
						rowStr = append(rowStr, fmt.Sprintf("[%s](rules)", b.Text))
					} else {
						rowStr = append(rowStr, fmt.Sprintf("[%s](%s)", b.Text, string(b.Data)))
					}
				}
			}
			btnStr = append(btnStr, strings.Join(rowStr, " "))
		}
		rules.Buttons = strings.Join(btnStr, "\n")
	}

	if err := db.SetRulesWithMedia(m.ChatID(), rules); err != nil {
		m.Reply("Failed to save rules")
		return nil
	}

	mediaTag := ""
	if rules.FileID != "" {
		mediaTag = " [with media]"
	}

	m.Reply(fmt.Sprintf("Rules have been saved%s\nUse /rules to view them", mediaTag))
	return nil
}

func GetRulesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Rules can only be viewed in groups")
		return nil
	}

	rules, err := db.GetRulesWithMedia(m.ChatID())
	if err != nil || rules == nil || (rules.Content == "" && rules.FileID == "") {
		m.Reply("No rules set for this chat\nAdmins can use /setrules to set rules")
		return nil
	}

	var chatName string
	if m.Channel != nil {
		chatName = m.Channel.Title
	} else if m.Chat != nil {
		chatName = m.Chat.Title
	} else {
		chatName = "this chat"
	}

	response := fmt.Sprintf("<b>Rules for %s:</b>\n\n%s", chatName, rules.Content)

	var keyboard *tg.ReplyInlineMarkup
	if rules.Buttons != "" {
		_, buttons := parseButtons(rules.Buttons)
		keyboard = buildKeyboard(buttons)
	}

	opts := &tg.SendOptions{}
	if keyboard != nil {
		opts.ReplyMarkup = keyboard
	}

	if rules.FileID != "" {
		media, err := tg.ResolveBotFileID(rules.FileID)
		if err == nil {
			m.ReplyMedia(media, &tg.MediaOptions{Caption: response, ReplyMarkup: keyboard})
			return nil
		}
	}

	m.Reply(response, opts)
	return nil
}

func ClearRulesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Rules can only be cleared in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to clear rules")
		return nil
	}

	if !db.HasRules(m.ChatID()) {
		m.Reply("No rules set for this chat")
		return nil
	}

	if err := db.DeleteRules(m.ChatID()); err != nil {
		m.Reply("Failed to clear rules")
		return nil
	}

	m.Reply("Rules have been cleared")
	return nil
}

func RulesButtonCallback(c *tg.CallbackQuery) error {
	data := c.DataString()
	if data != "rules_show" && !strings.HasPrefix(data, "rules_") {
		return nil
	}
	rules, err := db.GetRulesWithMedia(c.ChatID)
	if err != nil || rules == nil || rules.Content == "" {
		c.Answer("No rules set for this chat", &tg.CallbackOptions{Alert: true})
		return nil
	}

	if len(rules.Content) < 200 {
		c.Answer(rules.Content, &tg.CallbackOptions{Alert: true})
		return nil
	}

	msg := "<b>Rules:</b>\n\n" + rules.Content
	if _, err := c.Client.SendMessage(c.SenderID, msg, &tg.SendOptions{ParseMode: "HTML"}); err == nil {
		c.Answer("Rules sent to your DM.")
		return nil
	}

	me := c.Client.Me()
	if me != nil && me.Username != "" {
		b := tg.Button
		kb := tg.NewKeyboard().AddRow(
			b.URL("Open bot to see rules", "https://t.me/"+me.Username+"?start=rules"),
		).Build()
		c.Edit("Please open the bot in DM to see the rules.", &tg.SendOptions{ReplyMarkup: kb})
		c.Answer("Open the bot in DM to view rules.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	c.Answer("Open the bot in DM first, then click again.", &tg.CallbackOptions{Alert: true})
	return nil
}

func registerRuleHandlers() {
	c := modules.Client
	c.On("cmd:setrules", SetRulesHandler)
	c.On("cmd:rules", GetRulesHandler)
	c.On("cmd:clearrules", ClearRulesHandler)
	c.On("callback:rules_", RulesButtonCallback)
}

func initFromSrc_rules_0_1() {
	modules.QueueHandlerRegistration(registerRuleHandlers)

	modules.Mods.AddModule("Rules", `<b>Rules Module</b>

Set and display group rules with optional media.

<b>Commands:</b>
 - /setrules <text> - Set the group rules (or reply to a message)
 - /rules - Display the group rules
 - /clearrules - Clear the rules

<b>Button Format:</b>
 - [Button Text](https://url) - URL button
 - [same:Button](https://url) - Same row as previous
 - [Rules](rules) - Shows rules popup

<b>Note:</b> Only admins with Change Info permission can modify rules.`)
}
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
	if modules.Client != nil {
		_, _ = modules.Client.UnpinMessage(r.ChatID, r.MsgID)
		_, _ = modules.Client.DeleteMessages(r.ChatID, []int32{r.MsgID})
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
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "pin") {
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

	if modules.CanBot(m.Client, m.Channel, "pin") {
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
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "pin") {
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

	if modules.CanBot(m.Client, m.Channel, "pin") {
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
	c := modules.Client
	c.On("cmd:tmprule", TmpRuleHandler)
	c.On("cmd:tmprules", TmpRulesListHandler)
	c.On("cmd:rmtmprule", RmTmpRuleHandler)

	go rescheduleTmpRules()

	modules.Mods.AddModule("TempRules", `<b>Temporary Rules Module</b>

Pin a rule for a limited time. After expiry it is auto-unpinned and deleted.

<b>Commands:</b>
 - /tmprule &lt;duration&gt; &lt;text&gt; - Post a temporary rule (admin)
 - /tmprules - List active temporary rules
 - /rmtmprule &lt;id&gt; - Remove a temporary rule early (admin)

<b>Duration format:</b> 30s, 5m, 2h, 1d, 1w`)
}

func initFromSrc_temprules_1_1() {
	modules.QueueHandlerRegistration(registerTmpRulesHandlers)
}

func init() {
	initFromSrc_rules_0_1()
	initFromSrc_temprules_1_1()
}
