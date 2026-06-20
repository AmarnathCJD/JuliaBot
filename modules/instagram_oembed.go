package modules

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var igOgMetaPattern = regexp.MustCompile(`<meta\s+property="og:([a-z_]+)"\s+content="([^"]*)"`)
var igURLPattern = regexp.MustCompile(`https?://(?:www\.)?instagram\.com/[^\s]+`)

type igMeta struct {
	Title       string
	Description string
	Image       string
	URL         string
	SiteName    string
	Type        string
}

func fetchInstagramMeta(target string) (*igMeta, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	matches := igOgMetaPattern.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no og metadata found")
	}
	meta := &igMeta{}
	for _, mt := range matches {
		key := mt[1]
		val := html.UnescapeString(mt[2])
		switch key {
		case "title":
			meta.Title = val
		case "description":
			meta.Description = val
		case "image":
			meta.Image = val
		case "url":
			meta.URL = val
		case "site_name":
			meta.SiteName = val
		case "type":
			meta.Type = val
		}
	}
	if meta.Title == "" && meta.Description == "" && meta.Image == "" {
		return nil, fmt.Errorf("no og metadata found")
	}
	return meta, nil
}

func normalizeIgURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if m := igURLPattern.FindString(raw); m != "" {
		return m
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		if strings.HasPrefix(raw, "instagram.com") || strings.HasPrefix(raw, "www.instagram.com") {
			return "https://" + raw
		}
	}
	if strings.Contains(raw, "instagram.com") {
		return raw
	}
	return ""
}

func IgMetaHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/igmeta &lt;instagram url&gt;</code>")
		return nil
	}
	target := normalizeIgURL(arg)
	if target == "" {
		m.Reply("<b>Not a valid instagram.com URL.</b>")
		return nil
	}
	status, _ := m.Reply("Fetching <code>" + html.EscapeString(target) + "</code>...")
	meta, err := fetchInstagramMeta(target)
	if err != nil {
		status.Edit("<b>Failed:</b> " + html.EscapeString(err.Error()) + "\n<i>Instagram may require login for posts/reels. Try a profile URL.</i>")
		return nil
	}
	var b strings.Builder
	b.WriteString("<b>Instagram Metadata</b>\n")
	if meta.Title != "" {
		b.WriteString("\n<b>Title:</b> ")
		b.WriteString(html.EscapeString(meta.Title))
	}
	if meta.Type != "" {
		b.WriteString("\n<b>Type:</b> ")
		b.WriteString(html.EscapeString(meta.Type))
	}
	if meta.SiteName != "" {
		b.WriteString("\n<b>Site:</b> ")
		b.WriteString(html.EscapeString(meta.SiteName))
	}
	if meta.Description != "" {
		b.WriteString("\n<b>Description:</b> ")
		b.WriteString(html.EscapeString(meta.Description))
	}
	if meta.URL != "" {
		b.WriteString("\n<b>URL:</b> <a href=\"")
		b.WriteString(html.EscapeString(meta.URL))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(meta.URL))
		b.WriteString("</a>")
	}
	caption := b.String()
	if meta.Image != "" {
		if _, err := m.ReplyMedia(meta.Image, &tg.MediaOptions{Caption: caption}); err != nil {
			status.Edit(caption+"\n\n<b>Image:</b> <a href=\""+html.EscapeString(meta.Image)+"\">link</a>", &tg.SendOptions{LinkPreview: false})
			return nil
		}
		status.Delete()
		return nil
	}
	status.Edit(caption, &tg.SendOptions{LinkPreview: false})
	return nil
}

func registerIgMetaHandlers() {
	c := Client
	c.On("cmd:igmeta", IgMetaHandler)
}

func init() { QueueHandlerRegistration(registerIgMetaHandlers) }
