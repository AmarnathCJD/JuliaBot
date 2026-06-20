package modules

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func fetchScreenshot(target string) (string, error) {
	endpoint := "https://image.thum.io/get/width/1280/" + target
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("thum.io returned status %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return "", fmt.Errorf("unexpected content-type: %s", ct)
	}
	f, err := os.CreateTemp("", "ss-*.png")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 15*1024*1024)); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	st, err := os.Stat(f.Name())
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	if st.Size() == 0 {
		os.Remove(f.Name())
		return "", fmt.Errorf("empty screenshot response")
	}
	return f.Name(), nil
}

func ScreenshotHandler(m *tg.NewMessage) error {
	target := strings.TrimSpace(m.Args())
	if target == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			target = strings.TrimSpace(r.Text())
		}
	}
	if target == "" {
		m.Reply("usage: <code>/ss &lt;url&gt;</code>")
		return nil
	}

	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	u, err := url.Parse(target)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		m.Reply("invalid url, must be http(s)")
		return nil
	}

	status, _ := m.Reply("<code>capturing screenshot...</code>")

	path, err := fetchScreenshot(target)
	if err != nil {
		msg := "error fetching screenshot: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(path)

	if status != nil {
		status.Delete()
	}

	caption := "<b>Screenshot</b>\n<a href=\"" + html.EscapeString(target) + "\">" + html.EscapeString(target) + "</a>"
	if _, merr := m.ReplyMedia(path, &tg.MediaOptions{
		Caption:  caption,
		FileName: "screenshot.png",
		MimeType: "image/png",
	}); merr != nil {
		m.Reply("error sending: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func init() { QueueHandlerRegistration(registerScreenshotHandlers) }

func registerScreenshotHandlers() {
	c := Client
	c.On("cmd:ss", ScreenshotHandler)
}
