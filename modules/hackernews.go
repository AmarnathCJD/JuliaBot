package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var hnHTTPClient = &http.Client{Timeout: 30 * time.Second}

type hnStoryItem struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Score       int    `json:"score"`
	URL         string `json:"url"`
	Descendants int    `json:"descendants"`
	By          string `json:"by"`
	Type        string `json:"type"`
}

func hnFetchTopStoryIDs() ([]int64, error) {
	req, err := http.NewRequest(http.MethodGet, "https://hacker-news.firebaseio.com/v0/topstories.json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := hnHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("topstories status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var ids []int64
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func hnFetchItem(id int64) (*hnStoryItem, error) {
	endpoint := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := hnHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("item %d status %d", id, resp.StatusCode)
	}
	var item hnStoryItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

func hnCommentsURL(id int64) string {
	return fmt.Sprintf("https://news.ycombinator.com/item?id=%d", id)
}

func HackerNewsHandler(m *tg.NewMessage) error {
	ids, err := hnFetchTopStoryIDs()
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch Hacker News top stories.")
		return nil
	}
	if len(ids) == 0 {
		m.Reply("<b>Error:</b> no stories available.")
		return nil
	}
	limit := 10
	if len(ids) < limit {
		limit = len(ids)
	}
	selected := ids[:limit]

	results := make([]*hnStoryItem, limit)
	var wg sync.WaitGroup
	for i, id := range selected {
		wg.Add(1)
		go func(idx int, sid int64) {
			defer wg.Done()
			item, err := hnFetchItem(sid)
			if err == nil {
				results[idx] = item
			}
		}(i, id)
	}
	wg.Wait()

	var b strings.Builder
	b.WriteString("<b>Hacker News Top 10</b>\n\n")
	count := 0
	for i, item := range results {
		if item == nil || item.Title == "" {
			continue
		}
		count++
		link := item.URL
		if link == "" {
			link = hnCommentsURL(item.ID)
		}
		b.WriteString(fmt.Sprintf("<b>%d.</b> <a href=\"%s\">%s</a>\n", i+1, html.EscapeString(link), html.EscapeString(item.Title)))
		b.WriteString(fmt.Sprintf("    <b>Score:</b> %d | <b>Comments:</b> <a href=\"%s\">%d</a>\n\n", item.Score, html.EscapeString(hnCommentsURL(item.ID)), item.Descendants))
	}
	if count == 0 {
		m.Reply("<b>Error:</b> failed to fetch any story details.")
		return nil
	}
	m.Reply(b.String(), &tg.SendOptions{LinkPreview: false})
	return nil
}

func registerHackerNewsHandlers() {
	c := Client
	c.On("cmd:hn", HackerNewsHandler)
}

func init() {
	QueueHandlerRegistration(registerHackerNewsHandlers)
}
