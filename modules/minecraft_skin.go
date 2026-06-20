package modules

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var mcskinNameRe = regexp.MustCompile(`^[A-Za-z0-9_]{1,16}$`)

func McSkinHandler(m *tg.NewMessage) error {
	username := strings.TrimSpace(m.Args())
	if username == "" {
		m.Reply("usage: <code>/mcskin &lt;username&gt;</code>")
		return nil
	}
	username = strings.Fields(username)[0]
	if !mcskinNameRe.MatchString(username) {
		m.Reply("invalid minecraft username: <code>" + html.EscapeString(username) + "</code>")
		return nil
	}

	status, _ := m.Reply("fetching skin head for <code>" + html.EscapeString(username) + "</code>...")

	url := "https://mc-heads.net/avatar/" + username + "/256"
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		if status != nil {
			status.Edit("failed to fetch: " + html.EscapeString(err.Error()))
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if status != nil {
			status.Edit(fmt.Sprintf("mc-heads returned status <code>%d</code>", resp.StatusCode))
		}
		return nil
	}

	valid := strings.ToLower(resp.Header.Get("X-Account-Valid"))
	accountStatus := "unknown"
	if valid == "true" {
		accountStatus = "valid"
	} else if valid == "false" {
		accountStatus = "invalid (showing default Steve)"
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("mcskin_%s_%d.png", username, time.Now().UnixNano()))
	f, err := os.Create(tmp)
	if err != nil {
		if status != nil {
			status.Edit("failed to create temp file: " + html.EscapeString(err.Error()))
		}
		return nil
	}
	size, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmp)
		if status != nil {
			status.Edit("failed to save image: " + html.EscapeString(err.Error()))
		}
		return nil
	}
	defer os.Remove(tmp)

	caption := fmt.Sprintf(
		"<b>Minecraft Head</b>\n<b>User:</b> <code>%s</code>\n<b>Account:</b> %s\n<b>Size:</b> <code>%d B</code>",
		html.EscapeString(username),
		accountStatus,
		size,
	)

	if _, merr := m.ReplyMedia(tmp, &tg.MediaOptions{
		Caption:  caption,
		FileName: fmt.Sprintf("mcskin_%s.png", username),
		MimeType: "image/png",
	}); merr != nil {
		if status != nil {
			status.Edit("upload failed: " + html.EscapeString(merr.Error()))
		}
		return nil
	}

	if status != nil {
		status.Delete()
	}
	return nil
}

func registerMcSkinHandlers() {
	c := Client
	c.On("cmd:mcskin", McSkinHandler)
}

func init() {
	QueueHandlerRegistration(registerMcSkinHandlers)
}
