package extras

import (
	"context"
	"fmt"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

func Gban(m *telegram.NewMessage) error {
	user, reason, err := modules.GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	message, _ := m.Reply("Enforcing global ban...")
	done := 0
	m.Client.Broadcast(context.Background(), nil, func(c telegram.Chat) error {
		_, err := m.Client.EditBanned(c, user, &telegram.BannedOptions{Ban: true})
		if err == nil {
			done++
		}
		return nil
	}, 600)

	message.Edit(fmt.Sprintf("Global ban enforced in %d groups.\nReason: %s", done, reason))
	return nil
}

func Ungban(m *telegram.NewMessage) error {
	user, _, err := modules.GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	message, _ := m.Reply("Removing global ban...")
	done := 0
	m.Client.Broadcast(context.Background(), nil, func(c telegram.Chat) error {
		_, err := m.Client.EditBanned(c, user, &telegram.BannedOptions{Ban: false})
		if err == nil {
			done++
		}
		return nil
	}, 600)
	message.Edit(fmt.Sprintf("Global ban removed in %d groups.", done))
	return nil
}

func NightModeHandler(m *telegram.NewMessage) error {
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info rights to use this command")
		return nil
	}

	args := m.Args()
	if args == "" {
		m.Reply("Usage: /nightmode on/off")
		return nil
	}

	var enable bool
	switch strings.ToLower(args) {
	case "on":
		enable = true
	case "off":
		enable = false
	default:
		m.Reply("Usage: /nightmode on/off")
		return nil
	}

	chat, err := m.Client.GetChat(m.ChatID())
	if err != nil {
		m.Reply("Error fetching chat info")
		return nil
	}

	current := chat.DefaultBannedRights
	if current == nil {
		current = &telegram.ChatBannedRights{}
	}

	current.SendMessages = enable

	_, err = m.Client.MessagesEditChatDefaultBannedRights(m.Peer, current)
	if err != nil {
		m.Reply("Failed to toggle night mode: " + err.Error())
		return nil
	}

	if enable {
		m.Reply("Night mode enabled. Messages are restricted.")
	} else {
		m.Reply("Night mode disabled. Messages allowed.")
	}
	return nil
}

func registerMiscHandlers() {
	c := modules.Client
	c.On("cmd:help", modules.HelpHandle)
	c.On("cmd:nightmode", NightModeHandler)
	c.On("cmd:tempnote", SaveTempNoteHandler)
	c.On("callback:verify_op_", modules.AdminVerifyCallback)
	c.On("callback:help_back", modules.HelpBackCallback)
	c.On("cmd:gban", Gban, tg.CustomFilter(modules.FilterOwner))
	c.On("cmd:ungban", Ungban, tg.CustomFilter(modules.FilterOwner))
}

func init() {
	modules.QueueHandlerRegistration(registerMiscHandlers)
}
