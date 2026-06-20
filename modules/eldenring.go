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

type elderingNameAmount struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type elderingNameScaling struct {
	Name    string `json:"name"`
	Scaling string `json:"scaling"`
}

type elderingItem struct {
	ID                 string                `json:"id"`
	Name               string                `json:"name"`
	Image              string                `json:"image"`
	Description        string                `json:"description"`
	Category           string                `json:"category"`
	Weight             float64               `json:"weight"`
	Attack             []elderingNameAmount  `json:"attack"`
	Defence            []elderingNameAmount  `json:"defence"`
	ScalesWith         []elderingNameScaling `json:"scalesWith"`
	RequiredAttributes []elderingNameAmount  `json:"requiredAttributes"`
	DmgNegation        []elderingNameAmount  `json:"dmgNegation"`
	Resistance         []elderingNameAmount  `json:"resistance"`
	Region             string                `json:"region"`
	Location           string                `json:"location"`
	Drops              []string              `json:"drops"`
	HealthPoints       string                `json:"healthPoints"`
	Type               string                `json:"type"`
	Effect             string                `json:"effect"`
}

type elderingResp struct {
	Success bool           `json:"success"`
	Count   int            `json:"count"`
	Total   int            `json:"total"`
	Data    []elderingItem `json:"data"`
}

func elderingFetch(category, name string) (*elderingItem, int, error) {
	endpoint := fmt.Sprintf("https://eldenring.fanapis.com/api/%s?name=%s&limit=1", category, url.QueryEscape(name))
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, nil
	}
	var r elderingResp
	if jerr := json.Unmarshal(body, &r); jerr != nil {
		return nil, resp.StatusCode, jerr
	}
	if !r.Success || len(r.Data) == 0 {
		return nil, resp.StatusCode, nil
	}
	return &r.Data[0], resp.StatusCode, nil
}

func elderingDownloadImage(imgURL string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", imgURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("image http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func elderingResolveCategory(s string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "weapon", "weapons":
		return "weapons", "Weapon"
	case "armor", "armors", "armour", "armours":
		return "armors", "Armor"
	case "boss", "bosses":
		return "bosses", "Boss"
	case "item", "items":
		return "items", "Item"
	}
	return "", ""
}

func elderingFormatNameAmount(items []elderingNameAmount) string {
	if len(items) == 0 {
		return ""
	}
	parts := []string{}
	for _, it := range items {
		if it.Amount == 0 {
			continue
		}
		val := fmt.Sprintf("%g", it.Amount)
		parts = append(parts, fmt.Sprintf("%s %s", html.EscapeString(it.Name), val))
	}
	return strings.Join(parts, " · ")
}

func elderingFormatScaling(items []elderingNameScaling) string {
	if len(items) == 0 {
		return ""
	}
	parts := []string{}
	for _, it := range items {
		if it.Scaling == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s", html.EscapeString(it.Name), html.EscapeString(it.Scaling)))
	}
	return strings.Join(parts, " · ")
}

func elderingTrunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func elderingBuildCaption(category, label string, it *elderingItem) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("⚔️ <b>%s</b>\n", html.EscapeString(it.Name)))
	b.WriteString(fmt.Sprintf("<i>%s</i>\n", html.EscapeString(label)))
	b.WriteString("━━━━━━━━━━━━\n")

	if it.Category != "" {
		b.WriteString(fmt.Sprintf("<b>Category:</b> <code>%s</code>\n", html.EscapeString(it.Category)))
	}
	if it.Weight > 0 {
		b.WriteString(fmt.Sprintf("<b>Weight:</b> <code>%g</code>\n", it.Weight))
	}

	switch category {
	case "weapons":
		if s := elderingFormatNameAmount(it.Attack); s != "" {
			b.WriteString(fmt.Sprintf("<b>Attack:</b> %s\n", s))
		}
		if s := elderingFormatNameAmount(it.Defence); s != "" {
			b.WriteString(fmt.Sprintf("<b>Defence:</b> %s\n", s))
		}
		if s := elderingFormatScaling(it.ScalesWith); s != "" {
			b.WriteString(fmt.Sprintf("<b>Scales With:</b> %s\n", s))
		}
		if s := elderingFormatNameAmount(it.RequiredAttributes); s != "" {
			b.WriteString(fmt.Sprintf("<b>Requires:</b> %s\n", s))
		}
	case "armors":
		if s := elderingFormatNameAmount(it.DmgNegation); s != "" {
			b.WriteString(fmt.Sprintf("<b>Dmg Negation:</b> %s\n", s))
		}
		if s := elderingFormatNameAmount(it.Resistance); s != "" {
			b.WriteString(fmt.Sprintf("<b>Resistance:</b> %s\n", s))
		}
	case "bosses":
		if it.Region != "" {
			b.WriteString(fmt.Sprintf("<b>Region:</b> <code>%s</code>\n", html.EscapeString(it.Region)))
		}
		if it.Location != "" {
			b.WriteString(fmt.Sprintf("<b>Location:</b> <code>%s</code>\n", html.EscapeString(it.Location)))
		}
		if it.HealthPoints != "" {
			b.WriteString(fmt.Sprintf("<b>HP:</b> <code>%s</code>\n", html.EscapeString(it.HealthPoints)))
		}
		if len(it.Drops) > 0 {
			b.WriteString(fmt.Sprintf("<b>Drops:</b> <code>%s</code>\n", html.EscapeString(strings.Join(it.Drops, ", "))))
		}
	case "items":
		if it.Type != "" && it.Type != "-" {
			b.WriteString(fmt.Sprintf("<b>Type:</b> <code>%s</code>\n", html.EscapeString(it.Type)))
		}
		if it.Effect != "" {
			b.WriteString(fmt.Sprintf("<b>Effect:</b> <code>%s</code>\n", html.EscapeString(it.Effect)))
		}
	}

	if it.Description != "" {
		b.WriteString(fmt.Sprintf("\n<b>Description:</b>\n<i>%s</i>\n", html.EscapeString(elderingTrunc(it.Description, 600))))
	}
	b.WriteString("\n<i>Source: eldenring.fanapis.com</i>")
	return b.String()
}

func EldenRingHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/eldenring &lt;weapon|armor|boss|item&gt; &lt;name&gt;</code>\n<b>Example:</b> <code>/eldenring weapon Uchigatana</code>")
		return nil
	}

	parts := strings.Fields(arg)
	if len(parts) < 2 {
		m.Reply("<b>Usage:</b> <code>/eldenring &lt;weapon|armor|boss|item&gt; &lt;name&gt;</code>")
		return nil
	}

	category, label := elderingResolveCategory(parts[0])
	if category == "" {
		m.Reply("<b>Invalid type.</b> Choose from: <code>weapon</code>, <code>armor</code>, <code>boss</code>, <code>item</code>.")
		return nil
	}

	name := strings.TrimSpace(strings.Join(parts[1:], " "))
	if name == "" {
		m.Reply("<b>Provide a name to search.</b>")
		return nil
	}

	status, _ := m.Reply("<i>Searching the Lands Between...</i>")

	it, code, err := elderingFetch(category, name)
	if err != nil {
		if status != nil {
			status.Edit("<b>Request failed.</b> Could not reach eldenring.fanapis.com.")
		}
		return nil
	}
	if code != 200 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", code))
		}
		return nil
	}
	if it == nil {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>No %s found:</b> <code>%s</code>", html.EscapeString(strings.ToLower(label)), html.EscapeString(name)))
		}
		return nil
	}

	caption := elderingBuildCaption(category, label, it)

	if it.Image != "" {
		imgBytes, ierr := elderingDownloadImage(it.Image)
		if ierr == nil && len(imgBytes) > 0 {
			if status != nil {
				status.Delete()
			}
			fname := strings.ToLower(strings.ReplaceAll(it.Name, " ", "_"))
			if _, merr := m.ReplyMedia(imgBytes, &tg.MediaOptions{
				Caption:  caption,
				FileName: fmt.Sprintf("%s.png", fname),
				MimeType: "image/png",
			}); merr == nil {
				return nil
			}
		}
	}

	if status != nil {
		status.Edit(caption)
	} else {
		m.Reply(caption)
	}
	return nil
}

func registerEldenRingHandlers() {
	c := Client
	c.On("cmd:eldenring", EldenRingHandler)
}

func init() {
	QueueHandlerRegistration(registerEldenRingHandlers)
}
