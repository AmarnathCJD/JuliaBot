package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type officialJokeResponse struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Setup     string `json:"setup"`
	Punchline string `json:"punchline"`
}

type dadJokeResponse struct {
	ID     string `json:"id"`
	Joke   string `json:"joke"`
	Status int    `json:"status"`
}

type chuckJokeResponse struct {
	Categories []string `json:"categories"`
	IconURL    string   `json:"icon_url"`
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	Value      string   `json:"value"`
}

func JokeHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://official-joke-api.appspot.com/jokes/random")
	if err != nil {
		m.Reply("couldn't fetch joke: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data officialJokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't fetch joke: " + err.Error())
		return nil
	}
	if data.Setup == "" && data.Punchline == "" {
		m.Reply("couldn't fetch joke: empty response")
		return nil
	}
	out := "<b>Joke</b>\n\n" + html.EscapeString(data.Setup) + "\n\n<i>" + html.EscapeString(data.Punchline) + "</i>"
	if data.Type != "" {
		out += "\n\n<b>Type:</b> " + html.EscapeString(data.Type)
	}
	m.Reply(out)
	return nil
}

func DadJokeHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", "https://icanhazdadjoke.com/", nil)
	if err != nil {
		m.Reply("couldn't fetch dad joke: " + err.Error())
		return nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "JuliaBot (https://github.com/amarnathcjd)")
	resp, err := client.Do(req)
	if err != nil {
		m.Reply("couldn't fetch dad joke: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data dadJokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't fetch dad joke: " + err.Error())
		return nil
	}
	if data.Joke == "" {
		m.Reply("couldn't fetch dad joke: empty response")
		return nil
	}
	m.Reply("<b>Dad Joke</b>\n\n" + html.EscapeString(data.Joke))
	return nil
}

func ChuckHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://api.chucknorris.io/jokes/random")
	if err != nil {
		m.Reply("couldn't fetch chuck joke: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data chuckJokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't fetch chuck joke: " + err.Error())
		return nil
	}
	if data.Value == "" {
		m.Reply("couldn't fetch chuck joke: empty response")
		return nil
	}
	m.Reply("<b>Chuck Norris</b>\n\n" + html.EscapeString(data.Value))
	return nil
}

func init() { QueueHandlerRegistration(registerJokesV2Handlers) }
func registerJokesV2Handlers() {
	c := Client
	c.On("cmd:joke", JokeHandler)
	c.On("cmd:dadjoke", DadJokeHandler)
	c.On("cmd:chuck", ChuckHandler)
}
