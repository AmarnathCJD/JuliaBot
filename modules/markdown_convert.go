package modules

import (
	"html"
	"regexp"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	mdCodeBlockRe = regexp.MustCompile("(?s)```([a-zA-Z0-9_+\\-]*)\\n?(.*?)```")
	mdInlineRe    = regexp.MustCompile("`([^`\\n]+)`")
	mdImageRe     = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	mdLinkRe      = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	mdAutolinkRe  = regexp.MustCompile(`<((?:https?|ftp|mailto):[^>\s]+)>`)
	mdBoldStarRe  = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	mdBoldUndRe   = regexp.MustCompile(`__([^_\n]+)__`)
	mdItalicStRe  = regexp.MustCompile(`(^|[^*])\*([^*\n]+)\*([^*]|$)`)
	mdItalicUnRe  = regexp.MustCompile(`(^|[^_\w])_([^_\n]+)_([^_\w]|$)`)
	mdStrikeRe    = regexp.MustCompile(`~~([^~\n]+)~~`)
	mdSpoilerRe   = regexp.MustCompile(`\|\|([^|\n]+)\|\|`)
	mdHrRe        = regexp.MustCompile(`^\s*(?:-{3,}|\*{3,}|_{3,})\s*$`)
	mdBlockquote  = regexp.MustCompile(`^\s*>\s?(.*)$`)
	mdOrderedRe   = regexp.MustCompile(`^(\s*)(\d+)\.\s+(.*)$`)
	mdUnorderedRe = regexp.MustCompile(`^(\s*)[-*+]\s+(.*)$`)
	mdHeadingRe   = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
)

func markdownGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

type mdPlaceholder struct {
	token   string
	content string
}

func markdownConvert(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")

	var placeholders []mdPlaceholder
	addPlaceholder := func(html string) string {
		tok := "\x00MDPH" + intToToken(len(placeholders)) + "\x00"
		placeholders = append(placeholders, mdPlaceholder{token: tok, content: html})
		return tok
	}

	src = mdCodeBlockRe.ReplaceAllStringFunc(src, func(match string) string {
		sub := mdCodeBlockRe.FindStringSubmatch(match)
		lang := strings.TrimSpace(sub[1])
		body := sub[2]
		body = strings.TrimRight(body, "\n")
		escaped := html.EscapeString(body)
		var out string
		if lang != "" {
			out = "<pre><code class=\"language-" + html.EscapeString(lang) + "\">" + escaped + "</code></pre>"
		} else {
			out = "<pre>" + escaped + "</pre>"
		}
		return addPlaceholder(out)
	})

	src = mdInlineRe.ReplaceAllStringFunc(src, func(match string) string {
		sub := mdInlineRe.FindStringSubmatch(match)
		return addPlaceholder("<code>" + html.EscapeString(sub[1]) + "</code>")
	})

	src = mdImageRe.ReplaceAllStringFunc(src, func(match string) string {
		sub := mdImageRe.FindStringSubmatch(match)
		alt := sub[1]
		if alt == "" {
			alt = "image"
		}
		return addPlaceholder("<a href=\"" + html.EscapeString(sub[2]) + "\">" + html.EscapeString(alt) + "</a>")
	})

	src = mdLinkRe.ReplaceAllStringFunc(src, func(match string) string {
		sub := mdLinkRe.FindStringSubmatch(match)
		return addPlaceholder("<a href=\"" + html.EscapeString(sub[2]) + "\">" + html.EscapeString(sub[1]) + "</a>")
	})

	src = mdAutolinkRe.ReplaceAllStringFunc(src, func(match string) string {
		sub := mdAutolinkRe.FindStringSubmatch(match)
		return addPlaceholder("<a href=\"" + html.EscapeString(sub[1]) + "\">" + html.EscapeString(sub[1]) + "</a>")
	})

	lines := strings.Split(src, "\n")
	var out []string
	inQuote := false
	var quoteBuf []string
	flushQuote := func() {
		if len(quoteBuf) > 0 {
			out = append(out, "<blockquote>"+strings.Join(quoteBuf, "\n")+"</blockquote>")
			quoteBuf = nil
		}
		inQuote = false
	}

	for _, ln := range lines {
		if strings.Contains(ln, "\x00MDPH") && strings.TrimSpace(ln) == strings.TrimSpace(ln) {
			trim := strings.TrimSpace(ln)
			if strings.HasPrefix(trim, "\x00MDPH") && strings.HasSuffix(trim, "\x00") {
				if isPureBlockPlaceholder(trim, placeholders) {
					flushQuote()
					out = append(out, ln)
					continue
				}
			}
		}

		if mdHrRe.MatchString(ln) {
			flushQuote()
			out = append(out, "—")
			continue
		}

		if hm := mdHeadingRe.FindStringSubmatch(ln); hm != nil {
			flushQuote()
			level := len(hm[1])
			content := inlineMarkdown(hm[2])
			tag := "b"
			if level >= 3 {
				tag = "i"
			}
			out = append(out, "<"+tag+"><u>"+content+"</u></"+tag+">")
			continue
		}

		if qm := mdBlockquote.FindStringSubmatch(ln); qm != nil {
			inQuote = true
			quoteBuf = append(quoteBuf, inlineMarkdown(qm[1]))
			continue
		}

		if om := mdOrderedRe.FindStringSubmatch(ln); om != nil {
			flushQuote()
			out = append(out, om[1]+om[2]+". "+inlineMarkdown(om[3]))
			continue
		}

		if um := mdUnorderedRe.FindStringSubmatch(ln); um != nil {
			flushQuote()
			out = append(out, um[1]+"• "+inlineMarkdown(um[2]))
			continue
		}

		flushQuote()
		out = append(out, inlineMarkdown(ln))
	}
	if inQuote {
		flushQuote()
	}

	result := strings.Join(out, "\n")

	for i := len(placeholders) - 1; i >= 0; i-- {
		result = strings.ReplaceAll(result, placeholders[i].token, placeholders[i].content)
	}

	return result
}

