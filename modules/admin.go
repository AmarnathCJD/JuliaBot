package modules

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func PromoteUserHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "promote") {
		m.Reply("You don't have permission to do that here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "promote") {
		m.Reply("I need admin permission to add admins in this chat.")
		return nil
	}

	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to promote. " + adminUsage("promote"))
		return nil
	}

	if reason == "" {
		reason = "Admin"
	}

	done, err := m.Client.EditAdmin(m.ChatID(), user, &tg.AdminOptions{IsAdmin: true, Rank: reason, Rights: &tg.ChatAdminRights{
		ChangeInfo:     true,
		PinMessages:    true,
		DeleteMessages: true,
		ManageCall:     true,
		BanUsers:       true,
	}})

	if err != nil || !done {
		m.Reply(adminFriendlyError(err, "promote"))
		return nil
	}

	m.Reply("Done. Promoted with title: <code>" + reason + "</code>")
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
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "promote") {
		m.Reply("You don't have permission to do that here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "promote") {
		m.Reply("I need admin permission to manage admins in this chat.")
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to demote. " + adminUsage("demote"))
		return nil
	}

	done, err := m.Client.EditAdmin(m.ChatID(), user, &tg.AdminOptions{IsAdmin: false})
	if err != nil || !done {
		m.Reply(adminFriendlyError(err, "demote"))
		return nil
	}

	m.Reply("Done. Admin rights removed.")
	return nil
}

func BanUserHandle(m *tg.NewMessage) error {
	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to ban. " + adminUsage("ban"))
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("ban_%d", targetID)) {
		return nil
	}

	msg, opErr := performBan(m.Client, m.ChatID(), user, reason, m.SenderID())
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "ban"))
		return nil
	}

	b := tg.Button
	m.Reply(msg, &tg.SendOptions{
		ReplyMarkup: tg.NewKeyboard().AddRow(
			b.Data("Undo Ban", fmt.Sprintf("undo_ban_%d_%d", targetID, m.SenderID())).Danger(),
		).Build(),
	})
	return nil
}

func performBan(client *tg.Client, chatID int64, user tg.InputPeer, reason string, adminID int64) (string, error) {
	channel, _ := client.GetChannel(chatID)
	if channel != nil && !CanBot(client, channel, "ban") {
		return "", errors.New("missing bot rights")
	}

	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Ban: true})
	if err != nil || !done {
		return "", err
	}

	userID := client.GetPeerID(user)
	RecordAction(chatID, userID, adminID, "ban", map[string]interface{}{"reason": reason})

	name := GetPeerDisplayName(client, user)
	msg := fmt.Sprintf("Done. %s has been banned.", name)
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	return msg, nil
}

func UnbanUserHandle(m *tg.NewMessage) error {
	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to unban. " + adminUsage("unban"))
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("unban_%d", targetID)) {
		return nil
	}

	msg, opErr := performUnban(m.Client, m.ChatID(), user)
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "unban"))
		return nil
	}
	m.Reply(msg)
	return nil
}

func performUnban(client *tg.Client, chatID int64, user tg.InputPeer) (string, error) {
	channel, _ := client.GetChannel(chatID)
	if channel != nil && !CanBot(client, channel, "ban") {
		return "", errors.New("missing bot rights")
	}

	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Unban: true})
	if err != nil || !done {
		return "", err
	}
	name := GetPeerDisplayName(client, user)
	return fmt.Sprintf("Done. %s has been unbanned.", name), nil
}

func KickUserHandle(m *tg.NewMessage) error {
	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to kick. " + adminUsage("kick"))
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("kick_%d", targetID)) {
		return nil
	}

	msg, opErr := performKick(m.Client, m.ChatID(), user, reason)
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "kick"))
		return nil
	}
	m.Reply(msg)
	return nil
}

func performKick(client *tg.Client, chatID int64, user tg.InputPeer, reason string) (string, error) {
	channel, _ := client.GetChannel(chatID)
	if channel != nil && !CanBot(client, channel, "ban") {
		return "", errors.New("missing bot rights")
	}

	done, err := client.KickParticipant(chatID, user)
	if err != nil || !done {
		return "", err
	}

	name := GetPeerDisplayName(client, user)
	msg := fmt.Sprintf("Done. %s has been kicked.", name)
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	return msg, nil
}

func FullPromoteHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "promote") {
		m.Reply("You don't have permission to do that here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "promote") {
		m.Reply("I need admin permission to add admins in this chat.")
		return nil
	}

	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to promote. " + adminUsage("promote"))
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
		m.Reply(adminFriendlyError(err, "promote"))
		return nil
	}

	m.Reply("Done. Promoted with full admin rights.\n<b>Title:</b> <code>" + reason + "</code>")
	return nil
}

