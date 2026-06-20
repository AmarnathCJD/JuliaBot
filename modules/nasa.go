package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type nasaApodResponse struct {
	Date           string `json:"date"`
	Title          string `json:"title"`
	Explanation    string `json:"explanation"`
	URL            string `json:"url"`
	HDURL          string `json:"hdurl"`
	MediaType      string `json:"media_type"`
	ThumbnailURL   string `json:"thumbnail_url"`
	Copyright      string `json:"copyright"`
	ServiceVersion string `json:"service_version"`
}

type nasaCacheEntry struct {
	data      nasaApodResponse
	expiresAt time.Time
	negative  bool
}

var (
	nasaCacheMu sync.Mutex
	nasaCache   *nasaCacheEntry
)

func nasaGetAPIKey() string {
	if k := strings.TrimSpace(os.Getenv("NASA_API_KEY")); k != "" {
		return k
	}
	return "DEMO_KEY"
}

func nasaYouTubeID(u string) string {
	low := strings.ToLower(u)
	if !strings.Contains(low, "youtube.com") && !strings.Contains(low, "youtu.be") {
		return ""
	}
	if i := strings.Index(u, "youtu.be/"); i != -1 {
		id := u[i+len("youtu.be/"):]
		if j := strings.IndexAny(id, "?&/"); j != -1 {
			id = id[:j]
		}
		return id
	}
	if i := strings.Index(u, "/embed/"); i != -1 {
		id := u[i+len("/embed/"):]
		if j := strings.IndexAny(id, "?&/"); j != -1 {
			id = id[:j]
		}
		return id
	}
	if i := strings.Index(u, "v="); i != -1 {
		id := u[i+2:]
		if j := strings.IndexAny(id, "&#"); j != -1 {
			id = id[:j]
		}
		return id
	}
	return ""
}

func nasaFormatCaption(d nasaApodResponse) string {
	out := "<b>NASA Astronomy Picture of the Day</b>\n\n"
	if d.Title != "" {
		out += "<b>Title:</b> " + html.EscapeString(d.Title) + "\n"
	}
	if d.Date != "" {
		out += "<b>Date:</b> " + html.EscapeString(d.Date) + "\n"
	}
	if d.Copyright != "" {
		out += "<b>Copyright:</b> " + html.EscapeString(strings.TrimSpace(d.Copyright)) + "\n"
	}
	if d.MediaType != "" {
		out += "<b>Type:</b> " + html.EscapeString(d.MediaType) + "\n"
	}
	exp := strings.TrimSpace(d.Explanation)
	if exp != "" {
		if len(exp) > 700 {
			exp = exp[:700] + "..."
		}
		out += "\n" + html.EscapeString(exp) + "\n"
	}
	if d.URL != "" {
		out += "\n<a href=\"" + html.EscapeString(d.URL) + "\">Source</a>"
	}
	return out
}

func nasaFetchAPOD() (nasaApodResponse, int, error) {
	key := nasaGetAPIKey()
	url := "https://api.nasa.gov/planetary/apod?thumbs=true&api_key=" + key
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nasaApodResponse{}, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nasaApodResponse{}, resp.StatusCode, fmt.Errorf("status %d", resp.StatusCode)
	}
	var data nasaApodResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nasaApodResponse{}, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

func nasaGetCached() (nasaApodResponse, bool, bool) {
	nasaCacheMu.Lock()
	defer nasaCacheMu.Unlock()
	if nasaCache == nil {
		return nasaApodResponse{}, false, false
	}
	if time.Now().After(nasaCache.expiresAt) {
		return nasaApodResponse{}, false, false
	}
	return nasaCache.data, true, nasaCache.negative
}

func nasaSetCache(d nasaApodResponse, ttl time.Duration, negative bool) {
	nasaCacheMu.Lock()
	defer nasaCacheMu.Unlock()
	nasaCache = &nasaCacheEntry{data: d, expiresAt: time.Now().Add(ttl), negative: negative}
}

func ApodHandler(m *tg.NewMessage) error {
	if cached, ok, neg := nasaGetCached(); ok {
		if neg {
			m.Reply("NASA APOD is rate limited. Please try again later.")
			return nil
		}
		sendAPOD(m, cached)
		return nil
	}
	data, status, err := nasaFetchAPOD()
	if err != nil {
		if status == 429 {
			nasaSetCache(nasaApodResponse{}, 30*time.Minute, true)
			m.Reply("NASA APOD is rate limited. Please try again later.")
			return nil
		}
		m.Reply("Failed to fetch NASA APOD. Try again later.")
		return nil
	}
	if data.Title == "" && data.URL == "" {
		m.Reply("NASA APOD returned no data.")
		return nil
	}
	nasaSetCache(data, 1*time.Hour, false)
	sendAPOD(m, data)
	return nil
}

func sendAPOD(m *tg.NewMessage, data nasaApodResponse) {
	caption := nasaFormatCaption(data)
	if strings.EqualFold(data.MediaType, "video") {
		thumb := data.ThumbnailURL
		if thumb == "" {
			if id := nasaYouTubeID(data.URL); id != "" {
				thumb = "https://img.youtube.com/vi/" + id + "/hqdefault.jpg"
			}
		}
		if thumb != "" {
			if _, err := m.ReplyMedia(thumb, &tg.MediaOptions{Caption: caption}); err == nil {
				return
			}
		}
		link := data.URL
		title := data.Title
		if title == "" {
			title = "Video"
		}
		m.Reply(caption + "\n\n<a href=\"" + html.EscapeString(link) + "\">" + html.EscapeString(title) + "</a>")
		return
	}
	candidates := []string{}
	if data.HDURL != "" {
		candidates = append(candidates, data.HDURL)
	}
	if data.URL != "" {
		candidates = append(candidates, data.URL)
	}
	for _, c := range candidates {
		if _, err := m.ReplyMedia(c, &tg.MediaOptions{Caption: caption}); err == nil {
			return
		}
	}
	m.Reply(caption)
}

func init() { QueueHandlerRegistration(registerNasaHandlers) }
func registerNasaHandlers() {
	c := Client
	c.On("cmd:apod", ApodHandler)
}
