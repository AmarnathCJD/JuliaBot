package extras

import (
	"bytes"
	"encoding/json"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"html"
	"io"
	modules "main/modules"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var charsX = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+/"

type SnapResponse struct {
	Images      []string   `json:"images,omitempty"`
	Videos      []string   `json:"videos,omitempty"`
	Audio       []string   `json:"audio,omitempty"`
	Username    string     `json:"username,omitempty"`
	Description string     `json:"description,omitempty"`
	Statistics  Statistics `json:"statistics,omitempty"`
	Downloads   Downloads  `json:"downloads,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type Statistics struct {
	LikeCount    string `json:"likeCount,omitempty"`
	CommentCount string `json:"commentCount,omitempty"`
	ShareCount   string `json:"shareCount,omitempty"`
}

type Downloads struct {
	AvatarUrl  string `json:"avatarUrl,omitempty"`
	OverlayUrl string `json:"overlayUrl,omitempty"`
	VideoUrl   string `json:"videoUrl,omitempty"`
	MusicUrl   string `json:"musicUrl,omitempty"`
}

func decodeStringX(d string, e, f int) string {
	baseChars := charsX[:e]
	targetChars := charsX[:f]

	reversedString := 0
	for i, char := range reverseX(d) {
		index := strings.IndexRune(baseChars, char)
		if index != -1 {
			reversedString += index * int(math.Pow(float64(e), float64(i)))
		}
	}

	result := ""
	for reversedString > 0 {
		result = string(targetChars[reversedString%f]) + result
		reversedString /= f
	}

	if result == "" {
		return "0"
	}
	return result
}

func reverseX(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func decodeX(encodedString string, _ int, alphabet string, shift, offset int, _ int) string {
	var decoded strings.Builder

	for i := 0; i < len(encodedString); {
		segment := ""
		for i < len(encodedString) && string(encodedString[i]) != string(alphabet[offset]) {
			segment += string(encodedString[i])
			i++
		}
		i++

		for j := 0; j < len(alphabet); j++ {
			segment = strings.ReplaceAll(segment, string(alphabet[j]), strconv.Itoa(j))
		}

		decodedCharCode := decodeStringX(segment, offset, 10)
		decodedValue, _ := strconv.Atoi(decodedCharCode)
		decoded.WriteRune(rune(decodedValue - shift))
	}

	return decoded.String()
}

func extractJSArgs(input string) (string, int, string, int, int, int, error) {
	pattern := `decodeURIComponent\(escape\(r\)\)}\("([^"]*)",(\d+),"([^"]*)",(\d+),(\d+),(\d+)\)`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)

	if len(matches) != 7 {
		return "", 0, "", 0, 0, 0, fmt.Errorf("failed to extract all arguments")
	}

	arg2, _ := strconv.Atoi(matches[2])
	arg4, _ := strconv.Atoi(matches[4])
	arg5, _ := strconv.Atoi(matches[5])
	arg6, _ := strconv.Atoi(matches[6])

	return matches[1], arg2, matches[3], arg4, arg5, arg6, nil
}

func isTikTokURL(urlStr string) bool {
	return strings.Contains(urlStr, "tiktok.com") || strings.Contains(urlStr, "vm.tiktok.com") || strings.Contains(urlStr, "vt.tiktok.com")
}

func FetchInstagramMedia(url string) (*SnapResponse, error) {
	if url == "" {
		return &SnapResponse{Error: "URL cannot be empty"}, nil
	}

	if isTikTokURL(url) {
		return fetchTikTokMedia(url)
	}

	resp, err := fetchInstagramMedia(url)
	if err == nil && resp != nil && resp.Error == "" && (len(resp.Images) > 0 || len(resp.Videos) > 0) {
		return resp, nil
	}
	fallback, fErr := fetchInstagramMediaFastdl(url)
	if fErr == nil && fallback != nil && fallback.Error == "" && (len(fallback.Images) > 0 || len(fallback.Videos) > 0) {
		return fallback, nil
	}
	if fallback != nil && fallback.Error != "" {
		return fallback, nil
	}
	return resp, err
}

func fetchInstagramMedia(url string) (*SnapResponse, error) {
	hostUrl := "https://snapsave.app/action.php?lang=en"

	headers := map[string]string{
		"accept":          "*/*",
		"accept-language": "en-US,en;q=0.9",
		"origin":          "https://snapsave.app",
		"referer":         "https://snapsave.app/",
		"user-agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	}

	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)
	formField, _ := writer.CreateFormField("url")
	formField.Write([]byte(strings.TrimSpace(url)))
	writer.Close()

	client := &http.Client{}
	req, _ := http.NewRequest("POST", hostUrl, form)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return &SnapResponse{Error: fmt.Sprintf("failed to fetch URL: %v", err)}, nil
	}
	defer resp.Body.Close()
	bodyText, _ := io.ReadAll(resp.Body)

	encodedString, base, alphabet, shift, offset, length, err := extractJSArgs(string(bodyText))
	if err != nil {
		return &SnapResponse{Error: "failed to extract media info"}, nil
	}

	result := decodeX(encodedString, base, alphabet, shift, offset, length)
	if result == "" {
		return &SnapResponse{Error: "failed to decode media info"}, nil
	}

	if strings.Contains(result, "error_api_get_instagram") || strings.Contains(result, "Unable to connect to Instagram") {
		return &SnapResponse{Error: "could not fetch post (may be private, deleted, or age-restricted)"}, nil
	}

	pattern := regexp.MustCompile(`class=\\"icon\s+icon-dl(image|video)\\"[^>]*>.*?<a[^>]+href=\\"([^"]+)\\"`)
	matches := pattern.FindAllStringSubmatch(result, -1)

	if len(matches) == 0 {
		return &SnapResponse{Error: "failed to extract download URLs"}, nil
	}

	var images, videos []string
	for _, match := range matches {
		if len(match) >= 3 {
			mediaType := match[1]
			mediaURL := strings.ReplaceAll(match[2], "\\", "")

			switch mediaType {
			case "image":
				images = append(images, mediaURL)
			case "video":
				videos = append(videos, mediaURL)
			}
		}
	}

	return &SnapResponse{
		Images: images,
		Videos: videos,
	}, nil
}

