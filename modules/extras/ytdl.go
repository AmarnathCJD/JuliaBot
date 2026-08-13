package extras

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

const (
	ssyouUA          = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36"
	ssyouReferer     = "https://ssyou.online/en1401Yd/"
	ssyouOrigin      = "https://ssyou.online"
	ssyouDetailURL   = "https://ssyou.online/yt-video-detail/"
	ssyouAjaxURL     = "https://ssyou.online/wp-admin/admin-ajax.php"
	ytdlMaxBytes     = int64(1900 * 1024 * 1024) // 1.9 GB Telegram-friendly ceiling
	ytdlPickerTTL    = 10 * time.Minute
	ytdlRenderPoll   = 2 * time.Second
	ytdlRenderMaxDur = 8 * time.Minute
)

var (
	ytdlHTTP = &http.Client{
		Timeout: 5 * time.Minute,
	}
	ytdlSessions sync.Map // key "chatID:msgID" -> *ytdlSession
	ytdlURLRe    = regexp.MustCompile(`(?i)https?://(?:www\.|m\.)?(?:youtube\.com/(?:watch\?[^\s]*v=|shorts/|live/|embed/|v/)|youtu\.be/)[A-Za-z0-9_\-]{11}[^\s]*`)
	ytdlIDRe     = regexp.MustCompile(`(?i)(?:youtu\.be/|youtube\.com/(?:watch\?(?:[^"' ]*&)?v=|shorts/|live/|embed/|v/))([A-Za-z0-9_\-]{11})`)

	reFormatURL   = regexp.MustCompile(`'([0-9]{3,4}p|MP3)':\s*'(https://[^']+)'`)
	reAudioURL    = regexp.MustCompile(`(?s)const\s+audioUrl\s*=\s*'([^']+)'`)
	reMergeNonce  = regexp.MustCompile(`'nonce',\s*'([a-f0-9]+)'`)
	reVideoID     = regexp.MustCompile(`name="video_id"\s+value="([A-Za-z0-9_\-]{11})"`)
	reVideoTitle  = regexp.MustCompile(`(?s)class="col-lg-12[^"]*videoTitle"[^>]*title="([^"]+)"`)
	reDuration    = regexp.MustCompile(`<label class="duration">Duration:\s*([^<]+)</label>`)
	reThumbnail   = regexp.MustCompile(`<img class="thumbnail"\s+src="(https?://[^"]+)"`)
	reQualityRow  = regexp.MustCompile(`(?s)data-quality="([^"]+)"[^>]*?data-size="([0-9]+)"[^>]*?onclick="([a-zA-Z0-9]+)\(`)
	reHasAudioIn  = regexp.MustCompile(`data-has-audio="([^"]+)"`)
)

type ytdlFormat struct {
	Quality  string // "MP3", "144p", ..., "2160p"
	Size     int64
	HasAudio bool
	Kind     string // "mp3" | "direct" | "merge"
	URL      string // populated when Kind is "mp3" or "direct"; empty for merge
}

type ytdlVideo struct {
	VideoID   string
	Title     string
	Duration  string
	Thumbnail string
	AudioURL  string
	Nonce     string
	Formats   []ytdlFormat
	formatURL map[string]string // quality -> googlevideo URL
	fetched   time.Time
}

type ytdlSession struct {
	OwnerID int64
	YTLink  string
	Video   *ytdlVideo
}

func extractYTLink(s string) string {
	if s == "" {
		return ""
	}
	return ytdlURLRe.FindString(s)
}

