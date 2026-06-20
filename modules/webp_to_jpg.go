package modules

import (
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	_ "golang.org/x/image/webp"
)

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
	c := Client
	c.On("cmd:tojpg", WebpToJpgHandler)
}

func init() {
	QueueHandlerRegistration(registerWebpToJpgHandlers)
}