type fastdlResp struct {
	Status string `json:"status"`
	Mess   string `json:"mess"`
	Data   string `json:"data"`
}

func fetchInstagramMediaFastdl(target string) (*SnapResponse, error) {
	form := url.Values{}
	form.Set("q", strings.TrimSpace(target))
	form.Set("t", "media")
	form.Set("lang", "en")
	form.Set("v", "v2")
	form.Set("html", "")

	req, err := http.NewRequest("POST", "https://fastdl.to/api/ajaxSearch", strings.NewReader(form.Encode()))
	if err != nil {
		return &SnapResponse{Error: "failed to fetch media"}, nil
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://fastdl.to")
	req.Header.Set("Referer", "https://fastdl.to/en")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &SnapResponse{Error: "failed to fetch media"}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var parsed fastdlResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return &SnapResponse{Error: "invalid response from provider"}, nil
	}
	if parsed.Status != "ok" {
		return &SnapResponse{Error: "provider request failed"}, nil
	}
	if parsed.Mess != "" {
		clean := sanitizeProviderMessage(stripHTMLTags(parsed.Mess))
		return &SnapResponse{Error: clean}, nil
	}

	itemRe := regexp.MustCompile(`(?s)<li>\s*<div class="download-items">(.*?)</li>`)
	iconRe := regexp.MustCompile(`icon icon-dl(image|video)`)
	hrefRe := regexp.MustCompile(`href="(https://dl\.snapcdn\.app/[^"]+)"`)

	var images, videos []string
	for _, item := range itemRe.FindAllStringSubmatch(parsed.Data, -1) {
		block := item[1]
		iconMatch := iconRe.FindStringSubmatch(block)
		hrefMatch := hrefRe.FindStringSubmatch(block)
		if iconMatch == nil || hrefMatch == nil {
			continue
		}
		mediaURL := html.UnescapeString(hrefMatch[1])
		switch iconMatch[1] {
		case "image":
			images = append(images, mediaURL)
		case "video":
			videos = append(videos, mediaURL)
		}
	}

	if len(images) == 0 && len(videos) == 0 {
		return &SnapResponse{Error: "fastdl: no media found"}, nil
	}
	return &SnapResponse{Images: images, Videos: videos}, nil
}

func stripHTMLTags(s string) string {
	return regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
}