func fetchYTDLDetail(ctx context.Context, ytURL string) (*ytdlVideo, error) {
	form := url.Values{"videoURL": {ytURL}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ssyouDetailURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("user-agent", ssyouUA)
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("origin", ssyouOrigin)
	req.Header.Set("referer", ssyouReferer)
	req.AddCookie(&http.Cookie{Name: "pll_language", Value: "en"})

	resp, err := ytdlHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ssyou returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	page := string(body)

	video := &ytdlVideo{formatURL: map[string]string{}, fetched: time.Now()}
	if m := reVideoID.FindStringSubmatch(page); len(m) == 2 {
		video.VideoID = m[1]
	} else if m := ytdlIDRe.FindStringSubmatch(ytURL); len(m) == 2 {
		video.VideoID = m[1]
	}
	if m := reVideoTitle.FindStringSubmatch(page); len(m) == 2 {
		video.Title = strings.TrimSpace(m[1])
	}
	if m := reDuration.FindStringSubmatch(page); len(m) == 2 {
		video.Duration = strings.TrimSpace(m[1])
	}
	if m := reThumbnail.FindStringSubmatch(page); len(m) == 2 {
		video.Thumbnail = m[1]
	}
	if m := reAudioURL.FindStringSubmatch(page); len(m) == 2 {
		video.AudioURL = m[1]
	}
	if m := reMergeNonce.FindStringSubmatch(page); len(m) == 2 {
		video.Nonce = m[1]
	}

	for _, m := range reFormatURL.FindAllStringSubmatch(page, -1) {
		q, u := m[1], m[2]
		if q == "audio" {
			continue
		}
		video.formatURL[q] = u
	}

	rowSeen := map[string]bool{}
	// findIndex of every match to slice out the surrounding button chunk so we
	// can look for optional data-has-audio without regex backtracking games.
	for _, loc := range reQualityRow.FindAllStringSubmatchIndex(page, -1) {
		// loc: [start, end, g1s, g1e, g2s, g2e, g3s, g3e]
		quality := page[loc[2]:loc[3]]
		if rowSeen[quality] {
			continue
		}
		rowSeen[quality] = true
		var size int64
		fmt.Sscan(page[loc[4]:loc[5]], &size)
		action := page[loc[6]:loc[7]]

		// look for data-has-audio inside the matched span (which stops just
		// before the onclick payload — plenty of room for the attribute)
		chunk := page[loc[0]:loc[1]]
		hasAudio := false
		if hm := reHasAudioIn.FindStringSubmatch(chunk); len(hm) == 2 {
			hasAudio = hm[1] == "true"
		}

		f := ytdlFormat{Quality: quality, Size: size, HasAudio: hasAudio}
		switch {
		case action == "mp3Conversion":
			f.Kind = "mp3"
			f.URL = video.AudioURL
		case hasAudio || action == "openInNewTab":
			f.Kind = "direct"
			f.URL = video.formatURL[quality]
		default:
			f.Kind = "merge"
		}
		video.Formats = append(video.Formats, f)
	}
	if len(video.Formats) == 0 {
		return nil, fmt.Errorf("no formats found in response")
	}
	if video.VideoID == "" {
		return nil, fmt.Errorf("video ID missing from response")
	}
	return video, nil
}

func humanBytes(n int64) string {
	const (
		kb = 1024
		mb = 1024 * 1024
		gb = 1024 * 1024 * 1024
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/gb)
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/mb)
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/kb)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

var qualityRank = map[string]int{
	"MP3": 0, "144p": 1, "240p": 2, "360p": 3, "480p": 4,
	"720p": 5, "1080p": 6, "1440p": 7, "2160p": 8,
}

func qualityBadge(q string) string {
	switch q {
	case "MP3":
		return "HQ"
	case "2160p":
		return "4K"
	case "1440p":
		return "2K"
	case "1080p":
		return "FHD"
	case "720p":
		return "HD"
	}
	return ""
}

func buildPickerText(v *ytdlVideo) string {
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(v.Title))
	b.WriteString("</b>\n")
	if v.Duration != "" {
		b.WriteString("<i>")
		b.WriteString(html.EscapeString(v.Duration))
		b.WriteString("</i>\n")
	}
	b.WriteString("\nPick a quality to download:")
	return b.String()
}

