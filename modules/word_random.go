package modules

import (
	"encoding/json"
	"html"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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

func init() { QueueHandlerRegistration(registerRandomWordHandlers) }
func registerRandomWordHandlers() {
	c := Client
	c.On("cmd:randomword", RandomWordHandler)
}
