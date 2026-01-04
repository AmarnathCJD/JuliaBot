package modules

import (
	"fmt"
	"main/modules/db"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	pendingCaptcha   = make(map[string]*captchaSession)
	captchaMutex     sync.RWMutex
	mathCaptchaCache = make(map[string]int)
)

type captchaSession struct {
	ChatID    int64
	UserID    int64
	MessageID int
	ExpiresAt time.Time
	Answer    string
}

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

func SetCaptchaHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Captcha can only be configured in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to configure captcha")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))
	settings, _ := db.GetCaptchaSettings(m.ChatID())
	if settings == nil {
		settings = &db.CaptchaSettings{Mode: "button", TimeLimit: 120}
	}

	if args == "" {
		status := "disabled"
		if settings.Enabled {
			status = "enabled"
		}
		m.Reply(fmt.Sprintf(`<b>Captcha Settings</b>

Status: <b>%s</b>
Mode: <b>%s</b>
Time Limit: <b>%d seconds</b>
Mute New Users: <b>%v</b>
Kick on Timeout: <b>%v</b>

<b>Commands:</b>
 /captcha on/off - Toggle captcha
 /setcaptchamode <mode> - Set mode (button/math)
 /setcaptchatime <seconds> - Set time limit
 /captchamute on/off - Mute until verified
 /captchakick on/off - Kick on timeout`, status, settings.Mode, settings.TimeLimit, settings.MuteNewUsers, settings.KickTimeout))
		return nil
	}

	switch args {
	case "on", "enable", "yes":
		settings.Enabled = true
		db.SetCaptchaSettings(m.ChatID(), settings)
		m.Reply("Captcha enabled")
	case "off", "disable", "no":
		settings.Enabled = false
		db.SetCaptchaSettings(m.ChatID(), settings)
		m.Reply("Captcha disabled")
	default:
		m.Reply("Usage: /captcha on/off")
	}
	return nil
}

func SetCaptchaModeHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Captcha can only be configured in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to configure captcha")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))
	settings, _ := db.GetCaptchaSettings(m.ChatID())
	if settings == nil {
		settings = &db.CaptchaSettings{Mode: "button", TimeLimit: 120}
	}

	switch args {
	case "button":
		settings.Mode = "button"
		db.SetCaptchaSettings(m.ChatID(), settings)
		m.Reply("Captcha mode set to <b>button</b> - User must click a button to verify")
	case "math":
		settings.Mode = "math"
		db.SetCaptchaSettings(m.ChatID(), settings)
		m.Reply("Captcha mode set to <b>math</b> - User must solve a simple math problem")
	default:
		m.Reply("Available modes:\n - <code>button</code> - Click to verify\n - <code>math</code> - Solve a math problem")
	}
	return nil
}

func SetCaptchaTimeHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Captcha can only be configured in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to configure captcha")
		return nil
	}

	args := strings.TrimSpace(m.Args())
	seconds, err := strconv.Atoi(args)
	if err != nil || seconds < 30 || seconds > 600 {
		m.Reply("Time limit must be between 30 and 600 seconds")
		return nil
	}

	settings, _ := db.GetCaptchaSettings(m.ChatID())
	if settings == nil {
		settings = &db.CaptchaSettings{Mode: "button", TimeLimit: 120}
	}

	settings.TimeLimit = seconds
	db.SetCaptchaSettings(m.ChatID(), settings)
	m.Reply(fmt.Sprintf("Captcha time limit set to <b>%d seconds</b>", seconds))
	return nil
}

func CaptchaMuteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Captcha can only be configured in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to configure captcha")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))
	settings, _ := db.GetCaptchaSettings(m.ChatID())
	if settings == nil {
		settings = &db.CaptchaSettings{Mode: "button", TimeLimit: 120}
	}

	switch args {
	case "on", "yes", "enable":
		settings.MuteNewUsers = true
		db.SetCaptchaSettings(m.ChatID(), settings)
		m.Reply("New users will be muted until they pass captcha")
	case "off", "no", "disable":
		settings.MuteNewUsers = false
		db.SetCaptchaSettings(m.ChatID(), settings)
		m.Reply("New users will not be muted")
	default:
		status := "disabled"
		if settings.MuteNewUsers {
			status = "enabled"
		}
		m.Reply(fmt.Sprintf("Mute new users: <b>%s</b>\n\nUsage: /captchamute on/off", status))
	}
	return nil
}

func CaptchaKickHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Captcha can only be configured in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to configure captcha")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))
	settings, _ := db.GetCaptchaSettings(m.ChatID())
	if settings == nil {
		settings = &db.CaptchaSettings{Mode: "button", TimeLimit: 120}
	}

	switch args {
	case "on", "yes", "enable":
		settings.KickTimeout = true
		db.SetCaptchaSettings(m.ChatID(), settings)
		m.Reply("Users who fail captcha will be kicked")
	case "off", "no", "disable":
		settings.KickTimeout = false
		db.SetCaptchaSettings(m.ChatID(), settings)
		m.Reply("Users who fail captcha will not be kicked")
	default:
		status := "disabled"
		if settings.KickTimeout {
			status = "enabled"
		}
		m.Reply(fmt.Sprintf("Kick on timeout: <b>%s</b>\n\nUsage: /captchakick on/off", status))
	}
	return nil
}

