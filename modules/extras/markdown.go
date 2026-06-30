package extras

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	modules "main/modules"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// === from markdown_convert.go ===
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
		if strings.Contains(ln, "\x00MDPH") && ln == strings.TrimSpace(ln) {
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

func initFromSrc_markdown_convert_0_1() {
	modules.QueueHandlerRegistration(registerMarkdownConvertHandlers)
}
func registerMarkdownConvertHandlers() {
	c := modules.Client
	c.On("cmd:markdown", MarkdownConvertHandler)
}

// === from json_pretty.go ===
func extractJSONPayload(m *tg.NewMessage) string {
	args := strings.TrimSpace(m.Args())
	if args == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			args = strings.TrimSpace(r.Text())
		}
	}
	return args
}

func formatJSONErr(err error, data []byte) string {
	if se, ok := err.(*json.SyntaxError); ok {
		offset := int(se.Offset)
		line := 1
		col := 1
		if offset > len(data) {
			offset = len(data)
		}
		for i := 0; i < offset; i++ {
			if data[i] == '\n' {
				line++
				col = 1
			} else {
				col++
			}
		}
		return fmt.Sprintf("%s (line %d, col %d, offset %d)", se.Error(), line, col, offset)
	}
	return err.Error()
}

func JsonPrettyHandler(m *tg.NewMessage) error {
	payload := extractJSONPayload(m)
	if payload == "" {
		m.Reply("<b>Usage:</b> <code>/jsonpretty &lt;json&gt;</code>\nor reply to a message containing JSON.")
		return nil
	}
	var v any
	data := []byte(payload)
	if err := json.Unmarshal(data, &v); err != nil {
		m.Reply("<b>Invalid JSON:</b> <code>" + html.EscapeString(formatJSONErr(err, data)) + "</code>")
		return nil
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		m.Reply("<b>Error:</b> <code>" + html.EscapeString(err.Error()) + "</code>")
		return nil
	}
	body := string(out)
	if len(body) > 3500 {
		body = body[:3500] + "\n... (truncated)"
	}
	m.Reply("<pre>" + html.EscapeString(body) + "</pre>")
	return nil
}

func JsonMinHandler(m *tg.NewMessage) error {
	payload := extractJSONPayload(m)
	if payload == "" {
		m.Reply("<b>Usage:</b> <code>/jsonmin &lt;json&gt;</code>\nor reply to a message containing JSON.")
		return nil
	}
	data := []byte(payload)
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		m.Reply("<b>Invalid JSON:</b> <code>" + html.EscapeString(formatJSONErr(err, data)) + "</code>")
		return nil
	}
	body := buf.String()
	if len(body) > 3500 {
		body = body[:3500] + "... (truncated)"
	}
	m.Reply("<pre>" + html.EscapeString(body) + "</pre>")
	return nil
}

func JsonValidHandler(m *tg.NewMessage) error {
	payload := extractJSONPayload(m)
	if payload == "" {
		m.Reply("<b>Usage:</b> <code>/jsonvalid &lt;json&gt;</code>\nor reply to a message containing JSON.")
		return nil
	}
	data := []byte(payload)
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		m.Reply("<b>Invalid JSON</b>\n<code>" + html.EscapeString(formatJSONErr(err, data)) + "</code>")
		return nil
	}
	kind := "unknown"
	switch v.(type) {
	case map[string]any:
		kind = "object"
	case []any:
		kind = "array"
	case string:
		kind = "string"
	case float64:
		kind = "number"
	case bool:
		kind = "boolean"
	case nil:
		kind = "null"
	}
	m.Reply(fmt.Sprintf("<b>Valid JSON</b>\n<b>Top-level type:</b> <code>%s</code>\n<b>Bytes:</b> <code>%d</code>", kind, len(data)))
	return nil
}

func countLeadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 2
		default:
			return n
		}
	}
	return n
}

type yamlLine struct {
	indent  int
	content string
	isList  bool
}

