package extras

import (
	"encoding/json"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"html"
	"io"
	modules "main/modules"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
)

type datamuseSyllableEntry struct {
	Word         string `json:"word"`
	NumSyllables int    `json:"numSyllables"`
	Tags         []string `json:"tags"`
}

func countSyllablesHeuristic(word string) int {
	w := strings.ToLower(strings.TrimSpace(word))
	if w == "" {
		return 0
	}
	vowels := "aeiouy"
	count := 0
	prevVowel := false
	for _, r := range w {
		isVowel := strings.ContainsRune(vowels, r)
		if isVowel && !prevVowel {
			count++
		}
		prevVowel = isVowel
	}
	if strings.HasSuffix(w, "e") && count > 1 {
		runes := []rune(w)
		if len(runes) >= 2 {
			penult := runes[len(runes)-2]
			if !strings.ContainsRune(vowels, penult) {
				count--
			}
		}
	}
	if strings.HasSuffix(w, "le") && len(w) > 2 {
		runes := []rune(w)
		third := runes[len(runes)-3]
		if !strings.ContainsRune(vowels, third) {
			count++
		}
	}
	if count < 1 {
		count = 1
	}
	return count
}

func fetchSyllablesDatamuse(word string) (int, bool) {
	endpoint := fmt.Sprintf("https://api.datamuse.com/words?sp=%s&qe=sp&md=s&max=1", url.QueryEscape(word))
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return 0, false
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false
	}
	var entries []datamuseSyllableEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return 0, false
	}
	if len(entries) == 0 {
		return 0, false
	}
	for _, e := range entries {
		if strings.EqualFold(e.Word, word) && e.NumSyllables > 0 {
			return e.NumSyllables, true
		}
	}
	if entries[0].NumSyllables > 0 {
		return entries[0].NumSyllables, true
	}
	return 0, false
}

func splitSyllableInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || r == '-' || r == '\''
}

func SyllablesHandler(m *tg.NewMessage) error {
	input := splitSyllableInput(m)
	if input == "" {
		m.Reply("usage: /syllables &lt;word&gt;")
		return nil
	}
	fields := strings.Fields(input)
	if len(fields) > 8 {
		fields = fields[:8]
	}
	var b strings.Builder
	b.WriteString("<b>Syllable count</b>\n")
	total := 0
	for _, f := range fields {
		clean := strings.Map(func(r rune) rune {
			if isWordRune(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, f)
		if clean == "" {
			continue
		}
		count, ok := fetchSyllablesDatamuse(clean)
		source := "api"
		if !ok {
			count = countSyllablesHeuristic(clean)
			source = "heuristic"
		}
		total += count
		fmt.Fprintf(&b, "• <code>%s</code> — %d (%s)\n", html.EscapeString(clean), count, source)
	}
	if len(fields) > 1 {
		fmt.Fprintf(&b, "\n<b>Total:</b> %d", total)
	}
	m.Reply(b.String())
	return nil
}

func initFromSrc_word_define_offline_0_1() { modules.QueueHandlerRegistration(registerWordDefineOfflineHandlers) }
func registerWordDefineOfflineHandlers() {
	c := modules.Client
	c.On("cmd:syllables", SyllablesHandler)
}
func RandomWordHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://random-word-api.herokuapp.com/word")
	if err != nil {
		m.Reply("<b>Failed to fetch a random word.</b>")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("<b>Failed to fetch a random word.</b>")
		return nil
	}
	var words []string
	if err := json.NewDecoder(resp.Body).Decode(&words); err != nil || len(words) == 0 || words[0] == "" {
		m.Reply("<b>Failed to fetch a random word.</b>")
		return nil
	}
	out := "<b>Random Word</b>\n\n<code>" + html.EscapeString(words[0]) + "</code>\n\n<i>Source: random-word-api.herokuapp.com</i>"
	m.Reply(out)
	return nil
}

func initFromSrc_word_random_1_1() { modules.QueueHandlerRegistration(registerRandomWordHandlers) }
func registerRandomWordHandlers() {
	c := modules.Client
	c.On("cmd:randomword", RandomWordHandler)
}
type datamuseWord struct {
	Word  string `json:"word"`
	Score int    `json:"score"`
}

func fetchDatamuse(rel, word string, max int) ([]datamuseWord, error) {
	endpoint := fmt.Sprintf("https://api.datamuse.com/words?%s=%s&max=%d", rel, url.QueryEscape(word), max)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("datamuse returned status %d", resp.StatusCode)
	}
	var data []datamuseWord
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func renderWordList(title, emoji, word string, results []datamuseWord) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s <b>%s for:</b> <code>%s</code>\n", emoji, title, html.EscapeString(word)))
	b.WriteString(fmt.Sprintf("<i>%d results</i>\n\n", len(results)))
	b.WriteString("<blockquote>")
	parts := make([]string, 0, len(results))
	for _, r := range results {
		w := strings.TrimSpace(r.Word)
		if w == "" {
			continue
		}
		parts = append(parts, html.EscapeString(w))
	}
	b.WriteString(strings.Join(parts, ", "))
	b.WriteString("</blockquote>")
	return b.String()
}

