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

type steamSearchResponse struct {
	Total int               `json:"total"`
	Items []steamSearchItem `json:"items"`
}

type steamSearchItem struct {
	Type      string           `json:"type"`
	Name      string           `json:"name"`
	ID        int              `json:"id"`
	Price     *steamPriceBlock `json:"price"`
	TinyImage string           `json:"tiny_image"`
	Metascore string           `json:"metascore"`
	Platforms steamPlatforms   `json:"platforms"`
}

type steamPriceBlock struct {
	Currency string `json:"currency"`
	Initial  int    `json:"initial"`
	Final    int    `json:"final"`
}

type steamPlatforms struct {
	Windows bool `json:"windows"`
	Mac     bool `json:"mac"`
	Linux   bool `json:"linux"`
}

func formatSteamPrice(p *steamPriceBlock) string {
	if p == nil {
		return "Free / N/A"
	}
	final := fmt.Sprintf("%s %.2f", p.Currency, float64(p.Final)/100.0)
	if p.Initial > p.Final && p.Initial > 0 {
		discount := int(((float64(p.Initial) - float64(p.Final)) / float64(p.Initial)) * 100)
		return fmt.Sprintf("<s>%s %.2f</s> %s (-%d%%)", p.Currency, float64(p.Initial)/100.0, final, discount)
	}
	if p.Final == 0 {
		return "Free"
	}
	return final
}

func formatSteamPlatforms(p steamPlatforms) string {
	var parts []string
	if p.Windows {
		parts = append(parts, "Windows")
	}
	if p.Mac {
		parts = append(parts, "Mac")
	}
	if p.Linux {
		parts = append(parts, "Linux")
	}
	return strings.Join(parts, ", ")
}

func SteamSearchHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("<b>Usage:</b> <code>/steam &lt;game&gt;</code>")
		return nil
	}
	status, _ := m.Reply("Searching Steam for <code>" + html.EscapeString(query) + "</code>...")
	client := &http.Client{Timeout: 30 * time.Second}
	apiURL := "https://store.steampowered.com/api/storesearch?term=" + url.QueryEscape(query) + "&l=en&cc=US"
	resp, err := client.Get(apiURL)
	if err != nil {
		status.Edit("couldn't search Steam: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		status.Edit(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data steamSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		status.Edit("couldn't parse Steam response: " + html.EscapeString(err.Error()))
		return nil
	}
	if len(data.Items) == 0 {
		status.Edit("<b>No Steam results for:</b> <code>" + html.EscapeString(query) + "</code>")
		return nil
	}
	limit := 3
	if len(data.Items) < limit {
		limit = len(data.Items)
	}
	var b strings.Builder
	b.WriteString("<b>Steam Search:</b> <code>")
	b.WriteString(html.EscapeString(query))
	b.WriteString("</code>\n")
	for i := 0; i < limit; i++ {
		it := data.Items[i]
		storeURL := fmt.Sprintf("https://store.steampowered.com/app/%d", it.ID)
		b.WriteString("\n<b>")
		b.WriteString(fmt.Sprintf("%d. ", i+1))
		b.WriteString("<a href=\"")
		b.WriteString(storeURL)
		b.WriteString("\">")
		b.WriteString(html.EscapeString(it.Name))
		b.WriteString("</a></b>")
		b.WriteString("\n  Price: ")
		b.WriteString(formatSteamPrice(it.Price))
		if strings.TrimSpace(it.Metascore) != "" {
			b.WriteString("\n  Metascore: ")
			b.WriteString(html.EscapeString(it.Metascore))
		}
		plat := formatSteamPlatforms(it.Platforms)
		if plat != "" {
			b.WriteString("\n  Platforms: ")
			b.WriteString(plat)
		}
		b.WriteString("\n")
	}
	caption := strings.TrimRight(b.String(), "\n")
	topImage := data.Items[0].TinyImage
	if strings.TrimSpace(topImage) != "" {
		if _, err := m.ReplyMedia(topImage, &tg.MediaOptions{Caption: caption}); err != nil {
			status.Edit(caption, &tg.SendOptions{LinkPreview: false})
			return nil
		}
		status.Delete()
		return nil
	}
	status.Edit(caption, &tg.SendOptions{LinkPreview: false})
	return nil
}

func init() { QueueHandlerRegistration(registerSteamSearchHandlers) }
func registerSteamSearchHandlers() {
	c := Client
	c.On("cmd:steam", SteamSearchHandler)
}