func parseYAMLLines(input string) []yamlLine {
	var out []yamlLine
	for _, raw := range strings.Split(input, "\n") {
		stripped := raw
		if i := strings.Index(stripped, "#"); i >= 0 {
			inQuote := false
			var qc byte
			cut := -1
			for j := 0; j < len(stripped); j++ {
				c := stripped[j]
				if inQuote {
					if c == qc {
						inQuote = false
					}
					continue
				}
				if c == '"' || c == '\'' {
					inQuote = true
					qc = c
					continue
				}
				if c == '#' {
					cut = j
					break
				}
			}
			if cut >= 0 {
				stripped = stripped[:cut]
			}
		}
		if strings.TrimSpace(stripped) == "" {
			continue
		}
		indent := countLeadingSpaces(stripped)
		content := strings.TrimLeft(stripped, " \t")
		isList := false
		if strings.HasPrefix(content, "- ") {
			isList = true
			content = strings.TrimSpace(content[2:])
		} else if content == "-" {
			isList = true
			content = ""
		}
		out = append(out, yamlLine{indent: indent, content: content, isList: isList})
	}
	return out
}

func splitYAMLKV(s string) (string, string, bool) {
	inQuote := false
	var qc byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == qc {
				inQuote = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			inQuote = true
			qc = c
			continue
		}
		if c == ':' {
			key := strings.TrimSpace(s[:i])
			val := ""
			if i+1 < len(s) {
				val = strings.TrimSpace(s[i+1:])
			}
			return key, val, true
		}
	}
	return "", "", false
}

func convertYAMLBlock(lines []yamlLine, start, baseIndent int) (any, int, error) {
	if start >= len(lines) {
		return nil, start, nil
	}
	first := lines[start]
	if first.isList {
		var arr []any
		i := start
		for i < len(lines) {
			ln := lines[i]
			if ln.indent < baseIndent || !ln.isList {
				if ln.indent < baseIndent {
					break
				}
				if ln.indent == baseIndent && !ln.isList {
					break
				}
			}
			if ln.indent != baseIndent {
				return nil, i, fmt.Errorf("inconsistent list indent at line %d", i+1)
			}
			if ln.content == "" {
				if i+1 < len(lines) && lines[i+1].indent > baseIndent {
					childIndent := lines[i+1].indent
					val, next, err := convertYAMLBlock(lines, i+1, childIndent)
					if err != nil {
						return nil, i, err
					}
					arr = append(arr, val)
					i = next
					continue
				}
				arr = append(arr, nil)
				i++
				continue
			}
			if k, v, ok := splitYAMLKV(ln.content); ok {
				obj := map[string]any{}
				if v != "" {
					obj[unquoteYAMLKey(k)] = rawYAMLValue(v)
					i++
				} else {
					i++
					if i < len(lines) && lines[i].indent > baseIndent {
						childIndent := lines[i].indent
						val, next, err := convertYAMLBlock(lines, i, childIndent)
						if err != nil {
							return nil, i, err
						}
						obj[unquoteYAMLKey(k)] = val
						i = next
					} else {
						obj[unquoteYAMLKey(k)] = nil
					}
				}
				for i < len(lines) {
					sub := lines[i]
					if sub.indent != baseIndent+2 || sub.isList {
						if sub.indent < baseIndent+2 {
							break
						}
						if sub.indent == baseIndent+2 && sub.isList {
							break
						}
					}
					if sub.indent <= baseIndent {
						break
					}
					if k2, v2, ok2 := splitYAMLKV(sub.content); ok2 && !sub.isList && sub.indent == baseIndent+2 {
						if v2 != "" {
							obj[unquoteYAMLKey(k2)] = rawYAMLValue(v2)
							i++
						} else {
							i++
							if i < len(lines) && lines[i].indent > baseIndent+2 {
								childIndent := lines[i].indent
								val, next, err := convertYAMLBlock(lines, i, childIndent)
								if err != nil {
									return nil, i, err
								}
								obj[unquoteYAMLKey(k2)] = val
								i = next
							} else {
								obj[unquoteYAMLKey(k2)] = nil
							}
						}
					} else {
						break
					}
				}
				arr = append(arr, obj)
				continue
			}
			arr = append(arr, rawYAMLValue(ln.content))
			i++
		}
		return arr, i, nil
	}

	obj := map[string]any{}
	i := start
	for i < len(lines) {
		ln := lines[i]
		if ln.indent < baseIndent {
			break
		}
		if ln.indent > baseIndent {
			return nil, i, fmt.Errorf("unexpected indent at line %d", i+1)
		}
		if ln.isList {
			break
		}
		k, v, ok := splitYAMLKV(ln.content)
		if !ok {
			return nil, i, fmt.Errorf("expected 'key: value' at line %d", i+1)
		}
		key := unquoteYAMLKey(k)
		if v != "" {
			obj[key] = rawYAMLValue(v)
			i++
			continue
		}
		i++
		if i < len(lines) && lines[i].indent > baseIndent {
			childIndent := lines[i].indent
			val, next, err := convertYAMLBlock(lines, i, childIndent)
			if err != nil {
				return nil, i, err
			}
			obj[key] = val
			i = next
		} else {
			obj[key] = nil
		}
	}
	return obj, i, nil
}

