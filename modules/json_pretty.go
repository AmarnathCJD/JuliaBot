package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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
		if r == ' ' {
			n++
		} else if r == '\t' {
			n += 2
		} else {
			break
		}
	}
	return n
}

func yamlScalarToJSON(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "\"\""
	}
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			inner := v[1 : len(v)-1]
			b, err := json.Marshal(inner)
			if err == nil {
				return string(b)
			}
		}
	}
	lv := strings.ToLower(v)
	if lv == "true" || lv == "false" || lv == "null" || lv == "~" {
		if lv == "~" {
			return "null"
		}
		return lv
	}
	if _, err := strconv.ParseInt(v, 10, 64); err == nil {
		return v
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return v
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "\"\""
	}
	return string(b)
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

func yamlKeyToJSON(k string) string {
	v := strings.TrimSpace(k)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "\"\""
	}
	return string(b)
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
	c := Client
	c.On("cmd:jsonpretty", JsonPrettyHandler)
	c.On("cmd:jsonmin", JsonMinHandler)
	c.On("cmd:jsonvalid", JsonValidHandler)
	c.On("cmd:yamlcvt", YamlCvtHandler)
}

func init() {
	QueueHandlerRegistration(registerJsonPrettyHandlers)
}
