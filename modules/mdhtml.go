package modules

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

func mdToTelegramHTML(md string) string {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	md = strings.ReplaceAll(md, "\r", "\n")
	lines := strings.Split(md, "\n")

	var out []string
	i := 0
	n := len(lines)

	for i < n {
		line := lines[i]

		if lang, ok := fenceOpen(line); ok {
			fenceCh, fenceLen := fenceInfo(line)
			var buf []string
			i++
			for i < n {
				if fenceCloseMatch(lines[i], fenceCh, fenceLen) {
					i++
					break
				}
				buf = append(buf, lines[i])
				i++
			}
			out = append(out, renderFence(lang, buf))
			continue
		}

		if isTableStart(lines, i) {
			var trows []string
			for i < n && isTableRow(lines[i]) {
				trows = append(trows, lines[i])
				i++
			}
			out = append(out, renderTable(trows))
			continue
		}

		if isHR(line) {
			out = append(out, "——————————")
			i++
			continue
		}

		if _, txt, ok := headingParse(line); ok {
			out = append(out, "<b>"+inlineSpans(txt)+"</b>")
			i++
			continue
		}

		if isBlockquote(line) {
			var qbuf []string
			for i < n && isBlockquote(lines[i]) {
				qbuf = append(qbuf, inlineSpans(stripBlockquote(lines[i])))
				i++
			}
			out = append(out, "<blockquote>"+strings.Join(qbuf, "\n")+"</blockquote>")
			continue
		}

		if indent, num, content, isOrd, ok := listItemParse(line); ok {
			pad := strings.Repeat(" ", indent)
			if isOrd {
				out = append(out, pad+num+". "+inlineSpans(content))
			} else {
				out = append(out, pad+"• "+inlineSpans(content))
			}
			i++
			continue
		}

		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			i++
			continue
		}

		out = append(out, inlineSpans(line))
		i++
	}

	res := strings.Join(out, "\n")
	res = collapseBlankRuns(res)
	return strings.TrimRight(strings.TrimLeft(res, "\n"), "\n")
}

func collapseBlankRuns(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}

var fenceOpenRe = regexp.MustCompile("^\\s{0,3}(`{3,}|~{3,})[ \\t]*([^`]*?)[ \\t]*$")

func fenceInfo(line string) (byte, int) {
	t := strings.TrimLeft(line, " \t")
	if len(t) == 0 {
		return 0, 0
	}
	ch := t[0]
	if ch != '`' && ch != '~' {
		return 0, 0
	}
	c := 0
	for c < len(t) && t[c] == ch {
		c++
	}
	return ch, c
}

func fenceOpen(line string) (string, bool) {
	m := fenceOpenRe.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	if strings.HasPrefix(m[1], "~") && strings.Contains(m[2], "~") {
		return "", false
	}
	return strings.TrimSpace(m[2]), true
}

func fenceCloseMatch(line string, ch byte, minLen int) bool {
	t := strings.TrimLeft(line, " \t")
	t = strings.TrimRight(t, " \t")
	if len(t) < minLen {
		return false
	}
	for k := 0; k < len(t); k++ {
		if t[k] != ch {
			return false
		}
	}
	return true
}

func renderFence(lang string, body []string) string {
	content := strings.Join(body, "\n")
	escaped := escapeText(content)
	lang = sanitizeLang(lang)
	if lang != "" {
		return "<pre language=\"" + lang + "\">" + escaped + "</pre>"
	}
	return "<pre>" + escaped + "</pre>"
}

