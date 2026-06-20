package modules

import (
	"encoding/json"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type foxApiResponse struct {
	Image string `json:"image"`
	Link  string `json:"link"`
}

func FoxHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://randomfox.ca/floof/")
	if err != nil {
		m.Reply("Failed to fetch a fox. Try again later.")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("Fox API returned an error. Try again later.")
		return nil
	}
	var data foxApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse fox response.")
		return nil
	}
	if data.Image == "" {
		m.Reply("No fox image found. Try again.")
		return nil
	}
	if _, err := m.ReplyMedia(data.Image, &tg.MediaOptions{Caption: "What does the fox say?"}); err != nil {
		m.Reply("<a href=\"" + data.Image + "\">Fox</a>")
	}
	return nil
}

func init() { QueueHandlerRegistration(registerFoxHandlers) }
func registerFoxHandlers() {
	c := Client
	c.On("cmd:fox", FoxHandler)
}
