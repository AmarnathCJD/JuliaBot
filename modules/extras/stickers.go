package extras

import (
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	_ "golang.org/x/image/webp"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	modules "main/modules"
	"main/modules/db"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// === from stickers.go ===
const MaxStickersPerPack = 120

func GifToSticker(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Error:</b> Please reply to a GIF message to convert it to a sticker.")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Error:</b> Unable to fetch the replied message.")
		return nil
	}

	if !r.IsMedia() {
		m.Reply("<b>Error:</b> The replied message is not a GIF.")
		return nil
	}

	fn := ""
	if r.File != nil {
		fn = r.File.Name
	}

	if fn != "" {
		lfn := strings.ToLower(fn)
		if !(strings.HasSuffix(lfn, ".mp4") || strings.HasSuffix(lfn, ".gif")) {
			m.Reply("Invalid media: only .mp4 or .gif files are supported.")
			return nil
		}
	}

	fi, err := r.Download(&tg.DownloadOptions{
		FileName: "gif.gif",
	})
	if err != nil {
		m.Reply("<b>Error:</b> Unable to download the GIF.")
		return nil
	}

	out := "sticker.webm"
	defer os.Remove("gif.gif")
	defer os.Remove(out)

	cmd := exec.Command("ffmpeg", "-i", fi, "-vf", "format=yuv420p", "-c:v", "libvpx-vp9", "-b:v", "0", "-crf", "30", "-an", "-y", out)
	_ = cmd.Run()
	m.ReplyMedia("sticker.webm", &tg.MediaOptions{
		Attributes: []tg.DocumentAttribute{
			&tg.DocumentAttributeSticker{
				Alt:        "😍",
				Stickerset: &tg.InputStickerSetEmpty{},
			},
			&tg.DocumentAttributeFilename{
				FileName: "sticker.webm",
			},
		},
	})

	return nil
}

