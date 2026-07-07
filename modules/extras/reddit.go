package extras

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	nhttp "net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	modules "main/modules"

	tg "github.com/amarnathcjd/gogram/telegram"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

var redditUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

type redditComment struct {
	ID         string `json:"id"`
	Author     string `json:"author"`
	Score      int    `json:"score"`
	Body       string `json:"body"`
	CreatedISO string `json:"created_iso"`
	Permalink  string `json:"permalink"`
}

type redditPost struct {
	Title       string          `json:"title"`
	Subreddit   string          `json:"subreddit"`
	Author      string          `json:"author"`
	Score       int             `json:"score"`
	NumComments int             `json:"num_comments"`
	CreatedISO  string          `json:"created_iso"`
	URL         string          `json:"url"`
	Selftext    string          `json:"selftext"`
	Permalink   string          `json:"permalink"`
	MediaURLs   []string        `json:"media_urls"`
	Comments    []redditComment `json:"comments"`
	Source      string          `json:"source"`
	HTML        string          `json:"-"`
}

func redditNewClient() (tls_client.HttpClient, error) {
	return tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_124),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
	)
}

func redditChromeHeaders() http.Header {
	h := http.Header{
		"sec-ch-ua":                 {`"Chromium";v="124", "Google Chrome";v="124", "Not_A Brand";v="99"`},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"Upgrade-Insecure-Requests": {"1"},
		"User-Agent":                {redditUA},
		"Accept": {"text/html,application/xhtml+xml,application/xml;q=0.9," +
			"image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"Sec-Fetch-Site":  {"none"},
		"Sec-Fetch-Mode":  {"navigate"},
		"Sec-Fetch-User":  {"?1"},
		"Sec-Fetch-Dest":  {"document"},
		"Accept-Encoding": {"gzip, deflate, br, zstd"},
		"Accept-Language": {"en-US,en;q=0.9"},
		"Referer":         {"https://www.google.com/"},
	}
	h[http.HeaderOrderKey] = []string{
		"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
		"upgrade-insecure-requests", "user-agent", "accept",
		"sec-fetch-site", "sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest",
		"accept-encoding", "accept-language", "referer",
	}
	return h
}

func redditNewReq(u string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header = redditChromeHeaders()
	return req, nil
}

// toOldRedditURL rewrites any reddit URL to old.reddit.com and strips a
// trailing .json suffix. old.reddit's HTML endpoint bypasses Reddit's Fastly
// WAF for browser-shaped requests; the .json endpoint does not.
func toOldRedditURL(raw string) (string, error) {
	if !strings.Contains(raw, "://") {
		trimmed := strings.TrimLeft(raw, "/")
		for _, host := range []string{"www.reddit.com/", "old.reddit.com/",
			"m.reddit.com/", "np.reddit.com/", "reddit.com/"} {
			if strings.HasPrefix(trimmed, host) {
				trimmed = strings.TrimPrefix(trimmed, host)
				break
			}
		}
		raw = "https://old.reddit.com/" + trimmed
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if strings.Contains(strings.ToLower(u.Host), "reddit.com") {
		u.Host = "old.reddit.com"
		u.Scheme = "https"
	}
	p := u.Path
	if strings.HasSuffix(p, ".json") {
		p = strings.TrimSuffix(p, ".json")
	}
	if strings.HasSuffix(p, ".json/") {
		p = strings.TrimSuffix(p, ".json/") + "/"
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	u.Path = p
	u.Fragment = ""
	return u.String(), nil
}

func redditFetchHTML(u string) (string, error) {
	client, err := redditNewClient()
	if err != nil {
		return "", err
	}
	target, err := toOldRedditURL(u)
	if err != nil {
		return "", err
	}
	req, err := redditNewReq(target)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		head := string(body)
		if len(head) > 200 {
			head = head[:200]
		}
		return "", fmt.Errorf("reddit fetch: HTTP %d for %s (body head: %q)",
			resp.StatusCode, target, head)
	}
	s := string(body)
	if strings.Contains(redditClip(s, 2000), "<title>Blocked</title>") {
		return "", fmt.Errorf("reddit served 'Blocked' page (200) for %s", target)
	}
	return s, nil
}

func redditFetch(u string) (*redditPost, error) {
	htmlStr, err := redditFetchHTML(u)
	if err != nil {
		return nil, err
	}
	post := redditParseThread(htmlStr)
	post.HTML = htmlStr
	src, _ := toOldRedditURL(u)
	post.Source = src
	return post, nil
}

var (
	redditTagRE          = regexp.MustCompile(`<[^>]+>`)
	redditTitleRE        = regexp.MustCompile(`(?s)<a[^>]+class="title[^"]*"[^>]*>(.*?)</a>`)
	redditSubRE          = regexp.MustCompile(`/r/([A-Za-z0-9_]+)/`)
	redditCanonicalRE    = regexp.MustCompile(`<link\s+rel="canonical"\s+href="([^"]+)"`)
	redditAuthorRE       = regexp.MustCompile(`<a[^>]+class="author[^"]*"[^>]*>([^<]+)</a>`)
	redditScoreTitleRE   = regexp.MustCompile(`<div[^>]+class="score unvoted"[^>]*title="([^"]+)"`)
	redditScoreSpanRE    = regexp.MustCompile(`<span[^>]+class="score unvoted"[^>]*>([^<]+)</span>`)
	redditOpThingRE      = regexp.MustCompile(`(?s)<div[^>]+id="thing_t3_[^"]+"[^>]*>`)
	redditDataURLRE      = regexp.MustCompile(`<div[^>]+id="thing_t3_[^"]+"[^>]*data-url="([^"]+)"`)
	redditUsertextBodyRE = regexp.MustCompile(`(?s)<div\s+class="usertext-body[^"]*"[^>]*>\s*<div\s+class="md"[^>]*>(.*?)</div>\s*</div>`)
	redditTimeRE         = regexp.MustCompile(`<time[^>]+datetime="([^"]+)"`)
	redditDataCommentsRE = regexp.MustCompile(`data-comments-count="(\d+)"`)
	redditCommentsFallRE = regexp.MustCompile(`(?i)>\s*(\d[\d,]*)\s+comment[s]?\s*<`)
	redditCommentIDRE    = regexp.MustCompile(`<div[^>]+id="thing_t1_([A-Za-z0-9]+)"[^>]+data-author="([^"]*)"[^>]*`)
	redditCommentScoreRE = regexp.MustCompile(`class="score unvoted"[^>]*title="([^"]*)"`)
	redditCommentPermRE  = regexp.MustCompile(`<a[^>]+href="(/r/[^"]+/comments/[^"]+)"[^>]+class="[^"]*bylink[^"]*"`)
	redditCommentTimeRE  = regexp.MustCompile(`<time[^>]+datetime="([^"]+)"`)
)

func redditStripTags(s string) string {
	return strings.TrimSpace(html.UnescapeString(redditTagRE.ReplaceAllString(s, "")))
}

func redditAtoi(s string, fallback int) int {
	s = strings.TrimSpace(s)
	sign := 1
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return fallback
	}
	n, err := strconv.Atoi(s[:end])
	if err != nil {
		return fallback
	}
	return sign * n
}

func redditClip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func redditParseThread(htmlStr string) *redditPost {
	p := &redditPost{}

	if m := redditTitleRE.FindStringSubmatch(htmlStr); m != nil {
		p.Title = redditStripTags(m[1])
	}
	if m := redditSubRE.FindStringSubmatch(htmlStr); m != nil {
		p.Subreddit = m[1]
	}
	if m := redditCanonicalRE.FindStringSubmatch(htmlStr); m != nil {
		p.Permalink = m[1]
	}
	if m := redditAuthorRE.FindStringSubmatch(htmlStr); m != nil {
		p.Author = m[1]
	}
	if m := redditScoreTitleRE.FindStringSubmatch(htmlStr); m != nil {
		p.Score = redditAtoi(m[1], 0)
	} else if m := redditScoreSpanRE.FindStringSubmatch(htmlStr); m != nil {
		p.Score = redditAtoi(m[1], 0)
	}
	if m := redditDataURLRE.FindStringSubmatch(htmlStr); m != nil {
		p.URL = html.UnescapeString(m[1])
	}
	if m := redditTimeRE.FindStringSubmatch(htmlStr); m != nil {
		p.CreatedISO = m[1]
	}
	if m := redditDataCommentsRE.FindStringSubmatch(htmlStr); m != nil {
		p.NumComments = redditAtoi(m[1], 0)
	} else if m := redditCommentsFallRE.FindStringSubmatch(htmlStr); m != nil {
		p.NumComments = redditAtoi(strings.ReplaceAll(m[1], ",", ""), 0)
	}

	if m := redditOpThingRE.FindStringIndex(htmlStr); m != nil {
		opStart := m[0]
		rest := htmlStr[opStart:]
		limit := len(rest)
		if i := strings.Index(rest, "commentarea"); i >= 0 {
			limit = i
		} else if limit > 30000 {
			limit = 30000
		}
		if b := redditUsertextBodyRE.FindStringSubmatch(rest[:limit]); b != nil {
			p.Selftext = redditStripTags(b[1])
		}
	}

	p.MediaURLs = redditExtractMediaURLs(p.URL, htmlStr)
	p.Comments = redditExtractComments(htmlStr, 100)
	return p
}

func redditCommentareaSlice(htmlStr string) string {
	i := strings.Index(htmlStr, "class='commentarea'")
	if i < 0 {
		i = strings.Index(htmlStr, `class="commentarea"`)
	}
	if i < 0 {
		return ""
	}
	return htmlStr[i:]
}

func redditExtractComments(htmlStr string, limit int) []redditComment {
	tail := redditCommentareaSlice(htmlStr)
	if tail == "" {
		return nil
	}
	matches := redditCommentIDRE.FindAllStringSubmatchIndex(tail, -1)
	out := make([]redditComment, 0, len(matches))
	for i, m := range matches {
		if len(out) >= limit {
			break
		}
		var end int
		if i+1 < len(matches) {
			end = matches[i+1][0]
		} else {
			end = len(tail)
		}
		block := tail[m[0]:end]
		cid := tail[m[2]:m[3]]
		author := tail[m[4]:m[5]]

		body := ""
		if b := redditUsertextBodyRE.FindStringSubmatch(block); b != nil {
			body = redditStripTags(b[1])
		}
		score := 0
		if s := redditCommentScoreRE.FindStringSubmatch(block); s != nil {
			score = redditAtoi(s[1], 0)
		}
		perm := ""
		if pm := redditCommentPermRE.FindStringSubmatch(block); pm != nil {
			perm = pm[1]
		}
		created := ""
		if tm := redditCommentTimeRE.FindStringSubmatch(block); tm != nil {
			created = tm[1]
		}

		out = append(out, redditComment{
			ID:         "t1_" + cid,
			Author:     author,
			Score:      score,
			Body:       body,
			CreatedISO: created,
			Permalink:  perm,
		})
	}
	return out
}

var (
	redditVReddItRE     = regexp.MustCompile(`https?://v\.redd\.it/([A-Za-z0-9]+)`)
	redditIReddItRE     = regexp.MustCompile(`https?://(?:i|preview)\.redd\.it/[A-Za-z0-9_.\-/]+\.(?:jpg|jpeg|png|gif|gifv|webp)`)
	redditGalleryItemRE = regexp.MustCompile(`https?://preview\.redd\.it/([A-Za-z0-9]+\.(?:jpg|jpeg|png|gif|webp))`)
	redditImgurRE       = regexp.MustCompile(`https?://(?:i\.)?imgur\.com/[A-Za-z0-9]+\.(?:jpg|jpeg|png|gif|gifv|mp4|webm|webp)`)
	redditRedgifsRE     = regexp.MustCompile(`https?://(?:www\.)?redgifs\.com/watch/([a-zA-Z]+)`)
	redditVReddItMP4RE  = regexp.MustCompile(`https?://v\.redd\.it/[A-Za-z0-9]+/DASH_\d+\.mp4`)
)

// redditExtractMediaURLs returns de-duplicated directly-downloadable media URLs.
// For v.redd.it link posts we return the DASHPlaylist.mpd URL; redditDownload
// dispatches to a v.redd.it-specific fetcher that parses the manifest.
func redditExtractMediaURLs(postURL, htmlStr string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(u string) {
		if u == "" {
			return
		}
		u = html.UnescapeString(u)
		if !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}

	if postURL != "" {
		if redditIReddItRE.MatchString(postURL) || redditImgurRE.MatchString(postURL) {
			add(postURL)
		}
		if m := redditVReddItRE.FindStringSubmatch(postURL); m != nil {
			add(fmt.Sprintf("https://v.redd.it/%s/DASHPlaylist.mpd", m[1]))
		}
		if redditRedgifsRE.MatchString(postURL) {
			add(postURL)
		}
	}
	for _, m := range redditGalleryItemRE.FindAllStringSubmatch(htmlStr, -1) {
		add(fmt.Sprintf("https://i.redd.it/%s", m[1]))
	}
	for _, m := range redditVReddItMP4RE.FindAllString(htmlStr, -1) {
		add(m)
	}
	return out
}

func redditDownload(u, outDir string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	if m := redditRedgifsRE.FindStringSubmatch(u); m != nil {
		mp4, err := redditResolveRedgifs(m[1])
		if err != nil {
			return "", fmt.Errorf("redgifs resolve: %w", err)
		}
		u = mp4
	}
	if strings.Contains(u, "v.redd.it/") {
		if strings.HasSuffix(u, "DASHPlaylist.mpd") {
			return redditDownloadVReddManifest(u, outDir)
		}
		if strings.HasSuffix(u, ".mp4") {
			return redditDownloadVRedd(u, outDir)
		}
	}
	return redditDownloadDirect(u, outDir)
}

func redditDownloadDirect(u, outDir string) (string, error) {
	client, err := redditNewClient()
	if err != nil {
		return "", err
	}
	req, err := redditNewReq(u)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Sec-Fetch-Dest", "image")
	req.Header.Set("Sec-Fetch-Mode", "no-cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Del("Upgrade-Insecure-Requests")
	req.Header.Del("Sec-Fetch-User")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, u)
	}

	name := path.Base(u)
	if q := strings.Index(name, "?"); q >= 0 {
		name = name[:q]
	}
	if name == "" || name == "/" || name == "." {
		name = "download.bin"
	}
	dst := filepath.Join(outDir, name)
	f, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return dst, nil
}

