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

func shortenQRIsgdShorten(longURL string) (string, error) {
	endpoint := fmt.Sprintf("https://is.gd/create.php?format=simple&url=%s", url.QueryEscape(longURL))
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}
	body := strings.TrimSpace(string(data))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("is.gd returned status %d: %s", resp.StatusCode, body)
	}
	if !strings.HasPrefix(body, "http://") && !strings.HasPrefix(body, "https://") {
		return "", fmt.Errorf("%s", body)
	}
	return body, nil
}

func shortenQRFetchPNG(text string) ([]byte, error) {
	endpoint := fmt.Sprintf(
		"https://api.qrserver.com/v1/create-qr-code/?size=512x512&data=%s",
		url.QueryEscape(text),
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

func ShortenQRHandler(m *tg.NewMessage) error {
	raw := strings.TrimSpace(m.Args())
	if raw == "" {
		m.Reply("usage: <code>/shortqr &lt;url&gt;</code>\nexample: <code>/shortqr https://example.com/very/long/path</code>")
		return nil
	}

	target := strings.Fields(raw)[0]
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	status, _ := m.Reply("<code>shortening url...</code>")

	short, err := shortenQRIsgdShorten(target)
	if err != nil {
		msg := "shorten failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if status != nil {
		status.Edit("<code>generating qr...</code>")
	}

	png, err := shortenQRFetchPNG(short)
	if err != nil {
		msg := "qr generation failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("shortqr_%d.png", time.Now().UnixNano()))
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

	caption := fmt.Sprintf("<b>Short URL:</b> <a href=\"%s\">%s</a>\n<b>Original:</b> <code>%s</code>",
		html.EscapeString(short), html.EscapeString(short), html.EscapeString(target))

	_, merr := m.ReplyMedia(tmp, &tg.MediaOptions{
		Caption:  caption,
		FileName: "shortqr.png",
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

func init() { QueueHandlerRegistration(registerShortenQRHandlers) }

func registerShortenQRHandlers() {
	c := Client
	c.On("cmd:shortqr", ShortenQRHandler)
}