func buildPickerKeyboard(v *ytdlVideo, sessionKey string) *tg.ReplyInlineMarkup {
	b := tg.Button
	kb := tg.NewKeyboard()

	// Sort formats: MP3 first, then video ascending resolution (so highest quality is at the bottom = easiest to tap on mobile? No — user expectation is highest first. Do descending video, mp3 last row.)
	video := make([]ytdlFormat, 0, len(v.Formats))
	var mp3 *ytdlFormat
	for i := range v.Formats {
		f := v.Formats[i]
		if f.Quality == "MP3" {
			mp3 = &f
			continue
		}
		video = append(video, f)
	}
	// sort desc by rank
	for i := 0; i < len(video); i++ {
		for j := i + 1; j < len(video); j++ {
			if qualityRank[video[j].Quality] > qualityRank[video[i].Quality] {
				video[i], video[j] = video[j], video[i]
			}
		}
	}

	for i := 0; i < len(video); i += 2 {
		row := []tg.KeyboardButton{makeQualityButton(video[i], sessionKey)}
		if i+1 < len(video) {
			row = append(row, makeQualityButton(video[i+1], sessionKey))
		}
		kb.AddRow(row...)
	}
	if mp3 != nil {
		kb.AddRow(makeQualityButton(*mp3, sessionKey).Success())
	}
	_ = b
	kb.AddRow(tg.Button.Data("Cancel", "yt:cancel:"+sessionKey).Danger())
	return kb.Build()
}

func makeQualityButton(f ytdlFormat, sessionKey string) *tg.KeyboardButtonCallback {
	label := f.Quality
	if bg := qualityBadge(f.Quality); bg != "" {
		label += " " + bg
	}
	if f.Size > 0 {
		label += " - " + humanBytes(f.Size)
	}
	btn := tg.Button.Data(label, "yt:pick:"+sessionKey+":"+f.Quality)
	// Style: give the top-tier ones a Primary (blue) tint, keep the rest neutral.
	switch f.Quality {
	case "2160p", "1440p":
		return btn.Danger()
	case "1080p", "720p":
		return btn.Primary()
	}
	return btn
}

func sessionKey(chatID int64, msgID int32) string {
	return fmt.Sprintf("%d:%d", chatID, msgID)
}

func gcYTDLSessions() {
	now := time.Now()
	ytdlSessions.Range(func(k, v any) bool {
		if s, ok := v.(*ytdlSession); ok && s.Video != nil {
			if now.Sub(s.Video.fetched) > ytdlPickerTTL {
				ytdlSessions.Delete(k)
			}
		}
		return true
	})
}

// YTDLHandler is /yt <youtube link>
func YTDLHandler(m *tg.NewMessage) error {
	gcYTDLSessions()
	link := extractYTLink(m.Args())
	if link == "" && m.IsReply() {
		if reply, err := m.GetReplyMessage(); err == nil {
			link = extractYTLink(reply.Text())
		}
	}
	if link == "" {
		m.Reply("Usage: <code>/yt &lt;youtube link&gt;</code>", &tg.SendOptions{ParseMode: "HTML"})
		return nil
	}

	status, _ := m.Reply("Fetching video info...")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	video, err := fetchYTDLDetail(ctx, link)
	if err != nil {
		if status != nil {
			status.Edit("Failed to fetch: " + html.EscapeString(err.Error()))
		}
		log.Printf("[ytdl] fetch failed url=%s err=%v", link, err)
		return nil
	}

	sess := &ytdlSession{OwnerID: m.SenderID(), YTLink: link, Video: video}
	// stash under a temp key until we know the sent message ID
	tmpKey := fmt.Sprintf("tmp:%d:%d", m.ChatID(), time.Now().UnixNano())
	ytdlSessions.Store(tmpKey, sess)

	kb := buildPickerKeyboard(video, tmpKey)
	if status != nil {
		if _, err := status.Edit(buildPickerText(video), &tg.SendOptions{ParseMode: "HTML", ReplyMarkup: kb}); err != nil {
			log.Printf("[ytdl] picker edit failed: %v", err)
		}
		// rekey under the final chat:msgID once we know it
		finalKey := sessionKey(status.ChatID(), status.ID)
		ytdlSessions.Delete(tmpKey)
		ytdlSessions.Store(finalKey, sess)
		// re-render callback data with final key
		if _, err := status.Edit(buildPickerText(video), &tg.SendOptions{ParseMode: "HTML", ReplyMarkup: buildPickerKeyboard(video, finalKey)}); err != nil {
			log.Printf("[ytdl] rekey edit failed: %v", err)
		}
	}
	return nil
}

