package modules

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var wordFreqStopwords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"if": true, "of": true, "at": true, "by": true, "for": true, "with": true,
	"about": true, "against": true, "between": true, "into": true, "through": true,
	"during": true, "before": true, "after": true, "above": true, "below": true,
	"to": true, "from": true, "up": true, "down": true, "in": true, "out": true,
	"on": true, "off": true, "over": true, "under": true, "again": true, "further": true,
	"then": true, "once": true, "here": true, "there": true, "when": true, "where": true,
	"why": true, "how": true, "all": true, "any": true, "both": true, "each": true,
	"few": true, "more": true, "most": true, "other": true, "some": true, "such": true,
	"no": true, "nor": true, "not": true, "only": true, "own": true, "same": true,
	"so": true, "than": true, "too": true, "very": true, "s": true, "t": true,
	"can": true, "will": true, "just": true, "don": true, "should": true, "now": true,
	"i": true, "me": true, "my": true, "myself": true, "we": true, "our": true,
	"ours": true, "ourselves": true, "you": true, "your": true, "yours": true,
	"yourself": true, "yourselves": true, "he": true, "him": true, "his": true,
	"himself": true, "she": true, "her": true, "hers": true, "herself": true,
	"it": true, "its": true, "itself": true, "they": true, "them": true, "their": true,
	"theirs": true, "themselves": true, "what": true, "which": true, "who": true,
	"whom": true, "this": true, "that": true, "these": true, "those": true,
	"am": true, "is": true, "are": true, "was": true, "were": true, "be": true,
	"been": true, "being": true, "have": true, "has": true, "had": true, "having": true,
	"do": true, "does": true, "did": true, "doing": true, "would": true, "could": true,
	"shall": true, "may": true, "might": true, "must": true, "ought": true,
	"im": true, "ive": true, "youre": true, "youve": true, "hes": true, "shes": true,
	"its2": true, "were2": true, "theyre": true, "ill": true, "youll": true,
	"hell": true, "shell2": true, "well": true, "theyll": true, "isnt": true,
	"arent": true, "wasnt": true, "werent": true, "hasnt": true, "havent": true,
	"hadnt": true, "doesnt": true, "dont": true, "didnt": true, "wont": true,
	"wouldnt": true, "shant": true, "shouldnt": true, "cant": true, "cannot": true,
	"couldnt": true, "mustnt": true, "lets": true, "thats": true, "whos": true,
	"whats": true, "heres": true, "theres": true, "whens": true, "wheres": true,
	"whys": true, "hows": true,
}

func tokenizeWords(text string) []string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}

func WordFreqHandler(m *tg.NewMessage) error {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/wordfreq &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	tokens := tokenizeWords(text)
	if len(tokens) == 0 {
		m.Reply("<b>Error:</b> no words found.")
		return nil
	}
	counts := make(map[string]int, len(tokens))
	total := 0
	for _, w := range tokens {
		if wordFreqStopwords[w] {
			continue
		}
		if len(w) == 1 {
			continue
		}
		counts[w]++
		total++
	}
	if len(counts) == 0 {
		m.Reply("<b>Error:</b> nothing left after removing stopwords.")
		return nil
	}
	type kv struct {
		k string
		v int
	}
	pairs := make([]kv, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].v != pairs[j].v {
			return pairs[i].v > pairs[j].v
		}
		return pairs[i].k < pairs[j].k
	})
	limit := 10
	if len(pairs) < limit {
		limit = len(pairs)
	}
	maxWordLen := 4
	for i := 0; i < limit; i++ {
		if l := len(pairs[i].k); l > maxWordLen {
			maxWordLen = l
		}
	}
	if maxWordLen > 20 {
		maxWordLen = 20
	}
	var sb strings.Builder
	sb.WriteString("<b>Top Words</b>\n")
	sb.WriteString(fmt.Sprintf("<i>Unique:</i> <code>%d</code> <i>Total:</i> <code>%d</code>\n\n", len(counts), total))
	sb.WriteString("<pre>")
	sb.WriteString(fmt.Sprintf("%-3s %-*s %5s %6s\n", "#", maxWordLen, "Word", "Count", "Pct"))
	sb.WriteString(strings.Repeat("-", 3+1+maxWordLen+1+5+1+6))
	sb.WriteString("\n")
	for i := 0; i < limit; i++ {
		w := pairs[i].k
		if len(w) > maxWordLen {
			w = w[:maxWordLen]
		}
		pct := float64(pairs[i].v) * 100.0 / float64(total)
		sb.WriteString(fmt.Sprintf("%-3d %-*s %5d %5.1f%%\n", i+1, maxWordLen, w, pairs[i].v, pct))
	}
	sb.WriteString("</pre>")
	m.Reply(sb.String())
	return nil
}

func CharStatsHandler(m *tg.NewMessage) error {
	var text string
	args := strings.TrimSpace(m.Args())
	if args != "" {
		text = args
	} else if m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = r.Text()
		}
	}
	if text == "" {
		m.Reply("<b>Usage:</b> reply to a message with <code>/charstats</code> or provide text.")
		return nil
	}
	vowelSet := map[rune]bool{
		'a': true, 'e': true, 'i': true, 'o': true, 'u': true,
		'A': true, 'E': true, 'I': true, 'O': true, 'U': true,
	}
	var vowels, consonants, digits, spaces, punct, other, total int
	for _, r := range text {
		total++
		switch {
		case unicode.IsSpace(r):
			spaces++
		case unicode.IsDigit(r):
			digits++
		case unicode.IsLetter(r):
			if vowelSet[r] {
				vowels++
			} else {
				consonants++
			}
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			punct++
		default:
			other++
		}
	}
	if total == 0 {
		m.Reply("<b>Error:</b> empty input.")
		return nil
	}
	letters := vowels + consonants
	pct := func(n int) float64 {
		if total == 0 {
			return 0
		}
		return float64(n) * 100.0 / float64(total)
	}
	var sb strings.Builder
	sb.WriteString("<b>Character Stats</b>\n")
	sb.WriteString(fmt.Sprintf("<i>Total characters:</i> <code>%d</code>\n", total))
	sb.WriteString(fmt.Sprintf("<i>Letters:</i> <code>%d</code>\n\n", letters))
	sb.WriteString("<pre>")
	sb.WriteString(fmt.Sprintf("%-12s %6s %6s\n", "Category", "Count", "Pct"))
	sb.WriteString(strings.Repeat("-", 27))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%-12s %6d %5.1f%%\n", "vowels", vowels, pct(vowels)))
	sb.WriteString(fmt.Sprintf("%-12s %6d %5.1f%%\n", "consonants", consonants, pct(consonants)))
	sb.WriteString(fmt.Sprintf("%-12s %6d %5.1f%%\n", "digits", digits, pct(digits)))
	sb.WriteString(fmt.Sprintf("%-12s %6d %5.1f%%\n", "spaces", spaces, pct(spaces)))
	sb.WriteString(fmt.Sprintf("%-12s %6d %5.1f%%\n", "punctuation", punct, pct(punct)))
	sb.WriteString(fmt.Sprintf("%-12s %6d %5.1f%%\n", "other", other, pct(other)))
	sb.WriteString("</pre>")
	m.Reply(sb.String())
	return nil
}

func registerWordCountHandlers() {
	c := Client
	c.On("cmd:wordfreq", WordFreqHandler)
	c.On("cmd:charstats", CharStatsHandler)
}

func init() { QueueHandlerRegistration(registerWordCountHandlers) }