func sanitizeLang(s string) string {
	s = firstWord(s)
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '+' || r == '-' || r == '_' || r == '#' || r == '.' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func firstWord(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

var headingRe = regexp.MustCompile(`^\s{0,3}(#{1,6})\s+(.*?)\s*#*\s*$`)

func headingParse(line string) (int, string, bool) {
	m := headingRe.FindStringSubmatch(line)
	if m == nil {
		return 0, "", false
	}
	return len(m[1]), m[2], true
}

func isHR(line string) bool {
	t := strings.TrimSpace(line)
	if len(t) < 3 {
		return false
	}
	var marker byte
	count := 0
	for i := 0; i < len(t); i++ {
		c := t[i]
		if c == ' ' || c == '\t' {
			continue
		}
		if c != '-' && c != '*' && c != '_' {
			return false
		}
		if marker == 0 {
			marker = c
		} else if c != marker {
			return false
		}
		count++
	}
	return count >= 3
}

func isBlockquote(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " "), ">")
}

func stripBlockquote(line string) string {
	t := strings.TrimLeft(line, " ")
	t = strings.TrimPrefix(t, ">")
	if strings.HasPrefix(t, " ") {
		t = t[1:]
	}
	return t
}

var ulRe = regexp.MustCompile(`^(\s*)[-*+]\s+(.*)$`)
var olRe = regexp.MustCompile(`^(\s*)(\d{1,9})[.)]\s+(.*)$`)

func listItemParse(line string) (int, string, string, bool, bool) {
	if m := olRe.FindStringSubmatch(line); m != nil {
		return len(m[1]), m[2], m[3], true, true
	}
	if m := ulRe.FindStringSubmatch(line); m != nil {
		return len(m[1]), "", m[2], false, true
	}
	return 0, "", "", false, false
}

func isTableRow(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return false
	}
	return countUnescapedPipes(t) >= 1
}

func countUnescapedPipes(s string) int {
	count := 0
	inCode := false
	var tick int
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if c == '`' {
			rl := 0
			for i+rl < len(runes) && runes[i+rl] == '`' {
				rl++
			}
			if !inCode {
				inCode = true
				tick = rl
			} else if rl == tick {
				inCode = false
				tick = 0
			}
			i += rl - 1
			continue
		}
		if c == '|' && !inCode {
			if i > 0 && runes[i-1] == '\\' {
				continue
			}
			count++
		}
	}
	return count
}

var tableDelimRe = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)

func isTableDelim(line string) bool {
	t := strings.TrimSpace(line)
	if !strings.Contains(t, "-") {
		return false
	}
	return tableDelimRe.MatchString(t)
}

func isTableStart(lines []string, i int) bool {
	if i+1 >= len(lines) {
		return false
	}
	if !isTableRow(lines[i]) {
		return false
	}
	if !isTableDelim(lines[i+1]) {
		return false
	}
	return len(splitTableRow(lines[i])) >= 1 && len(splitTableRow(lines[i+1])) >= 1
}

func splitTableRow(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	var cells []string
	var cur strings.Builder
	inCode := false
	var tick int
	runes := []rune(t)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if c == '`' {
			rl := 0
			for i+rl < len(runes) && runes[i+rl] == '`' {
				rl++
			}
			for k := 0; k < rl; k++ {
				cur.WriteByte('`')
			}
			if !inCode {
				inCode = true
				tick = rl
			} else if rl == tick {
				inCode = false
				tick = 0
			}
			i += rl - 1
			continue
		}
		if c == '\\' && i+1 < len(runes) && runes[i+1] == '|' {
			cur.WriteByte('|')
			i++
			continue
		}
		if c == '|' && !inCode {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteRune(c)
	}
	cells = append(cells, strings.TrimSpace(cur.String()))
	return cells
}

type tableAlign int

const (
	alignLeft tableAlign = iota
	alignRight
	alignCenter
)

func parseAligns(delim string) []tableAlign {
	cells := splitTableRow(delim)
	aligns := make([]tableAlign, len(cells))
	for i, c := range cells {
		c = strings.TrimSpace(c)
		left := strings.HasPrefix(c, ":")
		right := strings.HasSuffix(c, ":")
		switch {
		case left && right:
			aligns[i] = alignCenter
		case right:
			aligns[i] = alignRight
		default:
			aligns[i] = alignLeft
		}
	}
	return aligns
}

