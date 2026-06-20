package modules

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func WallpaperHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("usage: <code>/wallpaper &lt;query&gt;</code> or <code>/wallpaper random</code>")
		return nil
	}

	var endpoint string
	var label string
	if strings.EqualFold(query, "random") {
		endpoint = fmt.Sprintf("https://picsum.photos/1280/720?rand=%d", time.Now().UnixNano())
		label = "random"
	} else {
		endpoint = "https://picsum.photos/seed/" + url.PathEscape(query) + "/1280/720"
		label = query
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		m.Reply("error: " + html.EscapeString(err.Error()))
		return nil
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.Reply("error fetching wallpaper: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.Reply(fmt.Sprintf("wallpaper api returned status %d", resp.StatusCode))
		return nil
	}

	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("wallpaper_%d.jpg", time.Now().UnixNano()))
	out, err := os.Create(tmpPath)
	if err != nil {
		m.Reply("error creating temp file: " + html.EscapeString(err.Error()))
		return nil
	}

	if _, err := io.Copy(out, io.LimitReader(resp.Body, 15*1024*1024)); err != nil {
		out.Close()
		os.Remove(tmpPath)
		m.Reply("error writing wallpaper: " + html.EscapeString(err.Error()))
		return nil
	}
	out.Close()
	defer os.Remove(tmpPath)

	caption := "<b>Wallpaper</b>\n<code>" + html.EscapeString(label) + "</code>"
	_, err = m.ReplyMedia(tmpPath, &tg.MediaOptions{Caption: caption})
	if err != nil {
		m.Reply("error sending wallpaper: " + html.EscapeString(err.Error()))
		return nil
	}
	return nil
}

func init() { QueueHandlerRegistration(registerWallpaperHandlers) }

func registerWallpaperHandlers() {
	c := Client
	c.On("cmd:wallpaper", WallpaperHandler)
}