var providerBrandRe = regexp.MustCompile(`(?i)\b(fastdl|snapsave|saveinsta|ssstik|snapcdn)(\.(app|to|io|com|net))?\b`)

func sanitizeProviderMessage(s string) string {
	cleaned := providerBrandRe.ReplaceAllString(s, "")
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "media unavailable"
	}
	return cleaned
}

func fetchTikTokMedia(urx string) (*SnapResponse, error) {
	headers := map[string]string{
		"accept":             "*/*",
		"accept-language":    "en-US,en;q=0.9",
		"content-type":       "application/x-www-form-urlencoded",
		"dnt":                "1",
		"hx-current-url":     "https://ssstik.io/in",
		"hx-request":         "true",
		"hx-target":          "target",
		"hx-trigger":         "_gcaptcha_pt",
		"origin":             "https://ssstik.io",
		"priority":           "u=1, i",
		"referer":            "https://ssstik.io/in",
		"sec-ch-ua":          `"Google Chrome";v="143", "Chromium";v="143", "Not A(Brand";v="24"`,
		"sec-ch-ua-mobile":   "?1",
		"sec-ch-ua-platform": `"Android"`,
		"sec-fetch-dest":     "empty",
		"sec-fetch-mode":     "cors",
		"sec-fetch-site":     "same-origin",
		"user-agent":         "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Mobile Safari/537.36",
	}

	client := &http.Client{}
	getReq, _ := http.NewRequest("GET", "https://ssstik.io/in", nil)
	for key, value := range headers {
		getReq.Header.Set(key, value)
	}

	getResp, err := client.Do(getReq)
	if err != nil {
		return &SnapResponse{Error: fmt.Sprintf("failed to fetch ssstik.io/in: %v", err)}, nil
	}
	defer getResp.Body.Close()
	getBody, _ := io.ReadAll(getResp.Body)

	ttPattern := regexp.MustCompile(`s_tt\s*=\s*'([^']+)'`)
	ttMatches := ttPattern.FindStringSubmatch(string(getBody))

	tt := "aVYybTc_"
	if len(ttMatches) >= 2 {
		tt = ttMatches[1]
	}

	formValues := url.Values{}
	formValues.Set("id", urx)
	formValues.Set("locale", "in")
	formValues.Set("tt", tt)

	formDataStr := formValues.Encode()
	req, _ := http.NewRequest("POST", "https://ssstik.io/abc?url=dl", strings.NewReader(formDataStr))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return &SnapResponse{Error: fmt.Sprintf("failed to fetch video info: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &SnapResponse{Error: fmt.Sprintf("received non-200 response: %d", resp.StatusCode)}, nil
	}
	bodyText, _ := io.ReadAll(resp.Body)

	result := string(bodyText)
	if result == "" {
		return &SnapResponse{Error: "failed to get response - response is empty"}, nil
	}

	pattern := regexp.MustCompile(`<a\s+[^>]*href="([^"]+)"[^>]*class="[^"]*download_link[^"]*"`)
	matches := pattern.FindAllStringSubmatch(result, -1)

	if len(matches) == 0 {
		return &SnapResponse{Error: "no media found"}, nil
	}

	var images, videos, audio []string
	for _, match := range matches {
		if len(match) >= 2 {
			mediaURL := match[1]

			switch {
			case strings.Contains(mediaURL, "tikcdn.io/ssstik/s/"):
				images = append(images, mediaURL)
			case strings.Contains(mediaURL, "tikcdn.io/ssstik/m/"):
				audio = append(audio, mediaURL)
			case strings.Contains(mediaURL, "tikcdn.io/ssstik/"):
				videos = append(videos, mediaURL)
			}
		}
	}

	if len(images) > 0 || len(videos) > 0 || len(audio) > 0 {
		return &SnapResponse{
			Images: images,
			Videos: videos,
			Audio:  audio,
		}, nil
	}

	return &SnapResponse{Error: "failed to extract media URLs"}, nil
}

type snapMediaItem struct {
	path      string
	mimeType  string
	isVideo   bool
	duration  int
	width     int
	height    int
	audioSeen bool
}

func (s *snapMediaItem) hasAudio() bool { return s.audioSeen }

func downloadURL(target string) ([]byte, error) {
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)")
	req.Header.Set("referer", "https://fastdl.to/")
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func detectMimeAndExt(data []byte) (mime, ext string) {
	mime = http.DetectContentType(data)
	switch mime {
	case "image/jpeg":
		return mime, "jpg"
	case "image/png":
		return mime, "png"
	case "image/gif":
		return mime, "gif"
	case "image/webp":
		return mime, "webp"
	case "video/mp4":
		return mime, "mp4"
	case "video/webm":
		return mime, "webm"
	case "video/quicktime":
		return mime, "mov"
	case "audio/mpeg":
		return mime, "mp3"
	case "audio/mp4":
		return mime, "m4a"
	}
	// Heuristics for MP4 containers (ftyp box) that DetectContentType misses.
	if len(data) >= 12 && string(data[4:8]) == "ftyp" {
		return "video/mp4", "mp4"
	}
	return mime, "bin"
}

func probeVideo(path string) (durationSec, width, height int, hasAudio bool) {
	cmd := exec.Command("ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height:format=duration",
		"-of", "default=noprint_wrappers=1:nokey=0",
		path)
	out, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "width="):
				width, _ = strconv.Atoi(strings.TrimPrefix(line, "width="))
			case strings.HasPrefix(line, "height="):
				height, _ = strconv.Atoi(strings.TrimPrefix(line, "height="))
			case strings.HasPrefix(line, "duration="):
				f, _ := strconv.ParseFloat(strings.TrimPrefix(line, "duration="), 64)
				durationSec = int(f + 0.5)
			}
		}
	}
	aCmd := exec.Command("ffprobe", "-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		path)
	aOut, err := aCmd.Output()
	if err == nil {
		hasAudio = strings.Contains(string(aOut), "audio")
	}
	return
}