// YTDLCallback handles yt:pick:<sessKey>:<quality> and yt:cancel:<sessKey>
func YTDLCallback(c *tg.CallbackQuery) error {
	data := c.DataString()
	rest := strings.TrimPrefix(data, "yt:")
	action, tail, ok := strings.Cut(rest, ":")
	if !ok {
		c.Answer("Bad data.", &tg.CallbackOptions{Alert: false})
		return nil
	}

	if action == "cancel" {
		key := tail
		if v, ok := ytdlSessions.Load(key); ok {
			if s, ok2 := v.(*ytdlSession); ok2 && s.OwnerID != c.SenderID {
				c.Answer("Not for you.", &tg.CallbackOptions{Alert: true})
				return nil
			}
		}
		ytdlSessions.Delete(key)
		c.Edit("Cancelled.")
		c.Answer("", &tg.CallbackOptions{Alert: false})
		return nil
	}

	if action != "pick" {
		c.Answer("Unknown action.", &tg.CallbackOptions{Alert: false})
		return nil
	}
	// tail = "chatID:msgID:quality"
	// split from the RIGHT since chatID may be negative and contain a colon? Actually chatID:msgID is two ints separated by colon.
	lastColon := strings.LastIndex(tail, ":")
	if lastColon <= 0 {
		c.Answer("Bad data.", &tg.CallbackOptions{Alert: false})
		return nil
	}
	sessKey := tail[:lastColon]
	quality := tail[lastColon+1:]

	v, ok := ytdlSessions.Load(sessKey)
	if !ok {
		c.Answer("Session expired. Run /yt again.", &tg.CallbackOptions{Alert: true})
		return nil
	}
	sess := v.(*ytdlSession)
	if sess.OwnerID != c.SenderID {
		c.Answer("Not for you.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	var chosen *ytdlFormat
	for i := range sess.Video.Formats {
		if sess.Video.Formats[i].Quality == quality {
			chosen = &sess.Video.Formats[i]
			break
		}
	}
	if chosen == nil {
		c.Answer("Format not found.", &tg.CallbackOptions{Alert: true})
		return nil
	}
	if chosen.Size > ytdlMaxBytes {
		c.Answer(fmt.Sprintf("Too big (%s). Max %s.", humanBytes(chosen.Size), humanBytes(ytdlMaxBytes)),
			&tg.CallbackOptions{Alert: true})
		return nil
	}

	// Ack immediately so the client doesn't show a spinner forever.
	c.Answer("Preparing "+quality+"...", &tg.CallbackOptions{Alert: false})
	ytdlSessions.Delete(sessKey) // one-shot

	go runYTDLDownload(c, sess, *chosen)
	return nil
}

func runYTDLDownload(c *tg.CallbackQuery, sess *ytdlSession, f ytdlFormat) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ytdl] panic: %v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), ytdlRenderMaxDur+5*time.Minute)
	defer cancel()

	// Update the picker message to a progress line.
	c.Edit(fmt.Sprintf("<b>%s</b>\n\n%s - preparing...",
		html.EscapeString(sess.Video.Title), f.Quality),
		&tg.SendOptions{ParseMode: "HTML"})

	mediaURL := f.URL
	var outputName string

	switch f.Kind {
	case "mp3", "direct":
		if mediaURL == "" {
			mediaURL = sess.Video.formatURL[f.Quality]
		}
		if mediaURL == "" {
			c.Edit("No direct URL available for " + f.Quality + ". Try another quality.")
			return
		}
		if f.Kind == "mp3" {
			outputName = fmt.Sprintf("%s.m4a", sanitizeFileName(sess.Video.Title))
		} else {
			outputName = fmt.Sprintf("%s_%s.mp4", sanitizeFileName(sess.Video.Title), f.Quality)
		}

	case "merge":
		videoURL := sess.Video.formatURL[f.Quality]
		if videoURL == "" || sess.Video.AudioURL == "" {
			c.Edit("Missing video or audio URL for " + f.Quality + ".")
			return
		}
		c.Edit(fmt.Sprintf("<b>%s</b>\n\n%s - downloading...",
			html.EscapeString(sess.Video.Title), f.Quality),
			&tg.SendOptions{ParseMode: "HTML"})
		merged, err := runYTDLMerge(ctx, sess.Video, f, videoURL, c)
		if err != nil {
			c.Edit("Download failed: " + html.EscapeString(err.Error()))
			log.Printf("[ytdl] merge failed: %v", err)
			return
		}
		mediaURL = merged
		outputName = fmt.Sprintf("%s_%s.mp4", sanitizeFileName(sess.Video.Title), f.Quality)
	}

	c.Edit(fmt.Sprintf("<b>%s</b>\n\n%s - downloading %s...",
		html.EscapeString(sess.Video.Title), f.Quality, humanBytes(f.Size)),
		&tg.SendOptions{ParseMode: "HTML"})

	tmpDir, err := os.MkdirTemp("", "ytdl-*")
	if err != nil {
		c.Edit("tmp dir: " + err.Error())
		return
	}
	defer os.RemoveAll(tmpDir)
	localPath := filepath.Join(tmpDir, outputName)
	if err := downloadToFile(ctx, mediaURL, localPath); err != nil {
		c.Edit("Download failed: " + html.EscapeString(err.Error()))
		return
	}

	c.Edit(fmt.Sprintf("<b>%s</b>\n\n%s - uploading...",
		html.EscapeString(sess.Video.Title), f.Quality),
		&tg.SendOptions{ParseMode: "HTML"})

	caption := fmt.Sprintf("<b>%s</b>\n%s - %s",
		html.EscapeString(sess.Video.Title),
		f.Quality, humanBytes(f.Size))

	mediaOpts := &tg.MediaOptions{
		Caption:   caption,
		ParseMode: "HTML",
		FileName:  outputName,
	}
	if f.Kind == "mp3" {
		mediaOpts.MimeType = "audio/mp4"
	} else {
		mediaOpts.MimeType = "video/mp4"
		mediaOpts.Attributes = []tg.DocumentAttribute{
			&tg.DocumentAttributeFilename{FileName: outputName},
			&tg.DocumentAttributeVideo{SupportsStreaming: true},
		}
	}

	client := modules.Client
	if _, err := client.SendMedia(c.ChatID, localPath, mediaOpts); err != nil {
		c.Edit("Upload failed: " + html.EscapeString(err.Error()))
		log.Printf("[ytdl] upload failed: %v", err)
		return
	}
	c.Delete()
}