func WelcomeHandler(p *tg.ParticipantUpdate) error {
	if !p.IsJoined() && !p.IsAdded() {
		return nil
	}

	chatID := p.ChannelID()
	user := p.User

	welcomeMsg, _ := db.GetWelcome(chatID)
	captchaSettings, _ := db.GetCaptchaSettings(chatID)

	if welcomeMsg != nil && welcomeMsg.DeletePrevious {
		if lastID, _ := db.GetLastWelcomeID(chatID); lastID > 0 {
			p.Client.DeleteMessages(chatID, []int32{int32(lastID)})
		}
	}

	if captchaSettings != nil && captchaSettings.Enabled {
		handleCaptcha(p.Client, chatID, user, captchaSettings)
	}

	if welcomeMsg == nil || (!welcomeMsg.Enabled && welcomeMsg.Content == "" && welcomeMsg.FileID == "") {
		return nil
	}

	if !welcomeMsg.Enabled {
		return nil
	}

	channel, _ := p.Client.GetChannel(chatID)
	text := formatWelcomeText(welcomeMsg.Content, user, channel)

	var keyboard *tg.ReplyInlineMarkup
	if welcomeMsg.Buttons != "" {
		_, buttons := parseButtons(welcomeMsg.Buttons)
		keyboard = buildKeyboard(buttons)
	}

	var sentMsg tg.NewMessage
	if welcomeMsg.FileID != "" {
		media, err := tg.ResolveBotFileID(welcomeMsg.FileID)
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

		if welcomeMsg.AutoDeleteSec > 0 {
			go func() {
				time.Sleep(time.Duration(welcomeMsg.AutoDeleteSec) * time.Second)
				p.Client.DeleteMessages(chatID, []int32{int32(sentMsg.ID)})
			}()
		}
	}

	return nil
}

func GoodbyeHandler(p *tg.ParticipantUpdate) error {
	if !p.IsLeft() && !p.IsKicked() {
		return nil
	}

	chatID := p.ChannelID()
	user := p.User

	goodbyeMsg, _ := db.GetGoodbye(chatID)
	if goodbyeMsg == nil || (!goodbyeMsg.Enabled && goodbyeMsg.Content == "" && goodbyeMsg.FileID == "") {
		return nil
	}

	if !goodbyeMsg.Enabled {
		return nil
	}

	channel, _ := p.Client.GetChannel(chatID)
	text := formatWelcomeText(goodbyeMsg.Content, user, channel)

	if goodbyeMsg.FileID != "" {
		media, err := tg.ResolveBotFileID(goodbyeMsg.FileID)
		if err == nil {
			p.Client.SendMedia(chatID, media, &tg.MediaOptions{Caption: text})
		}
	} else if text != "" {
		p.Client.SendMessage(chatID, text)
	}

	return nil
}

func handleCaptcha(client *tg.Client, chatID int64, user *tg.UserObj, settings *db.CaptchaSettings) {
	if settings.MuteNewUsers {
		peer, _ := client.ResolvePeer(user.ID)
		client.EditBanned(chatID, peer, &tg.BannedOptions{Mute: true})
	}

	sessionKey := fmt.Sprintf("%d_%d", chatID, user.ID)

	captchaMutex.Lock()
	if _, exists := pendingCaptcha[sessionKey]; exists {
		captchaMutex.Unlock()
		return
	}
	captchaMutex.Unlock()

	var msg *tg.NewMessage
	var answer string

	switch settings.Mode {
	case "math":
		a := rand.Intn(10) + 1
		b := rand.Intn(10) + 1
		answer = strconv.Itoa(a + b)
		mathCaptchaCache[sessionKey] = a + b

		text := fmt.Sprintf("<b>%s</b>, please solve this to verify you are human:\n\nWhat is <b>%d + %d</b>?\n\nYou have %d seconds.",
			user.FirstName, a, b, settings.TimeLimit)

		msg, _ = client.SendMessage(chatID, text)

	default:
		b := tg.Button
		keyboard := tg.NewKeyboard().AddRow(
			b.Data("I am human", fmt.Sprintf("captcha_verify_%d", user.ID)),
		).Build()

		text := fmt.Sprintf("<b>%s</b>, please click the button below to verify you are not a bot.\n\nYou have %d seconds.",
			user.FirstName, settings.TimeLimit)

		msg, _ = client.SendMessage(chatID, text, &tg.SendOptions{ReplyMarkup: keyboard})
		answer = "button"
	}

	if msg != nil {
		session := &captchaSession{
			ChatID:    chatID,
			UserID:    user.ID,
			MessageID: int(msg.ID),
			ExpiresAt: time.Now().Add(time.Duration(settings.TimeLimit) * time.Second),
			Answer:    answer,
		}

		captchaMutex.Lock()
		pendingCaptcha[sessionKey] = session
		captchaMutex.Unlock()

		go func() {
			time.Sleep(time.Duration(settings.TimeLimit) * time.Second)
			captchaMutex.Lock()
			sess, exists := pendingCaptcha[sessionKey]
			if exists {
				delete(pendingCaptcha, sessionKey)
				delete(mathCaptchaCache, sessionKey)
			}
			captchaMutex.Unlock()

			if exists && sess.MessageID > 0 {
				client.DeleteMessages(chatID, []int32{int32(sess.MessageID)})

				if settings.KickTimeout {
					peer, _ := client.ResolvePeer(user.ID)
					client.KickParticipant(chatID, peer)
					client.SendMessage(chatID, fmt.Sprintf("<b>%s</b> was removed for not completing verification", user.FirstName))
				}
			}
		}()
	}
}