func writeMediaToFile(data []byte, prefix string) (*snapMediaItem, error) {
	mime, ext := detectMimeAndExt(data)
	tmp, err := os.CreateTemp("", prefix+"_*."+ext)
	if err != nil {
		return nil, err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return nil, err
	}
	tmp.Close()

	item := &snapMediaItem{path: tmp.Name(), mimeType: mime}
	if strings.HasPrefix(mime, "video/") {
		item.isVideo = true
		item.duration, item.width, item.height, item.audioSeen = probeVideo(tmp.Name())
	}
	return item, nil
}

func downloadAllAsFiles(urls []string, cap int, prefix string) []*snapMediaItem {
	if cap > 0 && len(urls) > cap {
		urls = urls[:cap]
	}
	out := make([]*snapMediaItem, 0, len(urls))
	for _, u := range urls {
		data, err := downloadURL(u)
		if err != nil || len(data) == 0 {
			continue
		}
		item, err := writeMediaToFile(data, prefix)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func removeItems(items []*snapMediaItem) {
	for _, it := range items {
		if it != nil && it.path != "" {
			os.Remove(it.path)
		}
	}
}

func mediaOptsFor(item *snapMediaItem) *tg.MediaOptions {
	fileName := filepath.Base(item.path)
	opts := &tg.MediaOptions{
		FileName: fileName,
		MimeType: item.mimeType,
	}
	if item.isVideo {
		opts.Attributes = []tg.DocumentAttribute{
			&tg.DocumentAttributeVideo{
				Duration:          float64(item.duration),
				W:                 int32(item.width),
				H:                 int32(item.height),
				SupportsStreaming: true,
			},
			&tg.DocumentAttributeFilename{FileName: fileName},
		}
	} else {
		opts.Attributes = []tg.DocumentAttribute{
			&tg.DocumentAttributeFilename{FileName: fileName},
		}
	}
	return opts
}

func sendAsAlbums(m *tg.NewMessage, items []*snapMediaItem) {
	const batch = 10
	for i := 0; i < len(items); i += batch {
		end := min(i+batch, len(items))
		chunk := items[i:end]
		if len(chunk) == 1 {
			m.ReplyMedia(chunk[0].path, mediaOptsFor(chunk[0]))
			continue
		}
		paths := make([]string, len(chunk))
		hasVideo := false
		for i, it := range chunk {
			paths[i] = it.path
			if it.isVideo {
				hasVideo = true
			}
		}
		albumOpts := &tg.MediaOptions{}
		if hasVideo {
			albumOpts.Attributes = []tg.DocumentAttribute{
				&tg.DocumentAttributeVideo{SupportsStreaming: true},
			}
		}
		if _, err := m.ReplyAlbum(paths, albumOpts); err != nil {
			for _, it := range chunk {
				m.ReplyMedia(it.path, mediaOptsFor(it))
			}
		}
	}
}

func muxVideoFileWithAudio(videoPath string, audioBytes []byte) (*snapMediaItem, error) {
	aTmp, err := os.CreateTemp("", "aud_*.mp3")
	if err != nil {
		return nil, err
	}
	defer os.Remove(aTmp.Name())
	if _, err := aTmp.Write(audioBytes); err != nil {
		aTmp.Close()
		return nil, err
	}
	aTmp.Close()

	outTmp, err := os.CreateTemp("", "muxed_*.mp4")
	if err != nil {
		return nil, err
	}
	outTmp.Close()
	outPath := outTmp.Name()

	cmd := exec.Command("ffmpeg", "-y",
		"-i", videoPath,
		"-i", aTmp.Name(),
		"-map", "0:v:0", "-map", "1:a:0",
		"-c:v", "copy",
		"-c:a", "aac", "-b:a", "128k",
		"-shortest",
		"-movflags", "+faststart",
		outPath,
	)
	if err := cmd.Run(); err != nil {
		os.Remove(outPath)
		return nil, err
	}
	item := &snapMediaItem{path: outPath, mimeType: "video/mp4", isVideo: true}
	item.duration, item.width, item.height, item.audioSeen = probeVideo(outPath)
	return item, nil
}

func InstaHandler(m *tg.NewMessage) error {
	if m.Args() == "" {
		m.Reply("Usage: /snap <Instagram or TikTok URL>")
		return nil
	}

	msg, _ := m.Reply("Processing request...")

	resp, err := FetchInstagramMedia(m.Args())
	if err != nil || resp.Error != "" {
		msg.Edit("Failed to process URL")
		return nil
	}

	if len(resp.Images) == 0 && len(resp.Videos) == 0 && len(resp.Audio) == 0 {
		msg.Edit("No media found")
		return nil
	}

	defer msg.Delete()
	msg.Edit(fmt.Sprintf("Found %d image(s), %d video(s). Downloading...", len(resp.Images), len(resp.Videos)))

	isTikTok := isTikTokURL(m.Args())

	if len(resp.Videos) > 0 {
		vids := downloadAllAsFiles(resp.Videos, 20, "vid")
		defer removeItems(vids)

		// TikTok: if the video has no audio track and provider gave a separate audio stream, mux them.
		if isTikTok && len(vids) == 1 && len(resp.Audio) > 0 && !vids[0].hasAudio() {
			if audioData, aerr := downloadURL(resp.Audio[0]); aerr == nil {
				if muxed, err := muxVideoFileWithAudio(vids[0].path, audioData); err == nil {
					os.Remove(vids[0].path)
					vids[0] = muxed
				}
			}
		}

		sendAsAlbums(m, vids)
	}

	if len(resp.Images) > 0 {
		imgs := downloadAllAsFiles(resp.Images, 20, "img")
		defer removeItems(imgs)
		sendAsAlbums(m, imgs)
	}

	return nil
}

func registerInstaHandlers() {
	c := modules.Client
	c.On("cmd:snap", InstaHandler)
	c.On("cmd:insta", InstaHandler)
	c.On("cmd:tik", InstaHandler)
}

func initFromSrc_insta_0_1() {
	modules.QueueHandlerRegistration(registerInstaHandlers)
}
var igOgMetaPattern = regexp.MustCompile(`<meta\s+property="og:([a-z_]+)"\s+content="([^"]*)"`)
var igURLPattern = regexp.MustCompile(`https?://(?:www\.)?instagram\.com/[^\s]+`)

type igMeta struct {
	Title       string
	Description string
	Image       string
	URL         string
	SiteName    string
	Type        string
}

func fetchInstagramMeta(target string) (*igMeta, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	matches := igOgMetaPattern.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no og metadata found")
	}
	meta := &igMeta{}
	for _, mt := range matches {
		key := mt[1]
		val := html.UnescapeString(mt[2])
		switch key {
		case "title":
			meta.Title = val
		case "description":
			meta.Description = val
		case "image":
			meta.Image = val
		case "url":
			meta.URL = val
		case "site_name":
			meta.SiteName = val
		case "type":
			meta.Type = val
		}
	}
	if meta.Title == "" && meta.Description == "" && meta.Image == "" {
		return nil, fmt.Errorf("no og metadata found")
	}
	return meta, nil
}

func normalizeIgURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if m := igURLPattern.FindString(raw); m != "" {
		return m
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		if strings.HasPrefix(raw, "instagram.com") || strings.HasPrefix(raw, "www.instagram.com") {
			return "https://" + raw
		}
	}
	if strings.Contains(raw, "instagram.com") {
		return raw
	}
	return ""
}

func IgMetaHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/igmeta &lt;instagram url&gt;</code>")
		return nil
	}
	target := normalizeIgURL(arg)
	if target == "" {
		m.Reply("<b>Not a valid instagram.com URL.</b>")
		return nil
	}
	status, _ := m.Reply("Fetching <code>" + html.EscapeString(target) + "</code>...")
	meta, err := fetchInstagramMeta(target)
	if err != nil {
		status.Edit("<b>Failed:</b> " + html.EscapeString(err.Error()) + "\n<i>Instagram may require login for posts/reels. Try a profile URL.</i>")
		return nil
	}
	var b strings.Builder
	b.WriteString("<b>Instagram Metadata</b>\n")
	if meta.Title != "" {
		b.WriteString("\n<b>Title:</b> ")
		b.WriteString(html.EscapeString(meta.Title))
	}
	if meta.Type != "" {
		b.WriteString("\n<b>Type:</b> ")
		b.WriteString(html.EscapeString(meta.Type))
	}
	if meta.SiteName != "" {
		b.WriteString("\n<b>Site:</b> ")
		b.WriteString(html.EscapeString(meta.SiteName))
	}
	if meta.Description != "" {
		b.WriteString("\n<b>Description:</b> ")
		b.WriteString(html.EscapeString(meta.Description))
	}
	if meta.URL != "" {
		b.WriteString("\n<b>URL:</b> <a href=\"")
		b.WriteString(html.EscapeString(meta.URL))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(meta.URL))
		b.WriteString("</a>")
	}
	caption := b.String()
	if meta.Image != "" {
		if _, err := m.ReplyMedia(meta.Image, &tg.MediaOptions{Caption: caption}); err != nil {
			status.Edit(caption+"\n\n<b>Image:</b> <a href=\""+html.EscapeString(meta.Image)+"\">link</a>", &tg.SendOptions{LinkPreview: false})
			return nil
		}
		status.Delete()
		return nil
	}
	status.Edit(caption, &tg.SendOptions{LinkPreview: false})
	return nil
}

