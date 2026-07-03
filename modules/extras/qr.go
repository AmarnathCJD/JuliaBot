package extras

import (
	"bytes"
	"encoding/json"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"html"
	"image"
	_ "image/png"
	"io"
	modules "main/modules"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func initFromSrc_qr_0_1() { modules.QueueHandlerRegistration(registerQRHandlers) }

func registerQRHandlers() {
	c := modules.Client
	c.On("cmd:qr", QRHandler)
}
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

func initFromSrc_qr_image_1_1() { modules.QueueHandlerRegistration(registerQRColorHandlers) }

func registerQRColorHandlers() {
	c := modules.Client
	c.On("cmd:qrcolor", QRColorHandler)
}
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

func initFromSrc_qr_pretty_2_1() { modules.QueueHandlerRegistration(registerQRArtHandlers) }

func registerQRArtHandlers() {
	c := modules.Client
	c.On("cmd:qrart", QRArtHandler)
}
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

func initFromSrc_qrcode_decode_3_1() { modules.QueueHandlerRegistration(registerQRReadHandlers) }

func registerQRReadHandlers() {
	c := modules.Client
	c.On("cmd:qrread", QRReadHandler)
}

func init() {
	initFromSrc_qr_0_1()
	initFromSrc_qr_image_1_1()
	initFromSrc_qr_pretty_2_1()
	initFromSrc_qrcode_decode_3_1()
}
