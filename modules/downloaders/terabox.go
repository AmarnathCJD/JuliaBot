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

type TeraboxFile struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     string `json:"size"`
	Category string `json:"category"`
}

type TeraboxResponse struct {
	Files []TeraboxFile `json:"files"`
	Error string        `json:"error"`
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

	if len(teraResp.Files) == 0 {
		msg.Edit("No files found")
		return nil
	}

	if len(teraResp.Files) == 1 {
		msg.Edit("<i>Downloading file...</i>")

		proxiedURL := fmt.Sprintf("https://insta.gogram.fun/proxy?url=%s&accessToken=%s",
			url.QueryEscape(teraResp.Files[0].URL),
			url.QueryEscape(accessToken))

		filePath, err := downloadTeraboxFile(proxiedURL, msg, teraResp.Files[0].Filename)
		if err != nil {
			msg.Edit(fmt.Sprintf("Download failed: %v", err))
			return nil
		}

		defer os.Remove(filePath)
		defer msg.Delete()

		target, _ := m.ReplyMedia(filePath, &telegram.MediaOptions{
			//ProgressManager: telegram.NewProgressManager(5).SetMessage(msg),
			Caption: fmt.Sprintf("Downloaded from Terabox\n\n⚠️ Forward this message, as it will get auto-deleted in 5 minutes.\n\nFile Name: %s", teraResp.Files[0].Filename),
		})

		go func() {
			time.Sleep(5 * time.Minute)
			target.Delete()
		}()
	} else {
		msg.Edit(fmt.Sprintf("<b>Found %d files in directory</b>\n\nDownloading all files...", len(teraResp.Files)))

		var uploadedFiles []string
		for i, file := range teraResp.Files {
			msg.Edit(fmt.Sprintf("<i>Downloading file %d/%d: %s...</i>", i+1, len(teraResp.Files), file.Filename))

			proxiedURL := fmt.Sprintf("https://insta.gogram.fun/proxy?url=%s&accessToken=%s",
				url.QueryEscape(file.URL),
				url.QueryEscape(accessToken))

			filePath, err := downloadTeraboxFile(proxiedURL, msg, file.Filename)
			if err != nil {
				continue
			}

			uploadedFiles = append(uploadedFiles, filePath)
		}

		if len(uploadedFiles) == 0 {
			msg.Edit("Failed to download any files")
			return nil
		}

		msg.Edit(fmt.Sprintf("<i>Uploading %d files to Telegram...</i>", len(uploadedFiles)))

		var targets []*telegram.NewMessage
		for i, filePath := range uploadedFiles {
			defer os.Remove(filePath)

			target, _ := m.ReplyMedia(filePath, &telegram.MediaOptions{
				//ProgressManager: telegram.NewProgressManager(5).SetMessage(msg),
				Caption: fmt.Sprintf("File %d/%d from Terabox\n\n⚠️ Forward this message, as it will get auto-deleted in 5 minutes.", i+1, len(uploadedFiles)),
			})
			targets = append(targets, target)
		}

		msg.Delete()

		go func() {
			time.Sleep(5 * time.Minute)
			for _, target := range targets {
				target.Delete()
			}
		}()
	}

	return nil
}

func downloadTeraboxFile(directURL string, progressMsg *telegram.NewMessage, fn string) (string, error) {
	filePath := "tmp/" + fn + "_terabox_download.%(ext)s"
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
				message = fmt.Sprintf(text, fn, size, eta, speed, progressbar, percent)
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

	matches, err := filepath.Glob("tmp/" + fn + "_terabox_download.*")
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("downloaded file not found")
	}
	return matches[0], nil
}