func registerIgMetaHandlers() {
	c := modules.Client
	c.On("cmd:igmeta", IgMetaHandler)
}

func initFromSrc_instagram_oembed_1_1() { modules.QueueHandlerRegistration(registerIgMetaHandlers) }

func initFromSrc_instagram_0_1() {
	initFromSrc_insta_0_1()
	initFromSrc_instagram_oembed_1_1()
}
var tweetHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

type tweetOEmbed struct {
	URL          string `json:"url"`
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	HTML         string `json:"html"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	Type         string `json:"type"`
}

var tweetURLRegex = regexp.MustCompile(`https?://(?:www\.)?(?:x|twitter)\.com/[A-Za-z0-9_]+/status/\d+`)
var tweetTagRegex = regexp.MustCompile(`<[^>]+>`)
var tweetScriptRegex = regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
var tweetWSRegex = regexp.MustCompile(`\s+`)

func tweetExtractText(rawHTML string) string {
	s := tweetScriptRegex.ReplaceAllString(rawHTML, "")
	start := strings.Index(s, "<p")
	if start == -1 {
		return ""
	}
	close := strings.Index(s[start:], "</p>")
	if close == -1 {
		return ""
	}
	inner := s[start : start+close]
	gt := strings.Index(inner, ">")
	if gt == -1 {
		return ""
	}
	inner = inner[gt+1:]
	inner = strings.ReplaceAll(inner, "<br>", "\n")
	inner = strings.ReplaceAll(inner, "<br/>", "\n")
	inner = strings.ReplaceAll(inner, "<br />", "\n")
	inner = tweetTagRegex.ReplaceAllString(inner, "")
	inner = html.UnescapeString(inner)
	inner = strings.TrimSpace(inner)
	return inner
}

func tweetFetchOEmbed(tweetURL string) (*tweetOEmbed, error) {
	endpoint := "https://publish.twitter.com/oembed?url=" + url.QueryEscape(tweetURL)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := tweetHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oembed status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "{") {
		return nil, fmt.Errorf("tweet not found or unavailable")
	}
	var data tweetOEmbed
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if data.AuthorName == "" && data.HTML == "" {
		return nil, fmt.Errorf("empty oembed response")
	}
	return &data, nil
}