func TbanUserHandle(m *tg.NewMessage) error {
	user, args, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to temp-ban. " + adminUsage("tban"))
		return nil
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		m.Reply(adminUsage("tban"))
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

	msg, opErr := performTban(m.Client, m.ChatID(), user, parts[0], reason, m.SenderID())
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "temp-ban"))
		return nil
	}

	b := tg.Button
	m.Reply(msg, &tg.SendOptions{
		ReplyMarkup: tg.NewKeyboard().AddRow(
			b.Data("Undo Ban", fmt.Sprintf("undo_tban_%d_%d", targetID, m.SenderID())).Danger(),
		).Build(),
	})
	return nil
}

func performTban(client *tg.Client, chatID int64, user tg.InputPeer, durationStr, reason string, adminID int64) (string, error) {
	channel, _ := client.GetChannel(chatID)
	if channel != nil && !CanBot(client, channel, "ban") {
		return "", errors.New("missing bot rights")
	}

	duration, err := parseAdminDuration(durationStr)
	if err != nil || duration == 0 {
		return "", errors.New("invalid duration")
	}

	untilDate := int32(time.Now().Add(duration).Unix())
	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Ban: true, TillDate: untilDate})
	if err != nil || !done {
		return "", err
	}

	userID := client.GetPeerID(user)
	RecordAction(chatID, userID, adminID, "tban", map[string]interface{}{"reason": reason, "duration": duration.String()})

	name := GetPeerDisplayName(client, user)
	msg := fmt.Sprintf("Done. %s has been banned for %s.", name, formatAdminDuration(duration))
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	return msg, nil
}

func TmuteUserHandle(m *tg.NewMessage) error {
	user, args, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to temp-mute. " + adminUsage("tmute"))
		return nil
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		m.Reply(adminUsage("tmute"))
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

	msg, opErr := performTmute(m.Client, m.ChatID(), user, parts[0], reason, m.SenderID())
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "temp-mute"))
		return nil
	}

	b := tg.Button
	m.Reply(msg, &tg.SendOptions{
		ReplyMarkup: tg.NewKeyboard().AddRow(
			b.Data("Undo Mute", fmt.Sprintf("undo_tmute_%d_%d", targetID, m.SenderID())),
		).Build(),
	})
	return nil
}

func performTmute(client *tg.Client, chatID int64, user tg.InputPeer, durationStr, reason string, adminID int64) (string, error) {
	channel, _ := client.GetChannel(chatID)
	if channel != nil && !CanBot(client, channel, "ban") {
		return "", errors.New("missing bot rights")
	}

	duration, err := parseAdminDuration(durationStr)
	if err != nil || duration == 0 {
		return "", errors.New("invalid duration")
	}

	untilDate := int32(time.Now().Add(duration).Unix())
	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{
		Mute:     true,
		TillDate: untilDate,
	})
	if err != nil || !done {
		return "", err
	}

	userID := client.GetPeerID(user)
	RecordAction(chatID, userID, adminID, "tmute", map[string]interface{}{"reason": reason, "duration": duration.String()})

	name := GetPeerDisplayName(client, user)
	msg := fmt.Sprintf("Done. %s has been muted for %s.", name, formatAdminDuration(duration))
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	return msg, nil
}

func MuteUserHandle(m *tg.NewMessage) error {
	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to mute. " + adminUsage("mute"))
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("mute_%d", targetID)) {
		return nil
	}

	msg, opErr := performMute(m.Client, m.ChatID(), user, reason, m.SenderID())
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "mute"))
		return nil
	}

	b := tg.Button
	m.Reply(msg, &tg.SendOptions{
		ReplyMarkup: tg.NewKeyboard().AddRow(
			b.Data("Undo Mute", fmt.Sprintf("undo_mute_%d_%d", targetID, m.SenderID())),
		).Build(),
	})
	return nil
}

func performMute(client *tg.Client, chatID int64, user tg.InputPeer, reason string, adminID int64) (string, error) {
	channel, _ := client.GetChannel(chatID)
	if channel != nil && !CanBot(client, channel, "ban") {
		return "", errors.New("missing bot rights")
	}

	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Mute: true})
	if err != nil || !done {
		return "", err
	}

	userID := client.GetPeerID(user)
	RecordAction(chatID, userID, adminID, "mute", map[string]interface{}{"reason": reason})

	name := GetPeerDisplayName(client, user)
	msg := fmt.Sprintf("Done. %s has been muted.", name)
	if reason != "" {
		msg += "\n<b>Reason:</b> " + reason
	}
	return msg, nil
}

