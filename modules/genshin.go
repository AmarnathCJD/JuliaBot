package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type genshinAPIResp struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Vision      string `json:"vision"`
	Weapon      string `json:"weapon"`
	Gender      string `json:"gender"`
	Nation      string `json:"nation"`
	Affiliation string `json:"affiliation"`
	Rarity      int    `json:"rarity"`
	Release     string `json:"release"`
	Birthday    string `json:"birthday"`
	Description string `json:"description"`
}

type genshinCacheEntry struct {
	data    *genshinAPIResp
	expires time.Time
}

var (
	genshinCacheMu sync.Mutex
	genshinCache   = map[string]genshinCacheEntry{}
)

const genshinCacheTTL = 1 * time.Hour

func genshinSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func genshinFetch(query string) (*genshinAPIResp, int, error) {
	key := genshinSlug(query)
	genshinCacheMu.Lock()
	if e, ok := genshinCache[key]; ok && time.Now().Before(e.expires) {
		genshinCacheMu.Unlock()
		return e.data, 200, nil
	}
	genshinCacheMu.Unlock()

	endpoint := "https://genshin.jmp.blue/characters/" + key
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, nil
	}
	var g genshinAPIResp
	if jerr := json.Unmarshal(body, &g); jerr != nil {
		return nil, resp.StatusCode, jerr
	}
	genshinCacheMu.Lock()
	genshinCache[key] = genshinCacheEntry{data: &g, expires: time.Now().Add(genshinCacheTTL)}
	genshinCacheMu.Unlock()
	return &g, resp.StatusCode, nil
}

func genshinDownloadIcon(slug string) ([]byte, error) {
	iconURL := "https://genshin.jmp.blue/characters/" + slug + "/icon-big"
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", iconURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("icon http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func genshinRarityStars(r int) string {
	if r <= 0 {
		return "—"
	}
	if r > 6 {
		r = 6
	}
	return strings.Repeat("⭐", r)
}

func genshinTitleCase(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func genshinFallback(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func GenshinHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/genshin &lt;character&gt;</code>\n<b>Example:</b> <code>/genshin raiden</code>, <code>/genshin hu tao</code>")
		return nil
	}

	status, _ := m.Reply("<i>Fetching character data...</i>")

	g, code, err := genshinFetch(arg)
	if err != nil {
		if status != nil {
			status.Edit("<b>Request failed.</b> Could not reach genshin.jmp.blue.")
		}
		return nil
	}
	if code == 404 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>Character not found:</b> <code>%s</code>", html.EscapeString(arg)))
		}
		return nil
	}
	if code != 200 || g == nil {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", code))
		}
		return nil
	}

	name := genshinFallback(g.Name)
	if name == "—" {
		name = genshinTitleCase(genshinSlug(arg))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("✨ <b>%s</b>\n", html.EscapeString(name)))
	if strings.TrimSpace(g.Title) != "" {
		b.WriteString(fmt.Sprintf("<i>%s</i>\n", html.EscapeString(g.Title)))
	}
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("<b>Rarity:</b> %s\n", genshinRarityStars(g.Rarity)))
	b.WriteString(fmt.Sprintf("<b>Vision:</b> <code>%s</code>\n", html.EscapeString(genshinFallback(g.Vision))))
	b.WriteString(fmt.Sprintf("<b>Weapon:</b> <code>%s</code>\n", html.EscapeString(genshinFallback(g.Weapon))))
	b.WriteString(fmt.Sprintf("<b>Nation:</b> <code>%s</code>\n", html.EscapeString(genshinFallback(g.Nation))))
	b.WriteString(fmt.Sprintf("<b>Affiliation:</b> <code>%s</code>\n", html.EscapeString(genshinFallback(g.Affiliation))))
	if strings.TrimSpace(g.Description) != "" {
		b.WriteString("\n<b>Description:</b>\n")
		b.WriteString(html.EscapeString(g.Description))
		b.WriteString("\n")
	}
	b.WriteString("\n<i>Source: genshin.jmp.blue</i>")

	caption := b.String()

	slug := genshinSlug(arg)
	imgBytes, ierr := genshinDownloadIcon(slug)
	if ierr == nil && len(imgBytes) > 0 {
		if status != nil {
			status.Delete()
		}
		if _, merr := m.ReplyMedia(imgBytes, &tg.MediaOptions{
			Caption:  caption,
			FileName: fmt.Sprintf("%s.webp", slug),
			MimeType: "image/webp",
		}); merr == nil {
			return nil
		}
	}

	if status != nil {
		status.Edit(caption)
	} else {
		m.Reply(caption)
	}
	return nil
}

func registerGenshinHandlers() {
	c := Client
	c.On("cmd:genshin", GenshinHandler)
}

func init() {
	QueueHandlerRegistration(registerGenshinHandlers)
}