func wordToolsRun(m *tg.NewMessage, cmd, rel, title, emoji string, max int) error {
	word := strings.TrimSpace(m.Args())
	if word == "" {
		m.Reply(fmt.Sprintf("<b>Usage:</b> <code>/%s &lt;word&gt;</code>", cmd))
		return nil
	}
	if strings.ContainsAny(word, " \t\n") {
		fields := strings.Fields(word)
		if len(fields) > 0 {
			word = fields[0]
		}
	}

	status, _ := m.Reply(fmt.Sprintf("Looking up <code>%s</code>...", html.EscapeString(word)))

	results, err := fetchDatamuse(rel, word, max)
	if err != nil {
		msg := "Failed to fetch results: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if len(results) == 0 {
		msg := fmt.Sprintf("No %s found for <code>%s</code>", strings.ToLower(title), html.EscapeString(word))
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	text := renderWordList(title, emoji, word, results)
	if status != nil {
		status.Edit(text, &tg.SendOptions{LinkPreview: false})
	} else {
		m.Reply(text, &tg.SendOptions{LinkPreview: false})
	}
	return nil
}

func RhymeHandler(m *tg.NewMessage) error {
	return wordToolsRun(m, "rhyme", "rel_rhy", "Rhymes", "🎵", 20)
}

func SynonymHandler(m *tg.NewMessage) error {
	return wordToolsRun(m, "synonym", "rel_syn", "Synonyms", "🔁", 15)
}

func AntonymHandler(m *tg.NewMessage) error {
	return wordToolsRun(m, "antonym", "rel_ant", "Antonyms", "↔️", 15)
}

func registerWordToolsHandlers() {
	c := modules.Client
	c.On("cmd:rhyme", RhymeHandler)
	c.On("cmd:synonym", SynonymHandler)
	c.On("cmd:antonym", AntonymHandler)
}

func initFromSrc_word_tools_2_1() {
	modules.QueueHandlerRegistration(registerWordToolsHandlers)
}
var wordLadderDict = []string{
	"cat", "cot", "dot", "dog", "log", "bog", "bag", "bat", "bit", "big",
	"bug", "but", "hut", "hat", "hit", "hot", "lot", "lit", "fit", "fat",
	"fan", "man", "men", "ten", "tin", "win", "wig", "wag", "wad", "had",
	"hay", "bay", "ban", "bin", "pin", "pit", "pat", "pet", "pen", "ten",
	"ton", "ten", "den", "din", "dim", "rim", "ram", "raw", "row", "low",
	"law", "lay", "say", "sat", "set", "see", "bee", "beg", "bed", "red",
	"led", "let", "lot", "lop", "top", "tap", "map", "mat", "mad", "sad",
	"sap", "sip", "sin", "son", "sun", "fun", "run", "rug", "rag", "rat",
	"ear", "far", "for", "fog", "foe", "toe", "tow", "now", "new", "few",
	"fey", "key", "kit", "bit", "bid", "bad", "bud", "mud", "mug", "jug",
	"jog", "job", "mob", "rob", "rib", "ribs", "ride", "rude",
	"cold", "cord", "card", "ward", "warm", "worm", "word", "ford", "fort",
	"port", "post", "past", "pest", "test", "rest", "best", "bust", "bust",
	"head", "heat", "heal", "seal", "deal", "real", "read", "load", "road",
	"toad", "told", "tolt", "tilt", "till", "tall", "tale", "tame", "time",
	"lime", "line", "lane", "land", "band", "bond", "bend", "send", "sand",
	"hand", "hard", "harm", "warm", "ware", "wore", "wire", "fire", "five",
	"give", "live", "love", "dove", "dive", "dime", "dome", "home", "hope",
	"rope", "rose", "rise", "ride", "rode", "rude", "ruse", "muse", "mute",
	"cute", "cube", "tube", "tune", "tone", "tons", "tops", "tips", "ties",
	"like", "lake", "make", "mace", "race", "rice", "nice", "nine", "mine",
	"wine", "wind", "wand", "want", "rant", "rent", "tent", "tint", "lint",
	"link", "pink", "pine", "vine", "vane", "cane", "came", "case", "cast",
	"cost", "host", "most", "must", "bust", "busy", "bury", "burn", "barn",
	"born", "corn", "core", "bore", "bone", "bone", "gone", "gore", "gory",
	"glory",
	"heart", "heard", "beard", "bears", "bears", "years", "wears", "wares",
	"hares", "bares", "bores", "cores", "cares", "dares", "dates", "gates",
	"rates", "rites", "rides", "ridge", "bride", "pride", "price", "prize",
	"prose", "proud", "cloud", "clout", "clear", "clean", "lean", "learn",
	"earth", "earns", "yearn", "burns", "burst", "burnt", "blunt", "plant",
	"plait", "plain", "stain", "stair", "stare", "store", "stove", "stone",
	"shone", "shore", "shire", "share", "shape", "shame", "shams",
	"slime", "slim", "slip", "ship", "shop", "shot", "shut", "shun",
	"flame", "flake", "fluke", "flute", "fruit", "front", "frost", "first",
	"thirst", "thirty",
	"black", "block", "clock", "click", "stick", "stack", "shack", "shock",
	"smock", "smoke", "spoke", "spore", "spare", "scare", "score", "scope",
	"slope", "slate", "skate", "state", "stake", "snake", "shake", "shade",
	"shape", "shaft", "shift", "swift", "swirl",
	"world", "would", "could", "mould", "moult", "vault", "fault",
	"bread", "break", "creak", "creek", "cheek", "cheep", "sheep", "sleep",
	"sleek", "sleet", "fleet", "fleer", "flier", "flies", "fries", "tries",
	"trees", "frees", "freed", "greed", "green", "preen", "prone", "drone",
	"drove", "grove", "grave", "brave", "brake", "brace", "trace", "track",
	"trick", "trunk", "drunk", "drink",
}

func buildWordLadderDict(length int) map[string]bool {
	out := map[string]bool{}
	for _, w := range wordLadderDict {
		if len(w) == length {
			out[strings.ToLower(w)] = true
		}
	}
	return out
}

func wordLadderDiff(a, b string) int {
	d := 0
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			d++
		}
	}
	return d
}