func UnmuteUserHandle(m *tg.NewMessage) error {
	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("I couldn't find who to unmute. " + adminUsage("unmute"))
		return nil
	}

	targetID := m.Client.GetPeerID(user)
	if !checkAdmin(m, "ban", fmt.Sprintf("unmute_%d", targetID)) {
		return nil
	}

	msg, opErr := performUnmute(m.Client, m.ChatID(), user)
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "unmute"))
		return nil
	}
	m.Reply(msg)
	return nil
}

func performUnmute(client *tg.Client, chatID int64, user tg.InputPeer) (string, error) {
	channel, _ := client.GetChannel(chatID)
	if channel != nil && !CanBot(client, channel, "ban") {
		return "", errors.New("missing bot rights")
	}

	done, err := client.EditBanned(chatID, user, &tg.BannedOptions{Unmute: true})
	if err != nil || !done {
		return "", err
	}
	name := GetPeerDisplayName(client, user)
	return fmt.Sprintf("Done. %s has been unmuted.", name), nil
}

func SbanUserHandle(m *tg.NewMessage) error {
	m.Delete()

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		replyTemp(m, "You don't have permission to do that here.", 5)
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		replyTemp(m, "I need admin permission to ban users in this chat.", 5)
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		replyTemp(m, "I couldn't find who to ban. "+adminUsage("ban"), 6)
		return nil
	}
	_, opErr := performBan(m.Client, m.ChatID(), user, "", m.SenderID())
	if opErr != nil {
		replyTemp(m, adminFriendlyError(opErr, "ban"), 6)
		return nil
	}
	replyTemp(m, "Done.", 3)
	return nil
}

func SmuteUserHandle(m *tg.NewMessage) error {
	m.Delete()

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		replyTemp(m, "You don't have permission to do that here.", 5)
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		replyTemp(m, "I need admin permission to mute users in this chat.", 5)
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		replyTemp(m, "I couldn't find who to mute. "+adminUsage("mute"), 6)
		return nil
	}
	_, opErr := performMute(m.Client, m.ChatID(), user, "", m.SenderID())
	if opErr != nil {
		replyTemp(m, adminFriendlyError(opErr, "mute"), 6)
		return nil
	}
	replyTemp(m, "Done.", 3)
	return nil
}

func SkickUserHandle(m *tg.NewMessage) error {
	m.Delete()

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		replyTemp(m, "You don't have permission to do that here.", 5)
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		replyTemp(m, "I need admin permission to kick users in this chat.", 5)
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		replyTemp(m, "I couldn't find who to kick. "+adminUsage("kick"), 6)
		return nil
	}
	_, opErr := performKick(m.Client, m.ChatID(), user, "")
	if opErr != nil {
		replyTemp(m, adminFriendlyError(opErr, "kick"), 6)
		return nil
	}
	replyTemp(m, "Done.", 3)
	return nil
}

func DeleteMessageHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		m.Reply("You don't have permission to delete messages here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "delete") {
		m.Reply("I need admin permission to delete messages in this chat.")
		return nil
	}
	if !m.IsReply() {
		m.Reply(adminUsage("del"))
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("I couldn't read the replied message. Please try again.")
		return nil
	}
	m.Client.DeleteMessages(m.ChatID(), []int32{int32(reply.ID)})
	m.Reply("Done. Message deleted.")
	return nil
}

func DBanUserHandle(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply(adminUsage("dban"))
		return nil
	}
	// Need both: delete + ban.
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") || !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		m.Reply("You don't have permission to do that here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") || !CanBot(m.Client, m.Channel, "delete") {
		m.Reply("I need admin permission to delete messages and ban users in this chat.")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("I couldn't read the replied message. Please try again.")
		return nil
	}
	peer, err := m.Client.ResolvePeer(reply.Sender)
	if err != nil {
		m.Reply("I couldn't identify the sender of that message.")
		return nil
	}
	// Delete the offending message first.
	m.Client.DeleteMessages(m.ChatID(), []int32{int32(reply.ID)})
	msg, opErr := performBan(m.Client, m.ChatID(), peer, strings.TrimSpace(m.Args()), m.SenderID())
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "ban"))
		return nil
	}
	m.Reply(msg)
	return nil
}

func DMuteUserHandle(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply(adminUsage("dmute"))
		return nil
	}
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") || !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		m.Reply("You don't have permission to do that here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") || !CanBot(m.Client, m.Channel, "delete") {
		m.Reply("I need admin permission to delete messages and mute users in this chat.")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("I couldn't read the replied message. Please try again.")
		return nil
	}
	peer, err := m.Client.ResolvePeer(reply.Sender)
	if err != nil {
		m.Reply("I couldn't identify the sender of that message.")
		return nil
	}
	m.Client.DeleteMessages(m.ChatID(), []int32{int32(reply.ID)})
	msg, opErr := performMute(m.Client, m.ChatID(), peer, strings.TrimSpace(m.Args()), m.SenderID())
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "mute"))
		return nil
	}
	m.Reply(msg)
	return nil
}

