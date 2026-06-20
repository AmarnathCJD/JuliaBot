package modules

import (
	"fmt"
	"html"
	"strings"
	"unicode/utf8"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func echoReplyCode(m *tg.NewMessage, s string) {
	escaped := html.EscapeString(s)
	if len(escaped) > 4000 {
		escaped = escaped[:4000] + "\n... (truncated)"
	}
	m.Reply("<code>" + escaped + "</code>")
}

func EchoHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("usage: /echo &lt;text&gt;")
		return nil
	}
	m.Client.SendMessage(m.ChatID(), html.EscapeString(text))
	return nil
}

func ReverseItHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("usage: /reverseit &lt;text&gt;")
		return nil
	}
	runes := []rune(text)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	echoReplyCode(m, string(runes))
	return nil
}

func ClapHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("usage: /clap &lt;text&gt;")
		return nil
	}
	parts := strings.Fields(text)
	if len(parts) == 0 {
		m.Reply("usage: /clap &lt;text&gt;")
		return nil
	}
	echoReplyCode(m, strings.Join(parts, " 👏 "))
	return nil
}

func UpperHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("usage: /upper &lt;text&gt;")
		return nil
	}
	echoReplyCode(m, strings.ToUpper(text))
	return nil
}

func LowerHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("usage: /lower &lt;text&gt;")
		return nil
	}
	echoReplyCode(m, strings.ToLower(text))
	return nil
}

func TitleHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("usage: /title &lt;text&gt;")
		return nil
	}
	echoReplyCode(m, strings.Title(strings.ToLower(text)))
	return nil
}

func LenHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("usage: /len &lt;text&gt;")
		return nil
	}
	chars := utf8.RuneCountInString(text)
	bytes := len(text)
	words := len(strings.Fields(text))
	out := fmt.Sprintf("chars: %d\nbytes: %d\nwords: %d", chars, bytes, words)
	m.Reply("<code>" + html.EscapeString(out) + "</code>")
	return nil
}

func CountHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("reply to a message or supply text: /count &lt;text&gt;")
		return nil
	}
	chars := utf8.RuneCountInString(text)
	bytes := len(text)
	out := fmt.Sprintf("chars: %d\nbytes: %d", chars, bytes)
	m.Reply("<code>" + html.EscapeString(out) + "</code>")
	return nil
}

func init() { QueueHandlerRegistration(registerEchoHandlers) }
func registerEchoHandlers() {
	c := Client
	c.On("cmd:echo", EchoHandler, tg.CustomFilter(FilterOwner))
	c.On("cmd:reverseit", ReverseItHandler)
	c.On("cmd:clap", ClapHandler)
	c.On("cmd:upper", UpperHandler)
	c.On("cmd:lower", LowerHandler)
	c.On("cmd:title", TitleHandler)
	c.On("cmd:len", LenHandler)
	c.On("cmd:count", CountHandler)
}
