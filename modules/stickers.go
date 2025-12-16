package modules

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"main/modules/db"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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
	}

	userID := m.SenderID()
	username := m.Sender.Username
	if username == "" {
		username = fmt.Sprintf("user%d", userID)
	}

	if err := db.InitStickerDB(); err != nil {
		m.Reply("Database error: " + err.Error())
		return nil
	}

	pack, err := db.GetActivePack(userID, packType)

	var shortName, title string
	//var isNewPack bool

	if err != nil || pack == nil || pack.StickerCount >= MaxStickersPerPack {
		//isNewPack = true
		packs, _ := db.GetUserPacks(userID)
		packNumber := len(packs[packType]) + 1

		shortName = fmt.Sprintf("%s_%s_%d_by_%s", username, packType, packNumber, m.Client.Me().Username)
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

	var stickerFile *tg.InputDocument
	// if reply.Media() != nil {
	// 	if doc, ok := reply.Media().(*tg.MessageMediaDocument); ok {
	// 		if document, ok := doc.Document.(*tg.Document); ok {
	// 			stickerFile = &tg.InputDocument{
	// 				ID:            document.ID,
	// 				AccessHash:    document.AccessHash,
	// 				FileReference: document.FileReference,
	// 			}
	// 		}
	// 	}
	// }

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
	var packName string

	// for packType, packList := range packs {
	// 	for _, pack := range packList {
	// 		_, removeErr := m.Client.StickersRemoveStickerFromSet(&tg.InputDocumentObj{
	// 			ID:            stickerFile.ID,
	// 			AccessHash:    stickerFile.AccessHash,
	// 			FileReference: stickerFile.FileReference,
	// 		})

	// 		if removeErr == nil {
	// 			removed = true
	// 			packName = pack.Title
	// 			db.DecrementPackCount(userID, pack)
	// 			break
	// 		}
	// 	}
	// 	if removed {
	// 		break
	// 	}
	// }

	if removed {
		m.Reply(fmt.Sprintf("✅ Removed sticker from <b>%s</b>", packName))
		return nil
	}

	m.Reply("❌ Sticker not found in your packs or you don't own this sticker.")
	return nil
}
