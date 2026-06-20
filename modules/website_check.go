package modules

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func uptimeNormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	low := strings.ToLower(raw)
	if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
		raw = "https://" + raw
	}
	return raw
}

func uptimeStatusLabel(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "Up"
	case code >= 300 && code < 400:
		return "Redirect"
	case code >= 400 && code < 500:
		return "Client Error"
	case code >= 500 && code < 600:
		return "Server Error"
	default:
		return "Unknown"
	}
}

func UptimeHandler(m *tg.NewMessage) error {
	target := strings.TrimSpace(m.Args())
	if target == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			target = strings.TrimSpace(r.Text())
		}
	}
	if target == "" {
		m.Reply("usage: /uptime &lt;url&gt;")
		return nil
	}
	target = uptimeNormalizeURL(target)

	req, err := http.NewRequest(http.MethodHead, target, nil)
	if err != nil {
		m.Reply("invalid URL: " + html.EscapeString(err.Error()))
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (JuliaBot Uptime)")

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		out := "<b>Website Uptime Check</b>\n\n"
		out += "<b>URL:</b> " + html.EscapeString(target) + "\n"
		out += "<b>Status:</b> Down\n"
		out += "<b>Error:</b> " + html.EscapeString(err.Error()) + "\n"
		m.Reply(out)
		return nil
	}
	latency := time.Since(start)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented {
		getReq, gerr := http.NewRequest(http.MethodGet, target, nil)
		if gerr == nil {
			getReq.Header.Set("User-Agent", "Mozilla/5.0 (JuliaBot Uptime)")
			startG := time.Now()
			gResp, gErr := client.Do(getReq)
			if gErr == nil {
				latency = time.Since(startG)
				resp.Body.Close()
				resp = gResp
				defer resp.Body.Close()
			}
		}
	}

	server := resp.Header.Get("Server")
	if server == "" {
		server = "N/A"
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "N/A"
	}

	out := "<b>Website Uptime Check</b>\n\n"
	out += "<b>URL:</b> " + html.EscapeString(target) + "\n"
	out += fmt.Sprintf("<b>Status:</b> %s (%d %s)\n", uptimeStatusLabel(resp.StatusCode), resp.StatusCode, html.EscapeString(http.StatusText(resp.StatusCode)))
	out += fmt.Sprintf("<b>Latency:</b> %d ms\n", latency.Milliseconds())
	out += "<b>Server:</b> " + html.EscapeString(server) + "\n"
	out += "<b>Content-Type:</b> " + html.EscapeString(contentType) + "\n"
	if loc := resp.Header.Get("Location"); loc != "" {
		out += "<b>Location:</b> " + html.EscapeString(loc) + "\n"
	}
	m.Reply(out)
	return nil
}

func init() { QueueHandlerRegistration(registerUptimeHandlers) }
func registerUptimeHandlers() {
	c := Client
	c.On("cmd:uptime", UptimeHandler)
}
