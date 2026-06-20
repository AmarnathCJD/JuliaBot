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

var nytHTTPClient = &http.Client{Timeout: 30 * time.Second}

type nytItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Creator     string `xml:"http://purl.org/dc/elements/1.1/ creator"`
	PubDate     string `xml:"pubDate"`
}

type nytRSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title string    `xml:"title"`
		Items []nytItem `xml:"item"`
	} `xml:"channel"`
}

func nytFetchTopStories() (*nytRSS, error) {
	req, err := http.NewRequest(http.MethodGet, "https://rss.nytimes.com/services/xml/rss/nyt/HomePage.xml", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 JuliaBot/1.0")
	resp, err := nytHTTPClient.Do(req)
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
	var feed nytRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func nytFormatDate(s string) string {
	t, err := time.Parse(time.RFC1123Z, s)
	if err != nil {
		return ""
	}
	return t.UTC().Format("Jan 02, 15:04 MST")
}

func NYTimesHandler(m *tg.NewMessage) error {
	feed, err := nytFetchTopStories()
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch NYT top stories.")
		return nil
	}
	if len(feed.Channel.Items) == 0 {
		m.Reply("<b>Error:</b> no NYT stories available.")
		return nil
	}
	limit := 10
	if len(feed.Channel.Items) < limit {
		limit = len(feed.Channel.Items)
	}

	var b strings.Builder
	b.WriteString("<b>NYT Top Stories</b>\n\n")
	count := 0
	for i := 0; i < limit; i++ {
		it := feed.Channel.Items[i]
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
		meta := []string{}
		if c := strings.TrimSpace(it.Creator); c != "" {
			meta = append(meta, "<b>By:</b> "+html.EscapeString(c))
		}
		if d := nytFormatDate(it.PubDate); d != "" {
			meta = append(meta, "<b>Date:</b> "+html.EscapeString(d))
		}
		if len(meta) > 0 {
			b.WriteString("    " + strings.Join(meta, " | ") + "\n")
		}
		b.WriteString("\n")
	}
	if count == 0 {
		m.Reply("<b>Error:</b> failed to parse NYT stories.")
		return nil
	}
	m.Reply(b.String(), &tg.SendOptions{LinkPreview: false})
	return nil
}

func registerNYTimesHandlers() {
	c := Client
	c.On("cmd:nyt", NYTimesHandler)
}

func init() {
	QueueHandlerRegistration(registerNYTimesHandlers)
}
