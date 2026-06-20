package modules

import (
	"fmt"
	"html"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func StickerInfoHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a sticker with <code>/stickerinfo</code>")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Error:</b> unable to fetch replied message.")
		return nil
	}
	if !reply.IsMedia() {
		m.Reply("<b>Error:</b> please reply to a sticker.")
		return nil
	}
	var stickerAttr *tg.DocumentAttributeSticker
	var mime string
	if doc, ok := reply.Media().(*tg.MessageMediaDocument); ok {
		if document, ok := doc.Document.(*tg.DocumentObj); ok {
			mime = document.MimeType
			for _, attr := range document.Attributes {
				if s, ok := attr.(*tg.DocumentAttributeSticker); ok {
					stickerAttr = s
					break
				}
			}
		}
	}
	if stickerAttr == nil || stickerAttr.Stickerset == nil {
		m.Reply("<b>Error:</b> not a valid sticker or has no pack.")
		return nil
	}
	if _, ok := stickerAttr.Stickerset.(*tg.InputStickerSetEmpty); ok {
		m.Reply("<b>Error:</b> sticker has no associated pack.")
		return nil
	}
	result, err := m.Client.MessagesGetStickerSet(stickerAttr.Stickerset, 0)
	if err != nil {
		m.Reply(fmt.Sprintf("<b>Error:</b> failed to fetch pack info: %s", html.EscapeString(err.Error())))
		return nil
	}
	resp, ok := result.(*tg.MessagesStickerSetObj)
	if !ok || resp.Set == nil {
		m.Reply("<b>Error:</b> unexpected response from Telegram.")
		return nil
	}
	set := resp.Set
	kind := "Static"
	switch {
	case strings.Contains(mime, "x-tgsticker"):
		kind = "Animated (.tgs)"
	case strings.Contains(mime, "video/webm"):
		kind = "Video (.webm)"
	case strings.Contains(mime, "image/webp"):
		kind = "Static (.webp)"
	}
	if set.Emojis {
		kind += " · Emoji pack"
	}
	if set.Masks {
		kind += " · Mask pack"
	}
	addLink := fmt.Sprintf("https://t.me/addstickers/%s", set.ShortName)
	text := fmt.Sprintf(
		"<b>Sticker Pack Info</b>\n\n"+
			"<b>Title:</b> %s\n"+
			"<b>Short Name:</b> <code>%s</code>\n"+
			"<b>Stickers:</b> %d\n"+
			"<b>Type:</b> %s\n"+
			"<b>Link:</b> <a href=\"%s\">Add Pack</a>",
		html.EscapeString(set.Title),
		html.EscapeString(set.ShortName),
		set.Count,
		html.EscapeString(kind),
		addLink,
	)
	m.Reply(text)
	return nil
}

func MeActionHandler(m *tg.NewMessage) error {
	action := strings.TrimSpace(m.Args())
	if action == "" {
		m.Reply("<b>Usage:</b> <code>/me &lt;action&gt;</code>\nExample: <code>/me waves hello</code>")
		return nil
	}
	name := ""
	if m.Sender != nil {
		name = strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
		if name == "" && m.Sender.Username != "" {
			name = "@" + m.Sender.Username
		}
	}
	if name == "" {
		name = fmt.Sprintf("user%d", m.SenderID())
	}
	text := fmt.Sprintf("<i>* %s %s</i>", html.EscapeString(name), html.EscapeString(action))
	m.Reply(text)
	return nil
}

func registerStickerPackInfoHandlers() {
	c := Client
	c.On("cmd:stickerinfo", StickerInfoHandler)
	c.On("cmd:me", MeActionHandler)
	c.On("cmd:myself", MeActionHandler)
}

func init() {
	QueueHandlerRegistration(registerStickerPackInfoHandlers)
}