func TweetHandler(m *tg.NewMessage) error {
	target := strings.TrimSpace(m.Args())
	if target == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			target = strings.TrimSpace(r.Text())
		}
	}
	if target == "" {
		m.Reply("usage: /tweet &lt;url&gt;")
		return nil
	}
	match := tweetURLRegex.FindString(target)
	if match == "" {
		m.Reply("<b>Error:</b> please provide a valid x.com or twitter.com status URL.")
		return nil
	}

	data, err := tweetFetchOEmbed(match)
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}

	text := tweetExtractText(data.HTML)
	text = tweetWSRegex.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	var b strings.Builder
	b.WriteString("<b>Tweet</b>\n\n")
	if data.AuthorName != "" {
		if data.AuthorURL != "" {
			b.WriteString(fmt.Sprintf("<b>Author:</b> <a href=\"%s\">%s</a>\n", html.EscapeString(data.AuthorURL), html.EscapeString(data.AuthorName)))
		} else {
			b.WriteString("<b>Author:</b> ")
			b.WriteString(html.EscapeString(data.AuthorName))
			b.WriteString("\n")
		}
	}
	if text != "" {
		b.WriteString("\n")
		b.WriteString(html.EscapeString(text))
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("\n<a href=\"%s\">View on X</a>", html.EscapeString(data.URL)))

	m.Reply(b.String(), &tg.SendOptions{LinkPreview: false})
	return nil
}

func initFromSrc_twitter_thread_1_1() { modules.QueueHandlerRegistration(registerTweetHandlers) }
func registerTweetHandlers() {
	c := modules.Client
	c.On("cmd:tweet", TweetHandler)
}
type itunesTrack struct {
	ArtistName       string  `json:"artistName"`
	CollectionName   string  `json:"collectionName"`
	TrackName        string  `json:"trackName"`
	TrackViewURL     string  `json:"trackViewUrl"`
	PreviewURL       string  `json:"previewUrl"`
	ArtworkURL100    string  `json:"artworkUrl100"`
	ReleaseDate      string  `json:"releaseDate"`
	PrimaryGenreName string  `json:"primaryGenreName"`
	TrackTimeMillis  int     `json:"trackTimeMillis"`
	TrackPrice       float64 `json:"trackPrice"`
	Currency         string  `json:"currency"`
	Country          string  `json:"country"`
}

type itunesSearchResponse struct {
	ResultCount int           `json:"resultCount"`
	Results     []itunesTrack `json:"results"`
}

type spotifyLyricsOvhResponse struct {
	Lyrics string `json:"lyrics"`
	Error  string `json:"error"`
}

func formatTrackDuration(ms int) string {
	if ms <= 0 {
		return ""
	}
	total := ms / 1000
	mins := total / 60
	secs := total % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}

func upgradeArtwork(u string) string {
	if u == "" {
		return u
	}
	return strings.Replace(u, "/100x100bb.jpg", "/600x600bb.jpg", 1)
}

func fetchLyricsSnippet(artist, title string) string {
	client := &http.Client{Timeout: 12 * time.Second}
	endpoint := "https://api.lyrics.ovh/v1/" + url.PathEscape(artist) + "/" + url.PathEscape(title)
	resp, err := client.Get(endpoint)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	var data spotifyLyricsOvhResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	lyrics := strings.TrimSpace(data.Lyrics)
	if lyrics == "" {
		return ""
	}
	lines := strings.Split(lyrics, "\n")
	var picked []string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		picked = append(picked, ln)
		if len(picked) >= 6 {
			break
		}
	}
	if len(picked) == 0 {
		return ""
	}
	return strings.Join(picked, "\n")
}

func SpotifyMetaHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("<b>Usage:</b> <code>/spotify &lt;track query&gt;</code>\n<i>Example:</i> <code>/spotify imagine dragons believer</code>")
		return nil
	}
	status, _ := m.Reply("Searching <code>" + html.EscapeString(query) + "</code>...")
	client := &http.Client{Timeout: 20 * time.Second}
	endpoint := "https://itunes.apple.com/search?media=music&entity=song&limit=1&term=" + url.QueryEscape(query)
	resp, err := client.Get(endpoint)
	if err != nil {
		status.Edit("couldn't search track: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		status.Edit(fmt.Sprintf("iTunes API HTTP %d", resp.StatusCode))
		return nil
	}
	var data itunesSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		status.Edit("couldn't parse response: " + html.EscapeString(err.Error()))
		return nil
	}
	if data.ResultCount == 0 || len(data.Results) == 0 {
		status.Edit("<b>No track found for:</b> <code>" + html.EscapeString(query) + "</code>")
		return nil
	}
	track := data.Results[0]

	var lyrics string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		lyrics = fetchLyricsSnippet(track.ArtistName, track.TrackName)
	}()
	wg.Wait()

	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(track.TrackName))
	b.WriteString("</b>\n")
	b.WriteString("<i>by </i><b>")
	b.WriteString(html.EscapeString(track.ArtistName))
	b.WriteString("</b>")
	if strings.TrimSpace(track.CollectionName) != "" {
		b.WriteString("\n<b>Album:</b> ")
		b.WriteString(html.EscapeString(track.CollectionName))
	}
	if strings.TrimSpace(track.PrimaryGenreName) != "" {
		b.WriteString("\n<b>Genre:</b> ")
		b.WriteString(html.EscapeString(track.PrimaryGenreName))
	}
	if dur := formatTrackDuration(track.TrackTimeMillis); dur != "" {
		b.WriteString("\n<b>Duration:</b> ")
		b.WriteString(dur)
	}
	if len(track.ReleaseDate) >= 10 {
		b.WriteString("\n<b>Released:</b> ")
		b.WriteString(html.EscapeString(track.ReleaseDate[:10]))
	}
	if track.TrackPrice > 0 && strings.TrimSpace(track.Currency) != "" {
		b.WriteString(fmt.Sprintf("\n<b>Price:</b> %.2f %s", track.TrackPrice, html.EscapeString(track.Currency)))
	}
	b.WriteString("\n")
	if strings.TrimSpace(track.PreviewURL) != "" {
		b.WriteString("\n<a href=\"")
		b.WriteString(html.EscapeString(track.PreviewURL))
		b.WriteString("\">Preview</a>")
	}
	if strings.TrimSpace(track.TrackViewURL) != "" {
		b.WriteString(" | <a href=\"")
		b.WriteString(html.EscapeString(track.TrackViewURL))
		b.WriteString("\">Apple Music</a>")
	}
	spotifySearch := "https://open.spotify.com/search/" + url.PathEscape(track.ArtistName+" "+track.TrackName)
	b.WriteString(" | <a href=\"")
	b.WriteString(html.EscapeString(spotifySearch))
	b.WriteString("\">Spotify Search</a>")
	if lyrics != "" {
		b.WriteString("\n\n<b>Lyrics snippet:</b>\n<blockquote expandable>")
		b.WriteString(html.EscapeString(lyrics))
		b.WriteString("</blockquote>")
	}

	caption := b.String()
	artwork := upgradeArtwork(track.ArtworkURL100)
	if strings.TrimSpace(artwork) != "" {
		if _, err := m.ReplyMedia(artwork, &tg.MediaOptions{Caption: caption}); err != nil {
			status.Edit(caption, &tg.SendOptions{LinkPreview: false})
			return nil
		}
		status.Delete()
		return nil
	}
	status.Edit(caption, &tg.SendOptions{LinkPreview: false})
	return nil
}

func initFromSrc_spotify_meta_2_1() { modules.QueueHandlerRegistration(registerSpotifyMetaHandlers) }
func registerSpotifyMetaHandlers() {
	c := modules.Client
	c.On("cmd:spotify", SpotifyMetaHandler)
}

func init() {
	initFromSrc_instagram_0_1()
	initFromSrc_twitter_thread_1_1()
	initFromSrc_spotify_meta_2_1()
}