func KangSticker(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a sticker to kang it!\nUsage: <code>/kang [emoji]</code>")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Failed to get replied message.")

	}

	if !reply.IsMedia() {
		m.Reply("Please reply to a sticker!")
	}

	var packType string
	var emoji string = "👍"

	args := m.Args()
	if len(args) > 0 {
		emoji = args
	}

	var stickerFile struct {
		ID            int64
		AccessHash    int64
		FileReference []byte
		Type          string
		fi            tg.MessageMedia
	}

	if reply.Media() != nil {
		stickerFile.fi = reply.Media()
		if doc, ok := reply.Media().(*tg.MessageMediaDocument); ok {
			if document, ok := doc.Document.(*tg.DocumentObj); ok {
				for _, attr := range document.Attributes {
					if stickerAttr, ok := attr.(*tg.DocumentAttributeSticker); ok {
						if emoji == "👍" && stickerAttr.Alt != "" {
							emoji = stickerAttr.Alt
						}
					}
					if _, ok := attr.(*tg.DocumentAttributeVideo); ok {
						packType = "webm"
					}
					if fileName, ok := attr.(*tg.DocumentAttributeFilename); ok {
						if strings.HasSuffix(fileName.FileName, ".tgs") {
							packType = "tgs"
						}
					}
				}

				if packType == "" {
					if strings.Contains(document.MimeType, "video") {
						packType = "webm"
					} else if strings.Contains(document.MimeType, "application/x-tgsticker") {
						packType = "tgs"
					} else {
						packType = "normal"
					}
				}

				stickerFile.ID = document.ID
				stickerFile.AccessHash = document.AccessHash
				stickerFile.FileReference = document.FileReference
				stickerFile.Type = packType
			}
		}
		if reply.Document().MimeType == "application/x-tgsticker" {
			packType = "tgs"
		} else if strings.HasPrefix(reply.Document().MimeType, "video/") {
			packType = "webm"
		} else {
			packType = "normal"
		}
	}

	userID := m.SenderID()
	username := m.Sender.Username
	if username == "" {
		username = fmt.Sprintf("user%d", userID)
	}

	pack, err := db.GetActivePack(userID, packType)

	var shortName, title string
	//var isNewPack bool

	if err != nil || pack == nil || pack.StickerCount >= MaxStickersPerPack {
		//isNewPack = true
		packs, _ := db.GetUserPacks(userID)
		packNumber := len(packs[packType]) + 1

		shortName = fmt.Sprintf("x%s_%s_%d_by_%s", username, packType, packNumber, m.Client.Me().Username)
		title = fmt.Sprintf("%s's %s Stickers #%d", username, strings.Title(packType), packNumber)

		pack = &db.PackInfo{
			ShortName:    shortName,
			Title:        title,
			Type:         packType,
			StickerCount: 0,
			PackNumber:   packNumber,
		}

		var createErr error
		switch packType {
		case "tgs", "webm":
			fi, err := m.Client.DownloadMedia(stickerFile.fi)
			if err != nil {
				m.Reply("Failed to download sticker media.")
				return nil
			}
			defer os.Remove(fi)

			var mediaSendable *tg.InputMediaDocument
			if packType == "webm" {
				ext := filepath.Ext(fi)
				out := fi + "_resized" + ext
				cmd := exec.Command("ffmpeg", "-i", fi, "-vf", "scale=512:512:force_original_aspect_ratio=decrease,pad=512:512:(512-iw)/2:(512-ih)/2:color=black@0,format=yuva420p", "-c:v", "libvpx-vp9", "-auto-alt-ref", "0", "-b:v", "0", "-crf", "30", "-an", "-y", out)
				if err := cmd.Run(); err == nil {
					defer os.Remove(out)
					media, err := m.Client.GetSendableMedia(out, &tg.MediaMetadata{Inline: true})
					if err != nil {
						media, err = m.Client.GetSendableMedia(fi, &tg.MediaMetadata{Inline: true})
						if err != nil {
							m.Reply("Failed to prepare sticker media.")
							return nil
						}
					}
					mediaSendable = media.(*tg.InputMediaDocument)
				} else {
					media, err := m.Client.GetSendableMedia(fi, &tg.MediaMetadata{Inline: true})
					if err != nil {
						m.Reply("Failed to prepare sticker media.")
						return nil
					}
					mediaSendable = media.(*tg.InputMediaDocument)
				}
			} else {
				media, err := m.Client.GetSendableMedia(fi, &tg.MediaMetadata{Inline: true})
				if err != nil {
					m.Reply("Failed to prepare sticker media.")
					return nil
				}
				mediaSendable = media.(*tg.InputMediaDocument)
			}

			_, createErr = m.Client.StickersCreateStickerSet(&tg.StickersCreateStickerSetParams{
				UserID:    &tg.InputUserObj{UserID: userID, AccessHash: m.Sender.AccessHash},
				Title:     title,
				ShortName: shortName,
				Stickers: []*tg.InputStickerSetItem{
					{
						Document: mediaSendable.ID,
						Emoji:    emoji,
					},
				},
			})
		default:
			_, createErr = m.Client.StickersCreateStickerSet(&tg.StickersCreateStickerSetParams{
				UserID:    &tg.InputUserObj{UserID: userID, AccessHash: m.Sender.AccessHash},
				Title:     title,
				ShortName: shortName,
				Stickers: []*tg.InputStickerSetItem{
					{
						Document: &tg.InputDocumentObj{
							ID:            stickerFile.ID,
							AccessHash:    stickerFile.AccessHash,
							FileReference: stickerFile.FileReference,
						},
						Emoji: emoji,
					},
				},
			})
		}

		if createErr != nil {
			m.Reply(fmt.Sprintf("Failed to create sticker pack: %v", createErr))
			return nil
		}

		pack.StickerCount = 1
		db.SavePack(userID, pack)

		m.Reply(fmt.Sprintf(
			"<b>Created new %s sticker pack!</b>\n"+
				"Pack: <a href='https://t.me/addstickers/%s'>%s</a>\n"+
				"Stickers: 1/%d",
			packType, shortName, title, MaxStickersPerPack,
		))
		return nil
	}

	var doc tg.InputDocument

	switch packType {
	case "tgs", "webm":
		fi, err := m.Client.DownloadMedia(stickerFile.fi)
		if err != nil {
			m.Reply("Failed to download sticker media.")
			return nil
		}
		defer os.Remove(fi)

		if packType == "webm" {
			ext := filepath.Ext(fi)
			out := fi + "_resized" + ext
			cmd := exec.Command("ffmpeg", "-i", fi, "-vf", "scale=512:512:force_original_aspect_ratio=decrease,pad=512:512:(512-iw)/2:(512-ih)/2:color=black@0,format=yuva420p", "-c:v", "libvpx-vp9", "-auto-alt-ref", "0", "-b:v", "0", "-crf", "30", "-an", "-y", out)
			if err := cmd.Run(); err == nil {
				defer os.Remove(out)
				media, err := m.Client.GetSendableMedia(out, &tg.MediaMetadata{Inline: true})
				if err != nil {
					media, err = m.Client.GetSendableMedia(fi, &tg.MediaMetadata{Inline: true})
					if err != nil {
						m.Reply("Failed to prepare sticker media.")
						return nil
					}
				}
				doc = media.(*tg.InputMediaDocument).ID
			} else {
				media, err := m.Client.GetSendableMedia(fi, &tg.MediaMetadata{Inline: true})
				if err != nil {
					m.Reply("Failed to prepare sticker media.")
					return nil
				}
				doc = media.(*tg.InputMediaDocument).ID
			}
		} else {
			media, err := m.Client.GetSendableMedia(fi, &tg.MediaMetadata{Inline: true})
			if err != nil {
				m.Reply("Failed to prepare sticker media.")
				return nil
			}
			doc = media.(*tg.InputMediaDocument).ID
		}
	default:
		fi, err := m.Client.DownloadMedia(stickerFile.fi)
		if err != nil {
			m.Reply("Failed to download sticker media.")
			return nil
		}
		defer os.Remove(fi)

		ext := filepath.Ext(fi)
		out := fi + "_resized" + ext

		cmd := exec.Command("ffmpeg", "-i", fi, "-vf", "scale=w=512:h=512:force_original_aspect_ratio=decrease", "-y", out)
		if err := cmd.Run(); err != nil {
			media, err := m.Client.GetSendableMedia(fi, &tg.MediaMetadata{Inline: true})
			if err != nil {
				m.Reply("Failed to prepare sticker media.")
				return nil
			}
			doc = media.(*tg.InputMediaDocument).ID
		} else {
			defer os.Remove(out)
			media, err := m.Client.GetSendableMedia(out, &tg.MediaMetadata{Inline: true})
			if err != nil {
				media, err = m.Client.GetSendableMedia(fi, &tg.MediaMetadata{Inline: true})
				if err != nil {
					m.Reply("Failed to prepare sticker media.")
					return nil
				}
			}
			doc = media.(*tg.InputMediaDocument).ID
		}
	}

	_, addErr := m.Client.StickersAddStickerToSet(&tg.InputStickerSetShortName{ShortName: pack.ShortName}, &tg.InputStickerSetItem{
		Document: doc,
		Emoji:    emoji,
	})

	if addErr != nil {
		m.Reply(fmt.Sprintf("Failed to add sticker: %v", addErr))
		return nil
	}

	db.IncrementPackCount(userID, pack)

	msg := fmt.Sprintf(
		"<b>Added to pack!</b>\n"+
			"Pack: <a href='https://t.me/addstickers/%s'>%s</a>\n"+
			"Stickers: %d/%d",
		pack.ShortName, pack.Title, pack.StickerCount, MaxStickersPerPack,
	)

	if pack.StickerCount >= MaxStickersPerPack {
		msg += "\n\n⚠️ <b>Pack is full!</b> Next sticker will create a new pack."
	}

	m.Reply(msg)
	return nil
}

