package modules

import (
	"fmt"
	"main/modules/db"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func SetRulesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Rules can only be set in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to set rules")
		return nil
	}

	rules := m.Args()

	if m.IsReply() && rules == "" {
		reply, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error getting reply message")
			return nil
		}
		rules = reply.Text()
	}

	if rules == "" {
		m.Reply("Usage: /setrules <rules text> or reply to a message with /setrules")
		return nil
	}

	if err := db.SetRules(m.ChatID(), rules); err != nil {
		m.Reply("Failed to save rules")
		return nil
	}

	m.Reply("Rules have been saved successfully.\nUse /rules to view them.")
	return nil
}

func GetRulesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Rules can only be viewed in groups")
		return nil
	}

	rules, err := db.GetRules(m.ChatID())
	if err != nil || rules == "" {
		m.Reply("No rules set for this chat.\nAdmins can use /setrules to set rules.")
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

	response := fmt.Sprintf("<b>Rules for %s:</b>\n\n%s", chatName, rules)
	m.Reply(response)
	return nil
}

func ClearRulesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Rules can only be cleared in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
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
	if strings.HasPrefix(c.DataString(), "rules_") {
		rules, err := db.GetRules(c.ChatID)
		if err != nil || rules == "" {
			c.Answer("No rules set for this chat", &tg.CallbackOptions{Alert: true})
			return nil
		}

		c.Answer("Showing rules...", nil)
		if len(rules) < 200 {
			c.Answer(rules, &tg.CallbackOptions{Alert: true})
		} else {
			c.Respond("Rules:\n\n" + rules)
		}
	}
	return nil
}

func init() {
	Mods.AddModule("Rules", `<b>Rules Module</b>

Set and display group rules.

<b>Commands:</b>
 - /setrules <text> - Set the group rules (or reply to a message)
 - /rules - Display the group rules
 - /clearrules - Clear the rules

<b>Note:</b> Only admins with Change Info permission can modify rules.`)
}