func findWordLadder(start, end string, dict map[string]bool) []string {
	if start == end {
		return []string{start}
	}
	if !dict[start] || !dict[end] {
		return nil
	}
	type node struct {
		word string
		path []string
	}
	visited := map[string]bool{start: true}
	queue := []node{{start, []string{start}}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if len(cur.path) > 12 {
			continue
		}
		bytes := []byte(cur.word)
		for i := 0; i < len(bytes); i++ {
			orig := bytes[i]
			for c := byte('a'); c <= 'z'; c++ {
				if c == orig {
					continue
				}
				bytes[i] = c
				candidate := string(bytes)
				if dict[candidate] && !visited[candidate] {
					newPath := make([]string, len(cur.path)+1)
					copy(newPath, cur.path)
					newPath[len(cur.path)] = candidate
					if candidate == end {
						return newPath
					}
					visited[candidate] = true
					queue = append(queue, node{candidate, newPath})
				}
			}
			bytes[i] = orig
		}
	}
	return nil
}

func WordLadderHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/wordladder &lt;start&gt; &lt;end&gt;</code>\nBoth words must be the same length (3-5 letters).")
		return nil
	}
	parts := strings.Fields(args)
	if len(parts) < 2 {
		m.Reply("<b>Usage:</b> <code>/wordladder &lt;start&gt; &lt;end&gt;</code>\nProvide two words separated by a space.")
		return nil
	}
	start := strings.ToLower(parts[0])
	end := strings.ToLower(parts[1])
	if len(start) != len(end) {
		m.Reply(fmt.Sprintf("<b>Error:</b> words must be the same length. Got <code>%s</code> (%d) and <code>%s</code> (%d).",
			html.EscapeString(start), len(start), html.EscapeString(end), len(end)))
		return nil
	}
	if len(start) < 3 || len(start) > 5 {
		m.Reply("<b>Error:</b> word length must be between 3 and 5 letters.")
		return nil
	}
	for _, r := range start + end {
		if r < 'a' || r > 'z' {
			m.Reply("<b>Error:</b> only lowercase letters a-z are allowed.")
			return nil
		}
	}
	dict := buildWordLadderDict(len(start))
	if !dict[start] {
		m.Reply(fmt.Sprintf("<b>Error:</b> <code>%s</code> is not in the built-in dictionary.", html.EscapeString(start)))
		return nil
	}
	if !dict[end] {
		m.Reply(fmt.Sprintf("<b>Error:</b> <code>%s</code> is not in the built-in dictionary.", html.EscapeString(end)))
		return nil
	}
	if start == end {
		m.Reply(fmt.Sprintf("<b>Word Ladder</b>\n\n<i>Start equals End.</i>\n\n<code>%s</code>", html.EscapeString(start)))
		return nil
	}
	path := findWordLadder(start, end, dict)
	if path == nil {
		m.Reply(fmt.Sprintf("<b>No ladder found</b> between <code>%s</code> and <code>%s</code> in the built-in dictionary.",
			html.EscapeString(start), html.EscapeString(end)))
		return nil
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<b>Word Ladder</b>\n\n<b>From:</b> <code>%s</code>\n<b>To:</b> <code>%s</code>\n<b>Steps:</b> %d\n\n",
		html.EscapeString(start), html.EscapeString(end), len(path)-1))
	for i, w := range path {
		marker := "  "
		if i == 0 {
			marker = "<b>Start</b>"
		} else if i == len(path)-1 {
			marker = "<b>End</b>"
		} else {
			marker = fmt.Sprintf("Step %d", i)
		}
		diffStr := ""
		if i > 0 {
			diffStr = fmt.Sprintf(" <i>(changed %d letter)</i>", wordLadderDiff(path[i-1], w))
		}
		b.WriteString(fmt.Sprintf("%s: <code>%s</code>%s\n", marker, html.EscapeString(w), diffStr))
	}
	m.Reply(b.String())
	return nil
}

func registerWordLadderHandlers() {
	c := modules.Client
	c.On("cmd:wordladder", WordLadderHandler)
}

func initFromSrc_word_unscrambler_3_1() {
	modules.QueueHandlerRegistration(registerWordLadderHandlers)
}
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
	c := modules.Client
	c.On("cmd:wordfreq", WordFreqHandler)
	c.On("cmd:charstats", CharStatsHandler)
}

func initFromSrc_wordcount_4_1() { modules.QueueHandlerRegistration(registerWordCountHandlers) }

func init() {
	initFromSrc_word_define_offline_0_1()
	initFromSrc_word_random_1_1()
	initFromSrc_word_tools_2_1()
	initFromSrc_word_unscrambler_3_1()
	initFromSrc_wordcount_4_1()
}
