package modules

import (
	"strconv"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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
		m.Reply("Failed to promote user!")
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
		m.Reply("Failed to demote user!")
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
		m.Reply("Failed to ban user!")
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
		m.Reply("Failed to unban user!")
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

	// Kick = ban then immediately unban
	done, err := m.Client.KickParticipant(m.ChatID(), user)
	if err != nil || !done {
		m.Reply("Failed to kick user!")
		return nil
	}

	msg := "User kicked"
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	m.Reply(msg)
	return nil
}
