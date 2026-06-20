package modules

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func qrColorNormalizeHex(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	s = strings.ToLower(s)
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return "", fmt.Errorf("invalid hex length")
	}
	for i := 0; i < 6; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return "", fmt.Errorf("invalid hex char")
		}
	}
	return s, nil
}

func qrColorFetchPNG(text, fg, bg string) ([]byte, error) {
	endpoint := fmt.Sprintf(
		"https://api.qrserver.com/v1/create-qr-code/?size=512x512&data=%s&color=%s&bgcolor=%s",
		url.QueryEscape(text), fg, bg,
	)
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
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return nil, fmt.Errorf("unexpected content-type: %s", ct)
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

func QRColorHandler(m *tg.NewMessage) error {
	raw := strings.TrimSpace(m.Args())
	if raw == "" {
		m.Reply("usage: <code>/qrcolor &lt;text&gt; &lt;fg_hex&gt; &lt;bg_hex&gt;</code>\nexample: <code>/qrcolor hello ff0000 ffffff</code>")
		return nil
	}

	fields := strings.Fields(raw)
	if len(fields) < 3 {
		m.Reply("need at least 3 args: <code>&lt;text&gt; &lt;fg_hex&gt; &lt;bg_hex&gt;</code>")
		return nil
	}

	bgRaw := fields[len(fields)-1]
	fgRaw := fields[len(fields)-2]
	text := strings.TrimSpace(strings.Join(fields[:len(fields)-2], " "))
	if text == "" {
		m.Reply("text cannot be empty")
		return nil
	}
	if len(text) > 900 {
		m.Reply("text too long, max 900 characters")
		return nil
	}

	fg, err := qrColorNormalizeHex(fgRaw)
	if err != nil {
		m.Reply("invalid fg hex: " + html.EscapeString(fgRaw))
		return nil
	}
	bg, err := qrColorNormalizeHex(bgRaw)
	if err != nil {
		m.Reply("invalid bg hex: " + html.EscapeString(bgRaw))
		return nil
	}

	status, _ := m.Reply("<code>generating colored qr...</code>")

	png, err := qrColorFetchPNG(text, fg, bg)
	if err != nil {
		msg := "error generating qr: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("qrcolor_%d.png", time.Now().UnixNano()))
	if werr := os.WriteFile(tmp, png, 0644); werr != nil {
		msg := "error saving qr: " + html.EscapeString(werr.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(tmp)

	preview := text
	if len(preview) > 80 {
		preview = preview[:77] + "..."
	}
	caption := fmt.Sprintf("<b>QR Colored</b>\n<b>Text:</b> <code>%s</code>\n<b>FG:</b> <code>#%s</code>  <b>BG:</b> <code>#%s</code>",
		html.EscapeString(preview), fg, bg)

	_, merr := m.ReplyMedia(tmp, &tg.MediaOptions{
		Caption:  caption,
		FileName: "qrcolor.png",
		MimeType: "image/png",
	})
	if merr != nil {
		msg := "upload failed: " + html.EscapeString(merr.Error())
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

func init() { QueueHandlerRegistration(registerQRColorHandlers) }

func registerQRColorHandlers() {
	c := Client
	c.On("cmd:qrcolor", QRColorHandler)
}
