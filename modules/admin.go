package modules

import (
	"os/exec"
	"strconv"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func PromoteUserHandle(m *tg.NewMessage) error {
	m.Reply("Umbikko Myre >> 1")
	return nil
}

var (
	spotifyRestartCMD = "sudo systemctl restart spotdl.service"
	proxyRestartCMD   = "sudo systemctl restart wireproxy.service"
	selfRestartCMD    = "sudo systemctl restart rustybot.service"
)

func RestartSpotify(m *tg.NewMessage) error {
	msg, _ := m.Reply("Restarting Spotify...")
	if err := execCommand(spotifyRestartCMD); err != nil {
		return err
	}
	msg.Edit("Spotify restarted successfully.")
	return nil
}

func RestartProxy(m *tg.NewMessage) error {
	msg, _ := m.Reply("Restarting WProxy...")
	if err := execCommand(proxyRestartCMD); err != nil {
		return err
	}
	msg.Edit("Proxy restarted successfully.")
	return nil
}

func RestartHandle(m *tg.NewMessage) error {
	msg, _ := m.Reply("Restarting bot...")
	if err := execCommand(selfRestartCMD); err != nil {
		return err
	}
	msg.Edit("Bot restarted successfully.")
	return nil
}

func execCommand(cmd string) error {
	command := exec.Command("bash", "-c", cmd)
	_, err := command.CombinedOutput()
	if err != nil {
		return err
	}
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
	output += "📋 <b>Message Details</b>\n"
	output += "────────────────────\n"
	output += "👤 <b>UserID</b>: <code>" + strconv.Itoa(int(senderID)) + "</code>\n"
	output += "💬 <b>ChatID</b>: <code>" + strconv.Itoa(int(chatID)) + "</code>\n"

	if forwardedID != 0 {
		output += "\n📨 <b>Forward Details</b>\n"
		output += "────────────────────\n"
		output += "🔗 <b>ForwardedID</b>: <code>" + strconv.Itoa(forwardedID) + "</code>\n"
		output += "📌 <b>ForwardType</b>: <code>" + forwardedType + "</code>\n"
	}

	if repliedToUserID != 0 {
		output += "\n↩️ <b>Reply Details</b>\n"
		output += "────────────────────\n"
		output += "👤 <b>RepliedToUserID</b>: <code>" + strconv.Itoa(repliedToUserID) + "</code>\n"
		output += "📜 <b>RepliedMessageID</b>: <code>" + strconv.Itoa(repliedMessageID) + "</code>\n"
		if repliedMediaID != "" {
			output += "📎 <b>RepliedMediaID</b>: <code>" + repliedMediaID + "</code>\n"
		}

		if repliedForwardedID != 0 {
			output += "\n📨 <b>RepliedForward Details</b>\n"
			output += "────────────────────\n"
			output += "🔗 <b>RepliedForwardedID</b>: <code>" + strconv.Itoa(repliedForwardedID) + "</code>\n"
			output += "📌 <b>RepliedForwardType</b>: <code>" + repliedForwardedType + "</code>\n"
		}
	}

	message.Reply(output)
	return nil
}
