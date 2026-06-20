package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type ddgRelatedTopic struct {
	FirstURL string            `json:"FirstURL"`
	Text     string            `json:"Text"`
	Topics   []ddgRelatedTopic `json:"Topics"`
}

type ddgResponse struct {
	Heading        string            `json:"Heading"`
	AbstractText   string            `json:"AbstractText"`
	AbstractURL    string            `json:"AbstractURL"`
	AbstractSource string            `json:"AbstractSource"`
	Answer         string            `json:"Answer"`
	AnswerType     string            `json:"AnswerType"`
	Definition     string            `json:"Definition"`
	DefinitionURL  string            `json:"DefinitionURL"`
	Image          string            `json:"Image"`
	Entity         string            `json:"Entity"`
	RelatedTopics  []ddgRelatedTopic `json:"RelatedTopics"`
}

func flattenDDGTopics(topics []ddgRelatedTopic) []ddgRelatedTopic {
	var out []ddgRelatedTopic
	for _, t := range topics {
		if len(t.Topics) > 0 {
			out = append(out, flattenDDGTopics(t.Topics)...)
		} else if t.Text != "" {
			out = append(out, t)
		}
	}
	return out
}

func SearchHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("Usage: <code>/search &lt;query&gt;</code>")
		return nil
	}
	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json&no_html=1&skip_disambig=1"
	resp, err := client.Get(endpoint)
	if err != nil {
		m.Reply("couldn't reach DuckDuckGo: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("DuckDuckGo HTTP %d", resp.StatusCode))
		return nil
	}
	var data ddgResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't parse response: " + html.EscapeString(err.Error()))
		return nil
	}
	heading := data.Heading
	if heading == "" {
		heading = query
	}
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(heading))
	b.WriteString("</b>")
	if data.Entity != "" {
		b.WriteString(" <i>(")
		b.WriteString(html.EscapeString(data.Entity))
		b.WriteString(")</i>")
	}
	b.WriteString("\n")
	abstract := data.AbstractText
	if abstract == "" {
		abstract = data.Definition
	}
	if abstract == "" && data.Answer != "" {
		abstract = data.Answer
		if data.AnswerType != "" {
			abstract = "[" + data.AnswerType + "] " + abstract
		}
	}
	if abstract != "" {
		if len(abstract) > 700 {
			abstract = abstract[:700] + "..."
		}
		b.WriteString("\n")
		b.WriteString(html.EscapeString(abstract))
		b.WriteString("\n")
	}
	src := data.AbstractSource
	srcURL := data.AbstractURL
	if srcURL == "" {
		srcURL = data.DefinitionURL
	}
	if srcURL != "" {
		b.WriteString("\n<b>Source:</b> <a href=\"")
		b.WriteString(html.EscapeString(srcURL))
		b.WriteString("\">")
		if src != "" {
			b.WriteString(html.EscapeString(src))
		} else {
			b.WriteString("link")
		}
		b.WriteString("</a>")
	}
	if data.Image != "" {
		imgURL := data.Image
		if strings.HasPrefix(imgURL, "/") {
			imgURL = "https://duckduckgo.com" + imgURL
		}
		b.WriteString("\n<b>Image:</b> <a href=\"")
		b.WriteString(html.EscapeString(imgURL))
		b.WriteString("\">preview</a>")
	}
	flat := flattenDDGTopics(data.RelatedTopics)
	if len(flat) > 0 {
		b.WriteString("\n\n<b>Related:</b>")
		max := 5
		if len(flat) < max {
			max = len(flat)
		}
		for i := 0; i < max; i++ {
			t := flat[i]
			text := t.Text
			if len(text) > 120 {
				text = text[:120] + "..."
			}
			b.WriteString("\n• ")
			if t.FirstURL != "" {
				b.WriteString("<a href=\"")
				b.WriteString(html.EscapeString(t.FirstURL))
				b.WriteString("\">")
				b.WriteString(html.EscapeString(text))
				b.WriteString("</a>")
			} else {
				b.WriteString(html.EscapeString(text))
			}
		}
	}
	if abstract == "" && len(flat) == 0 && srcURL == "" {
		fallback := "https://duckduckgo.com/?q=" + url.QueryEscape(query)
		m.Reply("No instant answer for <b>" + html.EscapeString(query) + "</b>\nTry: <a href=\"" + html.EscapeString(fallback) + "\">DuckDuckGo</a>")
		return nil
	}
	m.Reply(b.String())
	return nil
}

func init() { QueueHandlerRegistration(registerOpenSearchHandlers) }
func registerOpenSearchHandlers() {
	c := Client
	c.On("cmd:search", SearchHandler)
}