func unquoteYAMLKey(k string) string {
	v := strings.TrimSpace(k)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

func rawYAMLValue(s string) any {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	lv := strings.ToLower(v)
	if lv == "true" {
		return true
	}
	if lv == "false" {
		return false
	}
	if lv == "null" || lv == "~" {
		return nil
	}
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return v
}

func YamlCvtHandler(m *tg.NewMessage) error {
	payload := extractJSONPayload(m)
	if payload == "" {
		m.Reply("<b>Usage:</b> <code>/yamlcvt &lt;yaml&gt;</code>\nor reply to a message containing YAML.\n<i>Note:</i> only flat documents with simple scalars, lists, and nested maps are supported.")
		return nil
	}
	if strings.Contains(payload, "&") || strings.Contains(payload, "*") || strings.Contains(payload, "<<:") || strings.Contains(payload, "!!") {
		m.Reply("<b>Refused:</b> document uses YAML features (anchors, aliases, tags, merge keys) that this naive converter does not support.")
		return nil
	}
	if strings.Contains(payload, "|") || strings.Contains(payload, ">\n") {
		m.Reply("<b>Refused:</b> block scalars (<code>|</code> / <code>&gt;</code>) are not supported.")
		return nil
	}
	lines := parseYAMLLines(payload)
	if len(lines) == 0 {
		m.Reply("<b>Error:</b> no non-empty YAML content found.")
		return nil
	}
	baseIndent := lines[0].indent
	val, consumed, err := convertYAMLBlock(lines, 0, baseIndent)
	if err != nil {
		m.Reply("<b>YAML parse error:</b> <code>" + html.EscapeString(err.Error()) + "</code>")
		return nil
	}
	if consumed != len(lines) {
		m.Reply("<b>YAML parse error:</b> <code>document too complex for naive parser (stopped at line " + strconv.Itoa(consumed+1) + ")</code>")
		return nil
	}
	out, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		m.Reply("<b>Error:</b> <code>" + html.EscapeString(err.Error()) + "</code>")
		return nil
	}
	body := string(out)
	if len(body) > 3500 {
		body = body[:3500] + "\n... (truncated)"
	}
	m.Reply("<pre>" + html.EscapeString(body) + "</pre>")
	return nil
}

func registerJsonPrettyHandlers() {
	c := modules.Client
	c.On("cmd:jsonpretty", JsonPrettyHandler)
	c.On("cmd:jsonmin", JsonMinHandler)
	c.On("cmd:jsonvalid", JsonValidHandler)
	c.On("cmd:yamlcvt", YamlCvtHandler)
}

func initFromSrc_json_pretty_2_1() {
	modules.QueueHandlerRegistration(registerJsonPrettyHandlers)
}

func init() {
	initFromSrc_markdown_convert_0_1()
	initFromSrc_json_pretty_2_1()
}
