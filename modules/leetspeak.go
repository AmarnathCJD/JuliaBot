package modules

import (
	"html"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var leetMap = map[rune]rune{
	'a': '4', 'A': '4',
	'b': '8', 'B': '8',
	'e': '3', 'E': '3',
	'g': '6', 'G': '6',
	'i': '1', 'I': '1',
	'l': '1', 'L': '1',
	'o': '0', 'O': '0',
	's': '5', 'S': '5',
	't': '7', 'T': '7',
	'z': '2', 'Z': '2',
}

var unleetMap = map[rune]rune{
	'4': 'a',
	'8': 'b',
	'3': 'e',
	'6': 'g',
	'1': 'i',
	'0': 'o',
	'5': 's',
	'7': 't',
	'2': 'z',
}

func leetGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func leetReplyCode(m *tg.NewMessage, s string) {
	escaped := html.EscapeString(s)
	if len(escaped) > 4000 {
		escaped = escaped[:4000] + "\n... (truncated)"
	}
	m.Reply("<code>" + escaped + "</code>")
}

func LeetHandler(m *tg.NewMessage) error {
	text := leetGetInput(m)
	if text == "" {
		m.Reply("usage: /leet &lt;text&gt;")
		return nil
	}
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if v, ok := leetMap[r]; ok {
			b.WriteRune(v)
		} else {
			b.WriteRune(r)
		}
	}
	leetReplyCode(m, b.String())
	return nil
}

func UnleetHandler(m *tg.NewMessage) error {
	text := leetGetInput(m)
	if text == "" {
		m.Reply("usage: /unleet &lt;text&gt;")
		return nil
	}
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if v, ok := unleetMap[r]; ok {
			b.WriteRune(v)
		} else {
			b.WriteRune(r)
		}
	}
	leetReplyCode(m, b.String())
	return nil
}

func VaporHandler(m *tg.NewMessage) error {
	text := leetGetInput(m)
	if text == "" {
		m.Reply("usage: /vapor &lt;text&gt;")
		return nil
	}
	var b strings.Builder
	for _, r := range text {
		switch {
		case r == ' ':
			b.WriteRune('　')
		case r >= '!' && r <= '~':
			b.WriteRune(r - 0x21 + 0xFF01)
		default:
			b.WriteRune(r)
		}
	}
	leetReplyCode(m, b.String())
	return nil
}

func init() { QueueHandlerRegistration(registerLeetspeakHandlers) }
func registerLeetspeakHandlers() {
	c := Client
	c.On("cmd:leet", LeetHandler)
	c.On("cmd:unleet", UnleetHandler)
	c.On("cmd:vapor", VaporHandler)
}
