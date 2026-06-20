package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type jokeAPIResponse struct {
	Error    bool   `json:"error"`
	Category string `json:"category"`
	Type     string `json:"type"`
	Setup    string `json:"setup"`
	Delivery string `json:"delivery"`
	Joke     string `json:"joke"`
	ID       int    `json:"id"`
	Safe     bool   `json:"safe"`
	Lang     string `json:"lang"`
}

func fetchJokeAPI(url string) (*jokeAPIResponse, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var data jokeAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error {
		return nil, fmt.Errorf("api returned error")
	}
	return &data, nil
}

func formatJokeAPI(data *jokeAPIResponse, title string) string {
	out := "<b>" + title + "</b>"
	if data.Category != "" {
		out += " <i>(" + html.EscapeString(data.Category) + ")</i>"
	}
	out += "\n\n"
	if data.Type == "twopart" {
		out += "<blockquote>" + html.EscapeString(data.Setup) + "</blockquote>\n"
		out += "<blockquote>" + html.EscapeString(data.Delivery) + "</blockquote>"
	} else {
		out += "<blockquote>" + html.EscapeString(data.Joke) + "</blockquote>"
	}
	return out
}

func ProgramJokeHandler(m *tg.NewMessage) error {
	data, err := fetchJokeAPI("https://v2.jokeapi.dev/joke/Programming?type=twopart")
	if err != nil {
		m.Reply("couldn't fetch programming joke: " + err.Error())
		return nil
	}
	if data.Setup == "" || data.Delivery == "" {
		m.Reply("couldn't fetch programming joke: empty response")
		return nil
	}
	m.Reply(formatJokeAPI(data, "Programming Joke"))
	return nil
}

func AnyJokeHandler(m *tg.NewMessage) error {
	data, err := fetchJokeAPI("https://v2.jokeapi.dev/joke/Any")
	if err != nil {
		m.Reply("couldn't fetch joke: " + err.Error())
		return nil
	}
	if data.Type == "twopart" && (data.Setup == "" || data.Delivery == "") {
		m.Reply("couldn't fetch joke: empty response")
		return nil
	}
	if data.Type == "single" && data.Joke == "" {
		m.Reply("couldn't fetch joke: empty response")
		return nil
	}
	m.Reply(formatJokeAPI(data, "Joke"))
	return nil
}

func init() { QueueHandlerRegistration(registerJokesAPIHandlers) }
func registerJokesAPIHandlers() {
	c := Client
	c.On("cmd:programjoke", ProgramJokeHandler)
	c.On("cmd:anyjoke", AnyJokeHandler)
}
