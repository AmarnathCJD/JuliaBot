package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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
			b.WriteString("<b>Author:</b> " + html.EscapeString(data.AuthorName) + "\n")
		}
	}
	if text != "" {
		b.WriteString("\n" + html.EscapeString(text) + "\n")
	}
	b.WriteString(fmt.Sprintf("\n<a href=\"%s\">View on X</a>", html.EscapeString(data.URL)))

	m.Reply(b.String(), &tg.SendOptions{LinkPreview: false})
	return nil
}

func init() { QueueHandlerRegistration(registerTweetHandlers) }
func registerTweetHandlers() {
	c := Client
	c.On("cmd:tweet", TweetHandler)
}
