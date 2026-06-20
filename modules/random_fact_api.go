package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var randomFactAPIClient = &http.Client{Timeout: 30 * time.Second}

type uselessFactResp struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Source    string `json:"source"`
	SourceURL string `json:"source_url"`
	Language  string `json:"language"`
	Permalink string `json:"permalink"`
}

type zenQuoteResp struct {
	Q    string `json:"q"`
	A    string `json:"a"`
	I    string `json:"i"`
	H    string `json:"h"`
	Date string `json:"date"`
}

func fetchRandomFactAPIJSON(url string, out interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "JuliaBot (https://github.com/amarnathcjd)")
	resp, err := randomFactAPIClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func RandomFact2Handler(m *tg.NewMessage) error {
	var f uselessFactResp
	if err := fetchRandomFactAPIJSON("https://uselessfacts.jsph.pl/api/v2/facts/random?language=en", &f); err != nil {
		m.Reply("<b>Error:</b> failed to fetch random fact.")
		return nil
	}
	if f.Text == "" {
		m.Reply("<b>Error:</b> empty fact received.")
		return nil
	}
	source := f.Source
	if source == "" {
		source = "unknown"
	}
	msg := fmt.Sprintf("<b>Random Fact:</b>\n<i>%s</i>\n\n<b>Source:</b> <code>%s</code>", html.EscapeString(f.Text), html.EscapeString(source))
	if f.Permalink != "" {
		msg += fmt.Sprintf("\n<a href=\"%s\">permalink</a>", html.EscapeString(f.Permalink))
	}
	m.Reply(msg)
	return nil
}

func QuoteDailyHandler(m *tg.NewMessage) error {
	var arr []zenQuoteResp
	if err := fetchRandomFactAPIJSON("https://zenquotes.io/api/today", &arr); err != nil {
		m.Reply("<b>Error:</b> failed to fetch daily quote.")
		return nil
	}
	if len(arr) == 0 || arr[0].Q == "" {
		m.Reply("<b>Error:</b> empty quote received.")
		return nil
	}
	q := arr[0]
	author := q.A
	if author == "" {
		author = "Unknown"
	}
	msg := fmt.Sprintf("<b>Quote of the Day:</b>\n<i>%s</i>\n\n<b>— %s</b>", html.EscapeString(q.Q), html.EscapeString(author))
	if q.Date != "" {
		msg += fmt.Sprintf("\n<code>%s</code>", html.EscapeString(q.Date))
	}
	m.Reply(msg)
	return nil
}

func RandomQuoteHandler(m *tg.NewMessage) error {
	var arr []zenQuoteResp
	if err := fetchRandomFactAPIJSON("https://zenquotes.io/api/random", &arr); err != nil {
		m.Reply("<b>Error:</b> failed to fetch random quote.")
		return nil
	}
	if len(arr) == 0 || arr[0].Q == "" {
		m.Reply("<b>Error:</b> empty quote received.")
		return nil
	}
	q := arr[0]
	author := q.A
	if author == "" {
		author = "Unknown"
	}
	m.Reply(fmt.Sprintf("<b>Random Quote:</b>\n<i>%s</i>\n\n<b>— %s</b>", html.EscapeString(q.Q), html.EscapeString(author)))
	return nil
}

func registerRandomFactAPIHandlers() {
	c := Client
	c.On("cmd:randomfact2", RandomFact2Handler)
	c.On("cmd:quotedaily", QuoteDailyHandler)
	c.On("cmd:randomquote", RandomQuoteHandler)
}

func init() {
	QueueHandlerRegistration(registerRandomFactAPIHandlers)
}
