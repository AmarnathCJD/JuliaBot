package modules

import (
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type wikiThumbnail struct {
	Source string `json:"source"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type wikiContentURLs struct {
	Page string `json:"page"`
}

type wikiURLBundle struct {
	Desktop wikiContentURLs `json:"desktop"`
	Mobile  wikiContentURLs `json:"mobile"`
}

type wikiSummaryResponse struct {
	Type         string         `json:"type"`
	Title        string         `json:"title"`
	DisplayTitle string         `json:"displaytitle"`
	Description  string         `json:"description"`
	Extract      string         `json:"extract"`
	Thumbnail    *wikiThumbnail `json:"thumbnail"`
	Originalimg  *wikiThumbnail `json:"originalimage"`
	ContentURLs  wikiURLBundle  `json:"content_urls"`
	Detail       string         `json:"detail"`
	Title404     string         `json:"title"`
}

func wikiHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func wikiTrimExtract(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if idx := strings.Index(s, "\n"); idx > 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	cut := string(runes[:max])
	if i := strings.LastIndex(cut, " "); i > max/2 {
		cut = cut[:i]
	}
	return strings.TrimSpace(cut) + "..."
}

func wikiFetchSummary(term string) (*wikiSummaryResponse, int, error) {
	endpoint := "https://en.wikipedia.org/api/rest_v1/page/summary/" + url.PathEscape(strings.ReplaceAll(term, " ", "_"))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0 (https://github.com/amarnathcjd)")
	req.Header.Set("Accept", "application/json")
	resp, err := wikiHTTPClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var out wikiSummaryResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, resp.StatusCode, err
	}
	return &out, resp.StatusCode, nil
}

func wikiDownloadThumb(src string) (string, error) {
	if src == "" {
		return "", nil
	}
	req, err := http.NewRequest(http.MethodGet, src, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0 (https://github.com/amarnathcjd)")
	resp, err := wikiHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", nil
	}
	ext := strings.ToLower(filepath.Ext(src))
	if ext == "" || len(ext) > 5 {
		ext = ".jpg"
	}
	tmp, err := os.CreateTemp("", "wiki_thumb_*"+ext)
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func WikiHandler(m *tg.NewMessage) error {
	term := strings.TrimSpace(m.Args())
	if term == "" {
		m.Reply("<b>Usage:</b> <code>/wiki &lt;term&gt;</code>")
		return nil
	}
	status, _ := m.Reply("<i>Searching Wikipedia for <code>" + html.EscapeString(term) + "</code>...</i>")
	data, code, err := wikiFetchSummary(term)
	if err != nil {
		msg := "<b>Wikipedia lookup failed:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if code == 404 {
		msg := "<b>No Wikipedia article found for</b> <code>" + html.EscapeString(term) + "</code>."
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if code >= 400 || data == nil {
		detail := ""
		if data != nil {
			detail = data.Detail
		}
		if detail == "" {
			detail = "request failed"
		}
		msg := "<b>Wikipedia lookup failed:</b> <code>" + html.EscapeString(detail) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	title := data.Title
	if title == "" {
		title = data.DisplayTitle
	}
	if title == "" {
		title = term
	}
	pageURL := data.ContentURLs.Desktop.Page
	if pageURL == "" {
		pageURL = data.ContentURLs.Mobile.Page
	}
	if pageURL == "" {
		pageURL = "https://en.wikipedia.org/wiki/" + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	}

	var sb strings.Builder
	sb.WriteString("<b>")
	sb.WriteString(html.EscapeString(title))
	sb.WriteString("</b>")
	if data.Description != "" {
		sb.WriteString("\n<i>")
		sb.WriteString(html.EscapeString(data.Description))
		sb.WriteString("</i>")
	}
	sb.WriteString("\n\n")

	if strings.EqualFold(data.Type, "disambiguation") {
		sb.WriteString("<i>This term refers to multiple things. Please be more specific.</i>\n\n")
	}

	extract := wikiTrimExtract(data.Extract, 600)
	if extract != "" {
		sb.WriteString(html.EscapeString(extract))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("<i>No summary available.</i>\n\n")
	}

	sb.WriteString("<a href=\"")
	sb.WriteString(html.EscapeString(pageURL))
	sb.WriteString("\">Read on Wikipedia</a>")

	out := sb.String()

	thumbURL := ""
	if data.Thumbnail != nil {
		thumbURL = data.Thumbnail.Source
	}
	if thumbURL == "" && data.Originalimg != nil {
		thumbURL = data.Originalimg.Source
	}

	if thumbURL != "" && !strings.EqualFold(data.Type, "disambiguation") {
		path, err := wikiDownloadThumb(thumbURL)
		if err == nil && path != "" {
			defer os.Remove(path)
			if status != nil {
				status.Delete()
			}
			m.ReplyMedia(path, &tg.MediaOptions{Caption: out})
			return nil
		}
	}

	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func init() { QueueHandlerRegistration(registerWikiHandlers) }

func registerWikiHandlers() {
	c := Client
	c.On("cmd:wiki", WikiHandler)
}
