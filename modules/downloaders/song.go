package downloaders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	yt "github.com/lrstanley/go-ytdlp"
)

type YTVideoInfo struct {
	Title         string
	Image         string
	LengthSeconds string
	Formats       []YTFormat
	AudioURL      string
	UserID        int64
	ChatID        int64
	MessageID     int32
	OriginalURL   string
}

type YTFormat struct {
	Quality  string
	URL      string
	HasAudio bool
	FileSize string
	MimeType string
}

var (
	ytVideoCache   = make(map[string]*YTVideoInfo)
	ytVideoCacheMu sync.RWMutex
)

func YtVideoDL(m *telegram.NewMessage) error {
	args := m.Args()
	if args == "" {
		m.Reply("Provide video url~")
		return nil
	}

	msg, _ := m.Reply("Downloading video...")

	dl := yt.New().
		FormatSort("res:1080,tbr").
		Format("bv+ba/b").
		NoWarnings().
		RecodeVideo("mp4").
		Output("yt-video.mp4").
		Downloader("aria2c").
		DownloaderArgs("--console-log-level=warn --max-connection-per-server=16 --split=16 --min-split-size=1M").
		ProgressFunc(time.Second*7, func(update yt.ProgressUpdate) {
			text := "<b>Downloading Youtube Video</b>\n\n"
			text += "<b>Name:</b> <code>%s</code>\n"
			text += "<b>File Size:</b> <code>%.2f MiB</code>\n"
			text += "<b>ETA:</b> <code>%s</code>\n"
			text += "<b>Speed:</b> <code>%s</code>\n"
			text += "<b>Progress:</b> %s <code>%.2f%%</code>"

			size := float64(update.TotalBytes) / 1024 / 1024
			eta := func() string {
				elapsed := time.Now().Unix() - update.Started.Unix()
				remaining := float64(update.TotalBytes-update.DownloadedBytes) / float64(update.DownloadedBytes) * float64(elapsed)
				return (time.Second * time.Duration(remaining)).String()
			}()

			speed := func() string {
				elapsedTime := time.Since(time.Unix(update.Started.Unix(), 0))
				if int(elapsedTime.Seconds()) == 0 {
					return "0 B/s"
				}
				speedBps := float64(update.TotalBytes) / elapsedTime.Seconds()
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
				msg.Edit("Starting download...")
				return
			}

			progressbar := strings.Repeat("■", int(percent/10)) + strings.Repeat("□", 10-int(percent/10))

			message := fmt.Sprintf(text, *update.Info.Title, size, eta, speed, progressbar, percent)
			msg.Edit(message)
		}).
		Proxy("http://rose:0000@127.0.0.1:8001").
		NoWarnings()

	_, err := dl.Run(context.TODO(), args)
	if err != nil {
		m.Reply("<code>video not found.</code>")
		return nil
	}

	defer os.Remove("yt-video.mp4")
	defer msg.Delete()

	m.ReplyMedia("yt-video.mp4", &telegram.MediaOptions{
		Attributes: []telegram.DocumentAttribute{
			&telegram.DocumentAttributeFilename{
				FileName: "yt-video.mp4",
			},
		},
		ProgressManager: telegram.NewProgressManager(5).SetMessage(msg),
	})
	return nil
}

