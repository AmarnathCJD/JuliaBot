package modules

import (
	"fmt"
	"html"
	"strings"
	"unicode"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func classifyToken(s string) string {
	if s == "" {
		return "empty"
	}
	allDigit := true
	allSpace := true
	allLetter := true
	allPunct := true
	hasDigit := false
	hasLetter := false
	for _, r := range s {
		if !unicode.IsDigit(r) {
			allDigit = false
		} else {
			hasDigit = true
		}
		if !unicode.IsSpace(r) {
			allSpace = false
		}
		if !unicode.IsLetter(r) {
			allLetter = false
		} else {
			hasLetter = true
		}
		if !(unicode.IsPunct(r) || unicode.IsSymbol(r)) {
			allPunct = false
		}
	}
	switch {
	case allSpace:
		return "space"
	case allDigit:
		return "number"
	case allLetter:
		return "word"
	case allPunct:
		return "punctuation"
	case hasLetter && hasDigit:
		return "alphanumeric"
	default:
		return "mixed"
	}
}

func splitIntoTokens(text string) []string {
	var tokens []string
	var current strings.Builder
	currentKind := -1
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	kindOf := func(r rune) int {
		switch {
		case unicode.IsSpace(r):
			return 0
		case unicode.IsDigit(r):
			return 1
		case unicode.IsLetter(r):
			return 2
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			return 3
		default:
			return 4
		}
	}
	for _, r := range text {
		k := kindOf(r)
		if k == 3 {
			flush()
			tokens = append(tokens, string(r))
			currentKind = -1
			continue
		}
		if currentKind == -1 || k == currentKind {
			current.WriteRune(r)
			currentKind = k
		} else {
			flush()
			current.WriteRune(r)
			currentKind = k
		}
	}
	flush()
	return tokens
}

func displayToken(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case ' ':
			sb.WriteRune('·')
		case '\t':
			sb.WriteString("\\t")
		case '\n':
			sb.WriteString("\\n")
		case '\r':
			sb.WriteString("\\r")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func ParseTokensHandler(m *tg.NewMessage) error {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = r.Text()
		}
	}
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/parsetokens &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	tokens := splitIntoTokens(text)
	if len(tokens) == 0 {
		m.Reply("<b>Error:</b> no tokens found.")
		return nil
	}
	maxShown := 60
	truncated := false
	if len(tokens) > maxShown {
		tokens = tokens[:maxShown]
		truncated = true
	}
	maxTokLen := 5
	for _, t := range tokens {
		d := displayToken(t)
		if len([]rune(d)) > maxTokLen {
			maxTokLen = len([]rune(d))
		}
	}
	if maxTokLen > 18 {
		maxTokLen = 18
	}
	counts := map[string]int{}
	var sb strings.Builder
	sb.WriteString("<b>Token Parser</b>\n")
	sb.WriteString(fmt.Sprintf("<i>Tokens shown:</i> <code>%d</code>\n\n", len(tokens)))
	sb.WriteString("<pre>")
	sb.WriteString(fmt.Sprintf("%-3s %-*s %-12s %4s\n", "#", maxTokLen, "Token", "Type", "Len"))
	sb.WriteString(strings.Repeat("-", 3+1+maxTokLen+1+12+1+4))
	sb.WriteString("\n")
	for i, t := range tokens {
		kind := classifyToken(t)
		counts[kind]++
		disp := displayToken(t)
		runes := []rune(disp)
		if len(runes) > maxTokLen {
			disp = string(runes[:maxTokLen-1]) + "…"
		}
		sb.WriteString(fmt.Sprintf("%-3d %-*s %-12s %4d\n", i+1, maxTokLen, html.EscapeString(disp), kind, len([]rune(t))))
	}
	sb.WriteString("</pre>\n")
	if truncated {
		sb.WriteString("<i>Output truncated.</i>\n")
	}
	sb.WriteString("<b>Summary:</b> ")
	first := true
	for _, k := range []string{"word", "number", "punctuation", "space", "alphanumeric", "mixed"} {
		if v, ok := counts[k]; ok {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s=<code>%d</code>", k, v))
			first = false
		}
	}
	m.Reply(sb.String())
	return nil
}

func registerSentenceParserHandlers() {
	c := Client
	c.On("cmd:parsetokens", ParseTokensHandler)
}

func init() { QueueHandlerRegistration(registerSentenceParserHandlers) }
