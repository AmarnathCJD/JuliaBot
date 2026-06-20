package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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
	c := Client
	c.On("cmd:rhyme", RhymeHandler)
	c.On("cmd:synonym", SynonymHandler)
	c.On("cmd:antonym", AntonymHandler)
}

func init() {
	QueueHandlerRegistration(registerWordToolsHandlers)
}
