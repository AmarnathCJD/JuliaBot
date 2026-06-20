package modules

import (
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func pasteGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func pasteHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func tryDpasteCom(text string) (string, error) {
	form := url.Values{}
	form.Set("content", text)
	form.Set("syntax", "text")
	form.Set("expiry_days", "7")
	req, err := http.NewRequest("POST", "https://dpaste.com/api/v2/", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := pasteHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if loc := resp.Header.Get("Location"); loc != "" {
		return strings.TrimSpace(loc), nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(body))
	if strings.HasPrefix(s, "http") {
		return s, nil
	}
	return "", nil
}

func tryDpasteOrg(text string) (string, error) {
	form := url.Values{}
	form.Set("content", text)
	form.Set("lexer", "_text")
	form.Set("expires", "604800")
	form.Set("format", "url")
	req, err := http.NewRequest("POST", "https://dpaste.org/api/", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := pasteHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(body))
	s = strings.Trim(s, "\"")
	if strings.HasPrefix(s, "http") {
		return s, nil
	}
	return "", nil
}

func tryPasteRs(text string) (string, error) {
	req, err := http.NewRequest("POST", "https://paste.rs/", strings.NewReader(text))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := pasteHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(body))
	if strings.HasPrefix(s, "http") {
		return s, nil
	}
	return "", nil
}

func DpasteHandler(m *tg.NewMessage) error {
	text := pasteGetInput(m)
	if text == "" {
		m.Reply("usage: /dpaste &lt;text&gt; or reply to a message")
		return nil
	}
	edit, _ := m.Reply("uploading paste...")
	if u, err := tryDpasteCom(text); err == nil && u != "" {
		if edit != nil {
			edit.Edit("paste: " + html.EscapeString(u) + "\n<i>via dpaste.com</i>")
		} else {
			m.Reply("paste: " + html.EscapeString(u))
		}
		return nil
	}
	if u, err := tryDpasteOrg(text); err == nil && u != "" {
		if edit != nil {
			edit.Edit("paste: " + html.EscapeString(u) + "\n<i>via dpaste.org</i>")
		} else {
			m.Reply("paste: " + html.EscapeString(u))
		}
		return nil
	}
	if u, err := tryPasteRs(text); err == nil && u != "" {
		if edit != nil {
			edit.Edit("paste: " + html.EscapeString(u) + "\n<i>via paste.rs</i>")
		} else {
			m.Reply("paste: " + html.EscapeString(u))
		}
		return nil
	}
	if edit != nil {
		edit.Edit("all paste providers failed")
	} else {
		m.Reply("all paste providers failed")
	}
	return nil
}

func tryHastebin(text string) (string, error) {
	req, err := http.NewRequest("POST", "https://hastebin.com/documents", strings.NewReader(text))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := pasteHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(body))
	idx := strings.Index(s, "\"key\"")
	if idx == -1 {
		return "", nil
	}
	rest := s[idx+5:]
	colon := strings.Index(rest, ":")
	if colon == -1 {
		return "", nil
	}
	rest = rest[colon+1:]
	q1 := strings.Index(rest, "\"")
	if q1 == -1 {
		return "", nil
	}
	rest = rest[q1+1:]
	q2 := strings.Index(rest, "\"")
	if q2 == -1 {
		return "", nil
	}
	key := rest[:q2]
	if key == "" {
		return "", nil
	}
	return "https://hastebin.com/" + key, nil
}

func HastebinHandler(m *tg.NewMessage) error {
	text := pasteGetInput(m)
	if text == "" {
		m.Reply("usage: /hastebin &lt;text&gt; or reply to a message")
		return nil
	}
	edit, _ := m.Reply("uploading to hastebin...")
	if u, err := tryHastebin(text); err == nil && u != "" {
		if edit != nil {
			edit.Edit("paste: " + html.EscapeString(u))
		} else {
			m.Reply("paste: " + html.EscapeString(u))
		}
		return nil
	}
	if u, err := tryDpasteCom(text); err == nil && u != "" {
		if edit != nil {
			edit.Edit("hastebin failed, fallback: " + html.EscapeString(u))
		} else {
			m.Reply("paste: " + html.EscapeString(u))
		}
		return nil
	}
	if edit != nil {
		edit.Edit("hastebin upload failed")
	} else {
		m.Reply("hastebin upload failed")
	}
	return nil
}

func pasteNormalizeRawURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "dpaste.com/") && !strings.HasSuffix(raw, ".txt") {
		return strings.TrimRight(raw, "/") + ".txt"
	}
	if strings.Contains(raw, "dpaste.org/") {
		parts := strings.Split(strings.TrimRight(raw, "/"), "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if last != "" && !strings.Contains(last, ".") {
				return "https://dpaste.org/" + last + "/raw"
			}
		}
	}
	if strings.Contains(raw, "hastebin.com/") && !strings.Contains(raw, "/raw/") {
		parts := strings.Split(strings.TrimRight(raw, "/"), "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if last != "" {
				return "https://hastebin.com/raw/" + last
			}
		}
	}
	return raw
}

func PasteGetHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("usage: /paste_get &lt;url&gt;")
		return nil
	}
	if !strings.HasPrefix(arg, "http://") && !strings.HasPrefix(arg, "https://") {
		arg = "https://" + arg
	}
	raw := pasteNormalizeRawURL(arg)
	req, err := http.NewRequest("GET", raw, nil)
	if err != nil {
		m.Reply("request error: " + html.EscapeString(err.Error()))
		return nil
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := pasteHTTPClient().Do(req)
	if err != nil {
		m.Reply("fetch failed: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		m.Reply("fetch failed: HTTP " + html.EscapeString(resp.Status))
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 200000))
	if err != nil {
		m.Reply("read failed: " + html.EscapeString(err.Error()))
		return nil
	}
	content := strings.TrimSpace(string(body))
	if content == "" {
		m.Reply("paste appears empty")
		return nil
	}
	escaped := html.EscapeString(content)
	if len(escaped) > 3800 {
		escaped = escaped[:3800] + "\n... (truncated)"
	}
	m.Reply("<pre>" + escaped + "</pre>")
	return nil
}

func init() { QueueHandlerRegistration(registerPasteExtrasHandlers) }
func registerPasteExtrasHandlers() {
	c := Client
	c.On("cmd:dpaste", DpasteHandler)
	c.On("cmd:hastebin", HastebinHandler)
	c.On("cmd:paste_get", PasteGetHandler)
}
