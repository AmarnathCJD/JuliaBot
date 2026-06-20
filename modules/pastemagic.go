package modules

import (
	"html"
	"regexp"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	pmCodeBlockRe = regexp.MustCompile("(?s)```([a-zA-Z0-9_+-]*)\\n?(.*?)```")
	pmInlineCode  = regexp.MustCompile("`([^`\\n]+)`")
	pmHeaderRe    = regexp.MustCompile("(?m)^(#{1,6})\\s+(.+)$")
	pmBoldStarRe  = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	pmBoldUndRe   = regexp.MustCompile(`__([^_\n]+)__`)
	pmItalicStar  = regexp.MustCompile(`\*([^*\n]+)\*`)
	pmItalicUnd   = regexp.MustCompile(`_([^_\n]+)_`)
	pmStrikeRe    = regexp.MustCompile(`~~([^~\n]+)~~`)
	pmLinkRe      = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s]+)\)`)
	pmImageRe     = regexp.MustCompile(`!\[([^\]\n]*)\]\(([^)\s]+)\)`)
	pmHrRe        = regexp.MustCompile(`(?m)^\s*(?:-{3,}|\*{3,}|_{3,})\s*$`)
	pmBlockquote  = regexp.MustCompile(`(?m)^>\s?(.*)$`)
	pmListItemRe  = regexp.MustCompile(`(?m)^\s*(?:[-*+]|\d+\.)\s+(.+)$`)
	pmHtmlTagRe   = regexp.MustCompile(`<[^>]+>`)
)

func pmGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func pmConvertMarkdown(input string) string {
	type stash struct {
		key string
		val string
	}
	var stashes []stash
	counter := 0
	stashStore := func(v string) string {
		key := "\x00PMSTASH" + string(rune('A'+counter%26)) + string(rune('a'+(counter/26)%26)) + "\x00"
		counter++
		stashes = append(stashes, stash{key: key, val: v})
		return key
	}

	work := input

	work = pmCodeBlockRe.ReplaceAllStringFunc(work, func(s string) string {
		match := pmCodeBlockRe.FindStringSubmatch(s)
		body := match[2]
		out := "<pre>" + html.EscapeString(body) + "</pre>"
		return stashStore(out)
	})

	work = pmInlineCode.ReplaceAllStringFunc(work, func(s string) string {
		match := pmInlineCode.FindStringSubmatch(s)
		body := match[1]
		out := "<code>" + html.EscapeString(body) + "</code>"
		return stashStore(out)
	})

	work = pmImageRe.ReplaceAllStringFunc(work, func(s string) string {
		match := pmImageRe.FindStringSubmatch(s)
		alt := match[1]
		url := match[2]
		if alt == "" {
			alt = "image"
		}
		out := "<a href=\"" + html.EscapeString(url) + "\">" + html.EscapeString(alt) + "</a>"
		return stashStore(out)
	})

	work = pmLinkRe.ReplaceAllStringFunc(work, func(s string) string {
		match := pmLinkRe.FindStringSubmatch(s)
		label := match[1]
		url := match[2]
		out := "<a href=\"" + html.EscapeString(url) + "\">" + html.EscapeString(label) + "</a>"
		return stashStore(out)
	})

	work = pmHtmlTagRe.ReplaceAllStringFunc(work, func(s string) string {
		return html.EscapeString(s)
	})

	work = pmHeaderRe.ReplaceAllStringFunc(work, func(s string) string {
		match := pmHeaderRe.FindStringSubmatch(s)
		return "<b>" + match[2] + "</b>"
	})

	work = pmHrRe.ReplaceAllString(work, "—————————")

	work = pmBoldStarRe.ReplaceAllString(work, "<b>$1</b>")
	work = pmBoldUndRe.ReplaceAllString(work, "<b>$1</b>")
	work = pmStrikeRe.ReplaceAllString(work, "<s>$1</s>")
	work = pmItalicStar.ReplaceAllString(work, "<i>$1</i>")
	work = pmItalicUnd.ReplaceAllString(work, "<i>$1</i>")

	work = pmBlockquote.ReplaceAllString(work, "<blockquote>$1</blockquote>")

	work = pmListItemRe.ReplaceAllStringFunc(work, func(s string) string {
		match := pmListItemRe.FindStringSubmatch(s)
		return "• " + match[1]
	})

	for _, st := range stashes {
		work = strings.Replace(work, st.key, st.val, 1)
	}

	return work
}

func MarkdownHandler(m *tg.NewMessage) error {
	text := pmGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/markdown &lt;text&gt;</code>\nor reply to a message containing markdown.")
		return nil
	}
	out := pmConvertMarkdown(text)
	if strings.TrimSpace(out) == "" {
		m.Reply("conversion produced empty output")
		return nil
	}
	if len(out) > 4000 {
		out = out[:4000] + "\n... (truncated)"
	}
	if _, err := m.Reply(out); err != nil {
		m.Reply("<b>Render failed:</b> <code>" + html.EscapeString(err.Error()) + "</code>\n\n<pre>" + html.EscapeString(out) + "</pre>")
	}
	return nil
}

func init() { QueueHandlerRegistration(registerPasteMagicHandlers) }
func registerPasteMagicHandlers() {
	c := Client
	c.On("cmd:markdown", MarkdownHandler)
}
