package modules

import (
	"fmt"
	"html"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func binGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func binReplyCode(m *tg.NewMessage, s string) {
	escaped := html.EscapeString(s)
	if len(escaped) > 4000 {
		escaped = escaped[:4000] + "\n... (truncated)"
	}
	m.Reply("<code>" + escaped + "</code>")
}

func BinaryHandler(m *tg.NewMessage) error {
	text := binGetInput(m)
	if text == "" {
		m.Reply("usage: /binary &lt;text&gt;")
		return nil
	}
	parts := make([]string, 0, len(text))
	for _, b := range []byte(text) {
		parts = append(parts, fmt.Sprintf("%08b", b))
	}
	binReplyCode(m, strings.Join(parts, " "))
	return nil
}

func UnbinaryHandler(m *tg.NewMessage) error {
	text := binGetInput(m)
	if text == "" {
		m.Reply("usage: /unbinary &lt;binary&gt;")
		return nil
	}
	cleaned := strings.ReplaceAll(text, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\t", "")
	if len(cleaned) == 0 || len(cleaned)%8 != 0 {
		m.Reply("decode failed: binary length must be a multiple of 8")
		return nil
	}
	out := make([]byte, 0, len(cleaned)/8)
	for i := 0; i < len(cleaned); i += 8 {
		var v byte
		for j := 0; j < 8; j++ {
			c := cleaned[i+j]
			if c != '0' && c != '1' {
				m.Reply("decode failed: invalid binary character")
				return nil
			}
			v = v<<1 | (c - '0')
		}
		out = append(out, v)
	}
	binReplyCode(m, string(out))
	return nil
}

func OctalHandler(m *tg.NewMessage) error {
	text := binGetInput(m)
	if text == "" {
		m.Reply("usage: /octal &lt;text&gt;")
		return nil
	}
	parts := make([]string, 0, len(text))
	for _, b := range []byte(text) {
		parts = append(parts, fmt.Sprintf("%o", b))
	}
	binReplyCode(m, strings.Join(parts, " "))
	return nil
}

func DecimalHandler(m *tg.NewMessage) error {
	text := binGetInput(m)
	if text == "" {
		m.Reply("usage: /decimal &lt;text&gt;")
		return nil
	}
	parts := make([]string, 0)
	for _, r := range text {
		parts = append(parts, fmt.Sprintf("%d", r))
	}
	binReplyCode(m, strings.Join(parts, " "))
	return nil
}

func init() { QueueHandlerRegistration(registerBinaryHandlers) }
func registerBinaryHandlers() {
	c := Client
	c.On("cmd:binary", BinaryHandler)
	c.On("cmd:unbinary", UnbinaryHandler)
	c.On("cmd:octal", OctalHandler)
	c.On("cmd:decimal", DecimalHandler)
}
