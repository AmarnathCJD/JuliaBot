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
	return fetchInstagramMedia(url)
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

	var images, videos []string
	for _, match := range matches {
		if len(match) >= 2 {
			mediaURL := match[1]

			if strings.Contains(mediaURL, "tikcdn.io/ssstik/s/") {
				images = append(images, mediaURL)
			} else if strings.Contains(mediaURL, "tikcdn.io/ssstik/") {
				videos = append(videos, mediaURL)
			}
		}
	}

	if len(images) > 0 || len(videos) > 0 {
		return &SnapResponse{
			Images: images,
			Videos: videos,
		}, nil
	}

	return &SnapResponse{Error: "failed to extract media URLs"}, nil
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

	if len(resp.Images) == 0 && len(resp.Videos) == 0 {
		msg.Edit("No media found")
		return nil
	}

	defer msg.Delete()
	msg.Edit(fmt.Sprintf("Found %d image(s) and %d video(s). Downloading...", len(resp.Images), len(resp.Videos)))

	if len(resp.Videos) > 0 {
		for _, vid := range resp.Videos[:minVal(len(resp.Videos), 10)] {
			req, _ := http.NewRequest("GET", vid, nil)
			req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}

			fileBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			mimeType := http.DetectContentType(fileBytes)
			ext := ""
			switch mimeType {
			case "video/mp4":
				ext = "mp4"
			case "video/webm":
				ext = "webm"
			case "audio/mpeg":
				ext = "mp3"
			default:
				fmt.Println("Unknown MIME type:", mimeType)
				ext = "mp4"
			}

			filename := fmt.Sprintf("insta_video_%d.%s", time.Now().UnixNano(), ext)
			m.ReplyMedia(fileBytes, &tg.MediaOptions{
				FileName: filename,
				MimeType: mimeType,
			})
		}
	}

	if len(resp.Images) > 0 {
		for _, img := range resp.Images[:minVal(len(resp.Images), 10)] {
			req, _ := http.NewRequest("GET", img, nil)
			req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}

			fileBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			mimeType := http.DetectContentType(fileBytes)
			ext := ""
			switch mimeType {
			case "image/jpeg":
				ext = "jpg"
			case "image/png":
				ext = "png"
			case "image/gif":
				ext = "gif"
			default:
				fmt.Println("Unknown MIME type:", mimeType)
				ext = "jpg"
			}

			filename := fmt.Sprintf("insta_image_%d.%s", time.Now().UnixNano(), ext)
			m.ReplyMedia(fileBytes, &tg.MediaOptions{
				FileName: filename,
				MimeType: mimeType,
			})
		}
	}

	return nil
}

func minVal(a, b int) int {
	if a < b {
		return a
	}
	return b
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
