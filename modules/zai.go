package modules

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"golang.org/x/net/proxy"
)

const (
	zaiBase             = "https://chatglm.cn"
	zaiGuestEndpoint    = zaiBase + "/chatglm/user-api/guest/access"
	zaiStreamEndpoint   = zaiBase + "/chatglm/backend-api/assistant/stream"
	zaiSignSalt         = "8a1317a7468aa3ad86e997d08f3f31cb"
	zaiUserAgent        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36"
	zaiDefaultAssistant = "65940acff94777010aa6b796"
	zaiSessionTTL       = 25 * time.Minute
	zaiTrigger          = "@glm"
	zaiAIMsgHistoryMax  = 64
)

type zaiSession struct {
	token     string
	deviceID  string
	convID    string
	expiry    time.Time
}

var (
	zaiSessions   sync.Map
	zaiSessionsMu sync.Map
	zaiHTTPOnce   sync.Once
	zaiHTTP       *http.Client
)

func zaiClient() *http.Client {
	zaiHTTPOnce.Do(func() {
		transport := &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 20 * time.Second,
		}
		if raw := strings.TrimSpace(os.Getenv("PROXY")); raw != "" {
			if host, portStr, err := net.SplitHostPort(raw); err == nil {
				var auth *proxy.Auth
				if u := os.Getenv("PROXY_USERNAME"); u != "" {
					auth = &proxy.Auth{User: u, Password: os.Getenv("PROXY_PASSWORD")}
				}
				if dialer, derr := proxy.SOCKS5("tcp", net.JoinHostPort(host, portStr), auth, proxy.Direct); derr == nil {
					if cd, ok := dialer.(proxy.ContextDialer); ok {
						transport.DialContext = cd.DialContext
					} else {
						transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
							return dialer.Dial(network, addr)
						}
					}
				}
			}
		}
		zaiHTTP = &http.Client{Timeout: 120 * time.Second, Transport: transport}
	})
	return zaiHTTP
}

var (
	zaiUUIDCounter uint64
	zaiUUIDMu      sync.Mutex
)

func zaiHex32() string {
	zaiUUIDMu.Lock()
	zaiUUIDCounter++
	c := zaiUUIDCounter
	zaiUUIDMu.Unlock()
	b := make([]byte, 16)
	n := time.Now().UnixNano() ^ int64(c<<24)
	for i := range b {
		b[i] = byte((n >> (i % 8 * 8)) + int64(c) + int64(i*131))
	}
	return hex.EncodeToString(b)
}

func zaiMangleTimestamp() string {
	ms := strconv.FormatInt(time.Now().UnixMilli(), 10)
	e := len(ms)
	if e < 2 {
		return ms
	}
	sum := 0
	for _, ch := range ms {
		sum += int(ch - '0')
	}
	i := sum - int(ms[e-2]-'0')
	mod := ((i % 10) + 10) % 10
	return ms[:e-2] + strconv.Itoa(mod) + ms[e-1:e]
}

func zaiSignHeaders() (ts, nonce, sign string) {
	ts = zaiMangleTimestamp()
	nonce = zaiHex32()
	digest := md5.Sum([]byte(ts + "-" + nonce + "-" + zaiSignSalt))
	sign = hex.EncodeToString(digest[:])
	return ts, nonce, sign
}

func zaiSetHeaders(req *http.Request, deviceID, token string) {
	ts, nonce, sign := zaiSignHeaders()
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("App-Name", "chatglm")
	req.Header.Set("X-Device-Id", deviceID)
	req.Header.Set("X-App-Platform", "pc")
	req.Header.Set("X-App-Version", "0.0.1")
	req.Header.Set("X-App-fr", "default")
	req.Header.Set("X-Request-Id", zaiHex32())
	req.Header.Set("X-Exp-Groups", "")
	req.Header.Set("X-Device-Model", "")
	req.Header.Set("X-Device-Brand", "")
	req.Header.Set("X-Lang", "zh")
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Sign", sign)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", zaiBase)
	req.Header.Set("Referer", zaiBase+"/")
	req.Header.Set("User-Agent", zaiUserAgent)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

var (
	errZaiAuth  = errors.New("zai: unauthorized")
	errZaiEmpty = errors.New("zai: empty response")
)

