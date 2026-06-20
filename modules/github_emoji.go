package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	ghEmojiMu      sync.RWMutex
	ghEmojiCache   map[string]string
	ghEmojiFetched time.Time
)

func loadGithubEmojis() (map[string]string, error) {
	ghEmojiMu.RLock()
	if ghEmojiCache != nil && time.Since(ghEmojiFetched) < 24*time.Hour {
		c := ghEmojiCache
		ghEmojiMu.RUnlock()
		return c, nil
	}
	ghEmojiMu.RUnlock()

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/emojis", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "JuliaBot")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	ghEmojiMu.Lock()
	ghEmojiCache = data
	ghEmojiFetched = time.Now()
	ghEmojiMu.Unlock()
	return data, nil
}

func downloadGithubEmojiImage(url string) ([]byte, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "JuliaBot")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "image/png"
	}
	return body, mime, nil
}

func GitHubEmojiHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/ghemoji &lt;name&gt;</code>")
		return nil
	}
	name := strings.ToLower(strings.TrimSpace(strings.Trim(arg, ":")))
	if name == "" {
		m.Reply("<b>Usage:</b> <code>/ghemoji &lt;name&gt;</code>")
		return nil
	}

	emojis, err := loadGithubEmojis()
	if err != nil {
		m.Reply("Failed to fetch GitHub emojis. Try again later.")
		return nil
	}

	url, ok := emojis[name]
	if !ok {
		var suggestions []string
		for k := range emojis {
			if strings.Contains(k, name) {
				suggestions = append(suggestions, k)
				if len(suggestions) >= 8 {
					break
				}
			}
		}
		if len(suggestions) > 0 {
			var b strings.Builder
			for i, s := range suggestions {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString("<code>")
				b.WriteString(html.EscapeString(s))
				b.WriteString("</code>")
			}
			m.Reply(fmt.Sprintf("<b>No emoji found for:</b> <code>%s</code>\n<b>Did you mean:</b> %s", html.EscapeString(name), b.String()))
			return nil
		}
		m.Reply(fmt.Sprintf("<b>No GitHub emoji found for:</b> <code>%s</code>", html.EscapeString(name)))
		return nil
	}

	caption := fmt.Sprintf("<b>:%s:</b>\n<a href=\"%s\">source</a>", html.EscapeString(name), html.EscapeString(url))

	imgBytes, mime, derr := downloadGithubEmojiImage(url)
	if derr != nil || len(imgBytes) == 0 {
		m.Reply(caption, &tg.SendOptions{LinkPreview: true})
		return nil
	}

	ext := strings.ToLower(path.Ext(strings.SplitN(url, "?", 2)[0]))
	if ext == "" {
		ext = ".png"
	}
	fileName := name + ext

	if _, err := m.ReplyMedia(imgBytes, &tg.MediaOptions{
		Caption:  caption,
		FileName: fileName,
		MimeType: mime,
	}); err != nil {
		m.Reply(caption, &tg.SendOptions{LinkPreview: true})
	}
	return nil
}

func init() { QueueHandlerRegistration(registerGitHubEmojiHandlers) }
func registerGitHubEmojiHandlers() {
	c := Client
	c.On("cmd:ghemoji", GitHubEmojiHandler)
}
