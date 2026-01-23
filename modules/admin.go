package modules

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func parseAdminError(err error, action string) string {
	if err == nil {
		return "Unknown error occurred"
	}
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "CHAT_ADMIN_REQUIRED"):
		return "Unable to " + action + ", make sure I have the required admin rights"
	case strings.Contains(errStr, "USER_ADMIN_INVALID"):
		return "Unable to " + action + ", can't modify another admin's rights"
	case strings.Contains(errStr, "USER_NOT_PARTICIPANT"):
		return "Unable to " + action + ", user is not a member of this chat"
	case strings.Contains(errStr, "USER_CREATOR"):
		return "Unable to " + action + ", can't perform this action on the chat owner"
	case strings.Contains(errStr, "USER_ID_INVALID"):
		return "Unable to " + action + ", invalid user specified"
	case strings.Contains(errStr, "PEER_ID_INVALID"):
		return "Unable to " + action + ", invalid user or chat"
	case strings.Contains(errStr, "USER_PRIVACY_RESTRICTED"):
		return "Unable to " + action + ", user's privacy settings prevent this"
	case strings.Contains(errStr, "RIGHT_FORBIDDEN"):
		return "Unable to " + action + ", I don't have the required permission"
	case strings.Contains(errStr, "ADMIN_RANK_INVALID"):
		return "Unable to " + action + ", custom title is too long or contains invalid characters"
	case strings.Contains(errStr, "ADMIN_RANK_EMOJI_NOT_ALLOWED"):
		return "Unable to " + action + ", custom title cannot contain emojis"
	case strings.Contains(errStr, "USER_RESTRICTED"):
		return "Unable to " + action + ", user is globally restricted by Telegram"
	default:
		return "Failed to " + action + ": " + errStr
	}
}

func parseAdminDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, nil
	}

	s = strings.ReplaceAll(s, "minutes", "m")
	s = strings.ReplaceAll(s, "minute", "m")
	s = strings.ReplaceAll(s, "mins", "m")
	s = strings.ReplaceAll(s, "min", "m")
	s = strings.ReplaceAll(s, "hours", "h")
	s = strings.ReplaceAll(s, "hour", "h")
	s = strings.ReplaceAll(s, "hrs", "h")
	s = strings.ReplaceAll(s, "hr", "h")
	s = strings.ReplaceAll(s, "days", "d")
	s = strings.ReplaceAll(s, "day", "d")
	s = strings.ReplaceAll(s, "weeks", "w")
	s = strings.ReplaceAll(s, "week", "w")
	s = strings.ReplaceAll(s, "seconds", "s")
	s = strings.ReplaceAll(s, "secs", "s")
	s = strings.ReplaceAll(s, "sec", "s")

	var total time.Duration
	numBuf := ""

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			numBuf += string(c)
		} else if c == 'd' || c == 'w' || c == 'h' || c == 'm' || c == 's' {
			if numBuf == "" {
				numBuf = "1"
			}
			num := 0
			for _, d := range numBuf {
				num = num*10 + int(d-'0')
			}
			numBuf = ""

			switch c {
			case 'w':
				total += time.Duration(num) * 7 * 24 * time.Hour
			case 'd':
				total += time.Duration(num) * 24 * time.Hour
			case 'h':
				total += time.Duration(num) * time.Hour
			case 'm':
				total += time.Duration(num) * time.Minute
			case 's':
				total += time.Duration(num) * time.Second
			}
		}
	}

	if numBuf != "" {
		num := 0
		for _, d := range numBuf {
			num = num*10 + int(d-'0')
		}
		total += time.Duration(num) * time.Minute
	}

	if total == 0 {
		return time.ParseDuration(s)
	}

	return total, nil
}

func PromoteUserHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "promote") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "promote") {
		m.Reply("I need the 'Add Admins' right to promote users")
		return nil
	}

	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
	}

	if reason == "" {
		reason = "Admin"
	}

	done, err := m.Client.EditAdmin(m.ChatID(), user, &tg.AdminOptions{Rank: reason, Rights: &tg.ChatAdminRights{
		ChangeInfo:     true,
		PinMessages:    true,
		DeleteMessages: true,
		ManageCall:     true,
		BanUsers:       true,
	}})

	if err != nil || !done {
		m.Reply(parseAdminError(err, "promote user"))
		return nil
	}

	m.Reply("User promoted to admin with custom title: " + strconv.Quote(reason))
	return nil
}

