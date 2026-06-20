package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type itunesTrack struct {
	ArtistName       string  `json:"artistName"`
	CollectionName   string  `json:"collectionName"`
	TrackName        string  `json:"trackName"`
	TrackViewURL     string  `json:"trackViewUrl"`
	PreviewURL       string  `json:"previewUrl"`
	ArtworkURL100    string  `json:"artworkUrl100"`
	ReleaseDate      string  `json:"releaseDate"`
	PrimaryGenreName string  `json:"primaryGenreName"`
	TrackTimeMillis  int     `json:"trackTimeMillis"`
	TrackPrice       float64 `json:"trackPrice"`
	Currency         string  `json:"currency"`
	Country          string  `json:"country"`
}

type itunesSearchResponse struct {
	ResultCount int           `json:"resultCount"`
	Results     []itunesTrack `json:"results"`
}

type spotifyLyricsOvhResponse struct {
	Lyrics string `json:"lyrics"`
	Error  string `json:"error"`
}

func formatTrackDuration(ms int) string {
	if ms <= 0 {
		return ""
	}
	total := ms / 1000
	mins := total / 60
	secs := total % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}

func upgradeArtwork(u string) string {
	if u == "" {
		return u
	}
	return strings.Replace(u, "/100x100bb.jpg", "/600x600bb.jpg", 1)
}

func fetchLyricsSnippet(artist, title string) string {
	client := &http.Client{Timeout: 12 * time.Second}
	endpoint := "https://api.lyrics.ovh/v1/" + url.PathEscape(artist) + "/" + url.PathEscape(title)
	resp, err := client.Get(endpoint)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	var data spotifyLyricsOvhResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	lyrics := strings.TrimSpace(data.Lyrics)
	if lyrics == "" {
		return ""
	}
	lines := strings.Split(lyrics, "\n")
	var picked []string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		picked = append(picked, ln)
		if len(picked) >= 6 {
			break
		}
	}
	if len(picked) == 0 {
		return ""
	}
	return strings.Join(picked, "\n")
}

func SpotifyMetaHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("<b>Usage:</b> <code>/spotify &lt;track query&gt;</code>\n<i>Example:</i> <code>/spotify imagine dragons believer</code>")
		return nil
	}
	status, _ := m.Reply("Searching <code>" + html.EscapeString(query) + "</code>...")
	client := &http.Client{Timeout: 20 * time.Second}
	endpoint := "https://itunes.apple.com/search?media=music&entity=song&limit=1&term=" + url.QueryEscape(query)
	resp, err := client.Get(endpoint)
	if err != nil {
		status.Edit("couldn't search track: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		status.Edit(fmt.Sprintf("iTunes API HTTP %d", resp.StatusCode))
		return nil
	}
	var data itunesSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		status.Edit("couldn't parse response: " + html.EscapeString(err.Error()))
		return nil
	}
	if data.ResultCount == 0 || len(data.Results) == 0 {
		status.Edit("<b>No track found for:</b> <code>" + html.EscapeString(query) + "</code>")
		return nil
	}
	track := data.Results[0]

	var lyrics string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		lyrics = fetchLyricsSnippet(track.ArtistName, track.TrackName)
	}()
	wg.Wait()

	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(track.TrackName))
	b.WriteString("</b>\n")
	b.WriteString("<i>by </i><b>")
	b.WriteString(html.EscapeString(track.ArtistName))
	b.WriteString("</b>")
	if strings.TrimSpace(track.CollectionName) != "" {
		b.WriteString("\n<b>Album:</b> ")
		b.WriteString(html.EscapeString(track.CollectionName))
	}
	if strings.TrimSpace(track.PrimaryGenreName) != "" {
		b.WriteString("\n<b>Genre:</b> ")
		b.WriteString(html.EscapeString(track.PrimaryGenreName))
	}
	if dur := formatTrackDuration(track.TrackTimeMillis); dur != "" {
		b.WriteString("\n<b>Duration:</b> ")
		b.WriteString(dur)
	}
	if len(track.ReleaseDate) >= 10 {
		b.WriteString("\n<b>Released:</b> ")
		b.WriteString(html.EscapeString(track.ReleaseDate[:10]))
	}
	if track.TrackPrice > 0 && strings.TrimSpace(track.Currency) != "" {
		b.WriteString(fmt.Sprintf("\n<b>Price:</b> %.2f %s", track.TrackPrice, html.EscapeString(track.Currency)))
	}
	b.WriteString("\n")
	if strings.TrimSpace(track.PreviewURL) != "" {
		b.WriteString("\n<a href=\"")
		b.WriteString(html.EscapeString(track.PreviewURL))
		b.WriteString("\">Preview</a>")
	}
	if strings.TrimSpace(track.TrackViewURL) != "" {
		b.WriteString(" | <a href=\"")
		b.WriteString(html.EscapeString(track.TrackViewURL))
		b.WriteString("\">Apple Music</a>")
	}
	spotifySearch := "https://open.spotify.com/search/" + url.PathEscape(track.ArtistName+" "+track.TrackName)
	b.WriteString(" | <a href=\"")
	b.WriteString(html.EscapeString(spotifySearch))
	b.WriteString("\">Spotify Search</a>")
	if lyrics != "" {
		b.WriteString("\n\n<b>Lyrics snippet:</b>\n<blockquote expandable>")
		b.WriteString(html.EscapeString(lyrics))
		b.WriteString("</blockquote>")
	}

	caption := b.String()
	artwork := upgradeArtwork(track.ArtworkURL100)
	if strings.TrimSpace(artwork) != "" {
		if _, err := m.ReplyMedia(artwork, &tg.MediaOptions{Caption: caption}); err != nil {
			status.Edit(caption, &tg.SendOptions{LinkPreview: false})
			return nil
		}
		status.Delete()
		return nil
	}
	status.Edit(caption, &tg.SendOptions{LinkPreview: false})
	return nil
}

func init() { QueueHandlerRegistration(registerSpotifyMetaHandlers) }
func registerSpotifyMetaHandlers() {
	c := Client
	c.On("cmd:spotify", SpotifyMetaHandler)
}