func renderTable(rows []string) string {
	if len(rows) == 0 {
		return ""
	}
	var aligns []tableAlign
	dataStart := 1
	if len(rows) >= 2 && isTableDelim(rows[1]) {
		aligns = parseAligns(rows[1])
		dataStart = 2
	}

	var matrix [][]string
	matrix = append(matrix, cleanCells(splitTableRow(rows[0])))
	for r := dataStart; r < len(rows); r++ {
		matrix = append(matrix, cleanCells(splitTableRow(rows[r])))
	}

	cols := 0
	for _, row := range matrix {
		if len(row) > cols {
			cols = len(row)
		}
	}
	if cols == 0 {
		return ""
	}
	for ri := range matrix {
		for len(matrix[ri]) < cols {
			matrix[ri] = append(matrix[ri], "")
		}
	}
	for len(aligns) < cols {
		aligns = append(aligns, alignLeft)
	}

	widths := make([]int, cols)
	for _, row := range matrix {
		for c := 0; c < cols; c++ {
			w := utf8.RuneCountInString(row[c])
			if w > widths[c] {
				widths[c] = w
			}
		}
	}

	var b strings.Builder
	writeRow := func(cells []string) {
		parts := make([]string, cols)
		for c := 0; c < cols; c++ {
			parts[c] = padCell(cells[c], widths[c], aligns[c])
		}
		b.WriteString(strings.TrimRight(strings.Join(parts, " | "), " "))
	}

	var sep strings.Builder
	for c := 0; c < cols; c++ {
		if c > 0 {
			sep.WriteString("-+-")
		}
		sep.WriteString(strings.Repeat("-", widths[c]))
	}

	writeRow(matrix[0])
	b.WriteString("\n")
	b.WriteString(sep.String())
	for r := 1; r < len(matrix); r++ {
		b.WriteString("\n")
		writeRow(matrix[r])
	}

	return "<pre>" + escapeText(b.String()) + "</pre>"
}

func cleanCells(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = stripInlineToPlain(c)
	}
	return out
}

func padCell(s string, width int, a tableAlign) string {
	l := utf8.RuneCountInString(s)
	if l >= width {
		return s
	}
	diff := width - l
	switch a {
	case alignRight:
		return strings.Repeat(" ", diff) + s
	case alignCenter:
		left := diff / 2
		right := diff - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default:
		return s + strings.Repeat(" ", diff)
	}
}

var linkStripRe = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)