func RemoveKangedSticker(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a sticker in your pack to remove it!\nUsage: <code>/rmkang</code>")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Failed to get replied message.")
		return nil
	}

	if !reply.IsMedia() {
		m.Reply("Please reply to a sticker!")
		return nil
	}

	var stickerFile tg.InputDocument
	if reply.Media() != nil {
		if doc, ok := reply.Media().(*tg.MessageMediaDocument); ok {
			if document, ok := doc.Document.(*tg.DocumentObj); ok {
				stickerFile = &tg.InputDocumentObj{
					ID:            document.ID,
					AccessHash:    document.AccessHash,
					FileReference: document.FileReference,
				}
			}
		}
	}

	if stickerFile == nil {
		m.Reply("Unable to extract sticker file!")
		return nil
	}

	userID := m.Sender.ID

	packs, err := db.GetUserPacks(userID)
	if err != nil || len(packs) == 0 {
		m.Reply("You don't have any sticker packs!")
		return nil
	}

	var removed bool

	_, err = m.Client.StickersRemoveStickerFromSet(stickerFile)
	if err == nil {
		removed = true
	}

	if removed {
		m.Reply("✅ Removed sticker from your pack!")
		return nil
	}

	m.Reply("❌ Sticker not found in your packs or you don't own this sticker.")
	return nil
}

