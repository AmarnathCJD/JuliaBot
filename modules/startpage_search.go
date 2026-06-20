package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func StartpageSearchHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("<b>Usage:</b> <code>/swiki &lt;query&gt;</code>")
		return nil
	}

	status, _ := m.Reply("<i>Searching Wikipedia for <code>" + html.EscapeString(query) + "</code>...</i>")

	endpoint := "https://en.wikipedia.org/w/api.php?action=opensearch&search=" + url.QueryEscape(query) + "&limit=5&format=json"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		msg := "<b>Request error:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0 (https://github.com/amarnathcjd)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		msg := "<b>Wikipedia unreachable:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("<b>Wikipedia HTTP %d</b>", resp.StatusCode)
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		msg := "<b>Read failed:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		msg := "<b>Parse failed:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if len(raw) < 4 {
		msg := "<b>Unexpected response shape.</b>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	var titles, snippets, urls []string
	json.Unmarshal(raw[1], &titles)
	json.Unmarshal(raw[2], &snippets)
	json.Unmarshal(raw[3], &urls)

	if len(titles) == 0 {
		msg := "<b>No results for</b> <code>" + html.EscapeString(query) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	var sb strings.Builder
	sb.WriteString("<b>Wikipedia results for</b> <code>")
	sb.WriteString(html.EscapeString(query))
	sb.WriteString("</code>\n")

	for i, t := range titles {
		if i >= 5 {
			break
		}
		sb.WriteString("\n<b>")
		sb.WriteString(fmt.Sprintf("%d.", i+1))
		sb.WriteString("</b> ")
		link := ""
		if i < len(urls) {
			link = urls[i]
		}
		if link != "" {
			sb.WriteString("<a href=\"")
			sb.WriteString(html.EscapeString(link))
			sb.WriteString("\">")
			sb.WriteString(html.EscapeString(t))
			sb.WriteString("</a>")
		} else {
			sb.WriteString(html.EscapeString(t))
		}
		snippet := ""
		if i < len(snippets) {
			snippet = strings.TrimSpace(snippets[i])
		}
		if snippet != "" {
			if len(snippet) > 220 {
				snippet = snippet[:220] + "..."
			}
			sb.WriteString("\n<i>")
			sb.WriteString(html.EscapeString(snippet))
			sb.WriteString("</i>")
		}
		sb.WriteString("\n")
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func init() { QueueHandlerRegistration(registerStartpageSearchHandlers) }

func registerStartpageSearchHandlers() {
	c := Client
	c.On("cmd:swiki", StartpageSearchHandler)
}