func stripInlineToPlain(s string) string {
	s = linkStripRe.ReplaceAllString(s, "$1")
	var b strings.Builder
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		c := runes[i]
		switch c {
		case '`':
			j := i + 1
			for j < len(runes) && runes[j] != '`' {
				j++
			}
			if j < len(runes) {
				b.WriteString(string(runes[i+1 : j]))
				i = j + 1
				continue
			}
			b.WriteRune(c)
			i++
		case '*', '_', '~':
			j := i
			for j < len(runes) && runes[j] == c {
				j++
			}
			i = j
		case '\\':
			if i+1 < len(runes) {
				b.WriteRune(runes[i+1])
				i += 2
				continue
			}
			b.WriteRune(c)
			i++
		default:
			b.WriteRune(c)
			i++
		}
	}
	return strings.TrimSpace(b.String())
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func escapeHref(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			b.WriteString("%3C")
		case '>':
			b.WriteString("%3E")
		case '"':
			b.WriteString("%22")
		case '\'':
			b.WriteString("%27")
		case ' ':
			b.WriteString("%20")
		case '\t':
			b.WriteString("%09")
		case '\n':
			b.WriteString("%0A")
		case '\r':
			b.WriteString("%0D")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

type itok struct {
	kind     string
	text     string
	href     string
	count    int
	canOpen  bool
	canClose bool
}

func inlineSpans(s string) string {
	toks := tokenizeInline(s)
	var b strings.Builder
	for _, t := range toks {
		switch t.kind {
		case "text":
			b.WriteString(escapeText(t.text))
		case "code":
			b.WriteString("<code>" + escapeText(t.text) + "</code>")
		case "open_b":
			b.WriteString("<b>")
		case "close_b":
			b.WriteString("</b>")
		case "open_i":
			b.WriteString("<i>")
		case "close_i":
			b.WriteString("</i>")
		case "open_s":
			b.WriteString("<s>")
		case "close_s":
			b.WriteString("</s>")
		case "link":
			b.WriteString("<a href=\"" + escapeHref(t.href) + "\">" + escapeText(t.text) + "</a>")
		}
	}
	return b.String()
}

func tokenizeInline(s string) []itok {
	var pre []itok
	runes := []rune(s)
	i := 0
	n := len(runes)
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			pre = append(pre, itok{kind: "text", text: lit.String()})
			lit.Reset()
		}
	}

	prevRune := func() (rune, bool) {
		if i == 0 {
			return 0, false
		}
		return runes[i-1], true
	}

	for i < n {
		c := runes[i]

		if c == '\\' && i+1 < n {
			lit.WriteRune(runes[i+1])
			i += 2
			continue
		}

		if c == '`' {
			start := i
			ticks := 0
			for i < n && runes[i] == '`' {
				ticks++
				i++
			}
			contentStart := i
			closeAt := -1
			j := i
			for j < n {
				if runes[j] == '`' {
					cnt := 0
					k := j
					for k < n && runes[k] == '`' {
						cnt++
						k++
					}
					if cnt == ticks {
						closeAt = j
						break
					}
					j = k
					continue
				}
				j++
			}
			if closeAt >= 0 {
				flush()
				code := string(runes[contentStart:closeAt])
				if len(code) >= 2 && strings.HasPrefix(code, " ") && strings.HasSuffix(code, " ") && strings.TrimSpace(code) != "" {
					code = code[1 : len(code)-1]
				}
				pre = append(pre, itok{kind: "code", text: code})
				i = closeAt + ticks
				continue
			}
			for k := 0; k < ticks; k++ {
				lit.WriteRune(runes[start+k])
			}
			continue
		}

		if c == '!' && i+1 < n && runes[i+1] == '[' {
			if label, _, end, ok := parseLink(runes, i+1); ok {
				lit.WriteString(stripInlineToPlain(label))
				i = end
				continue
			}
		}

		if c == '[' {
			if label, href, end, ok := parseLink(runes, i); ok {
				flush()
				pre = append(pre, itok{kind: "link", text: stripInlineToPlain(label), href: href})
				i = end
				continue
			}
		}

		if c == '*' || c == '_' || c == '~' {
			before, hasBefore := prevRune()
			run := 0
			for i < n && runes[i] == c {
				run++
				i++
			}
			var after rune
			hasAfter := i < n
			if hasAfter {
				after = runes[i]
			}

			beforeWS := !hasBefore || unicode.IsSpace(before)
			afterWS := !hasAfter || unicode.IsSpace(after)
			beforePunct := hasBefore && isMdPunct(before)
			afterPunct := hasAfter && isMdPunct(after)

			leftFlank := !afterWS && (!afterPunct || beforeWS || beforePunct)
			rightFlank := !beforeWS && (!beforePunct || afterWS || afterPunct)

			canOpen := leftFlank
			canClose := rightFlank
			if c == '_' {
				canOpen = leftFlank && (!rightFlank || beforePunct)
				canClose = rightFlank && (!leftFlank || afterPunct)
			}

			if c == '~' {
				if run < 2 {
					lit.WriteString(strings.Repeat("~", run))
					continue
				}
				run = 2
			}

			flush()
			pre = append(pre, itok{kind: "delim", text: string(c), canOpen: canOpen, canClose: canClose, count: run})
			continue
		}

		lit.WriteRune(c)
		i++
	}
	flush()

	return resolveEmphasis(pre)
}

