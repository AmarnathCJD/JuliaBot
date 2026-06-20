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

var dadlifeClient = &http.Client{Timeout: 30 * time.Second}

type quotableQuote struct {
	Content string `json:"content"`
	Author  string `json:"author"`
}

type adviceSlipResp struct {
	Slip struct {
		ID     int    `json:"id"`
		Advice string `json:"advice"`
	} `json:"slip"`
}

type kanyeQuote struct {
	Quote string `json:"quote"`
}

type trumpQuote struct {
	Value     string `json:"value"`
	AppearedAt string `json:"appeared_at"`
	Embedded  struct {
		Author []struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"_embedded"`
}

func fetchDadlifeJSON(url string, headers map[string]string, out interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := dadlifeClient.Do(req)
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

func InspireHandler(m *tg.NewMessage) error {
	var q quotableQuote
	if err := fetchDadlifeJSON("https://api.quotable.io/random", nil, &q); err != nil {
		m.Reply("<b>Error:</b> failed to fetch inspirational quote.")
		return nil
	}
	if q.Content == "" {
		m.Reply("<b>Error:</b> empty quote received.")
		return nil
	}
	out := fmt.Sprintf("<i>%s</i>", html.EscapeString(q.Content))
	if q.Author != "" {
		out += fmt.Sprintf("\n\n<b>- %s</b>", html.EscapeString(q.Author))
	}
	m.Reply(out)
	return nil
}

func AdviceHandler(m *tg.NewMessage) error {
	var a adviceSlipResp
	if err := fetchDadlifeJSON("https://api.adviceslip.com/advice", nil, &a); err != nil {
		m.Reply("<b>Error:</b> failed to fetch advice.")
		return nil
	}
	if a.Slip.Advice == "" {
		m.Reply("<b>Error:</b> empty advice received.")
		return nil
	}
	m.Reply(fmt.Sprintf("<b>Advice:</b>\n<i>%s</i>", html.EscapeString(a.Slip.Advice)))
	return nil
}

func KanyeHandler(m *tg.NewMessage) error {
	var q kanyeQuote
	if err := fetchDadlifeJSON("https://api.kanye.rest/", nil, &q); err != nil {
		m.Reply("<b>Error:</b> failed to fetch Kanye quote.")
		return nil
	}
	if q.Quote == "" {
		m.Reply("<b>Error:</b> empty quote received.")
		return nil
	}
	m.Reply(fmt.Sprintf("<i>%s</i>\n\n<b>- Kanye West</b>", html.EscapeString(q.Quote)))
	return nil
}

func TrumpHandler(m *tg.NewMessage) error {
	var q trumpQuote
	headers := map[string]string{"Accept": "application/json"}
	if err := fetchDadlifeJSON("https://api.tronalddump.io/random/quote", headers, &q); err != nil {
		m.Reply("<b>Error:</b> failed to fetch Trump quote.")
		return nil
	}
	if q.Value == "" {
		m.Reply("<b>Error:</b> empty quote received.")
		return nil
	}
	out := fmt.Sprintf("<i>%s</i>\n\n<b>- Donald Trump</b>", html.EscapeString(q.Value))
	if q.AppearedAt != "" && len(q.AppearedAt) >= 10 {
		out += fmt.Sprintf("\n<code>%s</code>", html.EscapeString(q.AppearedAt[:10]))
	}
	m.Reply(out)
	return nil
}

func registerDadlifeHandlers() {
	c := Client
	c.On("cmd:inspire", InspireHandler)
	c.On("cmd:advice", AdviceHandler)
	c.On("cmd:kanye", KanyeHandler)
	c.On("cmd:trump", TrumpHandler)
}

func init() {
	QueueHandlerRegistration(registerDadlifeHandlers)
}
