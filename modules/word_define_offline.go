package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	tg "github.com/amarnathcjd/gogram/telegram"
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

func init() { QueueHandlerRegistration(registerWordDefineOfflineHandlers) }
func registerWordDefineOfflineHandlers() {
	c := Client
	c.On("cmd:syllables", SyllablesHandler)
}
