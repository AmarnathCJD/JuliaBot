package modules

import (
	"fmt"
	"main/modules/db"
	"regexp"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
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

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
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
			if reply.Photo() != nil {
				welcomeMsg.MediaType = "photo"
				welcomeMsg.FileID = reply.File.FileID
			} else if reply.Document() != nil {
				welcomeMsg.MediaType = "document"
				welcomeMsg.FileID = reply.File.FileID
			} else if reply.Video() != nil {
				welcomeMsg.MediaType = "video"
				welcomeMsg.FileID = reply.File.FileID
			} else if reply.Animation() != nil {
				welcomeMsg.MediaType = "animation"
				welcomeMsg.FileID = reply.File.FileID
			} else if reply.Sticker() != nil {
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

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
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

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
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

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
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

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
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

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
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

	chatID := p.ChatID()
	user := p.User

	if user == nil {
		return nil
	}

	welcomeMsg, err := db.GetWelcome(chatID)
	if err != nil {
		return nil
	}

	if welcomeMsg != nil && welcomeMsg.DeletePrevious {
		if lastID, _ := db.GetLastWelcomeID(chatID); lastID > 0 {
			p.Client.DeleteMessages(chatID, []int32{int32(lastID)})
		}
	}

	// Use default welcome if none configured
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

	// Default welcome message
	if content == "" && fileID == "" {
		content = "Hey {mention}, welcome to {chatname}!"
	}

	channel, _ := p.Client.GetChannel(chatID)
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
			msg, err := p.Client.SendMedia(chatID, media, &tg.MediaOptions{Caption: text, ReplyMarkup: keyboard})
			if err == nil {
				sentMsg = *msg
			}
		}
	} else if text != "" {
		opts := &tg.SendOptions{}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		msg, err := p.Client.SendMessage(chatID, text, opts)
		if err == nil {
			sentMsg = *msg
		}
	}

	if sentMsg.ID > 0 {
		db.SetLastWelcomeID(chatID, int(sentMsg.ID))

		if autoDelete > 0 {
			go func() {
				time.Sleep(time.Duration(autoDelete) * time.Second)
				p.Client.DeleteMessages(chatID, []int32{int32(sentMsg.ID)})
			}()
		}
	}

	return nil
}

func GoodbyeHandler(p *tg.ParticipantUpdate) error {
	return nil
	if !p.IsLeft() && !p.IsKicked() {
		return nil
	}

	chatID := p.ChatID()
	user := p.User

	if user == nil {
		return nil
	}

	goodbyeMsg, _ := db.GetGoodbye(chatID)

	// Use default goodbye if none configured
	content := ""
	fileID := ""
	if goodbyeMsg != nil {
		content = goodbyeMsg.Content
		fileID = goodbyeMsg.FileID
	}

	// Default goodbye message
	if content == "" && fileID == "" {
		content = "{first} has left the chat."
	}

	channel, _ := p.Client.GetChannel(chatID)
	text := formatWelcomeText(content, user, channel)

	if fileID != "" {
		media, err := tg.ResolveBotFileID(fileID)
		if err == nil {
			p.Client.SendMedia(chatID, media, &tg.MediaOptions{Caption: text})
		}
	} else if text != "" {
		p.Client.SendMessage(chatID, text)
	}

	return nil
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

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "delete") {
		m.Reply("You need Delete Messages permission")
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

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
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

func init() {
	Mods.AddModule("Welcome", `<b>Greetings Module</b>

Welcome new users and say goodbye to leaving users.

<b>Welcome Commands:</b>
 - /setwelcome <text> - Set welcome message
 - /welcome on/off - Toggle welcome messages
 - /clearwelcome - Clear welcome message
 - /cleanwelcome on/off - Delete previous welcome
 - /wautodelete <time> - Auto-delete welcome

<b>Goodbye Commands:</b>
 - /setgoodbye <text> - Set goodbye message
 - /goodbye on/off - Toggle goodbye messages
 - /cleargoodbye - Clear goodbye message



<b>Variables:</b>
 {first}, {last}, {fullname}, {username}
 {mention}, {id}, {chatname}

<b>Button Format:</b>
 [Button](https://url)
 [same:Button](https://url) - Same row
 [Rules](rules) - Show rules`)
}