func PackInfoHandle(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a sticker to get pack info!")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Failed to get replied message.")
		return nil
	}

	if !reply.IsMedia() {
		m.Reply("Please reply to a sticker!")
		return nil
	}

	var stickerAttr *tg.DocumentAttributeSticker
	if reply.Media() != nil {
		if doc, ok := reply.Media().(*tg.MessageMediaDocument); ok {
			if document, ok := doc.Document.(*tg.DocumentObj); ok {
				for _, attr := range document.Attributes {
					if sticker, ok := attr.(*tg.DocumentAttributeSticker); ok {
						stickerAttr = sticker
						break
					}
				}
			}
		}
	}

	if stickerAttr == nil || stickerAttr.Stickerset == nil {
		m.Reply("This is not a valid sticker or doesn't belong to a pack!")
		return nil
	}

	// Get the sticker set
	result, err := m.Client.MessagesGetStickerSet(stickerAttr.Stickerset, 0)
	if err != nil {
		m.Reply("Failed to get sticker pack info.")
		return nil
	}
	resp := result.(*tg.MessagesStickerSetObj)

	var creatorID, internalID int64
	stickerSetID := resp.Set.ID
	sid := stickerSetID
	creatorID = sid >> 32

	if ((sid >> 24) & 0xFF) == 0 {
		internalID = sid & 0xFFFFFFFF
	}

	text := fmt.Sprintf("🧩 <b>Sticker Pack Info</b>\n\n👤 <b>Creator ID:</b> <code>%d</code>\n", creatorID)

	if internalID != 0 {
		text += fmt.Sprintf("🆔 <b>Increment set ID:</b> <code>%d</code>\n", internalID)
	} else {
		text += "🆔 <b>Increment set ID:</b> <code>Unavailable</code>\n"
	}

	if creatorID > 0 {
		user, err := m.Client.GetUser(creatorID)
		if err == nil && user != nil {
			userName := user.FirstName
			if user.LastName != "" {
				userName += " " + user.LastName
			}
			if user.Username != "" {
				text += fmt.Sprintf("👤 <b>Creator Name:</b> <a href='https://t.me/%s'>%s</a>", user.Username, userName)
			} else {
				text += fmt.Sprintf("👤 <b>Creator Name:</b> %s", userName)
			}
		}
	}

	m.Reply(text)
	return nil
}

func registerStickersHandlers() {
	c := modules.Client
	c.OnCommand("gif", GifToSticker)
	c.OnCommand("kang", KangSticker)
	c.OnCommand("rmkang", RemoveKangedSticker)
	c.OnCommand("pack", PackInfoHandle)
	c.On("command:doge", modules.DogeSticker)
	c.On("inline:doge", modules.DogeStickerInline)
}

func initFromSrc_stickers_0_1() {
	modules.QueueHandlerRegistration(registerStickersHandlers)
}
// === from sticker_pack_info.go ===
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
	c := modules.Client
	c.On("cmd:stickerinfo", StickerInfoHandler)
	c.On("cmd:me", MeActionHandler)
	c.On("cmd:myself", MeActionHandler)
}

func initFromSrc_sticker_pack_info_1_1() {
	modules.QueueHandlerRegistration(registerStickerPackInfoHandlers)
}
// === from sticker_to_image.go ===
func stickerExtractDoc(reply *tg.NewMessage) (*tg.DocumentObj, string) {
	if reply.Media() == nil {
		return nil, ""
	}
	md, ok := reply.Media().(*tg.MessageMediaDocument)
	if !ok {
		return nil, ""
	}
	doc, ok := md.Document.(*tg.DocumentObj)
	if !ok {
		return nil, ""
	}
	kind := "static"
	for _, attr := range doc.Attributes {
		if _, ok := attr.(*tg.DocumentAttributeVideo); ok {
			kind = "video"
		}
		if fn, ok := attr.(*tg.DocumentAttributeFilename); ok {
			if strings.HasSuffix(strings.ToLower(fn.FileName), ".tgs") {
				kind = "tgs"
			}
		}
	}
	if strings.Contains(doc.MimeType, "application/x-tgsticker") {
		kind = "tgs"
	} else if strings.HasPrefix(doc.MimeType, "video/") {
		kind = "video"
	}
	return doc, kind
}

func StickerToImageHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a static sticker with <code>/towebp</code>")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Error:</b> unable to fetch reply: " + html.EscapeString(err.Error()))
		return nil
	}

	if !reply.IsMedia() {
		m.Reply("<b>Error:</b> reply is not a sticker")
		return nil
	}

	_, kind := stickerExtractDoc(reply)
	if kind == "" {
		m.Reply("<b>Error:</b> reply is not a sticker document")
		return nil
	}
	if kind == "tgs" {
		m.Reply("<b>Error:</b> animated (.tgs) stickers are not supported")
		return nil
	}
	if kind == "video" {
		m.Reply("<b>Error:</b> video (.webm) stickers are not supported")
		return nil
	}

	status, _ := m.Reply("<code>converting sticker...</code>")

	ts := time.Now().UnixNano()
	srcPath := filepath.Join(os.TempDir(), fmt.Sprintf("sticker_%d.webp", ts))
	pngPath := filepath.Join(os.TempDir(), fmt.Sprintf("sticker_%d.png", ts))
	jpgPath := filepath.Join(os.TempDir(), fmt.Sprintf("sticker_%d.jpg", ts))

	fi, err := reply.Download(&tg.DownloadOptions{FileName: srcPath})
	if err != nil {
		msg := "<b>Error:</b> download failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(fi)

	f, err := os.Open(fi)
	if err != nil {
		msg := "<b>Error:</b> open failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	img, fmtName, err := image.Decode(f)
	f.Close()
	if err != nil {
		msg := "<b>Error:</b> decode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	pngOut, err := os.Create(pngPath)
	if err != nil {
		msg := "<b>Error:</b> create png failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if err := png.Encode(pngOut, img); err != nil {
		pngOut.Close()
		os.Remove(pngPath)
		msg := "<b>Error:</b> png encode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	pngOut.Close()
	defer os.Remove(pngPath)

	bounds := img.Bounds()
	flat := image.NewRGBA(bounds)
	draw.Draw(flat, bounds, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(flat, bounds, img, bounds.Min, draw.Over)

	jpgOut, err := os.Create(jpgPath)
	if err != nil {
		msg := "<b>Error:</b> create jpg failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if err := jpeg.Encode(jpgOut, flat, &jpeg.Options{Quality: 92}); err != nil {
		jpgOut.Close()
		os.Remove(jpgPath)
		msg := "<b>Error:</b> jpg encode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	jpgOut.Close()
	defer os.Remove(jpgPath)

	caption := fmt.Sprintf("<b>Sticker -&gt; Image</b>\n<b>Source:</b> <code>%s</code>", html.EscapeString(fmtName))

	if _, err := m.ReplyMedia(pngPath, &tg.MediaOptions{
		Caption:       caption + "\n<b>Format:</b> PNG",
		FileName:      "sticker.png",
		MimeType:      "image/png",
		ForceDocument: false,
	}); err != nil {
		msg := "<b>Error:</b> png upload failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if _, err := m.ReplyMedia(jpgPath, &tg.MediaOptions{
		Caption:       caption + "\n<b>Format:</b> JPG",
		FileName:      "sticker.jpg",
		MimeType:      "image/jpeg",
		ForceDocument: false,
	}); err != nil {
		msg := "<b>Error:</b> jpg upload failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if status != nil {
		status.Delete()
	}
	return nil
}

func registerStickerToImageHandlers() {
	c := modules.Client
	c.On("cmd:towebp", StickerToImageHandler)
}

func initFromSrc_sticker_to_image_2_1() {
	modules.QueueHandlerRegistration(registerStickerToImageHandlers)
}
// === from webp_to_jpg.go ===
func webpToJpgExtractDoc(reply *tg.NewMessage) (*tg.DocumentObj, string, string) {
	if reply.Media() == nil {
		return nil, "", ""
	}
	md, ok := reply.Media().(*tg.MessageMediaDocument)
	if !ok {
		return nil, "", ""
	}
	doc, ok := md.Document.(*tg.DocumentObj)
	if !ok {
		return nil, "", ""
	}
	fileName := ""
	kind := "image"
	for _, attr := range doc.Attributes {
		if _, ok := attr.(*tg.DocumentAttributeVideo); ok {
			kind = "video"
		}
		if _, ok := attr.(*tg.DocumentAttributeAnimated); ok {
			kind = "animated"
		}
		if fn, ok := attr.(*tg.DocumentAttributeFilename); ok {
			fileName = fn.FileName
			if strings.HasSuffix(strings.ToLower(fn.FileName), ".tgs") {
				kind = "tgs"
			}
		}
		if _, ok := attr.(*tg.DocumentAttributeSticker); ok {
			if kind == "image" {
				kind = "sticker"
			}
		}
	}
	mime := strings.ToLower(doc.MimeType)
	if strings.Contains(mime, "application/x-tgsticker") {
		kind = "tgs"
	} else if strings.HasPrefix(mime, "video/") {
		kind = "video"
	}
	return doc, kind, fileName
}

func WebpToJpgHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a webp image or static sticker with <code>/tojpg</code>")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Error:</b> unable to fetch reply: " + html.EscapeString(err.Error()))
		return nil
	}

	if !reply.IsMedia() {
		m.Reply("<b>Error:</b> reply has no media")
		return nil
	}

	doc, kind, fileName := webpToJpgExtractDoc(reply)
	if doc == nil {
		m.Reply("<b>Error:</b> reply is not a document")
		return nil
	}
	if kind == "tgs" {
		m.Reply("<b>Error:</b> animated (.tgs) stickers are not supported")
		return nil
	}
	if kind == "video" {
		m.Reply("<b>Error:</b> video stickers are not supported")
		return nil
	}
	if kind == "animated" {
		m.Reply("<b>Error:</b> animated media is not supported")
		return nil
	}

	mime := strings.ToLower(doc.MimeType)
	lowerName := strings.ToLower(fileName)
	isWebp := strings.Contains(mime, "webp") || strings.HasSuffix(lowerName, ".webp")
	isImage := strings.HasPrefix(mime, "image/")
	if !isWebp && !isImage && kind != "sticker" {
		m.Reply("<b>Error:</b> reply is not a webp or image")
		return nil
	}

	status, _ := m.Reply("<code>converting to jpg...</code>")

	ts := time.Now().UnixNano()
	ext := ".webp"
	if strings.HasSuffix(lowerName, ".png") {
		ext = ".png"
	} else if strings.HasSuffix(lowerName, ".jpg") || strings.HasSuffix(lowerName, ".jpeg") {
		ext = ".jpg"
	}
	srcPath := filepath.Join(os.TempDir(), fmt.Sprintf("tojpg_%d%s", ts, ext))
	jpgPath := filepath.Join(os.TempDir(), fmt.Sprintf("tojpg_%d.jpg", ts))

	fi, err := reply.Download(&tg.DownloadOptions{FileName: srcPath})
	if err != nil {
		msg := "<b>Error:</b> download failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(fi)

	f, err := os.Open(fi)
	if err != nil {
		msg := "<b>Error:</b> open failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	img, fmtName, err := image.Decode(f)
	f.Close()
	if err != nil {
		msg := "<b>Error:</b> decode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	bounds := img.Bounds()
	flat := image.NewRGBA(bounds)
	draw.Draw(flat, bounds, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(flat, bounds, img, bounds.Min, draw.Over)

	jpgOut, err := os.Create(jpgPath)
	if err != nil {
		msg := "<b>Error:</b> create jpg failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if err := jpeg.Encode(jpgOut, flat, &jpeg.Options{Quality: 92}); err != nil {
		jpgOut.Close()
		os.Remove(jpgPath)
		msg := "<b>Error:</b> jpg encode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	jpgOut.Close()
	defer os.Remove(jpgPath)

	caption := fmt.Sprintf("<b>WebP -&gt; JPG</b>\n<b>Source:</b> <code>%s</code>\n<b>Size:</b> <code>%dx%d</code>", html.EscapeString(fmtName), bounds.Dx(), bounds.Dy())

	if _, err := m.ReplyMedia(jpgPath, &tg.MediaOptions{
		Caption:       caption,
		FileName:      "converted.jpg",
		MimeType:      "image/jpeg",
		ForceDocument: false,
	}); err != nil {
		msg := "<b>Error:</b> upload failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if status != nil {
		status.Delete()
	}
	return nil
}

func registerWebpToJpgHandlers() {
	c := modules.Client
	c.On("cmd:tojpg", WebpToJpgHandler)
}

func initFromSrc_webp_to_jpg_3_1() {
	modules.QueueHandlerRegistration(registerWebpToJpgHandlers)
}

func init() {
	initFromSrc_stickers_0_1()
	initFromSrc_sticker_pack_info_1_1()
	initFromSrc_sticker_to_image_2_1()
	initFromSrc_webp_to_jpg_3_1()
}
