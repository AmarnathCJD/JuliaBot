package modules

import (
	"fmt"
	"html"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

func circleCropApplyMask(src image.Image) image.Image {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	size := w
	if h < size {
		size = h
	}
	if size <= 0 {
		return src
	}

	offX := bounds.Min.X + (w-size)/2
	offY := bounds.Min.Y + (h-size)/2
	squareRect := image.Rect(offX, offY, offX+size, offY+size)

	square := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(square, square.Bounds(), src, squareRect.Min, draw.Src)

	dc := gg.NewContext(size, size)
	dc.DrawCircle(float64(size)/2, float64(size)/2, float64(size)/2)
	dc.Clip()
	dc.DrawImage(square, 0, 0)
	return dc.Image()
}

func circleCropDecodeFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func circleCropSavePNG(img image.Image, outPath string) error {
	return gg.NewContextForImage(img).SavePNG(outPath)
}

func CircleCropHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("reply to a photo with <code>/circle</code> to crop it into a circle")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("error: " + html.EscapeString(err.Error()))
		return nil
	}

	if r.Photo() == nil {
		m.Reply("the replied message is not a photo")
		return nil
	}

	status, _ := m.Reply("<code>cropping into circle...</code>")

	inPath := filepath.Join(os.TempDir(), fmt.Sprintf("circle_in_%d.jpg", time.Now().UnixNano()))
	_, err = m.Client.DownloadMedia(r.Media(), &tg.DownloadOptions{FileName: inPath})
	if err != nil {
		msg := "error downloading photo: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(inPath)

	src, err := circleCropDecodeFile(inPath)
	if err != nil {
		msg := "error decoding image: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	b := src.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		msg := "invalid image dimensions"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	out := circleCropApplyMask(src)

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("circle_out_%d.png", time.Now().UnixNano()))
	if err := circleCropSavePNG(out, outPath); err != nil {
		msg := "error saving png: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(outPath)

	side := int(math.Min(float64(b.Dx()), float64(b.Dy())))
	caption := fmt.Sprintf("<b>Circle Crop</b>\n<b>Size:</b> <code>%dx%d</code>", side, side)

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		Caption:  caption,
		FileName: "circle.png",
		MimeType: "image/png",
	})
	if merr != nil {
		m.Reply("upload failed: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerImageCropHandlers() {
	c := Client
	c.On("cmd:circle", CircleCropHandler)
}

func init() { QueueHandlerRegistration(registerImageCropHandlers) }
