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

type lyricsRecovOvhPayload struct {
	Lyrics string `json:"lyrics"`
	Error  string `json:"error"`
}

func LyricsHandler(m *tg.NewMessage) error {
	q := strings.TrimSpace(m.Args())
	if q == "" {
		m.Reply("<b>Usage:</b> <code>/lyrics &lt;artist&gt; - &lt;title&gt;</code>")
		return nil
	}
	parts := strings.SplitN(q, " - ", 2)
	if len(parts) != 2 {
		m.Reply("<b>Usage:</b> <code>/lyrics &lt;artist&gt; - &lt;title&gt;</code>")
		return nil
	}
	artist := strings.TrimSpace(parts[0])
	title := strings.TrimSpace(parts[1])
	if artist == "" || title == "" {
		m.Reply("<b>Usage:</b> <code>/lyrics &lt;artist&gt; - &lt;title&gt;</code>")
		return nil
	}
	endpoint := "https://api.lyrics.ovh/v1/" + url.PathEscape(artist) + "/" + url.PathEscape(title)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		m.Reply("couldn't fetch lyrics: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("lyrics not found (HTTP %d)", resp.StatusCode))
		return nil
	}
	var data lyricsRecovOvhPayload
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't parse lyrics response: " + html.EscapeString(err.Error()))
		return nil
	}
	if data.Error != "" {
		m.Reply("lyrics not found: " + html.EscapeString(data.Error))
		return nil
	}
	lyrics := strings.TrimSpace(data.Lyrics)
	if lyrics == "" {
		m.Reply("no lyrics found for that song.")
		return nil
	}
	if len(lyrics) > 4000 {
		lyrics = lyrics[:4000]
	}
	out := "<b>" + html.EscapeString(artist) + " - " + html.EscapeString(title) + "</b>\n\n<blockquote>" + html.EscapeString(lyrics) + "</blockquote>"
	m.Reply(out)
	return nil
}

func init() { QueueHandlerRegistration(registerLyricsHandlers) }
func registerLyricsHandlers() {
	c := Client
	c.On("cmd:lyrics", LyricsHandler)
}
