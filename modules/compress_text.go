package modules

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"html"
	"io"
	"net/url"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func ctGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func ctReplyCode(m *tg.NewMessage, s string) {
	escaped := html.EscapeString(s)
	if len(escaped) > 4000 {
		escaped = escaped[:4000] + "\n... (truncated)"
	}
	m.Reply("<code>" + escaped + "</code>")
}

func ctReplyHTML(m *tg.NewMessage, s string) {
	if len(s) > 4000 {
		s = s[:4000] + "\n... (truncated)"
	}
	m.Reply(s)
}

func ZipHandler(m *tg.NewMessage) error {
	text := ctGetInput(m)
	if text == "" {
		m.Reply("usage: /zip &lt;text&gt;")
		return nil
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(text)); err != nil {
		m.Reply("compress failed: " + html.EscapeString(err.Error()))
		return nil
	}
	if err := w.Close(); err != nil {
		m.Reply("compress failed: " + html.EscapeString(err.Error()))
		return nil
	}
	ctReplyCode(m, base64.StdEncoding.EncodeToString(buf.Bytes()))
	return nil
}

func UnzipHandler(m *tg.NewMessage) error {
	text := ctGetInput(m)
	if text == "" {
		m.Reply("usage: /unzip &lt;base64-gzip&gt;")
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
	if err != nil {
		m.Reply("decode failed: " + html.EscapeString(err.Error()))
		return nil
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		m.Reply("gunzip failed: " + html.EscapeString(err.Error()))
		return nil
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		m.Reply("gunzip failed: " + html.EscapeString(err.Error()))
		return nil
	}
	ctReplyCode(m, string(out))
	return nil
}

func B64Handler(m *tg.NewMessage) error {
	text := ctGetInput(m)
	if text == "" {
		m.Reply("usage: /b64 &lt;text&gt;")
		return nil
	}
	ctReplyCode(m, base64.StdEncoding.EncodeToString([]byte(text)))
	return nil
}

func UnB64Handler(m *tg.NewMessage) error {
	text := ctGetInput(m)
	if text == "" {
		m.Reply("usage: /unb64 &lt;base64&gt;")
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(text))
		if err != nil {
			m.Reply("decode failed: " + html.EscapeString(err.Error()))
			return nil
		}
	}
	ctReplyCode(m, string(data))
	return nil
}

func HexHandler(m *tg.NewMessage) error {
	text := ctGetInput(m)
	if text == "" {
		m.Reply("usage: /hex &lt;text&gt;")
		return nil
	}
	ctReplyCode(m, hex.EncodeToString([]byte(text)))
	return nil
}

func UnHexHandler(m *tg.NewMessage) error {
	text := ctGetInput(m)
	if text == "" {
		m.Reply("usage: /unhex &lt;hex&gt;")
		return nil
	}
	cleaned := strings.ReplaceAll(strings.TrimSpace(text), " ", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	data, err := hex.DecodeString(cleaned)
	if err != nil {
		m.Reply("decode failed: " + html.EscapeString(err.Error()))
		return nil
	}
	ctReplyCode(m, string(data))
	return nil
}

func URLHandler(m *tg.NewMessage) error {
	text := ctGetInput(m)
	if text == "" {
		m.Reply("usage: /url &lt;text&gt;")
		return nil
	}
	ctReplyCode(m, url.QueryEscape(text))
	return nil
}

func UnURLHandler(m *tg.NewMessage) error {
	text := ctGetInput(m)
	if text == "" {
		m.Reply("usage: /unurl &lt;encoded&gt;")
		return nil
	}
	out, err := url.QueryUnescape(strings.TrimSpace(text))
	if err != nil {
		m.Reply("decode failed: " + html.EscapeString(err.Error()))
		return nil
	}
	ctReplyCode(m, out)
	return nil
}

func jwtPrettyJSON(seg string) string {
	data, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(seg)
		if err != nil {
			return seg
		}
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return string(data)
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(data)
	}
	return string(pretty)
}

func JWTDecHandler(m *tg.NewMessage) error {
	text := ctGetInput(m)
	if text == "" {
		m.Reply("usage: /jwtdec &lt;token&gt;")
		return nil
	}
	parts := strings.Split(strings.TrimSpace(text), ".")
	if len(parts) != 3 {
		m.Reply("invalid jwt: expected 3 segments")
		return nil
	}
	header := jwtPrettyJSON(parts[0])
	payload := jwtPrettyJSON(parts[1])
	signature := parts[2]
	var b strings.Builder
	b.WriteString("<b>Header</b>\n<code>")
	b.WriteString(html.EscapeString(header))
	b.WriteString("</code>\n\n<b>Payload</b>\n<code>")
	b.WriteString(html.EscapeString(payload))
	b.WriteString("</code>\n\n<b>Signature</b>\n<code>")
	b.WriteString(html.EscapeString(signature))
	b.WriteString("</code>")
	ctReplyHTML(m, b.String())
	return nil
}

func init() { QueueHandlerRegistration(registerCompressTextHandlers) }
func registerCompressTextHandlers() {
	c := Client
	c.On("cmd:zip", ZipHandler)
	c.On("cmd:unzip", UnzipHandler)
	c.On("cmd:b64", B64Handler)
	c.On("cmd:unb64", UnB64Handler)
	c.On("cmd:hex", HexHandler)
	c.On("cmd:unhex", UnHexHandler)
	c.On("cmd:url", URLHandler)
	c.On("cmd:unurl", UnURLHandler)
	c.On("cmd:jwtdec", JWTDecHandler)
}
