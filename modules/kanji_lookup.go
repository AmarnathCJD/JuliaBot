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

type kanjiLookupResponse struct {
	Kanji        string   `json:"kanji"`
	Grade        int      `json:"grade"`
	StrokeCount  int      `json:"stroke_count"`
	Meanings     []string `json:"meanings"`
	KunReadings  []string `json:"kun_readings"`
	OnReadings   []string `json:"on_readings"`
	NameReadings []string `json:"name_readings"`
	JLPT         int      `json:"jlpt"`
	Unicode      string   `json:"unicode"`
	HeisigEn     string   `json:"heisig_en"`
}

func KanjiLookupHandler(m *tg.NewMessage) error {
	q := strings.TrimSpace(m.Args())
	if q == "" {
		m.Reply("Usage: <code>/kanji &lt;character&gt;</code>")
		return nil
	}
	runes := []rune(q)
	character := string(runes[0])
	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := "https://kanjiapi.dev/v1/kanji/" + url.PathEscape(character)
	resp, err := client.Get(endpoint)
	if err != nil {
		m.Reply("couldn't fetch kanji: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		m.Reply("No kanji found for <b>" + html.EscapeString(character) + "</b>")
		return nil
	}
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data kanjiLookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't parse kanji: " + err.Error())
		return nil
	}
	if data.Kanji == "" {
		m.Reply("No kanji found for <b>" + html.EscapeString(character) + "</b>")
		return nil
	}
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(data.Kanji))
	b.WriteString("</b>")
	if data.Unicode != "" {
		b.WriteString(" <i>(U+")
		b.WriteString(html.EscapeString(data.Unicode))
		b.WriteString(")</i>")
	}
	b.WriteString("\n")
	if len(data.Meanings) > 0 {
		b.WriteString("\n<b>Meanings:</b> ")
		b.WriteString(html.EscapeString(strings.Join(data.Meanings, ", ")))
	}
	if len(data.OnReadings) > 0 {
		b.WriteString("\n<b>On:</b> ")
		b.WriteString(html.EscapeString(strings.Join(data.OnReadings, ", ")))
	}
	if len(data.KunReadings) > 0 {
		b.WriteString("\n<b>Kun:</b> ")
		b.WriteString(html.EscapeString(strings.Join(data.KunReadings, ", ")))
	}
	if len(data.NameReadings) > 0 {
		b.WriteString("\n<b>Name:</b> ")
		b.WriteString(html.EscapeString(strings.Join(data.NameReadings, ", ")))
	}
	b.WriteString(fmt.Sprintf("\n<b>Strokes:</b> %d", data.StrokeCount))
	if data.JLPT > 0 {
		b.WriteString(fmt.Sprintf("\n<b>JLPT:</b> N%d", data.JLPT))
	}
	if data.Grade > 0 {
		b.WriteString(fmt.Sprintf("\n<b>Grade:</b> %d", data.Grade))
	}
	if data.HeisigEn != "" {
		b.WriteString("\n<b>Heisig:</b> ")
		b.WriteString(html.EscapeString(data.HeisigEn))
	}
	m.Reply(b.String())
	return nil
}

func init() { QueueHandlerRegistration(registerKanjiLookupHandlers) }
func registerKanjiLookupHandlers() {
	c := Client
	c.On("cmd:kanji", KanjiLookupHandler)
}
