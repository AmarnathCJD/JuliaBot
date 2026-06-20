package modules

import (
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func aiImgGenFetch(prompt string, seed int) ([]byte, error) {
	endpoint := fmt.Sprintf(
		"https://image.pollinations.ai/prompt/%s?width=1024&height=1024&seed=%d&nologo=true",
		url.PathEscape(prompt), seed,
	)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return nil, fmt.Errorf("unexpected content-type: %s", ct)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty image response")
	}
	return data, nil
}

func AIImgGenHandler(m *tg.NewMessage) error {
	prompt := strings.TrimSpace(m.Args())
	if prompt == "" {
		m.Reply("usage: <code>/aiimg &lt;prompt&gt;</code>\nexample: <code>/aiimg a cat astronaut on mars</code>")
		return nil
	}
	if len(prompt) > 900 {
		m.Reply("prompt too long, max 900 characters")
		return nil
	}

	status, _ := m.Reply("<code>generating image...</code>")

	seed := rand.Intn(1000000)
	img, err := aiImgGenFetch(prompt, seed)
	if err != nil {
		msg := "error generating image: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("aiimg_%d.jpg", time.Now().UnixNano()))
	if werr := os.WriteFile(tmp, img, 0644); werr != nil {
		msg := "error saving image: " + html.EscapeString(werr.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(tmp)

	preview := prompt
	if len(preview) > 200 {
		preview = preview[:197] + "..."
	}
	caption := fmt.Sprintf("<b>AI Image</b>\n<b>Prompt:</b> <code>%s</code>\n<b>Seed:</b> <code>%d</code>",
		html.EscapeString(preview), seed)

	_, merr := m.ReplyMedia(tmp, &tg.MediaOptions{
		Caption:  caption,
		FileName: "aiimg.jpg",
		MimeType: "image/jpeg",
	})
	if merr != nil {
		msg := "upload failed: " + html.EscapeString(merr.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if status != nil {
		status.Delete()
	}
	return nil
}

func init() { QueueHandlerRegistration(registerAIImgGenHandlers) }

func registerAIImgGenHandlers() {
	c := Client
	c.On("cmd:aiimg", AIImgGenHandler)
}