func YtSongDL(m *telegram.NewMessage) error {
	args := m.Args()
	if args == "" {
		m.Reply("Provide song url~")
		return nil
	}

	var channelId string = "(unknown)"
	var thumbImage string

	if !strings.Contains(args, "youtube.com") {
		vidId, channel, thumb, err := searchYouTube(args)
		if err != nil {
			m.Reply("<code>video not found.</code>")
			return nil
		}

		fmt.Println("Video ID:", vidId)
		fmt.Println("Channel:", channel)
		fmt.Println("Thumbnail:", thumb)

		args = "https://www.youtube.com/watch?v=" + vidId
		channelId = channel
		thumbImage = thumb
	}

	vid, err := getVid(args)
	if err != nil {
		m.Reply("<code>video not found.</code>")
		return nil
	}

	re := regexp.MustCompile(`onVideoOptionSelected\('(.+?)', '(.+?)', '(.+?)', (\d+), '(.+?)', '(.+?)'\)`)
	matches := re.FindAllStringSubmatch(vid, -1)
	for _, match := range matches {

		if match[5] == "mp4a" {
			fi, _ := http.Get(match[2])
			if fi.StatusCode != 200 {
				m.Reply("<code>video not found.</code>")
				return nil
			}

			defer fi.Body.Close()
			body, _ := io.ReadAll(fi.Body)
			os.WriteFile("song.mp3", body, 0644)
			defer os.Remove("song.mp3")

			m.ReplyMedia("song.mp3", &telegram.MediaOptions{
				Attributes: []telegram.DocumentAttribute{
					&telegram.DocumentAttributeFilename{
						FileName: strings.Split(match[3], "', '")[1] + ".mp3",
					},
					&telegram.DocumentAttributeAudio{
						Title:     strings.Split(match[3], "', '")[1],
						Performer: channelId,
					},
				},
				Thumb: thumbImage,
			})
		}
	}
	return nil
}

func searchYouTube(query string) (string, string, string, error) {
	searchQuery := strings.ReplaceAll(query, " ", "+")
	url := "https://www.youtube.com/results?search_query=" + searchQuery
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf("error reading response: %w", err)
	}

	bodyText := string(body)
	urlRegex := regexp.MustCompile(`https://i\.ytimg\.com/vi/[\w-]+/`)
	urls := urlRegex.FindAllString(bodyText, -1)

	channelRegex := regexp.MustCompile(`"\/@[\w-]+"`)
	channels := channelRegex.FindAllString(bodyText, -1)

	if len(urls) == 0 || len(channels) == 0 {
		return "", "", "", fmt.Errorf("no results found")
	}

	videoIDs := []string{}
	for _, url := range urls {
		parts := strings.Split(url, "/")
		if len(parts) >= 5 {
			videoIDs = append(videoIDs, parts[4])
		}
	}

	if len(videoIDs) == 0 {
		return "", "", "", fmt.Errorf("no video IDs found")
	}

	if len(channels) == 0 {
		return "", "", "", fmt.Errorf("no channels found")
	}

	return videoIDs[0], channels[0][2 : len(channels[0])-1], "https://i.ytimg.com/vi/" + videoIDs[0] + "/default.jpg", nil
}

