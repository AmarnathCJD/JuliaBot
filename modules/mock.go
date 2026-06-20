package modules

import (
	"html"
	"math/rand"
	"strings"
	"unicode"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func mockGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func mockReplyCode(m *tg.NewMessage, s string) {
	escaped := html.EscapeString(s)
	if len(escaped) > 4000 {
		escaped = escaped[:4000] + "\n... (truncated)"
	}
	m.Reply("<code>" + escaped + "</code>")
}

func MockHandler(m *tg.NewMessage) error {
	text := mockGetInput(m)
	if text == "" {
		m.Reply("usage: /mock &lt;text&gt; or reply")
		return nil
	}
	var b strings.Builder
	b.Grow(len(text))
	upper := false
	for _, r := range text {
		if unicode.IsLetter(r) {
			if upper {
				b.WriteRune(unicode.ToUpper(r))
			} else {
				b.WriteRune(unicode.ToLower(r))
			}
			upper = !upper
		} else {
			b.WriteRune(r)
		}
	}
	mockReplyCode(m, b.String())
	return nil
}

func Clap2Handler(m *tg.NewMessage) error {
	text := mockGetInput(m)
	if text == "" {
		m.Reply("usage: /clap2 &lt;text&gt;")
		return nil
	}
	parts := strings.Fields(text)
	out := strings.Join(parts, " 👏 ")
	mockReplyCode(m, out)
	return nil
}

func scrambleWord(w string) string {
	runes := []rune(w)
	n := len(runes)
	if n <= 3 {
		return w
	}
	prefix := 0
	for prefix < n && !unicode.IsLetter(runes[prefix]) {
		prefix++
	}
	suffix := n - 1
	for suffix >= 0 && !unicode.IsLetter(runes[suffix]) {
		suffix--
	}
	if suffix-prefix < 3 {
		return w
	}
	mid := make([]rune, suffix-prefix-1)
	copy(mid, runes[prefix+1:suffix])
	if len(mid) > 1 {
		rand.Shuffle(len(mid), func(i, j int) { mid[i], mid[j] = mid[j], mid[i] })
	}
	var b strings.Builder
	b.Grow(n)
	for i := 0; i <= prefix; i++ {
		b.WriteRune(runes[i])
	}
	for _, r := range mid {
		b.WriteRune(r)
	}
	for i := suffix; i < n; i++ {
		b.WriteRune(runes[i])
	}
	return b.String()
}

func ScrambleHandler(m *tg.NewMessage) error {
	text := mockGetInput(m)
	if text == "" {
		m.Reply("usage: /scramble &lt;text&gt;")
		return nil
	}
	parts := strings.Fields(text)
	for i, p := range parts {
		parts[i] = scrambleWord(p)
	}
	mockReplyCode(m, strings.Join(parts, " "))
	return nil
}

func init() { QueueHandlerRegistration(registerMockHandlers) }
func registerMockHandlers() {
	c := Client
	c.On("cmd:mock", MockHandler)
	c.On("cmd:clap2", Clap2Handler)
	c.On("cmd:scramble", ScrambleHandler)
}
