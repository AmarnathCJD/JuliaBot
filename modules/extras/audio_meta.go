package extras

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

type iTunesSongResult struct {
	ArtistName       string `json:"artistName"`
	TrackName        string `json:"trackName"`
	CollectionName   string `json:"collectionName"`
	PrimaryGenreName string `json:"primaryGenreName"`
	ReleaseDate      string `json:"releaseDate"`
	PreviewURL       string `json:"previewUrl"`
	ArtworkURL100    string `json:"artworkUrl100"`
	TrackViewURL     string `json:"trackViewUrl"`
	TrackTimeMillis  int    `json:"trackTimeMillis"`
	Country          string `json:"country"`
}

type iTunesSearchResponse struct {
	ResultCount int                `json:"resultCount"`
	Results     []iTunesSongResult `json:"results"`
}

func MusicInfoHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("Usage: <code>/musicinfo &lt;song name&gt;</code>")
		return nil
	}
	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := "https://itunes.apple.com/search?term=" + url.QueryEscape(query) + "&entity=song&limit=1"
	resp, err := client.Get(endpoint)
	if err != nil {
		m.Reply("Couldn't reach iTunes: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("iTunes API returned HTTP %d", resp.StatusCode))
		return nil
	}
	var data iTunesSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse iTunes response.")
		return nil
	}
	if data.ResultCount == 0 || len(data.Results) == 0 {
		m.Reply("No song found for <b>" + html.EscapeString(query) + "</b>")
		return nil
	}
	r := data.Results[0]
	releaseDate := r.ReleaseDate
	if len(releaseDate) >= 10 {
		releaseDate = releaseDate[:10]
	}
	duration := ""
	if r.TrackTimeMillis > 0 {
		secs := r.TrackTimeMillis / 1000
		duration = fmt.Sprintf("%d:%02d", secs/60, secs%60)
	}
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(r.TrackName))
	b.WriteString("</b>\n")
	if r.ArtistName != "" {
		b.WriteString("<b>Artist:</b> ")
		b.WriteString(html.EscapeString(r.ArtistName))
		b.WriteString("\n")
	}
	if r.CollectionName != "" {
		b.WriteString("<b>Album:</b> ")
		b.WriteString(html.EscapeString(r.CollectionName))
		b.WriteString("\n")
	}
	if r.PrimaryGenreName != "" {
		b.WriteString("<b>Genre:</b> ")
		b.WriteString(html.EscapeString(r.PrimaryGenreName))
		b.WriteString("\n")
	}
	if releaseDate != "" {
		b.WriteString("<b>Released:</b> ")
		b.WriteString(html.EscapeString(releaseDate))
		b.WriteString("\n")
	}
	if duration != "" {
		b.WriteString("<b>Duration:</b> ")
		b.WriteString(duration)
		b.WriteString("\n")
	}
	if r.PreviewURL != "" {
		b.WriteString("<a href=\"")
		b.WriteString(html.EscapeString(r.PreviewURL))
		b.WriteString("\">Preview</a>")
	}
	if r.TrackViewURL != "" {
		if r.PreviewURL != "" {
			b.WriteString(" | ")
		}
		b.WriteString("<a href=\"")
		b.WriteString(html.EscapeString(r.TrackViewURL))
		b.WriteString("\">Apple Music</a>")
	}
	caption := b.String()
	artwork := r.ArtworkURL100
	if artwork != "" {
		artwork = strings.Replace(artwork, "100x100bb.jpg", "600x600bb.jpg", 1)
	}
	if artwork != "" {
		if _, err := m.ReplyMedia(artwork, &tg.MediaOptions{Caption: caption}); err == nil {
			return nil
		}
	}
	m.Reply(caption)
	return nil
}

func init() { modules.QueueHandlerRegistration(registerAudioMetaHandlers) }
func registerAudioMetaHandlers() {
	c := modules.Client
	c.On("cmd:musicinfo", MusicInfoHandler)
}
