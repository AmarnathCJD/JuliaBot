package modules

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

const tenorAPIKey = "AIzaSyAyimkuYQYF_FXVALexPuGQctUWRURdCYQ"

type tenorMediaFormat struct {
	URL string `json:"url"`
}

type tenorResult struct {
	MediaFormats map[string]tenorMediaFormat `json:"media_formats"`
	ItemURL      string                      `json:"itemurl"`
	Title        string                      `json:"title"`
	ContentDesc  string                      `json:"content_description"`
}

type tenorResponse struct {
	Results []tenorResult `json:"results"`
	Error   string        `json:"error"`
}

func pickTenorGifURL(r tenorResult) string {
	keys := []string{"gif", "mediumgif", "tinygif", "nanogif"}
	for _, k := range keys {
		if v, ok := r.MediaFormats[k]; ok && v.URL != "" {
			return v.URL
		}
	}
	for _, v := range r.MediaFormats {
		if v.URL != "" {
			return v.URL
		}
	}
	return ""
}

func GifSearchHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("<b>GIF Search</b>\n\n<b>Usage:</b> <code>/gifs &lt;query&gt;</code>\nExample: <code>/gifs happy cat</code>")
		return nil
	}

	endpoint := "https://tenor.googleapis.com/v2/search?q=" + url.QueryEscape(query) + "&key=" + tenorAPIKey + "&limit=10&media_filter=gif&contentfilter=medium"

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		m.Reply("Failed to reach Tenor: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		m.Reply("Tenor returned status " + resp.Status)
		return nil
	}

	var data tenorResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse Tenor response: " + html.EscapeString(err.Error()))
		return nil
	}

	if data.Error != "" {
		m.Reply("Tenor error: " + html.EscapeString(data.Error))
		return nil
	}

	if len(data.Results) == 0 {
		m.Reply("No GIFs found for: <b>" + html.EscapeString(query) + "</b>")
		return nil
	}

	var gifURL string
	var picked tenorResult
	for _, r := range data.Results {
		if u := pickTenorGifURL(r); u != "" {
			gifURL = u
			picked = r
			break
		}
	}

	if gifURL == "" {
		m.Reply("No usable GIF URL found for: <b>" + html.EscapeString(query) + "</b>")
		return nil
	}

	caption := "<b>" + html.EscapeString(query) + "</b>"
	if picked.ContentDesc != "" {
		caption += "\n<i>" + html.EscapeString(picked.ContentDesc) + "</i>"
	} else if picked.Title != "" {
		caption += "\n<i>" + html.EscapeString(picked.Title) + "</i>"
	}

	_, merr := m.ReplyMedia(gifURL, &tg.MediaOptions{
		Caption:  caption,
		FileName: "tenor.gif",
		MimeType: "image/gif",
	})
	if merr != nil {
		m.Reply(caption + "\n\n" + gifURL)
	}
	return nil
}

func init() { QueueHandlerRegistration(registerGifSearchHandlers) }
func registerGifSearchHandlers() {
	c := Client
	c.On("cmd:gifs", GifSearchHandler)
	c.On("cmd:gify", GifSearchHandler)
}