func getVid(videoURL string) (string, error) {
	payload := []byte(`videoURL=` + videoURL)

	req, err := http.NewRequest("POST", "https://ssyoutube.online/yt-video-detail/", bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	return string(body), nil
}

type Sptfy struct {
	Artists    string `json:"artists"`
	Title      string `json:"title"`
	Image      string `json:"image"`
	IsPlaying  bool   `json:"is_playing"`
	DurationMs int    `json:"duration_ms"`
	ProgressMs int    `json:"progress_ms"`
	URL        string `json:"url"`
}

func InlineSpotify(m *telegram.InlineQuery) error {
	b := m.Builder()
	svg, _ := http.Get("https://spotify-now-playing-psi-silk.vercel.app/api/current-playing?s=1")
	if svg.StatusCode != 200 {
		b.Article("Error", "Failed to fetch data", "Failed to fetch data", &telegram.ArticleOptions{
			ReplyMarkup: telegram.NewKeyboard().AddRow(
				telegram.Button.SwitchInline("Retry", true, "sp"),
			).Build(),
		})
		m.Answer(b.Results())
		return nil
	}

	defer svg.Body.Close()
	var s Sptfy
	json.NewDecoder(svg.Body).Decode(&s)

	var caption string
	if s.IsPlaying {
		caption = fmt.Sprintf(
			"<b><i>Now Playing:</i></b> <a href=\"%s\">%s</a>\n"+
				"<b><i>Artist:</i></b> %s\n"+
				"<b><i>Time:</i></b> %s / %s",
			s.URL,
			s.Title,
			s.Artists,
			fmtDuration(s.ProgressMs),
			fmtDuration(s.DurationMs),
		)
	} else {
		caption = "<i>No song is currently playing</i>"
	}

	btn := telegram.Button

	if s.Image != "" {
		b.Document(s.Image, &telegram.ArticleOptions{
			Title:         "Spotify Now Playing",
			Description:   "Shows the currently playing song on Spotify",
			Caption:       caption,
			ForceDocument: true,
			ReplyMarkup: telegram.NewKeyboard().AddRow(
				btn.URL("Open in Spotify", s.URL),
			).Build(),
		})
	} else {
		b.Article("Spotify Now Playing", "Shows the currently playing song on Spotify", caption, &telegram.ArticleOptions{
			ReplyMarkup: telegram.NewKeyboard().AddRow(
				btn.URL("Open in Spotify", s.URL),
			).Build(),
		})
	}

	m.Answer(b.Results(), &telegram.InlineSendOptions{Gallery: true, CacheTime: 0})
	return nil
}

func fmtDuration(ms int) string {
	sec := ms / 1000
	min := sec / 60
	sec = sec % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}

func YTCustomHandler(m *telegram.NewMessage) error {
	query := m.Args()
	if query == "" {
		m.Reply("Provide youtube link~")
		return nil
	}

	var videoURL string
	if strings.Contains(query, "youtube.com") || strings.Contains(query, "youtu.be") {
		videoURL = query
	} else {
		vidId, _, _, err := searchYouTube(query)
		if err != nil {
			m.Reply("<code>Video not found.</code>")
			return nil
		}
		videoURL = "https://www.youtube.com/watch?v=" + vidId
	}

	msg, _ := m.Reply("<i>Fetching video info...</i>")

	info, err := fetchYTCustom(videoURL)
	if err != nil || info == nil || len(info.Formats) == 0 {
		msg.Edit("<i>Trying quick download...</i>")
		return ytQuickFallback(m, msg, videoURL)
	}

	cacheID := fmt.Sprintf("%d%d", m.ChatID(), time.Now().UnixNano())
	info.UserID = m.SenderID()
	info.ChatID = m.ChatID()
	info.OriginalURL = videoURL

	ytVideoCacheMu.Lock()
	ytVideoCache[cacheID] = info
	ytVideoCacheMu.Unlock()

	keyboard := telegram.NewKeyboard()
	row := []telegram.KeyboardButton{}
	for i, f := range info.Formats {
		label := f.Quality
		if f.HasAudio {
			label += " [Audio]"
		}
		if f.FileSize != "" {
			size, _ := strconv.ParseInt(f.FileSize, 10, 64)
			if size > 0 {
				label += fmt.Sprintf(" (%.1fMB)", float64(size)/1024/1024)
			}
		}
		callbackData := fmt.Sprintf("ytdl_%s_%d", cacheID, i)
		row = append(row, telegram.Button.Data(label, callbackData))
		if len(row) == 2 || i == len(info.Formats)-1 {
			keyboard.AddRow(row...)
			row = []telegram.KeyboardButton{}
		}
	}

	keyboard.AddRow(telegram.Button.Data("Cancel", fmt.Sprintf("ytdl_%s_cancel", cacheID)))

	caption := fmt.Sprintf(
		"<b>%s</b>\n\n"+
			"<b>Duration:</b> %s\n\n"+
			"<i>Select quality:</i>",
		info.Title,
		fmtDurationSec(info.LengthSeconds),
	)

	info.MessageID = msg.ID
	msg.Edit(caption, &telegram.SendOptions{ReplyMarkup: keyboard.Build()})

	return nil
}

func fmtDurationSec(sec string) string {
	s, _ := strconv.Atoi(sec)
	min := s / 60
	s = s % 60
	return fmt.Sprintf("%02d:%02d", min, s)
}

func fetchYTCustom(videoURL string) (*YTVideoInfo, error) {
	payload := map[string]string{
		"url":         videoURL,
		"accessToken": os.Getenv("INSTA_ACCESS"),
	}
	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://insta.gogram.fun/yt-custom", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var result struct {
		Title         string `json:"title"`
		Image         string `json:"image"`
		LengthSeconds string `json:"lengthSeconds"`
		FormatOptions struct {
			Video struct {
				MP4 []struct {
					Quality  string `json:"quality"`
					URL      string `json:"url"`
					HasAudio bool   `json:"hasAudio"`
					FileSize string `json:"fileSize"`
					MimeType string `json:"mimeType"`
				} `json:"mp4"`
			} `json:"video"`
		} `json:"format_options"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != "" {
		return nil, fmt.Errorf("%s", result.Error)
	}

	info := &YTVideoInfo{
		Title:         result.Title,
		Image:         result.Image,
		LengthSeconds: result.LengthSeconds,
	}

	var audioURL string
	for _, f := range result.FormatOptions.Video.MP4 {
		info.Formats = append(info.Formats, YTFormat{
			Quality:  f.Quality,
			URL:      f.URL,
			HasAudio: f.HasAudio,
			FileSize: f.FileSize,
			MimeType: f.MimeType,
		})
		if f.HasAudio && audioURL == "" {
			audioURL = f.URL
		}
	}
	info.AudioURL = audioURL

	return info, nil
}

func ytQuickFallback(m *telegram.NewMessage, msg *telegram.NewMessage, videoURL string) error {
	payload := map[string]string{
		"url":         videoURL,
		"accessToken": os.Getenv("INSTA_ACCESS"),
	}
	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://insta.gogram.fun/yt-quick", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		msg.Edit("<code>Failed to fetch video.</code>")
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		DownloadURL string `json:"downloadURL"`
		Status      string `json:"status"`
		Error       string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.DownloadURL == "" {
		msg.Edit("<code>Failed to get download URL.</code>")
		return nil
	}

	msg.Edit("<i>Downloading video...</i>")

	filePath, err := downloadFile(result.DownloadURL, "yt_quick_video.mp4", msg)
	if err != nil {
		msg.Edit("<code>Failed to download video.</code>")
		return nil
	}
	defer os.Remove(filePath)

	msg.Edit("<i>Uploading video...</i>")

	m.ReplyMedia(filePath, &telegram.MediaOptions{
		ProgressManager: telegram.NewProgressManager(5).SetMessage(msg),
	})
	msg.Delete()

	return nil
}

func YTCallbackHandler(cb *telegram.CallbackQuery) error {
	data := cb.DataString()
	if !strings.HasPrefix(data, "ytdl_") {
		return nil
	}

	data = strings.TrimPrefix(data, "ytdl_")
	lastUnderscore := strings.LastIndex(data, "_")
	if lastUnderscore == -1 {
		return nil
	}

	cacheID := data[:lastUnderscore]
	action := data[lastUnderscore+1:]

	if action == "cancel" {
		ytVideoCacheMu.Lock()
		delete(ytVideoCache, cacheID)
		ytVideoCacheMu.Unlock()
		cb.Edit("<b>Cancelled</b>")
		cb.Answer("Cancelled")
		return nil
	}

	ytVideoCacheMu.RLock()
	info, exists := ytVideoCache[cacheID]
	ytVideoCacheMu.RUnlock()

	if !exists {
		cb.Answer("Session expired. Please try again.", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	if cb.Sender.ID != info.UserID {
		cb.Answer("Only the requester can select quality.", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	formatIdx, err := strconv.Atoi(action)
	if err != nil || formatIdx < 0 || formatIdx >= len(info.Formats) {
		cb.Answer("Invalid selection", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	selectedFormat := info.Formats[formatIdx]
	cb.Answer(fmt.Sprintf("Downloading %s", selectedFormat.Quality))
	cb.Edit(fmt.Sprintf("<i>Downloading %s quality...</i>", selectedFormat.Quality))

	var finalPath string
	var cleanupFiles []string

	if selectedFormat.HasAudio {
		msg := &telegram.NewMessage{
			ID:     info.MessageID,
			Client: cb.Client,
		}
		filePath, err := downloadFile(selectedFormat.URL, fmt.Sprintf("yt_%s.mp4", selectedFormat.Quality), msg)
		if err != nil {
			cb.Edit("<code>Failed to download video.</code>")
			return nil
		}
		finalPath = filePath
		cleanupFiles = append(cleanupFiles, filePath)
	} else {
		if info.AudioURL == "" {
			cb.Edit("<code>No audio source available.</code>")
			return nil
		}

		cb.Edit(fmt.Sprintf("<i>Downloading %s video...</i>", selectedFormat.Quality))
		msg := &telegram.NewMessage{
			ID:     info.MessageID,
			Client: cb.Client,
		}
		videoPath, err := downloadFile(selectedFormat.URL, fmt.Sprintf("yt_video_%s.mp4", selectedFormat.Quality), msg)
		if err != nil {
			cb.Edit("<code>Failed to download video.</code>")
			return nil
		}
		cleanupFiles = append(cleanupFiles, videoPath)

		cb.Edit("<i>Downloading audio...</i>")
		audioPath, err := downloadFile(info.AudioURL, "yt_audio.mp4", msg)
		if err != nil {
			for _, f := range cleanupFiles {
				os.Remove(f)
			}
			cb.Edit("<code>Failed to download audio.</code>")
			return nil
		}
		cleanupFiles = append(cleanupFiles, audioPath)

		cb.Edit("<i>Merging video and audio...</i>")
		outputPath := fmt.Sprintf("yt_merged_%s.mp4", selectedFormat.Quality)
		if err := mergeVideoAudio(videoPath, audioPath, outputPath); err != nil {
			for _, f := range cleanupFiles {
				os.Remove(f)
			}
			cb.Edit("<code>Failed to merge video and audio.</code>")
			return nil
		}
		cleanupFiles = append(cleanupFiles, outputPath)
		finalPath = outputPath
	}

	ytVideoCacheMu.Lock()
	delete(ytVideoCache, cacheID)
	ytVideoCacheMu.Unlock()

	cb.Edit("<i>Uploading video...</i>")

	msg := &telegram.NewMessage{
		ID:     info.MessageID,
		Client: cb.Client,
	}

	// if finalPath size > 2GB, abort with error
	fileInfo, err := os.Stat(finalPath)
	if err != nil {
		cb.Edit("<code>Failed to access video file.</code>")
		return nil
	}
	if fileInfo.Size() > 2*1024*1024*1024 {
		cb.Edit("<code>> 2GB files cannot be uploaded to Telegram.</code>")
		for _, f := range cleanupFiles {
			os.Remove(f)
		}
		return nil
	}

	_, err = cb.Client.SendMedia(info.ChatID, finalPath, &telegram.MediaOptions{
		Caption:         fmt.Sprintf("<b>%s</b>\nQuality: %s", info.Title, selectedFormat.Quality),
		ProgressManager: telegram.NewProgressManager(5).SetMessage(msg),
	})

	for _, f := range cleanupFiles {
		os.Remove(f)
	}

	if err != nil {
		cb.Edit("<code>Failed to upload video.</code>")
		return nil
	}

	cb.Delete()
	return nil
}

func downloadFile(url, filename string, progressMsg *telegram.NewMessage) (string, error) {
	filePath := "tmp/" + filename
	os.MkdirAll("tmp", 0755)

	dl := yt.New().
		Output(filePath).
		Downloader("aria2c").
		DownloaderArgs("--console-log-level=warn --max-connection-per-server=16 --split=16 --min-split-size=1M").
		NoWarnings()

	if progressMsg != nil {
		dl = dl.ProgressFunc(time.Second*5, func(update yt.ProgressUpdate) {
			percent := float64(update.DownloadedBytes) / float64(update.TotalBytes) * 100
			if percent > 0 {
				progressMsg.Edit(fmt.Sprintf("<i>Downloading: %.1f%%</i>", percent))
			}
		})
	}

	_, err := dl.Run(context.TODO(), url)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func mergeVideoAudio(videoPath, audioPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-y",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %v, stderr: %s", err, stderr.String())
	}

	return nil
}
