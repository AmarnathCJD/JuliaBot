package modules

import (
	"fmt"
	"html"
	"strings"
	"unicode"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var morseTable = map[rune]string{
	'A': ".-", 'B': "-...", 'C': "-.-.", 'D': "-..", 'E': ".",
	'F': "..-.", 'G': "--.", 'H': "....", 'I': "..", 'J': ".---",
	'K': "-.-", 'L': ".-..", 'M': "--", 'N': "-.", 'O': "---",
	'P': ".--.", 'Q': "--.-", 'R': ".-.", 'S': "...", 'T': "-",
	'U': "..-", 'V': "...-", 'W': ".--", 'X': "-..-", 'Y': "-.--",
	'Z': "--..",
	'0': "-----", '1': ".----", '2': "..---", '3': "...--", '4': "....-",
	'5': ".....", '6': "-....", '7': "--...", '8': "---..", '9': "----.",
}

var reverseMorseTable = func() map[string]rune {
	m := make(map[string]rune, len(morseTable))
	for k, v := range morseTable {
		m[v] = k
	}
	return m
}()

func MorseEncodeHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/morse &lt;text&gt;</code>")
		return nil
	}
	upper := strings.ToUpper(args)
	words := strings.Fields(upper)
	encodedWords := make([]string, 0, len(words))
	skipped := 0
	for _, w := range words {
		letters := make([]string, 0, len(w))
		for _, r := range w {
			if code, ok := morseTable[r]; ok {
				letters = append(letters, code)
			} else if unicode.IsSpace(r) {
				continue
			} else {
				skipped++
			}
		}
		if len(letters) > 0 {
			encodedWords = append(encodedWords, strings.Join(letters, " "))
		}
	}
	if len(encodedWords) == 0 {
		m.Reply("<b>Error:</b> nothing convertible (A-Z, 0-9 only).")
		return nil
	}
	encoded := strings.Join(encodedWords, " / ")
	note := ""
	if skipped > 0 {
		note = fmt.Sprintf("\n\n<i>Skipped %d unsupported character(s).</i>", skipped)
	}
	out := fmt.Sprintf("<b>Morse Code</b>\n\n<b>Input:</b> <code>%s</code>\n<b>Output:</b>\n<pre>%s</pre>%s",
		html.EscapeString(args),
		html.EscapeString(encoded),
		note,
	)
	m.Reply(out)
	return nil
}

func MorseDecodeHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/unmorse &lt;morse&gt;</code>\nUse <code>.</code>/<code>-</code> or <code>·</code>/<code>−</code>, space between letters, <code>/</code> between words.")
		return nil
	}
	normalized := strings.ReplaceAll(args, "·", ".")
	normalized = strings.ReplaceAll(normalized, "•", ".")
	normalized = strings.ReplaceAll(normalized, "−", "-")
	normalized = strings.ReplaceAll(normalized, "–", "-")
	normalized = strings.ReplaceAll(normalized, "—", "-")
	normalized = strings.ReplaceAll(normalized, "|", "/")
	wordParts := strings.Split(normalized, "/")
	decodedWords := make([]string, 0, len(wordParts))
	unknown := 0
	for _, wp := range wordParts {
		wp = strings.TrimSpace(wp)
		if wp == "" {
			continue
		}
		letters := strings.Fields(wp)
		var b strings.Builder
		for _, l := range letters {
			if r, ok := reverseMorseTable[l]; ok {
				b.WriteRune(r)
			} else {
				b.WriteRune('?')
				unknown++
			}
		}
		if b.Len() > 0 {
			decodedWords = append(decodedWords, b.String())
		}
	}
	if len(decodedWords) == 0 {
		m.Reply("<b>Error:</b> could not decode any letters. Separate letters with spaces and words with <code>/</code>.")
		return nil
	}
	decoded := strings.Join(decodedWords, " ")
	note := ""
	if unknown > 0 {
		note = fmt.Sprintf("\n\n<i>Replaced %d unknown sequence(s) with '?'.</i>", unknown)
	}
	out := fmt.Sprintf("<b>Morse Decode</b>\n\n<b>Input:</b> <code>%s</code>\n<b>Output:</b>\n<pre>%s</pre>%s",
		html.EscapeString(args),
		html.EscapeString(decoded),
		note,
	)
	m.Reply(out)
	return nil
}

func registerMorseHandlers() {
	c := Client
	c.On("cmd:morse", MorseEncodeHandler)
	c.On("cmd:unmorse", MorseDecodeHandler)
}

func init() {
	QueueHandlerRegistration(registerMorseHandlers)
}
