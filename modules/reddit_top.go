package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type redditTopChild struct {
	Data struct {
		Title       string  `json:"title"`
		Score       int     `json:"score"`
		Permalink   string  `json:"permalink"`
		URL         string  `json:"url"`
		Author      string  `json:"author"`
		Subreddit   string  `json:"subreddit"`
		NumComments int     `json:"num_comments"`
		UpvoteRatio float64 `json:"upvote_ratio"`
		Over18      bool    `json:"over_18"`
		IsSelf      bool    `json:"is_self"`
		Stickied    bool    `json:"stickied"`
	} `json:"data"`
}

type redditTopResponse struct {
	Data struct {
		Children []redditTopChild `json:"children"`
	} `json:"data"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

func formatRedditScore(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func RedditTopHandler(m *tg.NewMessage) error {
	sub := strings.TrimSpace(m.Args())
	sub = strings.TrimPrefix(sub, "r/")
	sub = strings.TrimPrefix(sub, "/r/")
	sub = strings.TrimSpace(sub)
	if sub == "" {
		m.Reply("<b>Reddit Top</b>\n\nUsage: <code>/redditop &lt;subreddit&gt;</code>")
		return nil
	}
	parts := strings.Fields(sub)
	sub = parts[0]
	for _, r := range sub {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			m.Reply("<b>Invalid subreddit name.</b>")
			return nil
		}
	}

	url := fmt.Sprintf("https://www.reddit.com/r/%s/top.json?limit=5&t=day", sub)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		m.Reply("couldn't build request: " + err.Error())
		return nil
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0 (Telegram Bot)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.Reply("couldn't reach Reddit: " + err.Error())
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		m.Reply("<b>Subreddit not found:</b> r/" + html.EscapeString(sub))
		return nil
	}
	if resp.StatusCode == 403 {
		m.Reply("<b>Subreddit is private or banned:</b> r/" + html.EscapeString(sub))
		return nil
	}
	if resp.StatusCode == 429 {
		m.Reply("Reddit rate limit reached. Try again shortly.")
		return nil
	}
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("Reddit returned HTTP %d", resp.StatusCode))
		return nil
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "json") {
		m.Reply("Reddit returned a non-JSON response (likely blocked or rate-limited).")
		return nil
	}

	var data redditTopResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't parse Reddit response: " + err.Error())
		return nil
	}
	if data.Error != 0 {
		msg := data.Message
		if msg == "" {
			msg = fmt.Sprintf("error %d", data.Error)
		}
		m.Reply("Reddit error: " + html.EscapeString(msg))
		return nil
	}
	if len(data.Data.Children) == 0 {
		m.Reply("<b>No posts found for r/" + html.EscapeString(sub) + ".</b>")
		return nil
	}

	subDisplay := sub
	if len(data.Data.Children) > 0 && data.Data.Children[0].Data.Subreddit != "" {
		subDisplay = data.Data.Children[0].Data.Subreddit
	}

	out := "<b>Top posts of the day in r/" + html.EscapeString(subDisplay) + "</b>\n\n"
	count := 0
	for _, child := range data.Data.Children {
		if count >= 5 {
			break
		}
		d := child.Data
		if d.Title == "" {
			continue
		}
		count++
		title := d.Title
		if len(title) > 180 {
			title = title[:177] + "..."
		}
		link := "https://www.reddit.com" + d.Permalink
		out += fmt.Sprintf("<b>%d.</b> <a href=\"%s\">%s</a>\n", count, html.EscapeString(link), html.EscapeString(title))
		out += fmt.Sprintf("    <b>Score:</b> %s", formatRedditScore(d.Score))
		if d.UpvoteRatio > 0 {
			out += fmt.Sprintf(" (%d%% upvoted)", int(d.UpvoteRatio*100))
		}
		out += fmt.Sprintf(" • <b>Comments:</b> %s", formatRedditScore(d.NumComments))
		if d.Author != "" {
			out += " • <b>By:</b> u/" + html.EscapeString(d.Author)
		}
		if d.Over18 {
			out += " <b>[NSFW]</b>"
		}
		out += "\n"
		if !d.IsSelf && d.URL != "" && d.URL != link {
			linkHost := d.URL
			if len(linkHost) > 80 {
				linkHost = linkHost[:77] + "..."
			}
			out += "    <a href=\"" + html.EscapeString(d.URL) + "\">" + html.EscapeString(linkHost) + "</a>\n"
		}
		out += "\n"
	}

	if count == 0 {
		m.Reply("<b>No posts found for r/" + html.EscapeString(sub) + ".</b>")
		return nil
	}

	m.Reply(out, &tg.SendOptions{LinkPreview: false})
	return nil
}

func init() { QueueHandlerRegistration(registerRedditTopHandlers) }
func registerRedditTopHandlers() {
	c := Client
	c.On("cmd:redditop", RedditTopHandler)

	Mods.AddModule("RedditTop", `<b>Reddit Top Module</b>

<b>Commands:</b>
 • /redditop &lt;subreddit&gt; - Top 5 posts of the day from a subreddit

<i>Fetches data from reddit.com. Requires the bot host to have Reddit access.</i>`)
}
