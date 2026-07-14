package extras

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"

	modules "main/modules"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func ephemeralRandomID() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int64(binary.LittleEndian.Uint64(b[:]))
}

func toInputUser(c *tg.Client, target any) (*tg.InputUserObj, error) {
	peer, err := c.ResolvePeer(target)
	if err != nil {
		return nil, err
	}
	u, ok := peer.(*tg.InputPeerUser)
	if !ok {
		return nil, fmt.Errorf("not a user: %T", peer)
	}
	return &tg.InputUserObj{UserID: u.UserID, AccessHash: u.AccessHash}, nil
}

func sendEphemeralIn(m *tg.NewMessage, chat tg.InputPeer, receiver *tg.InputUserObj, text string, deleteTrigger bool) error {
	_, err := m.Client.EphemeralSendMessage(&tg.EphemeralSendMessageParams{
		Peer:       chat,
		ReceiverID: receiver,
		Message:    text,
		RandomID:   ephemeralRandomID(),
	})
	if err != nil {
		m.Reply("Ephemeral send failed: " + err.Error())
		return nil
	}
	if deleteTrigger {
		m.Delete()
	} else {
		m.Reply("Whisper delivered.")
	}
	return nil
}

func EphemeralHandler(m *tg.NewMessage) error {
	text := strings.TrimSpace(m.Args())
	if text == "" {
		m.Reply("Usage: <code>/ep &lt;text&gt;</code> — sends a message only you can see.")
		return nil
	}
	chat, err := m.Client.ResolvePeer(m.ChatID())
	if err != nil {
		m.Reply("Failed to resolve chat: " + err.Error())
		return nil
	}
	receiver, err := toInputUser(m.Client, m.SenderID())
	if err != nil {
		m.Reply("Failed to resolve sender: " + err.Error())
		return nil
	}
	return sendEphemeralIn(m, chat, receiver, text, true)
}

const whisperUsage = "<b>Usage:</b>\n" +
	" - In a group: <code>/wsp @user text</code>, <code>/wsp &lt;userid&gt; text</code>, or reply with <code>/wsp text</code>\n" +
	" - In this DM (whisper text stays private): <code>/wsp &lt;chat&gt; &lt;user&gt; &lt;text&gt;</code>"

func WhisperHandler(m *tg.NewMessage) error {
	args := m.ArgsList()
	if len(args) == 0 {
		m.Reply(whisperUsage)
		return nil
	}

	if m.IsPrivate() {
		if len(args) < 3 {
			m.Reply(whisperUsage)
			return nil
		}
		chatTok, userTok := args[0], args[1]
		rest := strings.TrimSpace(m.Args())
		rest = strings.TrimSpace(strings.TrimPrefix(rest, chatTok))
		text := strings.TrimSpace(strings.TrimPrefix(rest, userTok))
		if text == "" {
			m.Reply("No text to whisper.")
			return nil
		}
		chat, err := m.Client.ResolvePeer(chatTok)
		if err != nil {
			m.Reply("Couldn't resolve chat <code>" + chatTok + "</code>: " + err.Error())
			return nil
		}
		if _, isUser := chat.(*tg.InputPeerUser); isUser {
			m.Reply("Chat argument resolved to a user, not a chat. Give me a group/channel.")
			return nil
		}
		receiver, err := toInputUser(m.Client, userTok)
		if err != nil {
			m.Reply("Couldn't resolve user <code>" + userTok + "</code>: " + err.Error())
			return nil
		}
		return sendEphemeralIn(m, chat, receiver, text, false)
	}

	chat, err := m.Client.ResolvePeer(m.ChatID())
	if err != nil {
		m.Reply("Failed to resolve chat: " + err.Error())
		return nil
	}

	var receiver *tg.InputUserObj
	var text string

	if m.IsReply() {
		reply, rerr := m.GetReplyMessage()
		if rerr != nil || reply == nil {
			m.Reply("Couldn't read the replied-to message.")
			return nil
		}
		receiver, err = toInputUser(m.Client, reply.SenderID())
		if err != nil {
			m.Reply("Couldn't resolve reply target: " + err.Error())
			return nil
		}
		text = strings.TrimSpace(m.Args())
	} else {
		target := args[0]
		text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(m.Args()), target))
		receiver, err = toInputUser(m.Client, target)
		if err != nil {
			m.Reply("Couldn't resolve <code>" + target + "</code>: " + err.Error())
			return nil
		}
	}

	if text == "" {
		m.Reply("No text to whisper.")
		return nil
	}
	return sendEphemeralIn(m, chat, receiver, text, true)
}

func registerEphemeralHandlers() {
	c := modules.Client
	c.On("cmd:ep", EphemeralHandler)
	c.On("cmd:wsp", WhisperHandler)
	c.On("cmd:whisper", WhisperHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerEphemeralHandlers)

	modules.Mods.AddModule("Ephemeral", `<b>Ephemeral & Whisper</b>

Send messages only specific people can see.

<b>In a group:</b>
 - /ep &lt;text&gt; - Send an ephemeral visible only to you
 - /wsp @user &lt;text&gt; - Whisper to that user
 - /wsp &lt;userid&gt; &lt;text&gt; - Whisper to that user by numeric ID
 - /wsp &lt;text&gt; while replying - Whisper to whoever sent the replied-to message

<b>In this bot's DM (whisper text stays private):</b>
 - /wsp &lt;chat&gt; &lt;user&gt; &lt;text&gt; - Deliver a whisper into a group without ever posting the text there

<b>Aliases:</b> /whisper = /wsp

Group triggers are deleted right after send; DM triggers are acknowledged with a delivery confirmation.`)
}
