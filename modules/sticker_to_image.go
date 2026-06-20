package modules

import (
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	_ "golang.org/x/image/webp"
)

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
	c := Client
	c.On("cmd:towebp", StickerToImageHandler)
}

func init() {
	QueueHandlerRegistration(registerStickerToImageHandlers)
}