func IDHandle(message *tg.NewMessage) error {
	senderID := message.SenderID()
	chatID := message.ChatID()

	var forwardedID int
	var forwardedType string

	var repliedToUserID int
	var repliedMessageID int
	var repliedMediaID string
	var repliedForwardedID int
	var repliedForwardedType string

	if message.IsForward() {
		forwardedFrom := message.Message.FwdFrom.FromID
		switch forwardedFrom.(type) {
		case *tg.PeerChannel:
			forwardedType = "Channel"
		case *tg.PeerUser:
			forwardedType = "User"
		case *tg.PeerChat:
			forwardedType = "Chat"
		}
		forwardedID = int(message.Client.GetPeerID(forwardedFrom))
	}

	if message.IsReply() {
		repliedMessage, err := message.GetReplyMessage()
		if err != nil {
			return err
		}
		repliedToUserID = int(repliedMessage.SenderID())
		repliedMessageID = int(repliedMessage.ID)
		if repliedMessage.IsMedia() {
			repliedMediaID = repliedMessage.File.FileID
		}

		if repliedMessage.IsForward() {
			repliedForwardedFrom := repliedMessage.Message.FwdFrom.FromID
			switch repliedForwardedFrom.(type) {
			case *tg.PeerChannel:
				repliedForwardedType = "Channel"
			case *tg.PeerUser:
				repliedForwardedType = "User"
			case *tg.PeerChat:
				repliedForwardedType = "Chat"
			}
			repliedForwardedID = int(message.Client.GetPeerID(repliedForwardedFrom))
		}
	}

	var output string
	output = "<b>User:</b> <code>" + strconv.Itoa(int(senderID)) + "</code>\n"
	output += "<b>Chat:</b> <code>" + strconv.Itoa(int(chatID)) + "</code>"

	if forwardedID != 0 {
		output += "\n\n<b>Forwarded From:</b> <code>" + strconv.Itoa(forwardedID) + "</code> (" + forwardedType + ")"
	}

	if repliedToUserID != 0 {
		output += "\n\n<b>Reply To:</b> <code>" + strconv.Itoa(repliedToUserID) + "</code>"
		output += "\n<b>Reply MsgID:</b> <code>" + strconv.Itoa(repliedMessageID) + "</code>"
		if repliedMediaID != "" {
			output += "\n<b>Reply FileID:</b> <code>" + repliedMediaID + "</code>"
		}
		if repliedForwardedID != 0 {
			output += "\n<b>Reply Fwd:</b> <code>" + strconv.Itoa(repliedForwardedID) + "</code> (" + repliedForwardedType + ")"
		}
	}

	message.Reply(output)
	return nil
}

func DemoteUserHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "promote") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "promote") {
		m.Reply("I need the 'Add Admins' right to demote users")
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	done, err := m.Client.EditAdmin(m.ChatID(), user, &tg.AdminOptions{IsAdmin: false})
	if err != nil || !done {
		m.Reply(parseAdminError(err, "demote user"))
		return nil
	}

	m.Reply("User demoted from admin")
	return nil
}

func BanUserHandle(m *tg.NewMessage) error {
	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("ban_%d", targetID)) {
		return nil
	}

	return performBan(m.Client, m.ChatID(), user, reason)
}

func performBan(client *tg.Client, chatID int64, user tg.InputPeer, reason string) error {
	channel, _ := client.GetChannel(chatID)
	if !CanBot(client, channel, "ban") {
		return errors.New("I need 'Ban Users' permission")
	}

	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Ban: true})
	if err != nil || !done {
		return err
	}

	msg := "User banned"
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	client.SendMessage(chatID, msg)
	return nil
}

func UnbanUserHandle(m *tg.NewMessage) error {
	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("unban_%d", targetID)) {
		return nil
	}

	return performUnban(m.Client, m.ChatID(), user)
}

func performUnban(client *tg.Client, chatID int64, user tg.InputPeer) error {
	channel, _ := client.GetChannel(chatID)
	if !CanBot(client, channel, "ban") {
		return errors.New("I need 'Ban Users' permission")
	}

	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Unban: true})
	if err != nil || !done {
		return err
	}

	client.SendMessage(chatID, "User unbanned")
	return nil
}

