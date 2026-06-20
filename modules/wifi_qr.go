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

func wifiQREscapeField(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', ';', ',', ':', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func wifiQRNormalizeSecurity(s string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "WPA", "WPA2", "WPA3":
		return "WPA", true
	case "WEP":
		return "WEP", true
	case "NOPASS", "NONE", "OPEN", "":
		return "nopass", true
	}
	return "", false
}

func wifiQRFetchPNG(payload string) ([]byte, error) {
	endpoint := fmt.Sprintf(
		"https://api.qrserver.com/v1/create-qr-code/?size=512x512&data=%s",
		url.QueryEscape(payload),
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

func WifiQRHandler(m *tg.NewMessage) error {
	raw := strings.TrimSpace(m.Args())
	if raw == "" {
		m.Reply("usage: <code>/wifiqr &lt;ssid&gt; &lt;password&gt; [security]</code>\nsecurity: <code>WPA</code> (default), <code>WEP</code>, <code>nopass</code>\nexample: <code>/wifiqr MyWifi mypass123 WPA</code>")
		return nil
	}

	fields := strings.Fields(raw)
	if len(fields) < 2 {
		m.Reply("need at least 2 args: <code>&lt;ssid&gt; &lt;password&gt; [security]</code>")
		return nil
	}

	var ssid, password, secRaw string
	if len(fields) == 2 {
		ssid = fields[0]
		password = fields[1]
		secRaw = "WPA"
	} else {
		last := fields[len(fields)-1]
		if _, ok := wifiQRNormalizeSecurity(last); ok {
			secRaw = last
			ssid = fields[0]
			password = strings.Join(fields[1:len(fields)-1], " ")
		} else {
			ssid = fields[0]
			password = strings.Join(fields[1:], " ")
			secRaw = "WPA"
		}
	}

	if ssid == "" {
		m.Reply("ssid cannot be empty")
		return nil
	}
	if len(ssid) > 128 {
		m.Reply("ssid too long, max 128 characters")
		return nil
	}
	if len(password) > 256 {
		m.Reply("password too long, max 256 characters")
		return nil
	}

	security, ok := wifiQRNormalizeSecurity(secRaw)
	if !ok {
		m.Reply("invalid security: " + html.EscapeString(secRaw) + "\nallowed: <code>WPA</code>, <code>WEP</code>, <code>nopass</code>")
		return nil
	}

	if security == "nopass" {
		password = ""
	} else if password == "" {
		m.Reply("password required for security <code>" + security + "</code>")
		return nil
	}

	payload := fmt.Sprintf("WIFI:T:%s;S:%s;P:%s;;",
		security,
		wifiQREscapeField(ssid),
		wifiQREscapeField(password),
	)

	status, _ := m.Reply("<code>generating wifi qr...</code>")

	png, err := wifiQRFetchPNG(payload)
	if err != nil {
		msg := "error generating qr: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("wifiqr_%d.png", time.Now().UnixNano()))
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

	pwDisplay := password
	if pwDisplay == "" {
		pwDisplay = "(none)"
	}
	caption := fmt.Sprintf("<b>WiFi QR</b>\n<b>SSID:</b> <code>%s</code>\n<b>Password:</b> <code>%s</code>\n<b>Security:</b> <code>%s</code>\nScan with your phone camera to connect.",
		html.EscapeString(ssid),
		html.EscapeString(pwDisplay),
		security,
	)

	_, merr := m.ReplyMedia(tmp, &tg.MediaOptions{
		Caption:  caption,
		FileName: "wifiqr.png",
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

func init() { QueueHandlerRegistration(registerWifiQRHandlers) }

func registerWifiQRHandlers() {
	c := Client
	c.On("cmd:wifiqr", WifiQRHandler)
}