func DKickUserHandle(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply(adminUsage("dkick"))
		return nil
	}
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") || !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		m.Reply("You don't have permission to do that here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") || !CanBot(m.Client, m.Channel, "delete") {
		m.Reply("I need admin permission to delete messages and kick users in this chat.")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("I couldn't read the replied message. Please try again.")
		return nil
	}
	peer, err := m.Client.ResolvePeer(reply.Sender)
	if err != nil {
		m.Reply("I couldn't identify the sender of that message.")
		return nil
	}
	m.Client.DeleteMessages(m.ChatID(), []int32{int32(reply.ID)})
	msg, opErr := performKick(m.Client, m.ChatID(), peer, strings.TrimSpace(m.Args()))
	if opErr != nil {
		m.Reply(adminFriendlyError(opErr, "kick"))
		return nil
	}
	m.Reply(msg)
	return nil
}

func PinMessageHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "pin") {
		m.Reply("You don't have permission to pin messages here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "pin") {
		m.Reply("I need admin permission to pin messages in this chat.")
		return nil
	}
	if !m.IsReply() {
		m.Reply("Reply to the message you want to pin.")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("I couldn't read the replied message. Please try again.")
		return nil
	}

	notify := true
	args := strings.ToLower(strings.TrimSpace(m.Args()))
	if strings.Contains(args, "silent") || strings.Contains(args, "notify") {
		notify = false
	}

	_, err = m.Client.PinMessage(m.ChatID(), int32(reply.ID), &tg.PinOptions{Silent: !notify})
	if err != nil {
		m.Reply(adminFriendlyError(err, "pin message"))
		return nil
	}
	m.Reply("Done. Message pinned.")
	return nil
}

func UnpinMessageHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "pin") {
		m.Reply("You don't have permission to unpin messages here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "pin") {
		m.Reply("I need admin permission to unpin messages in this chat.")
		return nil
	}

	if !m.IsReply() {
		m.Reply("Reply to the pinned message you want to unpin, or use `/unpin all` to unpin all messages.")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("I couldn't read the replied message. Please try again.")
		return nil
	}

	_, err = m.Client.UnpinMessage(m.ChatID(), int32(reply.ID))
	if err != nil {
		m.Reply(adminFriendlyError(err, "unpin message"))
		return nil
	}
	m.Reply("Done. Message unpinned.")
	return nil
}

func PurgeMessagesHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		m.Reply("You don't have permission to delete messages here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "delete") {
		m.Reply("I need admin permission to delete messages in this chat.")
		return nil
	}
	if !m.IsReply() {
		m.Reply("Reply to the starting message. I'll delete all messages from that message to this command.")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("I couldn't read the replied message. Please try again.")
		return nil
	}

	startMsgID := int32(reply.ID)
	endMsgID := int32(m.ID)

	if startMsgID >= endMsgID {
		m.Reply("The replied message must be older than this command message.")
		return nil
	}

	// Telegram allows max 100 messages per request
	var msgIDs []int32
	for i := startMsgID; i <= endMsgID; i++ {
		msgIDs = append(msgIDs, i)
	}

	if len(msgIDs) > 100 {
		m.Reply(fmt.Sprintf("⚠️ Purging %d messages...", len(msgIDs)))

		// Delete in batches of 100
		for i := 0; i < len(msgIDs); i += 100 {
			end := i + 100
			if end > len(msgIDs) {
				end = len(msgIDs)
			}
			m.Client.DeleteMessages(m.ChatID(), msgIDs[i:end])
			time.Sleep(500 * time.Millisecond) // Rate limiting
		}
	} else {
		m.Client.DeleteMessages(m.ChatID(), msgIDs)
	}

	statusMsg, _ := m.Reply(fmt.Sprintf("✅ Purged %d messages.", len(msgIDs)))

	// Auto-delete the status message after 5 seconds
	if statusMsg != nil {
		time.Sleep(5 * time.Second)
		m.Client.DeleteMessages(m.ChatID(), []int32{int32(statusMsg.ID)})
	}

	return nil
}

func LockHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		m.Reply("You don't have permission to lock chat settings here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need admin permission to manage chat restrictions.")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))
	if args == "" {
		m.Reply("Please specify what to lock: `all`, `messages`, `media`, `stickers`, `gifs`, `polls`, `invite`, `pin`, `info`")
		return nil
	}

	channel, err := m.Client.GetChannel(m.ChatID())
	if err != nil {
		m.Reply("This command only works in supergroups.")
		return nil
	}

	defaultRights := channel.DefaultBannedRights
	if defaultRights == nil {
		defaultRights = &tg.ChatBannedRights{}
	}

	updated := false
	lockMsg := "🔒 Locked: "

	switch args {
	case "all":
		defaultRights.SendMessages = true
		defaultRights.SendMedia = true
		defaultRights.SendStickers = true
		defaultRights.SendGifs = true
		defaultRights.SendGames = true
		defaultRights.SendInline = true
		defaultRights.SendPolls = true
		defaultRights.ChangeInfo = true
		defaultRights.InviteUsers = true
		defaultRights.PinMessages = true
		lockMsg += "All permissions"
		updated = true
	case "messages", "msg":
		defaultRights.SendMessages = true
		lockMsg += "Messages"
		updated = true
	case "media":
		defaultRights.SendMedia = true
		lockMsg += "Media"
		updated = true
	case "stickers", "sticker":
		defaultRights.SendStickers = true
		lockMsg += "Stickers"
		updated = true
	case "gifs", "gif", "animations":
		defaultRights.SendGifs = true
		lockMsg += "GIFs"
		updated = true
	case "games", "game":
		defaultRights.SendGames = true
		lockMsg += "Games"
		updated = true
	case "inline":
		defaultRights.SendInline = true
		lockMsg += "Inline bots"
		updated = true
	case "polls", "poll":
		defaultRights.SendPolls = true
		lockMsg += "Polls"
		updated = true
	case "invite", "invites":
		defaultRights.InviteUsers = true
		lockMsg += "Invite users"
		updated = true
	case "pin":
		defaultRights.PinMessages = true
		lockMsg += "Pin messages"
		updated = true
	case "info", "change_info":
		defaultRights.ChangeInfo = true
		lockMsg += "Change chat info"
		updated = true
	default:
		m.Reply("Unknown lock type. Available: `all`, `messages`, `media`, `stickers`, `gifs`, `polls`, `invite`, `pin`, `info`")
		return nil
	}

	if updated {
		peerChannel, _ := m.Client.ResolvePeer(m.ChatID())
		_, err := m.Client.MessagesEditChatDefaultBannedRights(peerChannel, defaultRights)
		if err != nil {
			m.Reply(adminFriendlyError(err, "update chat permissions"))
			return nil
		}
		m.Reply(lockMsg)
	}

	return nil
}

func UnlockHandle(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		m.Reply("You don't have permission to unlock chat settings here.")
		return nil
	}
	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need admin permission to manage chat restrictions.")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))
	if args == "" {
		m.Reply("Please specify what to unlock: `all`, `messages`, `media`, `stickers`, `gifs`, `polls`, `invite`, `pin`, `info`")
		return nil
	}

	channel, err := m.Client.GetChannel(m.ChatID())
	if err != nil {
		m.Reply("This command only works in supergroups.")
		return nil
	}

	defaultRights := channel.DefaultBannedRights
	if defaultRights == nil {
		defaultRights = &tg.ChatBannedRights{}
	}

	updated := false
	unlockMsg := "🔓 Unlocked: "

	switch args {
	case "all":
		defaultRights.SendMessages = false
		defaultRights.SendMedia = false
		defaultRights.SendStickers = false
		defaultRights.SendGifs = false
		defaultRights.SendGames = false
		defaultRights.SendInline = false
		defaultRights.SendPolls = false
		defaultRights.ChangeInfo = false
		defaultRights.InviteUsers = false
		defaultRights.PinMessages = false
		unlockMsg += "All permissions"
		updated = true
	case "messages", "msg":
		defaultRights.SendMessages = false
		unlockMsg += "Messages"
		updated = true
	case "media":
		defaultRights.SendMedia = false
		unlockMsg += "Media"
		updated = true
	case "stickers", "sticker":
		defaultRights.SendStickers = false
		unlockMsg += "Stickers"
		updated = true
	case "gifs", "gif", "animations":
		defaultRights.SendGifs = false
		unlockMsg += "GIFs"
		updated = true
	case "games", "game":
		defaultRights.SendGames = false
		unlockMsg += "Games"
		updated = true
	case "inline":
		defaultRights.SendInline = false
		unlockMsg += "Inline bots"
		updated = true
	case "polls", "poll":
		defaultRights.SendPolls = false
		unlockMsg += "Polls"
		updated = true
	case "invite", "invites":
		defaultRights.InviteUsers = false
		unlockMsg += "Invite users"
		updated = true
	case "pin":
		defaultRights.PinMessages = false
		unlockMsg += "Pin messages"
		updated = true
	case "info", "change_info":
		defaultRights.ChangeInfo = false
		unlockMsg += "Change chat info"
		updated = true
	default:
		m.Reply("Unknown unlock type. Available: `all`, `messages`, `media`, `stickers`, `gifs`, `polls`, `invite`, `pin`, `info`")
		return nil
	}

	if updated {
		peerChannel, _ := m.Client.ResolvePeer(m.ChatID())
		_, err := m.Client.MessagesEditChatDefaultBannedRights(peerChannel, defaultRights)
		if err != nil {
			m.Reply(adminFriendlyError(err, "update chat permissions"))
			return nil
		}
		m.Reply(unlockMsg)
	}

	return nil
}

