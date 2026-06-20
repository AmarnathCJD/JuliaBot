package modules

import (
	"html"
	"math/rand"
	"strings"
	"unicode"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var owoFaces = []string{
	"OwO", "UwU", "owo", "uwu", ">w<", "^w^", ":3", "x3", ">_<", "nya~",
}

var uwuFaces = []string{
	"UwU", "uwu", "OwO", ">w<", "^w^", "rawr x3", "nyaa~~", ":3", "x3",
	"(◕ᴗ◕✿)", "(*≧ω≦*)", "(´｡• ω •｡`)", ">.<", "ʕ•ᴥ•ʕ", "( ˘ ³˘)♥",
}

var catSuffixes = []string{
	"nya", "nyaa", "meow", "mrrp", "purr", "nya~", "meow~", "mrow",
	":3", "purrr", "nyaaa", "mreow",
}

func tsGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func tsOwofyTransform(text string, strong bool) string {
	var b strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		switch r {
		case 'r', 'l':
			b.WriteRune('w')
		case 'R', 'L':
			b.WriteRune('W')
		case 'n', 'N':
			if i+1 < len(runes) {
				next := runes[i+1]
				if next == 'a' || next == 'o' || next == 'u' || next == 'e' || next == 'i' ||
					next == 'A' || next == 'O' || next == 'U' || next == 'E' || next == 'I' {
					b.WriteRune(r)
					b.WriteRune('y')
					continue
				}
			}
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	result := b.String()
	if strong {
		result = tsStutter(result)
	}
	return result
}

func tsStutter(text string) string {
	words := strings.Fields(text)
	out := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) > 2 && unicode.IsLetter(rune(w[0])) && rand.Intn(100) < 35 {
			first := string(w[0])
			out = append(out, first+"-"+w)
		} else {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

func tsSprinkleFaces(text string, faces []string, every int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text + " " + faces[rand.Intn(len(faces))]
	}
	out := make([]string, 0, len(words)+4)
	for i, w := range words {
		out = append(out, w)
		if (i+1)%every == 0 && i != len(words)-1 {
			out = append(out, faces[rand.Intn(len(faces))])
		}
	}
	out = append(out, faces[rand.Intn(len(faces))])
	return strings.Join(out, " ")
}

func tsCatspeak(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text + " " + catSuffixes[rand.Intn(len(catSuffixes))]
	}
	out := make([]string, 0, len(words)+4)
	for i, w := range words {
		ww := w
		runes := []rune(ww)
		if len(runes) > 0 {
			last := runes[len(runes)-1]
			if last == '.' || last == '!' || last == '?' {
				core := string(runes[:len(runes)-1])
				ww = core + " " + catSuffixes[rand.Intn(len(catSuffixes))] + string(last)
			} else if rand.Intn(100) < 25 {
				ww = ww + "~"
			}
		}
		out = append(out, ww)
		if (i+1)%4 == 0 && i != len(words)-1 {
			out = append(out, catSuffixes[rand.Intn(len(catSuffixes))])
		}
	}
	out = append(out, catSuffixes[rand.Intn(len(catSuffixes))]+"!")
	return strings.Join(out, " ")
}

func OwofyHandler(m *tg.NewMessage) error {
	text := tsGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/owofy &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := tsOwofyTransform(text, false)
	out = tsSprinkleFaces(out, owoFaces, 5)
	m.Reply("<b>OwO:</b> " + html.EscapeString(out))
	return nil
}

func UwuHandler(m *tg.NewMessage) error {
	text := tsGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/uwu &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := tsOwofyTransform(text, true)
	out = tsSprinkleFaces(out, uwuFaces, 3)
	m.Reply("<b>UwU:</b> " + html.EscapeString(out))
	return nil
}

func CatspeakHandler(m *tg.NewMessage) error {
	text := tsGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/catspeak &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := tsCatspeak(text)
	m.Reply("<b>Catspeak:</b> " + html.EscapeString(out))
	return nil
}

func registerTranslateSpecialHandlers() {
	c := Client
	c.On("cmd:owofy", OwofyHandler)
	c.On("cmd:uwu", UwuHandler)
	c.On("cmd:catspeak", CatspeakHandler)
}

func init() {
	QueueHandlerRegistration(registerTranslateSpecialHandlers)
}
