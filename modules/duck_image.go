package modules

import (
	"encoding/json"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type duckApiResponse struct {
	Message string `json:"message"`
	URL     string `json:"url"`
}

func DuckHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://random-d.uk/api/v2/random")
	if err != nil {
		m.Reply("Failed to fetch a duck. Try again later.")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("Duck API returned an error. Try again later.")
		return nil
	}
	var data duckApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse duck response.")
		return nil
	}
	if data.URL == "" {
		m.Reply("No duck image found. Try again.")
		return nil
	}
	if _, err := m.ReplyMedia(data.URL, &tg.MediaOptions{Caption: "Quack!"}); err != nil {
		m.Reply("<a href=\"" + data.URL + "\">Duck</a>")
	}
	return nil
}

func init() { QueueHandlerRegistration(registerDuckHandlers) }
func registerDuckHandlers() {
	c := Client
	c.On("cmd:duck", DuckHandler)
}