func CaptchaVerifyCallback(c *tg.CallbackQuery) error {
	data := c.DataString()

	if !strings.HasPrefix(data, "captcha_verify_") {
		return nil
	}

	userIDStr := strings.TrimPrefix(data, "captcha_verify_")
	targetUserID, _ := strconv.ParseInt(userIDStr, 10, 64)

	if c.SenderID != targetUserID {
		c.Answer("This verification is not for you", &tg.CallbackOptions{Alert: true})
		return nil
	}

	sessionKey := fmt.Sprintf("%d_%d", c.ChatID, c.SenderID)

	captchaMutex.Lock()
	_, exists := pendingCaptcha[sessionKey]
	if exists {
		delete(pendingCaptcha, sessionKey)
	}
	captchaMutex.Unlock()

	if !exists {
		c.Answer("Verification expired or already completed", &tg.CallbackOptions{Alert: true})
		return nil
	}

	captchaSettings, _ := db.GetCaptchaSettings(c.ChatID)
	if captchaSettings != nil && captchaSettings.MuteNewUsers {
		peer, _ := c.Client.ResolvePeer(c.SenderID)
		c.Client.EditBanned(c.ChatID, peer, &tg.BannedOptions{Unmute: true})
	}

	c.Delete()
	c.Answer("Verification successful. Welcome!", &tg.CallbackOptions{Alert: true})

	return nil
}

func CaptchaMathHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	sessionKey := fmt.Sprintf("%d_%d", m.ChatID(), m.SenderID())

	captchaMutex.RLock()
	session, exists := pendingCaptcha[sessionKey]
	expectedAnswer, hasMath := mathCaptchaCache[sessionKey]
	captchaMutex.RUnlock()

	if !exists || !hasMath {
		return nil
	}

	answer, err := strconv.Atoi(strings.TrimSpace(m.Text()))
	if err != nil {
		return nil
	}

	if answer == expectedAnswer {
		captchaMutex.Lock()
		delete(pendingCaptcha, sessionKey)
		delete(mathCaptchaCache, sessionKey)
		captchaMutex.Unlock()

		m.Client.DeleteMessages(m.ChatID(), []int32{int32(session.MessageID)})

		captchaSettings, _ := db.GetCaptchaSettings(m.ChatID())
		if captchaSettings != nil && captchaSettings.MuteNewUsers {
			peer, _ := m.Client.ResolvePeer(m.SenderID())
			m.Client.EditBanned(m.ChatID(), peer, &tg.BannedOptions{Unmute: true})
		}

		m.Reply("Correct! Welcome to the group.")
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
	captchaSettings, _ := db.GetCaptchaSettings(m.ChatID())

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

	captchaStatus := "disabled"
	captchaMode := "button"
	captchaTime := 120
	if captchaSettings != nil {
		if captchaSettings.Enabled {
			captchaStatus = "enabled"
		}
		if captchaSettings.Mode != "" {
			captchaMode = captchaSettings.Mode
		}
		if captchaSettings.TimeLimit > 0 {
			captchaTime = captchaSettings.TimeLimit
		}
	}

	m.Reply(fmt.Sprintf(`<b>Greetings Settings</b>

<b>Welcome:</b> %s
<b>Goodbye:</b> %s
<b>Captcha:</b> %s
 - Mode: %s
 - Time: %ds

<b>Commands:</b>
 /setwelcome - Set welcome message
 /setgoodbye - Set goodbye message
 /welcome on/off - Toggle welcome
 /goodbye on/off - Toggle goodbye
 /clearwelcome - Clear welcome
 /cleargoodbye - Clear goodbye
 /captcha - Captcha settings`, welcomeStatus, goodbyeStatus, captchaStatus, captchaMode, captchaTime))
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

<b>Captcha Commands:</b>
 - /captcha on/off - Toggle captcha
 - /setcaptchamode <mode> - Set mode (button/math)
 - /setcaptchatime <sec> - Set time limit
 - /captchamute on/off - Mute until verified
 - /captchakick on/off - Kick on timeout

<b>Variables:</b>
 {first}, {last}, {fullname}, {username}
 {mention}, {id}, {chatname}

<b>Button Format:</b>
 [Button](https://url)
 [same:Button](https://url) - Same row
 [Rules](rules) - Show rules`)
}
