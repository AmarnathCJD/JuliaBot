package downloaders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	yt "github.com/lrstanley/go-ytdlp"
)

type TeraboxResponse struct {
	DirectURL string `json:"url"`
	Error     string `json:"error"`
}

func TeraboxHandler(m *telegram.NewMessage) error {
	args := m.Args()
	if args == "" {
		m.Reply("Provide Terabox URL")
		return nil
	}

	accessToken := os.Getenv("INSTA_ACCESS")
	if accessToken == "" {
		m.Reply("Terabox access token not configured")
		return nil
	}

	msg, _ := m.Reply("<i>Fetching download link...</i>")

	apiURL := fmt.Sprintf("https://insta.gogram.fun/teradl?url=%s&accessToken=%s",
		url.QueryEscape(args),
		url.QueryEscape(accessToken))

	resp, err := http.Get(apiURL)
	if err != nil {
		msg.Edit("Failed to fetch download link")
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		msg.Edit("Failed to read response")
		return nil
	}

	var teraResp TeraboxResponse
	if err := json.Unmarshal(body, &teraResp); err != nil {
		msg.Edit("Failed to parse response")
		return nil
	}

	if teraResp.Error != "" {
		msg.Edit(fmt.Sprintf("Error: %s", teraResp.Error))
		return nil
	}

	if teraResp.DirectURL == "" {
		msg.Edit("No download URL found")
		return nil
	}

	msg.Edit("<i>Downloading file...</i>")

	filePath, err := downloadTeraboxFile(teraResp.DirectURL, msg)
	if err != nil {
		msg.Edit(fmt.Sprintf("Download failed: %v", err))
		return nil
	}

	defer os.Remove(filePath)
	msg.Delete()

	m.ReplyMedia(filePath, &telegram.MediaOptions{
		ProgressManager: telegram.NewProgressManager(5).SetMessage(msg),
	})

	return nil
}

func downloadTeraboxFile(directURL string, progressMsg *telegram.NewMessage) (string, error) {
	filePath := "tmp/terabox_download.%(ext)s"
	os.MkdirAll("tmp", 0755)

	dl := yt.New().
		Output(filePath).
		ConcurrentFragments(16).
		NoWarnings().
		ProgressFunc(time.Second*3, func(update yt.ProgressUpdate) {
			text := "<b>Downloading Terabox File</b>\n\n"
			if update.Info != nil && update.Info.Title != nil {
				text += "<b>Name:</b> <code>%s</code>\n"
			}
			text += "<b>File Size:</b> <code>%.2f MiB</code>\n"
			text += "<b>ETA:</b> <code>%s</code>\n"
			text += "<b>Speed:</b> <code>%s</code>\n"
			text += "<b>Progress:</b> %s <code>%.2f%%</code>"

			size := float64(update.TotalBytes) / 1024 / 1024

			eta := func() string {
				elapsed := time.Now().Unix() - update.Started.Unix()
				if elapsed == 0 || update.DownloadedBytes == 0 {
					return "calculating..."
				}
				remaining := float64(update.TotalBytes-update.DownloadedBytes) / float64(update.DownloadedBytes) * float64(elapsed)
				return (time.Second * time.Duration(remaining)).String()
			}()

			speed := func() string {
				elapsedTime := time.Since(update.Started)
				if int(elapsedTime.Seconds()) == 0 {
					return "0 B/s"
				}
				speedBps := float64(update.DownloadedBytes) / elapsedTime.Seconds()
				if speedBps < 1024 {
					return fmt.Sprintf("%.2f B/s", speedBps)
				} else if speedBps < 1024*1024 {
					return fmt.Sprintf("%.2f KB/s", speedBps/1024)
				} else {
					return fmt.Sprintf("%.2f MB/s", speedBps/1024/1024)
				}
			}()

			percent := float64(update.DownloadedBytes) / float64(update.TotalBytes) * 100
			if percent == 0 {
				progressMsg.Edit("Starting download...")
				return
			}

			progressbar := func() string {
				bars := int(percent / 10)
				filled := ""
				empty := ""
				for i := 0; i < bars; i++ {
					filled += "■"
				}
				for i := 0; i < 10-bars; i++ {
					empty += "□"
				}
				return filled + empty
			}()

			var message string
			if update.Info != nil && update.Info.Title != nil {
				message = fmt.Sprintf(text, *update.Info.Title, size, eta, speed, progressbar, percent)
			} else {
				message = fmt.Sprintf("<b>Downloading Terabox File</b>\n\n<b>File Size:</b> <code>%.2f MiB</code>\n<b>ETA:</b> <code>%s</code>\n<b>Speed:</b> <code>%s</code>\n<b>Progress:</b> %s <code>%.2f%%</code>",
					size, eta, speed, progressbar, percent)
			}
			progressMsg.Edit(message)
		}).
		NoWarnings()

	_, err := dl.Run(context.TODO(), directURL)
	if err != nil {
		return "", err
	}

	matches, err := filepath.Glob("tmp/terabox_download.*")
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("downloaded file not found")
	}

	actualFilePath := matches[0]
	return actualFilePath, nil
}
