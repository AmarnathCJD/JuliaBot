package extras

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	modules "main/modules"
	"main/modules/db"

	tg "github.com/amarnathcjd/gogram/telegram"
)

const tamilMVBase = "https://www.1tamilmv.ing"

const tamilMVFallback = "https://www.1tamilmv.fi"

var (
	tamilMVHTTP      = &http.Client{Timeout: 20 * time.Second}
	tamilMVTopics    sync.Map
	tamilMVAnchor    = regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']*?/index\.php\?[^"']*/forums/topic/[^"']*)["'][^>]*>(.*?)</a>`)
	tamilMVSlug      = regexp.MustCompile(`/forums/topic/\d+-([^/?#]+)`)
	tamilMVTags      = regexp.MustCompile(`<[^>]+>`)
	tamilMVLinks     = regexp.MustCompile(`(?is)<a[^>]+href=["']((?:https?://|magnet:\?)[^"']+)["'][^>]*>(.*?)</a>`)
	tamilMVPoster    = regexp.MustCompile(`(?is)<img[^>]+src=["'](https://pbs\.twimg\.com/[^"']+)["']`)
	tamilMVReleaseRe = regexp.MustCompile(`(?is)(<strong[^>]*>(?:[^<]|<span[^>]*>[^<]*</span>)*</strong>)\s*<br>\s*<strong>\s*<a[^>]+data-fileext=["']torrent["'][^>]+href=["']([^"']+)["'][^>]*>.*?</a>.*?<a[^>]+href=["'](magnet:\?[^"']+)["'][^>]*>.*?</a>.*?<a[^>]+href=["'](https?://[^"']+)["'][^>]*>.*?</a>`)
)

func tamilMVTopicTitle(slug string) string {
	if decoded, err := url.PathUnescape(slug); err == nil {
		slug = decoded
	}
	slug = strings.ReplaceAll(slug, "-", " ")
	slug = strings.ReplaceAll(slug, "\u00a0", " ")
	slug = regexp.MustCompile(`(?i)\btrue\b`).ReplaceAllString(slug, "")
	for _, marker := range []string{" hq predvd", " predvd", " true web dl", " web dl", " bluray", " hd ", " uhd ", " 4k "} {
		if i := strings.Index(strings.ToLower(slug), marker); i > 0 {
			slug = slug[:i]
			break
		}
	}
	return strings.Title(tamilMVClean(slug))
}

func tamilMVLanguage(title string) string {
	lower := strings.ToLower(title)
	for _, lang := range []string{"malayalam", "tamil", "telugu", "hindi", "kannada", "english"} {
		if strings.Contains(lower, lang) {
			return lang
		}
	}
	return "other"
}

func tamilMVStripLanguage(title string) string {
	for _, lang := range []string{"malayalam", "tamil", "telugu", "hindi", "kannada", "english"} {
		title = regexp.MustCompile(`(?i)(^|\s)`+lang+`(\s|$)`).ReplaceAllString(title, " ")
	}
	return strings.Join(strings.Fields(title), " ")
}

type tamilMVItem struct {
	Title string
	URL   string
	Lang  string
}

type tamilMVRelease struct {
	Title, Torrent, Magnet, Direct string
}

var tamilMVReleases sync.Map

func tamilMVParseReleases(page string) (string, []tamilMVRelease) {
	poster := ""
	if m := tamilMVPoster.FindStringSubmatch(page); len(m) == 2 {
		poster = html.UnescapeString(m[1])
	}
	items := make([]tamilMVRelease, 0)
	for _, m := range tamilMVReleaseRe.FindAllStringSubmatch(page, -1) {
		items = append(items, tamilMVRelease{Title: tamilMVClean(m[1]), Torrent: html.UnescapeString(m[2]), Magnet: html.UnescapeString(m[3]), Direct: html.UnescapeString(m[4])})
	}
	return poster, items
}

func tamilMVFetch(ctx context.Context, target string) (string, error) {
	page, err := tamilMVFetchURL(ctx, target)
	if err == nil {
		return page, nil
	}

	current, resolveErr := tamilMVResolveBase(ctx)
	if resolveErr != nil {
		return "", err
	}

	if strings.HasPrefix(target, tamilMVBase) {
		target = current + strings.TrimPrefix(target, tamilMVBase)
	}
	return tamilMVFetchURL(ctx, target)
}

