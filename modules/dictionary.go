package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type dictionaryDefinition struct {
	Definition string `json:"definition"`
	Example    string `json:"example"`
}

type dictionaryMeaning struct {
	PartOfSpeech string                 `json:"partOfSpeech"`
	Definitions  []dictionaryDefinition `json:"definitions"`
}

type dictionaryPhonetic struct {
	Text string `json:"text"`
}

type dictionaryEntry struct {
	Word      string               `json:"word"`
	Phonetic  string               `json:"phonetic"`
	Phonetics []dictionaryPhonetic `json:"phonetics"`
	Meanings  []dictionaryMeaning  `json:"meanings"`
}

func DefineHandler(m *tg.NewMessage) error {
	word := strings.TrimSpace(m.Args())
	if word == "" {
		m.Reply("Usage: <code>/define &lt;word&gt;</code>")
		return nil
	}
	client := &http.Client{Timeout: 30 * time.Second}
	url := "https://api.dictionaryapi.dev/api/v2/entries/en/" + word
	resp, err := client.Get(url)
	if err != nil {
		m.Reply("couldn't fetch definition: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		m.Reply("No definition found for <b>" + html.EscapeString(word) + "</b>")
		return nil
	}
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var entries []dictionaryEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		m.Reply("couldn't parse definition: " + err.Error())
		return nil
	}
	if len(entries) == 0 {
		m.Reply("No definition found for <b>" + html.EscapeString(word) + "</b>")
		return nil
	}
	entry := entries[0]
	headword := entry.Word
	if headword == "" {
		headword = word
	}
	phonetic := entry.Phonetic
	if phonetic == "" {
		for _, p := range entry.Phonetics {
			if p.Text != "" {
				phonetic = p.Text
				break
			}
		}
	}
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(headword))
	b.WriteString("</b>")
	if phonetic != "" {
		b.WriteString(" <i>")
		b.WriteString(html.EscapeString(phonetic))
		b.WriteString("</i>")
	}
	b.WriteString("\n")
	count := 0
	for _, meaning := range entry.Meanings {
		if count >= 3 {
			break
		}
		for _, def := range meaning.Definitions {
			if count >= 3 {
				break
			}
			if def.Definition == "" {
				continue
			}
			count++
			b.WriteString(fmt.Sprintf("\n<b>%d.</b> <i>(%s)</i> %s", count, html.EscapeString(meaning.PartOfSpeech), html.EscapeString(def.Definition)))
			if def.Example != "" {
				b.WriteString("\n   <i>e.g. ")
				b.WriteString(html.EscapeString(def.Example))
				b.WriteString("</i>")
			}
		}
	}
	if count == 0 {
		m.Reply("No definition found for <b>" + html.EscapeString(word) + "</b>")
		return nil
	}
	m.Reply(b.String())
	return nil
}

func init() { QueueHandlerRegistration(registerDictionaryHandlers) }
func registerDictionaryHandlers() {
	c := Client
	c.On("cmd:define", DefineHandler)
}
