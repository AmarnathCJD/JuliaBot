package modules

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func httpStatusFetch(m *tg.NewMessage, urlTpl, label string) error {
	code := strings.TrimSpace(m.Args())
	if code == "" {
		m.Reply("Usage: provide an HTTP status code, e.g. <code>200</code>.")
		return nil
	}
	n, err := strconv.Atoi(code)
	if err != nil || n < 100 || n > 599 {
		m.Reply("Please provide a valid HTTP status code between 100 and 599.")
		return nil
	}
	url := strings.Replace(urlTpl, "{code}", strconv.Itoa(n), 1)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		m.Reply("Failed to build request.")
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		m.Reply("Failed to reach " + label + ". Try again later.")
		return nil
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("No image found for status <code>" + strconv.Itoa(n) + "</code>.")
		return nil
	}
	if _, err := m.ReplyMedia(url, &tg.MediaOptions{Caption: "HTTP " + strconv.Itoa(n)}); err != nil {
		m.Reply("<a href=\"" + url + "\">HTTP " + strconv.Itoa(n) + "</a>")
	}
	return nil
}

func HttpCatHandler(m *tg.NewMessage) error {
	return httpStatusFetch(m, "https://http.cat/{code}", "http.cat")
}

func HttpDogHandler(m *tg.NewMessage) error {
	return httpStatusFetch(m, "https://http.dog/{code}.jpg", "http.dog")
}

func init() { QueueHandlerRegistration(registerHttpStatusHandlers) }
func registerHttpStatusHandlers() {
	c := Client
	c.On("cmd:http", HttpCatHandler)
	c.On("cmd:httpdog", HttpDogHandler)
}