func tamilMVFetchURL(ctx context.Context, target string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; JuliaBot/1.0)")
	resp, err := tamilMVHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	return string(b), err
}

func tamilMVResolveBase(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tamilMVFallback, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; JuliaBot/1.0)")
	resp, err := tamilMVHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 || resp.Request == nil || resp.Request.URL == nil {
		return "", fmt.Errorf("fallback HTTP %d", resp.StatusCode)
	}
	return strings.TrimRight(resp.Request.URL.Scheme+"://"+resp.Request.URL.Host, "/"), nil
}

func tamilMVClean(s string) string {
	s = tamilMVTags.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(html.UnescapeString(s)), " ")
	return strings.TrimSpace(s)
}

func tamilMVParse(page string, filter string, limit int) []tamilMVItem {
	seen := map[string]bool{}
	out := make([]tamilMVItem, 0, limit)
	for _, m := range tamilMVAnchor.FindAllStringSubmatch(page, -1) {
		href := html.UnescapeString(m[1])
		title := tamilMVClean(m[2])
		if sm := tamilMVSlug.FindStringSubmatch(href); len(sm) == 2 {
			if sm[1] == "0" || len(sm[1]) < 8 {
				continue
			}
			title = tamilMVTopicTitle(sm[1])
		}
		if title == "" || len(title) < 3 {
			continue
		}
		if !strings.HasPrefix(href, "http") {
			href = tamilMVBase + "/" + strings.TrimPrefix(href, "/")
		}
		if !strings.Contains(strings.ToLower(title), strings.ToLower(filter)) || seen[href] {
			continue
		}
		seen[href] = true
		lang := tamilMVLanguage(title)
		title = tamilMVStripLanguage(title)
		out = append(out, tamilMVItem{Title: title, URL: href, Lang: lang})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func tamilMVSearchURL(q string) string {
	return tamilMVBase + "/index.php?/search/&q=" + url.QueryEscape(q) + "&type=forums_topic"
}

func tamilMVKeyboard(items []tamilMVItem) *tg.ReplyInlineMarkup {
	if len(items) == 0 {
		return nil
	}
	k := tg.NewKeyboard()
	for i, item := range items {
		token := fmt.Sprintf("%x", time.Now().UnixNano()+int64(i))
		tamilMVTopics.Store(token, item.URL)
		button := tg.Button.Data(item.Title[:min(len(item.Title), 55)], "tm:open:"+token)
		switch item.Lang {
		case "malayalam":
			button = button.Success()
		case "tamil":
			button = button.Danger()
		default:
			button = button.Primary()
		}
		k.AddRow(button)
	}
	return k.Build()
}

func tamilMVHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	target := tamilMVBase
	filter := ""
	switch strings.ToLower(arg) {
	case "":
	case "recent":
		target += "/"
		filter = ""
	case "mal", "malayalam":
		target += "/index.php?/forums/forum/34-malayalam-language/"
		filter = ""
	default:
		target = tamilMVSearchURL(arg)
	}
	status, _ := m.Reply("Fetching TamilMV releases...")
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	page, err := tamilMVFetch(ctx, target)
	if err != nil {
		if status != nil {
			status.Edit("TamilMV fetch failed: " + html.EscapeString(err.Error()))
		}
		return nil
	}
	items := tamilMVParse(page, filter, 25)
	if len(items) == 0 {
		if status != nil {
			status.Edit("No TamilMV releases found.")
		}
		return nil
	}
	text := "<b>TamilMV releases</b>\nTap a title to view its topic links."
	if status != nil {
		status.Edit(text, &tg.SendOptions{ParseMode: "HTML", ReplyMarkup: tamilMVKeyboard(items)})
	}
	return nil
}

func tamilMVCallback(c *tg.CallbackQuery) error {
	data := c.DataString()
	if after, ok := strings.CutPrefix(data, "tm:settings:"); ok {
		p := strings.Split(after, ":")
		if len(p) == 2 {
			id, _ := strconv.ParseInt(p[0], 10, 64)
			switch p[1] {
			case "on":
				db.SetTamilMVAlertEnabled(id, c.SenderID, true)
			case "off":
				db.SetTamilMVAlertEnabled(id, c.SenderID, false)
			case "5m":
				db.SetTamilMVAlertInterval(id, c.SenderID, 300)
			case "30m":
				db.SetTamilMVAlertInterval(id, c.SenderID, 1800)
			case "1h":
				db.SetTamilMVAlertInterval(id, c.SenderID, 3600)
			}
			c.Answer("Settings updated", &tg.CallbackOptions{Alert: false})
			c.Edit("Alert settings updated.")
			return nil
		}
	}
	if after, ok := strings.CutPrefix(data, "tm:quality:"); ok {
		key := after
		raw, ok := tamilMVReleases.Load(key)
		if !ok {
			c.Answer("This result expired.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		r := raw.(tamilMVRelease)
		c.Answer("Choose a link", &tg.CallbackOptions{Alert: false})
		k := tg.NewKeyboard().AddRow(tg.Button.URL("Torrent", r.Torrent), tg.Button.URL("Magnet", r.Magnet)).AddRow(tg.Button.URL("Direct Link", r.Direct))
		c.Edit("<b>"+html.EscapeString(r.Title)+"</b>", &tg.SendOptions{ParseMode: "HTML", ReplyMarkup: k.Build()})
		return nil
	}
	token := strings.TrimPrefix(c.DataString(), "tm:open:")
	raw, ok := tamilMVTopics.Load(token)
	if !ok {
		c.Answer("This result expired.", &tg.CallbackOptions{Alert: true})
		return nil
	}
	c.Answer("Loading topic...", &tg.CallbackOptions{Alert: false})
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	page, err := tamilMVFetch(ctx, raw.(string))
	if err != nil {
		c.Edit("Topic fetch failed: " + html.EscapeString(err.Error()))
		return nil
	}
	poster, releases := tamilMVParseReleases(page)
	if len(releases) > 0 {
		var b strings.Builder
		b.WriteString("<b>Select quality</b>")
		if poster != "" {
			b.WriteString("\nPoster: <a href=\"")
			b.WriteString(html.EscapeString(poster))
			b.WriteString("\">open image</a>")
		}
		k := tg.NewKeyboard()
		for i, r := range releases {
			key := fmt.Sprintf("%s:%d", token, i)
			tamilMVReleases.Store(key, r)
			k.AddRow(tg.Button.Data(r.Title, "tm:quality:"+key))
		}
		c.Edit(b.String(), &tg.SendOptions{ParseMode: "HTML", ReplyMarkup: k.Build()})
		return nil
	}
	var b strings.Builder
	b.WriteString("<b>Release topic</b>\n")
	count := 0
	for _, m := range tamilMVLinks.FindAllStringSubmatch(page, -1) {
		href, title := html.UnescapeString(m[1]), tamilMVClean(m[2])
		if title == "" {
			title = "Open link"
		}
		lowerHref := strings.ToLower(href)
		blocked := strings.Contains(lowerHref, "1tamilmv") || strings.Contains(lowerHref, "cdn.") || strings.Contains(lowerHref, "google.") || strings.Contains(lowerHref, "telegram.") || strings.Contains(lowerHref, "t.me/") || strings.Contains(lowerHref, "streamdady.") || strings.Contains(lowerHref, "invisioncommunity.") || strings.Contains(lowerHref, "ipbmafia.") || strings.Contains(lowerHref, "facebook.") || strings.Contains(lowerHref, "twitter.")
		if !blocked && (strings.Contains(lowerHref, "magnet:") || strings.Contains(lowerHref, ".torrent") || strings.HasPrefix(lowerHref, "http")) {
			b.WriteString(fmt.Sprintf("\n• <a href=\"%s\">%s</a>", html.EscapeString(href), html.EscapeString(title)))
			count++
		}
		if count >= 30 {
			break
		}
	}
	if count == 0 {
		b.WriteString("\nNo public links were found on this topic.")
	}
	c.Edit(b.String(), &tg.SendOptions{ParseMode: "HTML"})
	return nil
}

func registerTamilMVHandlers() {
	modules.Client.On("cmd:tm", tamilMVHandler)
	modules.Client.On("callback:tm:", tamilMVCallback)
}

const tamilMVAlertMin = time.Minute

func tamilMVAlertArgs(s string) (string, time.Duration, error) {
	title := strings.TrimSpace(s)
	if title == "" {
		return "", 0, fmt.Errorf("missing title")
	}
	return title, 24 * time.Hour, nil
}
func tamilMVAlertCreate(m *tg.NewMessage) error {
	title, d, e := tamilMVAlertArgs(m.Args())
	if e != nil {
		m.Reply("Usage: /tmalert <title> [interval]")
		return nil
	}
	a := &db.TamilMVAlert{ChatID: m.ChatID(), UserID: m.SenderID(), Title: title, Interval: int64(d / time.Second)}
	if e = db.SaveTamilMVAlert(a); e != nil {
		m.Reply("Alert save failed: " + e.Error())
		return nil
	}
	m.Reply(fmt.Sprintf("Alert #%d created for %s. Checking every %s.", a.ID, title, d))
	return nil
}
func tamilMVAlertList(m *tg.NewMessage) error {
	as, e := db.ListTamilMVAlerts(m.SenderID())
	if e != nil {
		m.Reply("Alert list failed: " + e.Error())
		return nil
	}
	if len(as) == 0 {
		m.Reply("No TamilMV alerts.")
		return nil
	}
	var b strings.Builder
	for _, a := range as {
		b.WriteString(fmt.Sprintf("#%d %s (%s)\n", a.ID, a.Title, time.Duration(a.Interval)*time.Second))
	}
	m.Reply(b.String())
	return nil
}

func tamilMVSettings(m *tg.NewMessage) error {
	as, e := db.ListTamilMVAlerts(m.SenderID())
	if e != nil || len(as) == 0 {
		m.Reply("No TamilMV alerts.")
		return nil
	}
	k := tg.NewKeyboard()
	for _, a := range as {
		state := "off"
		if a.Enabled != 0 {
			state = "on"
		}
		id := strconv.FormatInt(a.ID, 10)
		k.AddRow(tg.Button.Data(fmt.Sprintf("#%d %s [%s]", a.ID, a.Title, state), "tm:settings:"+id+":"+state))
		k.AddRow(tg.Button.Data("5m", "tm:settings:"+id+":5m"), tg.Button.Data("30m", "tm:settings:"+id+":30m"), tg.Button.Data("1h", "tm:settings:"+id+":1h"))
	}
	m.Reply("<b>TamilMV alert settings</b>", &tg.SendOptions{ParseMode: "HTML", ReplyMarkup: k.Build()})
	return nil
}
func tamilMVAlertDelete(m *tg.NewMessage) error {
	id, e := strconv.ParseInt(strings.TrimSpace(m.Args()), 10, 64)
	if e != nil {
		m.Reply("Usage: /tmalertoff <id>")
		return nil
	}
	if e = db.DeleteTamilMVAlert(id, m.SenderID()); e != nil {
		m.Reply("Alert removal failed: " + e.Error())
		return nil
	}
	m.Reply("Alert removed.")
	return nil
}
func tamilMVAlertWorker() {
	for range time.Tick(30 * time.Second) {
		as, e := db.DueTamilMVAlerts(time.Now().Unix())
		if e != nil {
			continue
		}
		for _, a := range as {
			ctx, c := context.WithTimeout(context.Background(), 25*time.Second)
			page, e := tamilMVFetch(ctx, tamilMVBase)
			c()
			if e != nil {
				continue
			}
			found := ""
			for _, it := range tamilMVParse(page, "", 100) {
				if strings.Contains(strings.ToLower(it.Title), strings.ToLower(a.Title)) {
					found = it.URL
					break
				}
			}
			if found != "" && found != a.LastSeen {
				modules.Client.SendMessage(a.ChatID, fmt.Sprintf("TamilMV alert: %s\n%s", a.Title, found))
				db.TouchTamilMVAlert(a.ID, found, time.Now().Unix())
			} else {
				db.TouchTamilMVAlert(a.ID, a.LastSeen, time.Now().Unix())
			}
		}
	}
}
func registerTamilMVAlertHandlers() {
	modules.Client.On("cmd:tmalert", tamilMVAlertCreate)
	modules.Client.On("cmd:tmalerts", tamilMVAlertList)
	modules.Client.On("cmd:tmalertoff", tamilMVAlertDelete)
	modules.Client.On("cmd:tmsettings", tamilMVSettings)
	go tamilMVAlertWorker()
}
func init() {
	modules.QueueHandlerRegistration(registerTamilMVHandlers)
	modules.QueueHandlerRegistration(registerTamilMVAlertHandlers)
}
