package extras

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

const tamilMVBase = "https://www.1tamilmv.ing"

const tamilMVFallback = "https://www.1tamilmv.fi"

var (
	tamilMVHTTP   = &http.Client{Timeout: 20 * time.Second}
	tamilMVTopics sync.Map
	tamilMVAnchor = regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']*?/index\.php\?[^"']*(?:topic|forums)[^"']*)["'][^>]*>(.*?)</a>`)
	tamilMVTags   = regexp.MustCompile(`<[^>]+>`)
	tamilMVLinks  = regexp.MustCompile(`(?is)<a[^>]+href=["']((?:https?://|magnet:\?)[^"']+)["'][^>]*>(.*?)</a>`)
)

type tamilMVItem struct{ Title, URL string }

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
		out = append(out, tamilMVItem{title, href})
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
		k.AddRow(tg.Button.Data(item.Title[:min(len(item.Title), 55)], "tm:open:"+token))
	}
	return k.Build()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	var b strings.Builder
	b.WriteString("<b>Release topic</b>\n")
	count := 0
	for _, m := range tamilMVLinks.FindAllStringSubmatch(page, -1) {
		href, title := html.UnescapeString(m[1]), tamilMVClean(m[2])
		if title == "" {
			title = "Open link"
		}
		if strings.Contains(href, "1tamilmv") || strings.Contains(strings.ToLower(href), "magnet:") || strings.Contains(strings.ToLower(href), ".torrent") {
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
	modules.Client.On("callback:tm:open:", tamilMVCallback)
}
func init() { modules.QueueHandlerRegistration(registerTamilMVHandlers) }