func isMdPunct(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

func parseLink(runes []rune, start int) (string, string, int, bool) {
	n := len(runes)
	if start >= n || runes[start] != '[' {
		return "", "", 0, false
	}
	depth := 1
	j := start + 1
	for j < n {
		if runes[j] == '\\' && j+1 < n {
			j += 2
			continue
		}
		if runes[j] == '[' {
			depth++
		} else if runes[j] == ']' {
			depth--
			if depth == 0 {
				break
			}
		}
		j++
	}
	if depth != 0 || j >= n {
		return "", "", 0, false
	}
	label := string(runes[start+1 : j])
	if j+1 >= n || runes[j+1] != '(' {
		return "", "", 0, false
	}
	pdepth := 1
	k := j + 2
	urlStart := k
	for k < n {
		if runes[k] == '\\' && k+1 < n {
			k += 2
			continue
		}
		if runes[k] == '(' {
			pdepth++
		} else if runes[k] == ')' {
			pdepth--
			if pdepth == 0 {
				break
			}
		}
		k++
	}
	if pdepth != 0 {
		return "", "", 0, false
	}
	url := string(runes[urlStart:k])
	url = strings.TrimSpace(url)
	if idx := strings.IndexByte(url, ' '); idx >= 0 {
		url = url[:idx]
	}
	url = strings.TrimPrefix(url, "<")
	url = strings.TrimSuffix(url, ">")
	return label, url, k + 1, true
}

type emphMark struct {
	char     byte
	count    int
	canOpen  bool
	canClose bool
	active   bool
}

func resolveEmphasis(pre []itok) []itok {
	delims := make([]emphMark, len(pre))
	hasDelim := make([]bool, len(pre))
	for i, t := range pre {
		if t.kind == "delim" {
			delims[i] = emphMark{char: t.text[0], count: t.count, canOpen: t.canOpen, canClose: t.canClose, active: true}
			hasDelim[i] = true
		}
	}

	opensAt := make([][]string, len(pre))
	closesAt := make([][]string, len(pre))

	var stack []int
	for ci := 0; ci < len(pre); ci++ {
		if !hasDelim[ci] {
			continue
		}
		closer := &delims[ci]
		if !closer.active || !closer.canClose {
			if closer.canOpen && closer.active {
				stack = append(stack, ci)
			}
			continue
		}

		for closer.count > 0 {
			found := -1
			for si := len(stack) - 1; si >= 0; si-- {
				oi := stack[si]
				op := &delims[oi]
				if !op.active || op.char != closer.char || !op.canOpen {
					continue
				}
				found = si
				break
			}
			if found < 0 {
				break
			}
			oi := stack[found]
			opener := &delims[oi]

			var unit int
			if closer.char == '~' {
				unit = 2
				if opener.count < 2 || closer.count < 2 {
					opener.active = false
					break
				}
			} else if opener.count >= 2 && closer.count >= 2 {
				unit = 2
			} else {
				unit = 1
			}

			var openTag, closeTag string
			switch closer.char {
			case '~':
				openTag, closeTag = "open_s", "close_s"
			default:
				if unit == 2 {
					openTag, closeTag = "open_b", "close_b"
				} else {
					openTag, closeTag = "open_i", "close_i"
				}
			}

			opensAt[oi] = append(opensAt[oi], openTag)
			closesAt[ci] = append([]string{closeTag}, closesAt[ci]...)

			opener.count -= unit
			closer.count -= unit

			for si := len(stack) - 1; si > found; si-- {
				delims[stack[si]].active = false
			}
			stack = stack[:found+1]

			if opener.count == 0 {
				opener.active = false
				stack = stack[:found]
			}
		}

		if closer.count > 0 && closer.canOpen {
			stack = append(stack, ci)
		}
	}

	var final []itok
	for i, t := range pre {
		if !hasDelim[i] {
			final = append(final, t)
			continue
		}
		for _, ct := range closesAt[i] {
			final = append(final, itok{kind: ct})
		}
		if delims[i].count > 0 {
			final = append(final, itok{kind: "text", text: strings.Repeat(string(delims[i].char), delims[i].count)})
		}
		for _, ot := range opensAt[i] {
			final = append(final, itok{kind: ot})
		}
	}
	return final
}