func zaiFetchGuest(ctx context.Context) (string, string, error) {
	deviceID := zaiHex32()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, zaiGuestEndpoint, bytes.NewReader([]byte{}))
	if err != nil {
		return "", "", err
	}
	zaiSetHeaders(req, deviceID, "")
	resp, err := zaiClient().Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("guest endpoint returned status %d", resp.StatusCode)
	}
	var out struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Result  struct {
			AccessToken string `json:"access_token"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	if out.Result.AccessToken == "" {
		return "", "", fmt.Errorf("guest endpoint: %s", out.Message)
	}
	return out.Result.AccessToken, deviceID, nil
}

func zaiChatLock(chatID int64) *sync.Mutex {
	v, _ := zaiSessionsMu.LoadOrStore(chatID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func zaiGetSession(ctx context.Context, chatID int64, fresh bool) (*zaiSession, error) {
	if !fresh {
		if v, ok := zaiSessions.Load(chatID); ok {
			s := v.(*zaiSession)
			if time.Now().Before(s.expiry) {
				return s, nil
			}
		}
	}
	token, deviceID, err := zaiFetchGuest(ctx)
	if err != nil {
		return nil, err
	}
	s := &zaiSession{token: token, deviceID: deviceID, expiry: time.Now().Add(zaiSessionTTL)}
	zaiSessions.Store(chatID, s)
	return s, nil
}

func zaiResetSession(chatID int64) {
	zaiSessions.Delete(chatID)
}

func zaiChat(ctx context.Context, chatID int64, assistant, prompt string) (string, error) {
	s, err := zaiGetSession(ctx, chatID, false)
	if err != nil {
		return "", err
	}
	answer, err := zaiDoChat(ctx, s, assistant, prompt)
	if errors.Is(err, errZaiAuth) {
		s, err = zaiGetSession(ctx, chatID, true)
		if err != nil {
			return "", err
		}
		answer, err = zaiDoChat(ctx, s, assistant, prompt)
	}
	return answer, err
}

func zaiDoChat(ctx context.Context, s *zaiSession, assistant, prompt string) (string, error) {
	body := map[string]any{
		"assistant_id":    assistant,
		"conversation_id": s.convID,
		"meta_data": map[string]any{
			"if_plus_model":       false,
			"is_test":             false,
			"input_question_type": "xxxx",
			"channel":             "",
			"draft_id":            "",
			"quote_log_id":        "",
			"platform":            "pc",
		},
		"messages": []map[string]any{
			{"role": "user", "content": []map[string]any{{"type": "text", "text": prompt}}},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, zaiStreamEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	zaiSetHeaders(req, s.deviceID, s.token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := zaiClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", errZaiAuth
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("stream endpoint returned status %d", resp.StatusCode)
	}

	answer, convID, err := zaiParseStream(resp.Body)
	if err != nil {
		return "", err
	}
	if convID != "" {
		s.convID = convID
		s.expiry = time.Now().Add(zaiSessionTTL)
	}
	return answer, nil
}

type zaiStreamPart struct {
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type zaiStreamEnvelope struct {
	Status         string          `json:"status"`
	ConversationID string          `json:"conversation_id"`
	Parts          []zaiStreamPart `json:"parts"`
}

func zaiParseStream(r interface {
	Read([]byte) (int, error)
}) (string, string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	var latest, convID string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var env zaiStreamEnvelope
		if json.Unmarshal([]byte(data), &env) != nil {
			continue
		}
		if env.ConversationID != "" {
			convID = env.ConversationID
		}
		for _, part := range env.Parts {
			if part.Role != "" && part.Role != "assistant" {
				continue
			}
			for _, c := range part.Content {
				if c.Type == "text" && c.Text != "" {
					latest = c.Text
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	latest = strings.TrimSpace(latest)
	if latest == "" {
		return "", "", errZaiEmpty
	}
	return latest, convID, nil
}

func zaiUserError(err error) string {
	switch {
	case errors.Is(err, errZaiAuth):
		return "the AI service rejected the session, please try again in a moment."
	case errors.Is(err, errZaiEmpty):
		return "the AI service returned an empty response, please try again."
	default:
		return "request failed, please try again later."
	}
}

func zaiRunChat(m *tg.NewMessage, prompt string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		m.Reply("usage: <code>/glm &lt;prompt&gt;</code>\nexample: <code>/glm explain quantum entanglement simply</code>\nyou can also start a message with <code>@glm</code> or reply to me.", &tg.SendOptions{ParseMode: "HTML"})
		return nil
	}
	if len(prompt) > 8000 {
		prompt = prompt[:8000]
	}

	chatID := m.ChatID()
	status, _ := m.Reply("<code>thinking...</code>", &tg.SendOptions{ParseMode: "HTML"})
	if status != nil {
		zaiMarkAIMessage(chatID, status.ID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	lock := zaiChatLock(chatID)
	lock.Lock()
	answer, err := zaiChat(ctx, chatID, zaiDefaultAssistant, prompt)
	lock.Unlock()
	if err != nil {
		msg := zaiUserError(err)
		if status != nil {
			status.Edit(msg)
		} else {
			if r, rerr := m.Reply(msg); rerr == nil && r != nil {
				zaiMarkAIMessage(chatID, r.ID)
			}
		}
		return nil
	}

	answer = strings.TrimSpace(answer)

	if len([]rune(answer)) > 4000 {
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("zai_%d.txt", time.Now().UnixNano()))
		if werr := os.WriteFile(tmp, []byte(answer), 0644); werr == nil {
			defer os.Remove(tmp)
			if media, merr := m.ReplyMedia(tmp, &tg.MediaOptions{Caption: "<i>response sent as file</i>", FileName: "response.txt"}); merr == nil {
				if media != nil {
					zaiMarkAIMessage(chatID, media.ID)
				}
				if status != nil {
					status.Delete()
				}
				return nil
			}
		}
		answer = string([]rune(answer)[:4000])
	}

	out := mdToTelegramHTML(answer)
	opts := &tg.SendOptions{ParseMode: "HTML"}
	if status != nil {
		if _, eerr := status.Edit(out, opts); eerr != nil {
			if _, e2 := status.Edit(html.EscapeString(answer), opts); e2 != nil {
				if r, rerr := m.Reply(out, opts); rerr == nil && r != nil {
					zaiMarkAIMessage(chatID, r.ID)
				}
			}
		}
	} else {
		if r, rerr := m.Reply(out, opts); rerr == nil && r != nil {
			zaiMarkAIMessage(chatID, r.ID)
		}
	}
	return nil
}

func ZaiHandler(m *tg.NewMessage) error {
	return zaiRunChat(m, m.Args())
}

func ZaiResetHandler(m *tg.NewMessage) error {
	zaiResetSession(m.ChatID())
	m.Reply("<b>conversation reset.</b>", &tg.SendOptions{ParseMode: "HTML"})
	return nil
}

var zaiTriggers = []string{"@glm", "@z"}

func zaiStripTrigger(text string) (string, bool) {
	trimmed := strings.TrimLeft(text, " \t")
	lower := strings.ToLower(trimmed)
	for _, trig := range zaiTriggers {
		if !strings.HasPrefix(lower, trig) {
			continue
		}
		after := trimmed[len(trig):]
		if after != "" {
			r := rune(after[0])
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
				continue
			}
		}
		rest := strings.TrimLeft(after, " \t:,\n")
		return rest, true
	}
	return "", false
}

var (
	zaiAIMsgMu      sync.Mutex
	zaiAIMsgIDs     = map[int64]map[int32]struct{}{}
	zaiAIMsgHistory = map[int64][]int32{}
)

func zaiMarkAIMessage(chatID int64, msgID int32) {
	if msgID == 0 {
		return
	}
	zaiAIMsgMu.Lock()
	defer zaiAIMsgMu.Unlock()
	set, ok := zaiAIMsgIDs[chatID]
	if !ok {
		set = map[int32]struct{}{}
		zaiAIMsgIDs[chatID] = set
	}
	if _, exists := set[msgID]; exists {
		return
	}
	set[msgID] = struct{}{}
	hist := append(zaiAIMsgHistory[chatID], msgID)
	if len(hist) > zaiAIMsgHistoryMax {
		drop := hist[:len(hist)-zaiAIMsgHistoryMax]
		hist = hist[len(hist)-zaiAIMsgHistoryMax:]
		for _, id := range drop {
			delete(set, id)
		}
	}
	zaiAIMsgHistory[chatID] = hist
}

func zaiIsAIMessage(chatID int64, msgID int32) bool {
	zaiAIMsgMu.Lock()
	defer zaiAIMsgMu.Unlock()
	if set, ok := zaiAIMsgIDs[chatID]; ok {
		_, hit := set[msgID]
		return hit
	}
	return false
}

func ZaiWatcher(m *tg.NewMessage) error {
	if m.IsCommand() {
		return nil
	}
	text := m.Text()
	if text == "" {
		return nil
	}

	if prompt, ok := zaiStripTrigger(text); ok {
		return zaiRunChat(m, prompt)
	}

	if m.IsReply() && zaiIsAIMessage(m.ChatID(), m.ReplyToMsgID()) {
		return zaiRunChat(m, text)
	}
	return nil
}

func init() { QueueHandlerRegistration(registerZaiHandlers) }

func registerZaiHandlers() {
	c := Client
	c.On("cmd:glmnew", ZaiResetHandler)
	c.On("cmd:aireset", ZaiResetHandler)
	c.On(tg.OnNewMessage, ZaiWatcher)
	c.On("cmd:zai", ZaiHandler)
	c.On("cmd:glm", ZaiHandler)
	c.On("cmd:ai", ZaiHandler)
}

func mdToTelegramHTML(md string) string {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	md = strings.ReplaceAll(md, "\r", "\n")
	lines := strings.Split(md, "\n")

	var out strings.Builder
	out.Grow(len(md) + 64)

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trim := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trim, "```") {
			lang := strings.TrimSpace(trim[3:])
			if lang != "" {
				out.WriteString(`<pre><code class="language-`)
				out.WriteString(html.EscapeString(lang))
				out.WriteString(`">`)
			} else {
				out.WriteString("<pre><code>")
			}
			i++
			first := true
			for i < len(lines) && !strings.HasPrefix(strings.TrimLeft(lines[i], " \t"), "```") {
				if !first {
					out.WriteByte('\n')
				}
				out.WriteString(html.EscapeString(lines[i]))
				first = false
				i++
			}
			out.WriteString("</code></pre>")
			if i+1 < len(lines) {
				out.WriteByte('\n')
			}
			continue
		}

		hdr := trim
		level := 0
		for strings.HasPrefix(hdr, "#") && level < 6 {
			hdr = hdr[1:]
			level++
		}
		if level > 0 && strings.HasPrefix(hdr, " ") {
			out.WriteString("<b>")
			out.WriteString(zaiInlineMD(strings.TrimSpace(hdr)))
			out.WriteString("</b>")
		} else {
			out.WriteString(zaiInlineMD(line))
		}
		if i+1 < len(lines) {
			out.WriteByte('\n')
		}
	}

	result := out.String()
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return strings.TrimRight(strings.TrimLeft(result, "\n"), "\n")
}

