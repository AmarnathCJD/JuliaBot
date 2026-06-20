package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type qrServerSymbol struct {
	Seq   int     `json:"seq"`
	Data  string  `json:"data"`
	Error *string `json:"error"`
}

type qrServerResult struct {
	Type   string           `json:"type"`
	Symbol []qrServerSymbol `json:"symbol"`
}

func decodeQRFromFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, "https://api.qrserver.com/v1/read-qr-code/", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("qr api returned status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}

	var results []qrServerResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return "", fmt.Errorf("invalid response: %s", strings.TrimSpace(string(raw)))
	}

	var decoded []string
	var lastErr string
	for _, r := range results {
		for _, s := range r.Symbol {
			if s.Error != nil && *s.Error != "" {
				lastErr = *s.Error
				continue
			}
			if strings.TrimSpace(s.Data) != "" {
				decoded = append(decoded, s.Data)
			}
		}
	}

	if len(decoded) == 0 {
		if lastErr != "" {
			return "", fmt.Errorf("%s", lastErr)
		}
		return "", fmt.Errorf("no qr code detected")
	}

	return strings.Join(decoded, "\n"), nil
}

func QRReadHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("usage: reply to a QR-code image with <code>/qrread</code>")
		return nil
	}
	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("error: " + html.EscapeString(err.Error()))
		return nil
	}
	if !r.IsMedia() {
		m.Reply("the replied message has no media")
		return nil
	}

	status, _ := m.Reply("<code>downloading image...</code>")

	path, err := m.Client.DownloadMedia(r.Media())
	if err != nil {
		msg := "error downloading: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(path)

	if status != nil {
		status.Edit("<code>decoding qr...</code>")
	}

	text, err := decodeQRFromFile(path)
	if err != nil {
		msg := "error decoding qr: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	out := "<b>QR Decoded</b>\n<code>" + html.EscapeString(text) + "</code>"
	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func init() { QueueHandlerRegistration(registerQRReadHandlers) }

func registerQRReadHandlers() {
	c := Client
	c.On("cmd:qrread", QRReadHandler)
}