func LocksHandle(m *tg.NewMessage) error {
	channel, err := m.Client.GetChannel(m.ChatID())
	if err != nil {
		m.Reply("This command only works in supergroups.")
		return nil
	}

	defaultRights := channel.DefaultBannedRights
	if defaultRights == nil {
		m.Reply("📋 <b>Current Locks:</b>\n\nNo restrictions are currently active.")
		return nil
	}

	locks := "<b>📋 Current Locks:</b>\n\n"
	lockCount := 0

	checkLock := func(locked bool, name string) {
		if locked {
			locks += "🔒 " + name + "\n"
			lockCount++
		} else {
			locks += "🔓 " + name + "\n"
		}
	}

	checkLock(defaultRights.SendMessages, "Messages")
	checkLock(defaultRights.SendMedia, "Media")
	checkLock(defaultRights.SendStickers, "Stickers")
	checkLock(defaultRights.SendGifs, "GIFs")
	checkLock(defaultRights.SendGames, "Games")
	checkLock(defaultRights.SendInline, "Inline bots")
	checkLock(defaultRights.SendPolls, "Polls")
	checkLock(defaultRights.InviteUsers, "Invite users")
	checkLock(defaultRights.PinMessages, "Pin messages")
	checkLock(defaultRights.ChangeInfo, "Change info")

	if lockCount == 0 {
		locks += "\n<i>No restrictions are currently active.</i>"
	} else {
		locks += fmt.Sprintf("\n<i>Total locked: %d/10</i>", lockCount)
	}

	m.Reply(locks)
	return nil
}

const AnonBotID = 1087968824

