package modules

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var bbcNewsHTTPClient = &http.Client{Timeout: 30 * time.Second}

type bbcNewsItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type bbcNewsRSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title string        `xml:"title"`
		Items []bbcNewsItem `xml:"item"`
	} `xml:"channel"`
}

func bbcNewsFetch(endpoint string) (*bbcNewsRSS, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 JuliaBot/1.0")
	resp, err := bbcNewsHTTPClient.Do(req)
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
	var feed bbcNewsRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func BBCNewsHandler(m *tg.NewMessage) error {
	feed, err := bbcNewsFetch("https://feeds.bbci.co.uk/news/rss.xml")
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch BBC News.")
		return nil
	}
	if len(feed.Channel.Items) == 0 {
		m.Reply("<b>Error:</b> no headlines available.")
		return nil
	}
	limit := 5
	if len(feed.Channel.Items) < limit {
		limit = len(feed.Channel.Items)
	}
	items := feed.Channel.Items[:limit]

	var b strings.Builder
	b.WriteString("<b>BBC News - Top Headlines</b>\n\n")
	count := 0
	for _, it := range items {
		title := strings.TrimSpace(it.Title)
		link := strings.TrimSpace(it.Link)
		if title == "" || link == "" {
			continue
		}
		count++
		b.WriteString(fmt.Sprintf("<b>%d.</b> <a href=\"%s\">%s</a>\n", count, html.EscapeString(link), html.EscapeString(title)))
		desc := strings.TrimSpace(it.Description)
		if desc != "" {
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			b.WriteString(fmt.Sprintf("    <i>%s</i>\n", html.EscapeString(desc)))
		}
		b.WriteString("\n")
	}
	if count == 0 {
		m.Reply("<b>Error:</b> no valid headlines found.")
		return nil
	}
	m.Reply(b.String(), &tg.SendOptions{LinkPreview: false})
	return nil
}

func registerBBCNewsHandlers() {
	c := Client
	c.On("cmd:bbcnews", BBCNewsHandler)
}

func init() {
	QueueHandlerRegistration(registerBBCNewsHandlers)
}
