package modules

import (
	"bytes"
	"fmt"
	"html"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func fetchQRPrettyPNG(text string) ([]byte, error) {
	endpoint := "https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=" + url.QueryEscape(text)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qr api returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty qr response")
	}
	return data, nil
}

func qrPNGToASCII(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return "", fmt.Errorf("invalid image dimensions")
	}

	cell := w / 50
	if cell < 1 {
		cell = 1
	}

	cols := w / cell
	rows := h / cell

	if cols > 80 {
		cell = w / 60
		if cell < 1 {
			cell = 1
		}
		cols = w / cell
		rows = h / cell
	}

	var sb strings.Builder
	for ry := 0; ry < rows; ry += 2 {
		for rx := 0; rx < cols; rx++ {
			px := bounds.Min.X + rx*cell + cell/2
			py1 := bounds.Min.Y + ry*cell + cell/2
			py2 := bounds.Min.Y + (ry+1)*cell + cell/2

			top := isDark(img, px, py1)
			bot := false
			if ry+1 < rows {
				bot = isDark(img, px, py2)
			}

			switch {
			case top && bot:
				sb.WriteString("█")
			case top && !bot:
				sb.WriteString("▀")
			case !top && bot:
				sb.WriteString("▄")
			default:
				sb.WriteString(" ")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func isDark(img image.Image, x, y int) bool {
	b := img.Bounds()
	if x < b.Min.X {
		x = b.Min.X
	}
	if y < b.Min.Y {
		y = b.Min.Y
	}
	if x >= b.Max.X {
		x = b.Max.X - 1
	}
	if y >= b.Max.Y {
		y = b.Max.Y - 1
	}
	r, g, bl, _ := img.At(x, y).RGBA()
	lum := (uint32(r) + uint32(g) + uint32(bl)) / 3
	return lum < 0x8000
}

func QRArtHandler(m *tg.NewMessage) error {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	if text == "" {
		m.Reply("usage: <code>/qrart &lt;text&gt;</code>\nor reply to a message with <code>/qrart</code>")
		return nil
	}
	if len(text) > 100 {
		m.Reply("text too long, max 100 characters")
		return nil
	}

	status, _ := m.Reply("<code>generating qr art...</code>")

	png, err := fetchQRPrettyPNG(text)
	if err != nil {
		msg := "error generating qr: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	art, err := qrPNGToASCII(png)
	if err != nil {
		msg := "error converting qr: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	out := "<pre>" + html.EscapeString(art) + "</pre>"
	if len(out) > 4000 {
		out = out[:3990] + "</pre>"
	}

	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func init() { QueueHandlerRegistration(registerQRArtHandlers) }

func registerQRArtHandlers() {
	c := Client
	c.On("cmd:qrart", QRArtHandler)
}
