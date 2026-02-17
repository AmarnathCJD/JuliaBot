package modules

import (
	"fmt"
	"main/modules/db"
	"regexp"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
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

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
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

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
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
	if c.DataString() == "rules_show" {
		rules, err := db.GetRulesWithMedia(c.ChatID)
		if err != nil || rules == nil || rules.Content == "" {
			c.Answer("No rules set for this chat", &tg.CallbackOptions{Alert: true})
			return nil
		}

		if len(rules.Content) < 200 {
			c.Answer(rules.Content, &tg.CallbackOptions{Alert: true})
		} else {
			c.Answer("Showing rules...", nil)
			c.Respond("Rules:\n\n" + rules.Content)
		}
	}

	if strings.HasPrefix(c.DataString(), "rules_") && c.DataString() != "rules_show" {
		rules, err := db.GetRulesWithMedia(c.ChatID)
		if err != nil || rules == nil || rules.Content == "" {
			c.Answer("No rules set for this chat", &tg.CallbackOptions{Alert: true})
			return nil
		}

		c.Answer("Showing rules...", nil)
		if len(rules.Content) < 200 {
			c.Answer(rules.Content, &tg.CallbackOptions{Alert: true})
		} else {
			c.Respond("Rules:\n\n" + rules.Content)
		}
	}
	return nil
}

func registerRuleHandlers() {
	c := Client
	c.On("cmd:setrules", SetRulesHandler)
	c.On("cmd:rules", GetRulesHandler)
	c.On("cmd:clearrules", ClearRulesHandler)
	c.On("callback:rules_", RulesButtonCallback)
}

func init() {
	QueueHandlerRegistration(registerRuleHandlers)

	Mods.AddModule("Rules", `<b>Rules Module</b>

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
