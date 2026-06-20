package modules

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var cryptoNewsHTTPClient = &http.Client{Timeout: 30 * time.Second}

type cryptoNewsItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

type cryptoNewsRSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title string           `xml:"title"`
		Items []cryptoNewsItem `xml:"item"`
	} `xml:"channel"`
}

func cryptoNewsFetch(endpoint string) (*cryptoNewsRSS, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 JuliaBot/1.0")
	resp, err := cryptoNewsHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var feed cryptoNewsRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func cryptoNewsSource(link string) string {
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return "Unknown"
	}
	host := strings.TrimPrefix(u.Host, "www.")
	return host
}

func CryptoNewsHandler(m *tg.NewMessage) error {
	feeds := []string{
		"https://cointelegraph.com/rss",
		"https://www.coindesk.com/arc/outboundfeeds/rss/",
	}
	var items []cryptoNewsItem
	for _, f := range feeds {
		feed, err := cryptoNewsFetch(f)
		if err != nil {
			continue
		}
		items = append(items, feed.Channel.Items...)
		if len(items) >= 5 {
			break
		}
	}
	if len(items) == 0 {
		m.Reply("<b>Error:</b> failed to fetch crypto news.")
		return nil
	}
	limit := 5
	if len(items) < limit {
		limit = len(items)
	}
	items = items[:limit]

	var b strings.Builder
	b.WriteString("<b>Top Crypto News</b>\n\n")
	for i, it := range items {
		title := strings.TrimSpace(it.Title)
		link := strings.TrimSpace(it.Link)
		if title == "" || link == "" {
			continue
		}
		src := cryptoNewsSource(link)
		b.WriteString(fmt.Sprintf("<b>%d.</b> <a href=\"%s\">%s</a>\n", i+1, html.EscapeString(link), html.EscapeString(title)))
		b.WriteString(fmt.Sprintf("    <b>Source:</b> %s\n\n", html.EscapeString(src)))
	}
	m.Reply(b.String(), &tg.SendOptions{LinkPreview: false})
	return nil
}

func registerCryptoNewsHandlers() {
	c := Client
	c.On("cmd:cryptonews", CryptoNewsHandler)
}

func init() {
	QueueHandlerRegistration(registerCryptoNewsHandlers)
}