func KickUserHandle(m *tg.NewMessage) error {
	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("kick_%d", targetID)) {
		return nil
	}

	return performKick(m.Client, m.ChatID(), user, reason)
}

func performKick(client *tg.Client, chatID int64, user tg.InputPeer, reason string) error {
	channel, _ := client.GetChannel(chatID)
	if !CanBot(client, channel, "ban") {
		return errors.New("I need 'Ban Users' permission")
	}

	done, err := client.KickParticipant(chatID, user)
	if err != nil || !done {
		return err
	}

	msg := "User kicked"
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	client.SendMessage(chatID, msg)
	return nil
}

func FullPromoteHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "promote") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "promote") {
		m.Reply("I need the 'Add Admins' right to promote users")
		return nil
	}

	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if reason == "" {
		reason = "Admin"
	}

	done, err := m.Client.EditAdmin(m.ChatID(), user, &tg.AdminOptions{Rank: reason, Rights: &tg.ChatAdminRights{
		ChangeInfo:     true,
		PostMessages:   true,
		EditMessages:   true,
		DeleteMessages: true,
		BanUsers:       true,
		InviteUsers:    true,
		PinMessages:    true,
		ManageCall:     true,
		AddAdmins:      true,
		Anonymous:      false,
		ManageTopics:   true,
	}})

	if err != nil || !done {
		m.Reply(parseAdminError(err, "promote user"))
		return nil
	}

	m.Reply("User fully promoted with all admin rights\n<b>Title:</b> " + strconv.Quote(reason))
	return nil
}

func TbanUserHandle(m *tg.NewMessage) error {
	user, args, err := GetUserFromContext(m)
	if err != nil {
		m.Reply(err.Error())
		return nil
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		m.Reply("Usage: /tban <user> <duration> [reason]")
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	op := fmt.Sprintf("tban_%d_%s", targetID, parts[0])

	if !checkAdmin(m, "ban", op) {
		return nil
	}

	reason := ""
	if len(parts) > 1 {
		reason = strings.Join(parts[1:], " ")
	}

	return performTban(m.Client, m.ChatID(), user, parts[0], reason)
}

func performTban(client *tg.Client, chatID int64, user tg.InputPeer, durationStr, reason string) error {
	channel, _ := client.GetChannel(chatID)
	if !CanBot(client, channel, "ban") {
		return errors.New("I need 'Ban Users' permission")
	}

	duration, err := parseAdminDuration(durationStr)
	if err != nil || duration == 0 {
		return errors.New("Invalid duration")
	}

	untilDate := int32(time.Now().Add(duration).Unix())
	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Ban: true, TillDate: untilDate})
	if err != nil || !done {
		return err
	}

	msg := "User temporarily banned for " + duration.String()
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	client.SendMessage(chatID, msg)
	return nil
}

func TmuteUserHandle(m *tg.NewMessage) error {
	user, args, err := GetUserFromContext(m)
	if err != nil {
		m.Reply(err.Error())
		return nil
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		m.Reply("Usage: /tmute <user> <duration> [reason]")
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	op := fmt.Sprintf("tmute_%d_%s", targetID, parts[0])

	if !checkAdmin(m, "ban", op) {
		return nil
	}

	reason := ""
	if len(parts) > 1 {
		reason = strings.Join(parts[1:], " ")
	}

	return performTmute(m.Client, m.ChatID(), user, parts[0], reason)
}

func performTmute(client *tg.Client, chatID int64, user tg.InputPeer, durationStr, reason string) error {
	channel, _ := client.GetChannel(chatID)
	if !CanBot(client, channel, "ban") {
		return errors.New("I need 'Ban Users' permission")
	}

	duration, err := parseAdminDuration(durationStr)
	if err != nil || duration == 0 {
		return errors.New("Invalid duration")
	}

	untilDate := int32(time.Now().Add(duration).Unix())
	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{
		Mute:     true,
		TillDate: untilDate,
	})
	if err != nil || !done {
		return err
	}

	msg := "User temporarily muted for " + duration.String()
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	client.SendMessage(chatID, msg)
	return nil
}

func MuteUserHandle(m *tg.NewMessage) error {
	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("mute_%d", targetID)) {
		return nil
	}

	return performMute(m.Client, m.ChatID(), user, reason)
}