func checkAdmin(m *tg.NewMessage, right, callbackData string) bool {
	if IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), right) {
		return true
	}

	if m.SenderID() == AnonBotID || m.SenderID() == m.ChatID() {
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

	requiresBan := map[string]bool{"ban": true, "unban": true, "kick": true, "mute": true, "unmute": true, "tban": true, "tmute": true, "dban": true, "dmute": true, "dkick": true}
	requiresDelete := map[string]bool{"dban": true, "dmute": true, "dkick": true}

	if requiresBan[action] && !IsUserAdmin(c.Client, c.SenderID, c.ChatID, "ban") {
		c.Answer("You don't have permission to do that here.", &tg.CallbackOptions{Alert: true})
		return nil
	}
	if requiresDelete[action] && !IsUserAdmin(c.Client, c.SenderID, c.ChatID, "delete") {
		c.Answer("You don't have permission to delete messages here.", &tg.CallbackOptions{Alert: true})
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
		c.Answer("I couldn't find that user.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	var msgID int32
	if requiresDelete[action] {
		if len(parts) < 3 {
			c.Answer("Missing message to delete.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		mid, err := strconv.Atoi(parts[2])
		if err != nil {
			c.Answer("Invalid message.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		msgID = int32(mid)
	}

	var opErr error
	var resultMsg string
	switch action {
	case "ban":
		resultMsg, opErr = performBan(c.Client, c.ChatID, user, "", c.SenderID)
	case "unban":
		resultMsg, opErr = performUnban(c.Client, c.ChatID, user)
	case "kick":
		resultMsg, opErr = performKick(c.Client, c.ChatID, user, "")
	case "mute":
		resultMsg, opErr = performMute(c.Client, c.ChatID, user, "", c.SenderID)
	case "unmute":
		resultMsg, opErr = performUnmute(c.Client, c.ChatID, user)
	case "tban":
		if len(parts) < 3 {
			opErr = errors.New("missing duration")
		} else {
			resultMsg, opErr = performTban(c.Client, c.ChatID, user, parts[2], "", c.SenderID)
		}
	case "tmute":
		if len(parts) < 3 {
			opErr = errors.New("missing duration")
		} else {
			resultMsg, opErr = performTmute(c.Client, c.ChatID, user, parts[2], "", c.SenderID)
		}
	case "dban":
		if msgID != 0 {
			c.Client.DeleteMessages(c.ChatID, []int32{msgID})
		}
		resultMsg, opErr = performBan(c.Client, c.ChatID, user, "", c.SenderID)
	case "dmute":
		if msgID != 0 {
			c.Client.DeleteMessages(c.ChatID, []int32{msgID})
		}
		resultMsg, opErr = performMute(c.Client, c.ChatID, user, "", c.SenderID)
	case "dkick":
		if msgID != 0 {
			c.Client.DeleteMessages(c.ChatID, []int32{msgID})
		}
		resultMsg, opErr = performKick(c.Client, c.ChatID, user, "")
	}

	if opErr != nil {
		c.Answer(adminFriendlyError(opErr, action), &tg.CallbackOptions{Alert: true})
	} else {
		if resultMsg != "" {
			c.Client.SendMessage(c.ChatID, resultMsg)
		}
		c.Answer("Done")
		c.Delete()
	}

	return nil
}

// HandleReactionUpdate handles emoji reactions on messages
// 📌 = Pin, 🗑️ = Delete, ❤️ = React with heart, ⭐ = React with star
// 🔁 = Forward, ✅ = React with check, 🔖 = Bookmark
func HandleReactionUpdate(update tg.Update, c *tg.Client) error {
	msgUpdate, ok := update.(*tg.UpdateBotMessageReaction)
	if !ok {
		return nil
	}

	peer := msgUpdate.Peer
	msgID := msgUpdate.MsgID

	for _, reaction := range msgUpdate.NewReactions {
		emoji := ""

		if simpleReaction, ok := reaction.(*tg.ReactionEmoji); ok {
			emoji = simpleReaction.Emoticon
		}

		if emoji == "" {
			continue
		}

		switch emoji {
		case "📌":
			if IsUserAdmin(c, c.GetPeerID(msgUpdate.Actor), c.GetPeerID(msgUpdate.Peer), "pin") {
				c.PinMessage(peer, msgID)
			}
		case "💩":
			if IsUserAdmin(c, c.GetPeerID(msgUpdate.Actor), c.GetPeerID(msgUpdate.Peer), "delete") {
				c.DeleteMessages(peer, []int32{msgID})
			}
		}
	}

	return nil
}

//  ------------------ Util Fns ------------------

func adminUsage(action string) string {
	switch action {
	case "promote":
		return "Reply to a user's message or pass their username/ID. You can add an optional custom title after it."
	case "demote":
		return "Reply to a user's message or pass their username/ID."
	case "ban", "unban", "kick", "mute", "unmute":
		return "Reply to a user's message or pass their username/ID. You can add an optional reason after it."
	case "tban", "tmute":
		return "Reply to a user's message or pass their username/ID, then a duration (e.g. 30m, 2h, 1d) and optional reason."
	case "del":
		return "Reply to the message you want to delete."
	case "dban":
		return "Reply to the message you want deleted. I will delete it and ban the sender. You can add an optional reason."
	case "dmute":
		return "Reply to the message you want deleted. I will delete it and mute the sender. You can add an optional reason."
	case "dkick":
		return "Reply to the message you want deleted. I will delete it and kick the sender. You can add an optional reason."
	case "pin":
		return "Reply to the message you want to pin. Add 'silent' or 'notify' to control notification behavior."
	case "unpin":
		return "Reply to the pinned message you want to unpin, or use 'all' to unpin all messages."
	case "purge":
		return "Reply to the starting message. I'll delete all messages from that message to this command."
	default:
		return "Reply to a user's message or pass their username/ID."
	}
}

func formatAdminDuration(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	week := 7 * 24 * time.Hour
	day := 24 * time.Hour
	if d%week == 0 {
		w := int(d / week)
		if w == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", w)
	}
	if d%day == 0 {
		days := int(d / day)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	if d%time.Hour == 0 {
		h := int(d / time.Hour)
		if h == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", h)
	}
	if d%time.Minute == 0 {
		m := int(d / time.Minute)
		if m == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", m)
	}
	return d.Round(time.Second).String()
}

func adminFriendlyError(err error, action string) string {
	msg := parseAdminError(err, action)
	if msg == "" {
		return "I couldn't do that right now. Please try again."
	}
	return msg
}

func replyTemp(m *tg.NewMessage, text string, seconds int) {
	msg, err := m.Reply(text)
	if err != nil || msg == nil || seconds <= 0 {
		return
	}
	go func(chatID int64, msgID int32, delay int) {
		time.Sleep(time.Duration(delay) * time.Second)
		m.Client.DeleteMessages(chatID, []int32{msgID})
	}(m.ChatID(), msg.ID, seconds)
}

func parseAdminError(err error, action string) string {
	if err == nil {
		return "I couldn't " + action + ". Please try again."
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
		return "I couldn't " + action + ". That account can't be managed here."
	case strings.Contains(errStr, "PARTICIPANT_ID_INVALID"):
		return "I couldn't " + action + ". The user might have left the chat."
	case strings.Contains(errStr, "CHAT_NOT_MODIFIED"):
		return "The chat permissions are already set to the requested state."
	case strings.Contains(errStr, "MESSAGE_ID_INVALID"):
		return "Unable to " + action + ", the message might have been deleted or is too old"
	default:
		log.Printf("Admin action '%s' failed: %v", action, err)
		return "I couldn't " + action + ". Please check my admin rights and try again."
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

func init() {
	QueueHandlerRegistration(registerAdminHandlers)
	Mods.AddModule("Admin", `<b>Admin Commands</b>

<b>Ban & Unban:</b>
/ban [user] [reason] - Ban a user permanently
/unban [user] - Remove ban from user
/tban [user] [time] [reason] - Ban for duration (1h, 30m, 2d, etc.)
/sban [user] [reason] - Ban silently (deletes command message)

<b>Kick & Mute:</b>
/kick [user] [reason] - Remove user from group
/mute [user] [reason] - Silence a user (can read, not write)
/unmute [user] - Restore user's ability to send messages
/tmute [user] [time] [reason] - Mute for specified duration
/smute [user] [reason] - Mute silently (deletes command message)
/skick [user] [reason] - Kick silently (deletes command message)

<b>Promote & Demote:</b>
/promote [user] [title] - Make user an admin with optional title
/fullpromote [user] [title] - Promote with full admin rights
/demote [user] - Remove admin status from user

<b>Delete & Act:</b>
/del - Delete a message (reply to the message)
/dban - Delete message and ban sender
/dmute - Delete message and mute sender
/dkick - Delete message and kick sender
/purge - Delete messages from replied message to command message

<b>Pin Management:</b>
/pin [silent|notify] - Pin the replied message
/unpin - Unpin the replied message
/unpin all - Unpin all messages in chat

<b>Lock/Unlock:</b>
/lock [type] - Lock chat permissions (messages, media, stickers, gifs, polls, invite, pin, info, all)
/unlock [type] - Unlock chat permissions
/locks - View current lock status

<b>Info:</b>
/id - Get user and chat IDs with detailed info

<b>Usage:</b>
Reply to a user's message OR provide their @username or ID.
All actions support optional reasons.

<i>💡 Click undo buttons within 5 minutes to reverse actions</i>`)
}

func registerAdminHandlers() {
	c := Client
	c.On("cmd:rspot", RestartSpotify, tg.FromUser(OwnerId))
	c.On("cmd:rproxy", RestartProxy, tg.CustomFilter(FilterOwnerAndAuth))
	c.On("cmd:promote", PromoteUserHandle)
	c.On("cmd:demote", DemoteUserHandle)
	c.On("cmd:ban", BanUserHandle)
	c.On("cmd:unban", UnbanUserHandle)
	c.On("cmd:kick", KickUserHandle)
	c.On("cmd:fullpromote", FullPromoteHandle)
	c.On("cmd:tban", TbanUserHandle)
	c.On("cmd:tmute", TmuteUserHandle)
	c.On("cmd:mute", MuteUserHandle)
	c.On("cmd:unmute", UnmuteUserHandle)
	c.On("cmd:sban", SbanUserHandle)
	c.On("cmd:smute", SmuteUserHandle)
	c.On("cmd:skick", SkickUserHandle)
	c.On("cmd:del", DeleteMessageHandle)
	c.On("cmd:dban", DBanUserHandle)
	c.On("cmd:dmute", DMuteUserHandle)
	c.On("cmd:dkick", DKickUserHandle)
	c.On("cmd:pin", PinMessageHandle)
	c.On("cmd:unpin", UnpinMessageHandle)
	c.On("cmd:purge", PurgeMessagesHandle)
	c.On("cmd:lock", LockHandle)
	c.On("cmd:unlock", UnlockHandle)
	c.On("cmd:locks", LocksHandle)
	c.On("cmd:restart", RestartHandle, tg.CustomFilter(FilterOwner))
	c.On("cmd:id", IDHandle)
	c.On("cmd:gban", Gban, tg.CustomFilter(FilterOwner))
	c.On("cmd:ungban", Ungban, tg.CustomFilter(FilterOwner))
}