type ytdlMergeAjax struct {
	Success bool `json:"success"`
	Data    struct {
		Success bool `json:"success"`
		Result  struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Monitor struct {
				HTTP      string `json:"http"`
				WebSocket string `json:"websocket"`
			} `json:"monitor"`
		} `json:"result"`
	} `json:"data"`
}

type ytdlMergeStatus struct {
	Success bool `json:"success"`
	Result  struct {
		JobID                     string `json:"job_id"`
		Status                    string `json:"status"`
		Progress                  int    `json:"progress"`
		FormattedProgressInPercent int   `json:"formatted_progress_in_percent"`
		Output                    *struct {
			Key       string `json:"key"`
			URL       string `json:"url"`
			ExpiresAt string `json:"expiresAt"`
		} `json:"output"`
		Error any `json:"error"`
	} `json:"result"`
}

func runYTDLMerge(ctx context.Context, v *ytdlVideo, f ytdlFormat, videoURL string, c *tg.CallbackQuery) (string, error) {
	renderID := v.VideoID + "_" + f.Quality
	reqBody := map[string]any{
		"id":  renderID,
		"ttl": 3600000,
		"inputs": []any{
			map[string]any{
				"url": videoURL,
				"ext": "mp4",
				"chunkDownload": map[string]any{
					"type":        "header",
					"size":        50 * 1024 * 1024,
					"concurrency": 3,
				},
			},
			map[string]any{
				"url": v.AudioURL,
				"ext": "m4a",
			},
		},
		"output": map[string]any{
			"ext":          "mp4",
			"downloadName": sanitizeFileName(v.Title) + "_" + f.Quality + ".mp4",
			"chunkUpload": map[string]any{
				"size":        200 * 1024 * 1024,
				"concurrency": 3,
			},
		},
		"operation": map[string]any{"type": "replace_audio_in_video"},
	}
	jsonBody, _ := json.Marshal(reqBody)

	form := url.Values{
		"action":       {"process_video_merge"},
		"nonce":        {v.Nonce},
		"request_data": {string(jsonBody)},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ssyouAjaxURL, strings.NewReader(form.Encode()))
	req.Header.Set("user-agent", ssyouUA)
	req.Header.Set("accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("origin", ssyouOrigin)
	req.Header.Set("referer", ssyouDetailURL)
	req.Header.Set("x-wp-nonce", v.Nonce)
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := ytdlHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("ajax: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ajax HTTP %d: %s", resp.StatusCode, snippet(body))
	}
	var envelope ytdlMergeAjax
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", fmt.Errorf("ajax decode: %w", err)
	}
	if !envelope.Success || !envelope.Data.Success {
		return "", fmt.Errorf("ajax rejected: %s", snippet(body))
	}
	statusURL := envelope.Data.Result.Monitor.HTTP
	if statusURL == "" {
		return "", fmt.Errorf("no status url in ajax response")
	}
	// Some monitor URLs come back with http://; upgrade to https so we don't
	// hop networks on every poll.
	statusURL = strings.Replace(statusURL, "http://", "https://", 1)

	deadline := time.Now().Add(ytdlRenderMaxDur)
	var lastProgress = -1
	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("render timed out after %s", ytdlRenderMaxDur)
		}
		s, err := pollYTDLStatus(ctx, statusURL)
		if err != nil {
			// transient network errors during poll; keep trying until deadline
			log.Printf("[ytdl] status poll err: %v", err)
			time.Sleep(ytdlRenderPoll)
			continue
		}
		if s.Result.Status == "done" && s.Result.Output != nil && s.Result.Output.URL != "" {
			return s.Result.Output.URL, nil
		}
		if s.Result.Status == "failed" || s.Result.Status == "error" {
			return "", fmt.Errorf("render status=%s", s.Result.Status)
		}
		if s.Result.Error != nil && fmt.Sprint(s.Result.Error) != "<nil>" && s.Result.Error != "" {
			// pointer-y JSON: only bail if the error field is a non-empty string/object
			if es, ok := s.Result.Error.(string); ok && es != "" {
				return "", fmt.Errorf("render error: %s", es)
			}
		}
		if s.Result.FormattedProgressInPercent != lastProgress {
			lastProgress = s.Result.FormattedProgressInPercent
			if c != nil && lastProgress > 0 {
				c.Edit(fmt.Sprintf("<b>%s</b>\n\n%s - downloading %d%%...",
					html.EscapeString(v.Title), f.Quality, lastProgress),
					&tg.SendOptions{ParseMode: "HTML"})
			}
		}
		time.Sleep(ytdlRenderPoll)
	}
}

func pollYTDLStatus(ctx context.Context, url string) (*ytdlMergeStatus, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("user-agent", ssyouUA)
	req.Header.Set("accept", "application/json")
	resp, err := ytdlHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet(body))
	}
	var s ytdlMergeStatus
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func downloadToFile(ctx context.Context, srcURL, dstPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("user-agent", ssyouUA)
	resp, err := ytdlHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func snippet(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "…"
	}
	return string(b)
}

var reFileNameSanitize = regexp.MustCompile(`[^A-Za-z0-9._\- ]+`)

func sanitizeFileName(name string) string {
	name = reFileNameSanitize.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	if len(name) > 80 {
		name = name[:80]
	}
	if name == "" {
		name = "video"
	}
	return name
}

func registerYTDLHandlers() {
	c := modules.Client
	c.On("cmd:yt", YTDLHandler)
	c.On("cmd:youtube", YTDLHandler)
	c.On("callback:yt:", YTDLCallback)
}

func init() {
	modules.QueueHandlerRegistration(registerYTDLHandlers)
}
