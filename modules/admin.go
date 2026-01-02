package modules

import (
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
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need the 'Ban Users' right to ban users")
		return nil
	}

	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	done, err := m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Ban: true})
	if err != nil || !done {
		m.Reply(parseAdminError(err, "ban user"))
		return nil
	}

	msg := "User banned"
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	m.Reply(msg)
	return nil
}

func UnbanUserHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need the 'Ban Users' right to unban users")
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	done, err := m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Unban: true})
	if err != nil || !done {
		m.Reply(parseAdminError(err, "unban user"))
		return nil
	}

	m.Reply("User unbanned")
	return nil
}

func KickUserHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need the 'Ban Users' right to kick users")
		return nil
	}

	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	done, err := m.Client.KickParticipant(m.ChatID(), user)
	if err != nil || !done {
		m.Reply(parseAdminError(err, "kick user"))
		return nil
	}

	msg := "User kicked"
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	m.Reply(msg)
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
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need the 'Ban Users' right to ban users")
		return nil
	}

	user, args, err := GetUserFromContext(m)
	if err != nil {
		m.Reply(err.Error())
		return nil
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		m.Reply("Usage: /tban <user> <duration> [reason]\nExample: /tban @user 1h Spamming")
		return nil
	}

	duration, err := parseAdminDuration(parts[0])
	if err != nil || duration == 0 {
		m.Reply("Invalid duration. Use formats like: 30m, 1h, 2d")
		return nil
	}

	reason := ""
	if len(parts) > 1 {
		reason = strings.Join(parts[1:], " ")
	}

	untilDate := int32(time.Now().Add(duration).Unix())
	done, err := m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Ban: true, TillDate: untilDate})
	if err != nil || !done {
		m.Reply(parseAdminError(err, "ban user"))
		return nil
	}

	msg := "User temporarily banned for " + duration.String()
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	m.Reply(msg)
	return nil
}

func TmuteUserHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need the 'Ban Users' right to mute users")
		return nil
	}

	user, args, err := GetUserFromContext(m)
	if err != nil {
		m.Reply(err.Error())
		return nil
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		m.Reply("Usage: /tmute <user> <duration> [reason]\nExample: /tmute @user 1h")
		return nil
	}

	duration, err := parseAdminDuration(parts[0])
	if err != nil || duration == 0 {
		m.Reply("Invalid duration. Use formats like: 30m, 1h, 2d")
		return nil
	}

	reason := ""
	if len(parts) > 1 {
		reason = strings.Join(parts[1:], " ")
	}

	untilDate := int32(time.Now().Add(duration).Unix())
	done, err := m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{
		Mute:     true,
		TillDate: untilDate,
	})
	if err != nil || !done {
		m.Reply(parseAdminError(err, "mute user"))
		return nil
	}

	msg := "User temporarily muted for " + duration.String()
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	m.Reply(msg)
	return nil
}

func MuteUserHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need the 'Ban Users' right to mute users")
		return nil
	}

	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply(err.Error())
		return nil
	}

	done, err := m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Mute: true})
	if err != nil || !done {
		m.Reply(parseAdminError(err, "mute user"))
		return nil
	}

	msg := "User muted"
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	m.Reply(msg)
	return nil
}

func UnmuteUserHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need to be an admin to use this command")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need the 'Ban Users' right to unmute users")
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply(err.Error())
		return nil
	}

	done, err := m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Unmute: true})
	if err != nil || !done {
		m.Reply(parseAdminError(err, "unmute user"))
		return nil
	}

	m.Reply("User unmuted")
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