func zaiInlineMD(s string) string {
	var out strings.Builder
	out.Grow(len(s) + 16)
	i := 0
	for i < len(s) {
		c := s[i]
		switch c {
		case '\\':
			if i+1 < len(s) {
				out.WriteString(html.EscapeString(string(s[i+1])))
				i += 2
				continue
			}
		case '`':
			end := strings.IndexByte(s[i+1:], '`')
			if end >= 0 {
				out.WriteString("<code>")
				out.WriteString(html.EscapeString(s[i+1 : i+1+end]))
				out.WriteString("</code>")
				i += end + 2
				continue
			}
		case '*':
			if i+1 < len(s) && s[i+1] == '*' {
				end := strings.Index(s[i+2:], "**")
				if end >= 0 {
					out.WriteString("<b>")
					out.WriteString(zaiInlineMD(s[i+2 : i+2+end]))
					out.WriteString("</b>")
					i += end + 4
					continue
				}
			}
			end := strings.IndexByte(s[i+1:], '*')
			if end >= 0 && end > 0 {
				out.WriteString("<i>")
				out.WriteString(zaiInlineMD(s[i+1 : i+1+end]))
				out.WriteString("</i>")
				i += end + 2
				continue
			}
		case '_':
			if i+1 < len(s) && s[i+1] == '_' {
				end := strings.Index(s[i+2:], "__")
				if end >= 0 {
					out.WriteString("<b>")
					out.WriteString(zaiInlineMD(s[i+2 : i+2+end]))
					out.WriteString("</b>")
					i += end + 4
					continue
				}
			}
		case '[':
			closeBracket := strings.IndexByte(s[i:], ']')
			if closeBracket > 0 && closeBracket+1 < len(s)-i && s[i+closeBracket+1] == '(' {
				closeParen := strings.IndexByte(s[i+closeBracket+2:], ')')
				if closeParen >= 0 {
					linkText := s[i+1 : i+closeBracket]
					linkURL := s[i+closeBracket+2 : i+closeBracket+2+closeParen]
					out.WriteString(`<a href="`)
					out.WriteString(html.EscapeString(linkURL))
					out.WriteString(`">`)
					out.WriteString(zaiInlineMD(linkText))
					out.WriteString("</a>")
					i += closeBracket + closeParen + 3
					continue
				}
			}
		}
		out.WriteString(html.EscapeString(string(c)))
		i++
	}
	return out.String()
}