func redditDownloadVRedd(u, outDir string) (string, error) {
	i := strings.LastIndex(u, "/")
	if i < 0 {
		return "", errors.New("bad v.redd.it URL")
	}
	base := u[:i+1]
	tries := []string{
		"CMAF_720.mp4", "CMAF_480.mp4", "CMAF_360.mp4", "CMAF_270.mp4", "CMAF_220.mp4",
		"DASH_1080.mp4", "DASH_720.mp4", "DASH_480.mp4", "DASH_360.mp4", "DASH_240.mp4",
	}
	for _, name := range tries {
		p, err := redditDownloadDirect(base+name, outDir)
		if err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no v.redd.it fallback MP4 available at %s", base)
}

func redditDownloadVReddManifest(u, outDir string) (string, error) {
	client, err := redditNewClient()
	if err != nil {
		return "", err
	}
	req, err := redditNewReq(u)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://www.reddit.com/")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("manifest HTTP %d for %s", resp.StatusCode, u)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	baseURLs := regexp.MustCompile(`<BaseURL[^>]*>([^<]+)</BaseURL>`).
		FindAllStringSubmatch(string(body), -1)
	if len(baseURLs) == 0 {
		return "", errors.New("no <BaseURL> in manifest")
	}
	base := strings.TrimSuffix(u, "DASHPlaylist.mpd")
	type track struct {
		name string
		res  int
	}
	var videos []track
	resRE := regexp.MustCompile(`(?i)(?:CMAF|DASH)_(\d+)\.mp4`)
	for _, b := range baseURLs {
		name := strings.TrimSpace(b[1])
		if strings.Contains(strings.ToUpper(name), "AUDIO") {
			continue
		}
		m := resRE.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		r, _ := strconv.Atoi(m[1])
		videos = append(videos, track{name: name, res: r})
	}
	for i := range videos {
		for j := i + 1; j < len(videos); j++ {
			if videos[j].res > videos[i].res {
				videos[i], videos[j] = videos[j], videos[i]
			}
		}
	}
	for _, v := range videos {
		p, err := redditDownloadDirect(base+v.name, outDir)
		if err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no downloadable video track in manifest %s", u)
}

func redditResolveRedgifs(id string) (string, error) {
	client, err := redditNewClient()
	if err != nil {
		return "", err
	}
	tokReq, err := redditNewReq("https://api.redgifs.com/v2/auth/temporary")
	if err != nil {
		return "", err
	}
	tokReq.Header.Set("Accept", "application/json")
	tokResp, err := client.Do(tokReq)
	if err != nil {
		return "", err
	}
	defer tokResp.Body.Close()
	tokBody, _ := io.ReadAll(tokResp.Body)
	var tok struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(tokBody, &tok); err != nil || tok.Token == "" {
		return "", fmt.Errorf("redgifs token: %v (body: %s)", err, redditClip(string(tokBody), 200))
	}
	metaReq, err := redditNewReq(fmt.Sprintf("https://api.redgifs.com/v2/gifs/%s",
		strings.ToLower(id)))
	if err != nil {
		return "", err
	}
	metaReq.Header.Set("Accept", "application/json")
	metaReq.Header.Set("Authorization", "Bearer "+tok.Token)
	metaResp, err := client.Do(metaReq)
	if err != nil {
		return "", err
	}
	defer metaResp.Body.Close()
	metaBody, _ := io.ReadAll(metaResp.Body)
	var meta struct {
		Gif struct {
			Urls struct {
				HD  string `json:"hd"`
				SD  string `json:"sd"`
				MP4 string `json:"mp4"`
			} `json:"urls"`
		} `json:"gif"`
	}
	if err := json.Unmarshal(metaBody, &meta); err != nil {
		return "", fmt.Errorf("redgifs meta: %w", err)
	}
	for _, u := range []string{meta.Gif.Urls.HD, meta.Gif.Urls.SD, meta.Gif.Urls.MP4} {
		if u != "" {
			return u, nil
		}
	}
	return "", errors.New("redgifs: no mp4 in metadata")
}

func RedditHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<i>Usage:</i> <code>/reddit &lt;url&gt;</code>")
		return nil
	}
	if !strings.Contains(arg, "reddit.com") && !strings.Contains(arg, "redd.it") {
		m.Reply("Not a reddit URL.")
		return nil
	}

	status, _ := m.Reply("Fetching reddit post...")

	post, err := redditFetch(arg)
	if err != nil {
		if status != nil {
			status.Edit(fmt.Sprintf("Failed: %s", redditClip(err.Error(), 200)))
		}
		return nil
	}

	title := post.Title
	if title == "" {
		title = "(no title)"
	}
	var cap strings.Builder
	cap.WriteString("<b>")
	cap.WriteString(html.EscapeString(title))
	cap.WriteString("</b>\n")
	if post.Subreddit != "" {
		cap.WriteString("r/")
		cap.WriteString(post.Subreddit)
	}
	if post.Author != "" {
		if cap.Len() > 0 {
			cap.WriteString(" • ")
		}
		cap.WriteString("u/")
		cap.WriteString(post.Author)
	}
	cap.WriteString(fmt.Sprintf(" • %d pts • %d comments\n", post.Score, post.NumComments))
	if post.Selftext != "" {
		cap.WriteString("\n")
		cap.WriteString(html.EscapeString(redditClip(post.Selftext, 700)))
		if len(post.Selftext) > 700 {
			cap.WriteString("…")
		}
		cap.WriteString("\n")
	}
	if len(post.Comments) > 0 {
		cap.WriteString("\n<b>Top comments:</b>\n")
		for i, c := range post.Comments {
			if i >= 3 {
				break
			}
			cap.WriteString(fmt.Sprintf("• <b>%s</b> (%d): %s\n",
				html.EscapeString(c.Author), c.Score,
				html.EscapeString(redditClip(c.Body, 180))))
		}
	}
	if post.Permalink != "" {
		cap.WriteString("\n<a href=\"")
		cap.WriteString(html.EscapeString(post.Permalink))
		cap.WriteString("\">source</a>")
	}
	caption := redditClip(cap.String(), 3800)

	if len(post.MediaURLs) == 0 {
		if status != nil {
			status.Edit(caption, &tg.SendOptions{LinkPreview: false})
		} else {
			m.Reply(caption, &tg.SendOptions{LinkPreview: false})
		}
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "reddit-")
	if err != nil {
		if status != nil {
			status.Edit(caption+"\n\n<i>(media download failed to init)</i>", &tg.SendOptions{LinkPreview: false})
		}
		return nil
	}
	defer os.RemoveAll(tmpDir)

	sentAny := false
	for i, mu := range post.MediaURLs {
		if i >= 10 {
			break
		}
		p, err := redditDownload(mu, tmpDir)
		if err != nil {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		mime := nhttp.DetectContentType(b)
		fname := filepath.Base(p)
		if fname == "" {
			fname = fmt.Sprintf("reddit_%d", time.Now().UnixNano())
		}
		opts := &tg.MediaOptions{FileName: fname, MimeType: mime}
		if !sentAny {
			opts.Caption = caption
			opts.ParseMode = "HTML"
		}
		if _, err := m.ReplyMedia(b, opts); err == nil {
			sentAny = true
		}
	}

	if status != nil {
		status.Delete()
	}
	if !sentAny {
		m.Reply(caption, &tg.SendOptions{LinkPreview: false})
	}
	return nil
}

func registerRedditHandlers() {
	c := modules.Client
	c.On("cmd:reddit", RedditHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerRedditHandlers)
	modules.Mods.AddModule("Reddit", `<b>Reddit</b>

Fetch a Reddit post and get its title, body, top comments, and any images/videos attached.

<b>Command:</b>
 - /reddit &lt;url&gt; - Fetch a post and send its media + summary

Works with www.reddit.com, old.reddit.com, m.reddit.com, np.reddit.com and any .json / bare-path variants. v.redd.it videos, redgifs, imgur direct URLs, i.redd.it images, and gallery posts are all supported.`)
}