func performMute(client *tg.Client, chatID int64, user tg.InputPeer, reason string) error {
	channel, _ := client.GetChannel(chatID)
	if !CanBot(client, channel, "ban") {
		return errors.New("I need 'Ban Users' permission")
	}

	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Mute: true})
	if err != nil || !done {
		return err
	}

	msg := "User muted"
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	client.SendMessage(chatID, msg)
	return nil
}

func UnmuteUserHandle(m *tg.NewMessage) error {
	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("unmute_%d", targetID)) {
		return nil
	}

	return performUnmute(m.Client, m.ChatID(), user)
}

func performUnmute(client *tg.Client, chatID int64, user tg.InputPeer) error {
	channel, _ := client.GetChannel(chatID)
	if !CanBot(client, channel, "ban") {
		return errors.New("I need 'Ban Users' permission")
	}

	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Unmute: true})
	if err != nil || !done {
		return err
	}

	client.SendMessage(chatID, "User unmuted")
	return nil
}

func SbanUserHandle(m *tg.NewMessage) error {
	m.Delete()

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		return nil
	}

	m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Ban: true})
	return nil
}

func SmuteUserHandle(m *tg.NewMessage) error {
	m.Delete()

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		return nil
	}

	m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Mute: true})
	return nil
}

func SkickUserHandle(m *tg.NewMessage) error {
	m.Delete()

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		return nil
	}

	m.Client.KickParticipant(m.ChatID(), user)
	return nil
}

const AnonBotID = 1087968824

func checkAdmin(m *tg.NewMessage, right, callbackData string) bool {
	senderID := int(m.SenderID())
	chatID := int(m.ChatID())

	if IsUserAdmin(m.Client, senderID, chatID, right) {
		return true
	}

	if senderID == AnonBotID || int64(senderID) == m.ChatID() {
		b := tg.Button
		kb := tg.NewKeyboard().AddRow(b.Data("Verify Admin Rights", "verify_op_"+callbackData)).Build()
		m.Reply("Click to verify admin privileges to perform this action.", &tg.SendOptions{ReplyMarkup: kb})
		return false
	}

	m.Reply("You need to be an admin to use this command")
	return false
}

func AdminVerifyCallback(c *tg.CallbackQuery) error {
	data := c.DataString()
	if !strings.HasPrefix(data, "verify_op_") {
		return nil
	}

	op := strings.TrimPrefix(data, "verify_op_")
	parts := strings.Split(op, "_")
	if len(parts) < 2 {
		c.Answer("Invalid callback data", &tg.CallbackOptions{Alert: true})
		return nil
	}

	action := parts[0]
	// right is assumed check for "ban" for now as most supported commands are ban/kick/mute
	right := "ban"

	if !IsUserAdmin(c.Client, int(c.SenderID), int(c.ChatID), right) {
		c.Answer("You don't have permission!", &tg.CallbackOptions{Alert: true})
		return nil
	}

	targetIDStr := parts[1]
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil {
		c.Answer("Invalid target peer", &tg.CallbackOptions{Alert: true})
		return nil
	}

	user, err := c.Client.ResolvePeer(targetID)
	if err != nil {
		c.Answer("Could not resolve user", &tg.CallbackOptions{Alert: true})
		return nil
	}

	var opErr error
	switch action {
	case "ban":
		opErr = performBan(c.Client, c.ChatID, user, "")
	case "unban":
		opErr = performUnban(c.Client, c.ChatID, user)
	case "kick":
		opErr = performKick(c.Client, c.ChatID, user, "")
	case "mute":
		opErr = performMute(c.Client, c.ChatID, user, "")
	case "unmute":
		opErr = performUnmute(c.Client, c.ChatID, user)
	case "tban":
		if len(parts) < 3 {
			opErr = errors.New("missing duration")
		} else {
			opErr = performTban(c.Client, c.ChatID, user, parts[2], "")
		}
	case "tmute":
		if len(parts) < 3 {
			opErr = errors.New("missing duration")
		} else {
			opErr = performTmute(c.Client, c.ChatID, user, parts[2], "")
		}
	}

	if opErr != nil {
		c.Answer("Error: "+opErr.Error(), &tg.CallbackOptions{Alert: true})
	} else {
		c.Answer("Action executed")
		c.Delete()
	}

	return nil
}
