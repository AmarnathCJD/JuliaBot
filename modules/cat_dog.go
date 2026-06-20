package modules

import (
	"encoding/json"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type catApiResponse struct {
	URL string `json:"url"`
}

type dogApiResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

func CatHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://api.thecatapi.com/v1/images/search")
	if err != nil {
		m.Reply("Failed to fetch a cat. Try again later.")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("Cat API returned an error. Try again later.")
		return nil
	}
	var data []catApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse cat response.")
		return nil
	}
	if len(data) == 0 || data[0].URL == "" {
		m.Reply("No cat image found. Try again.")
		return nil
	}
	if _, err := m.ReplyMedia(data[0].URL, &tg.MediaOptions{Caption: "Meow!"}); err != nil {
		m.Reply("<a href=\"" + data[0].URL + "\">Cat</a>")
	}
	return nil
}

func DogHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://dog.ceo/api/breeds/image/random")
	if err != nil {
		m.Reply("Failed to fetch a dog. Try again later.")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("Dog API returned an error. Try again later.")
		return nil
	}
	var data dogApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse dog response.")
		return nil
	}
	if data.Status != "success" || data.Message == "" {
		m.Reply("No dog image found. Try again.")
		return nil
	}
	if _, err := m.ReplyMedia(data.Message, &tg.MediaOptions{Caption: "Woof!"}); err != nil {
		m.Reply("<a href=\"" + data.Message + "\">Dog</a>")
	}
	return nil
}

func init() { QueueHandlerRegistration(registerCatDogHandlers) }
func registerCatDogHandlers() {
	c := Client
	c.On("cmd:cat", CatHandler)
	c.On("cmd:dog", DogHandler)
}
