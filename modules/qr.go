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

func QRHandler(m *tg.NewMessage) error {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	if text == "" {
		m.Reply("usage: /qr &lt;text&gt;")
		return nil
	}

	endpoint := "https://api.qrserver.com/v1/create-qr-code/?size=512x512&data=" + url.QueryEscape(text)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		m.Reply("error: " + html.EscapeString(err.Error()))
		return nil
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.Reply("error fetching qr: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.Reply(fmt.Sprintf("qr api returned status %d", resp.StatusCode))
		return nil
	}

	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("qr_%d.png", time.Now().UnixNano()))
	out, err := os.Create(tmpPath)
	if err != nil {
		m.Reply("error creating temp file: " + html.EscapeString(err.Error()))
		return nil
	}

	if _, err := io.Copy(out, io.LimitReader(resp.Body, 5*1024*1024)); err != nil {
		out.Close()
		os.Remove(tmpPath)
		m.Reply("error writing qr: " + html.EscapeString(err.Error()))
		return nil
	}
	out.Close()
	defer os.Remove(tmpPath)

	caption := "<b>QR Code</b>\n<code>" + html.EscapeString(text) + "</code>"
	_, err = m.ReplyMedia(tmpPath, &tg.MediaOptions{Caption: caption})
	if err != nil {
		m.Reply("error sending qr: " + html.EscapeString(err.Error()))
		return nil
	}
	return nil
}

func init() { QueueHandlerRegistration(registerQRHandlers) }

func registerQRHandlers() {
	c := Client
	c.On("cmd:qr", QRHandler)
}
