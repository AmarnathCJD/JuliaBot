package extras

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

var jikanClient = &http.Client{Timeout: 30 * time.Second}

type jikanImage struct {
	ImageURL string `json:"image_url"`
	LargeURL string `json:"large_image_url"`
	SmallURL string `json:"small_image_url"`
}

type jikanImages struct {
	JPG  jikanImage `json:"jpg"`
	WebP jikanImage `json:"webp"`
}

type jikanEntry struct {
	MalID    int         `json:"mal_id"`
	URL      string      `json:"url"`
	Images   jikanImages `json:"images"`
	Title    string      `json:"title"`
	TitleEng string      `json:"title_english"`
	Type     string      `json:"type"`
	Episodes int         `json:"episodes"`
	Chapters int         `json:"chapters"`
	Volumes  int         `json:"volumes"`
	Status   string      `json:"status"`
	Score    float64     `json:"score"`
	Synopsis string      `json:"synopsis"`
}

type jikanResponse struct {
	Data []jikanEntry `json:"data"`
}

func fetchJikan(endpoint, query string) (*jikanEntry, error) {
	apiURL := fmt.Sprintf("https://api.jikan.moe/v4/%s?q=%s&limit=1", endpoint, url.QueryEscape(query))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := jikanClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data jikanResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if len(data.Data) == 0 {
		return nil, nil
	}
	return &data.Data[0], nil
}

func downloadJikanImage(imgURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", imgURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func truncateJikan(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := n
	for cut > 0 && r[cut] != ' ' {
		cut--
	}
	if cut == 0 {
		cut = n
	}
	return string(r[:cut]) + "..."
}

func buildJikanCaption(e *jikanEntry, kind string) string {
	title := strings.TrimSpace(e.Title)
	if title == "" {
		title = strings.TrimSpace(e.TitleEng)
	}
	if title == "" {
		title = "Unknown"
	}
	synopsis := strings.TrimSpace(e.Synopsis)
	if synopsis == "" {
		synopsis = "No synopsis available."
	}
	synopsis = truncateJikan(synopsis, 700)

	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</b>\n\n")
	if strings.TrimSpace(e.Type) != "" {
		b.WriteString("<b>Type:</b> ")
		b.WriteString(html.EscapeString(e.Type))
		b.WriteString("\n")
	}
	if kind == "anime" {
		if e.Episodes > 0 {
			b.WriteString(fmt.Sprintf("<b>Episodes:</b> %d\n", e.Episodes))
		} else {
			b.WriteString("<b>Episodes:</b> N/A\n")
		}
	} else {
		if e.Chapters > 0 {
			b.WriteString(fmt.Sprintf("<b>Chapters:</b> %d\n", e.Chapters))
		} else {
			b.WriteString("<b>Chapters:</b> N/A\n")
		}
		if e.Volumes > 0 {
			b.WriteString(fmt.Sprintf("<b>Volumes:</b> %d\n", e.Volumes))
		}
	}
	if strings.TrimSpace(e.Status) != "" {
		b.WriteString("<b>Status:</b> ")
		b.WriteString(html.EscapeString(e.Status))
		b.WriteString("\n")
	}
	if e.Score > 0 {
		b.WriteString(fmt.Sprintf("<b>Score:</b> %.2f\n", e.Score))
	} else {
		b.WriteString("<b>Score:</b> N/A\n")
	}
	b.WriteString("\n<b>Synopsis:</b>\n<blockquote>")
	b.WriteString(html.EscapeString(synopsis))
	b.WriteString("</blockquote>\n")
	if strings.TrimSpace(e.URL) != "" {
		b.WriteString("\n<a href=\"")
		b.WriteString(html.EscapeString(e.URL))
		b.WriteString("\">View on MyAnimeList</a>")
	}
	return b.String()
}

func handleJikanLookup(m *tg.NewMessage, kind string) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply(fmt.Sprintf("Usage: <code>/%s &lt;title&gt;</code>", kind))
		return nil
	}
	status, _ := m.Reply(fmt.Sprintf("Searching %s...", kind))

	entry, err := fetchJikan(kind, query)
	if err != nil {
		msg := "<b>Error:</b> failed to fetch " + kind + " info."
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if entry == nil {
		msg := "<b>No results for:</b> " + html.EscapeString(query)
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	caption := buildJikanCaption(entry, kind)
	imgURL := strings.TrimSpace(entry.Images.JPG.LargeURL)
	if imgURL == "" {
		imgURL = strings.TrimSpace(entry.Images.JPG.ImageURL)
	}
	if imgURL == "" {
		imgURL = strings.TrimSpace(entry.Images.JPG.SmallURL)
	}

	if imgURL != "" {
		imgBytes, ierr := downloadJikanImage(imgURL)
		if ierr == nil && len(imgBytes) > 0 {
			if status != nil {
				status.Delete()
			}
			m.ReplyMedia(imgBytes, &tg.MediaOptions{
				Caption:  caption,
				FileName: kind + ".jpg",
				MimeType: "image/jpeg",
			})
			return nil
		}
	}

	if status != nil {
		status.Edit(caption, &tg.SendOptions{LinkPreview: true})
	} else {
		m.Reply(caption, &tg.SendOptions{LinkPreview: true})
	}
	return nil
}

func AnimeHandler(m *tg.NewMessage) error {
	return handleJikanLookup(m, "anime")
}

func MangaHandler(m *tg.NewMessage) error {
	return handleJikanLookup(m, "manga")
}

func registerAnimeHandlers() {
	c := modules.Client
	c.On("cmd:anime", AnimeHandler)
	c.On("cmd:manga", MangaHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerAnimeHandlers)
}