func isPureBlockPlaceholder(trim string, placeholders []mdPlaceholder) bool {
	for _, p := range placeholders {
		if p.token == trim && strings.HasPrefix(p.content, "<pre") {
			return true
		}
	}
	return false
}

func inlineMarkdown(s string) string {
	if strings.Contains(s, "\x00MDPH") {
		return processInlineWithPlaceholders(s)
	}
	return applyInlineFormatting(html.EscapeString(s))
}

func processInlineWithPlaceholders(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		idx := strings.Index(s[i:], "\x00MDPH")
		if idx < 0 {
			b.WriteString(applyInlineFormatting(html.EscapeString(s[i:])))
			break
		}
		if idx > 0 {
			b.WriteString(applyInlineFormatting(html.EscapeString(s[i : i+idx])))
		}
		start := i + idx
		end := strings.Index(s[start+1:], "\x00")
		if end < 0 {
			b.WriteString(s[start:])
			break
		}
		end = start + 1 + end + 1
		b.WriteString(s[start:end])
		i = end
	}
	return b.String()
}

func applyInlineFormatting(s string) string {
	s = mdBoldStarRe.ReplaceAllString(s, "<b>$1</b>")
	s = mdBoldUndRe.ReplaceAllString(s, "<b>$1</b>")
	s = mdStrikeRe.ReplaceAllString(s, "<s>$1</s>")
	s = mdSpoilerRe.ReplaceAllString(s, "<tg-spoiler>$1</tg-spoiler>")
	s = mdItalicStRe.ReplaceAllString(s, "$1<i>$2</i>$3")
	s = mdItalicUnRe.ReplaceAllString(s, "$1<i>$2</i>$3")
	return s
}

func intToToken(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func MarkdownConvertHandler(m *tg.NewMessage) error {
	text := markdownGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/markdown &lt;text&gt;</code>\nor reply to a message.\n\nConverts <b>**bold**</b> <i>_italic_</i> <code>`code`</code> [link](url) and # headers to Telegram HTML.")
		return nil
	}
	converted := markdownConvert(text)
	if len(converted) > 4000 {
		converted = converted[:4000] + "\n... (truncated)"
	}
	if strings.TrimSpace(converted) == "" {
		m.Reply("<i>(empty after conversion)</i>")
		return nil
	}
	m.Reply(converted)
	return nil
}

func init() { QueueHandlerRegistration(registerMarkdownConvertHandlers) }
func registerMarkdownConvertHandlers() {
	c := Client
	c.On("cmd:markdown", MarkdownConvertHandler)
}
