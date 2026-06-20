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

type cs2SkinNamed struct {
	Name string `json:"name"`
}

type cs2SkinRarity struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type cs2Skin struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Weapon      cs2SkinNamed    `json:"weapon"`
	Category    cs2SkinNamed    `json:"category"`
	Pattern     cs2SkinNamed    `json:"pattern"`
	MinFloat    float64         `json:"min_float"`
	MaxFloat    float64         `json:"max_float"`
	Rarity      cs2SkinRarity   `json:"rarity"`
	StatTrak    bool            `json:"stattrak"`
	Souvenir    bool            `json:"souvenir"`
	PaintIndex  string          `json:"paint_index"`
	Wears       []cs2SkinNamed  `json:"wears"`
	Collections []cs2SkinNamed  `json:"collections"`
	Crates      []cs2SkinNamed  `json:"crates"`
	Team        cs2SkinNamed    `json:"team"`
	Image       string          `json:"image"`
}

var (
	cs2SkinMu       sync.Mutex
	cs2SkinCache    []cs2Skin
	cs2SkinExpires  time.Time
	cs2SkinCacheTTL = 6 * time.Hour
)

const cs2SkinsURL = "https://raw.githubusercontent.com/ByMykel/CSGO-API/main/public/api/en/skins.json"

func cs2SkinsLoad() ([]cs2Skin, error) {
	cs2SkinMu.Lock()
	if cs2SkinCache != nil && time.Now().Before(cs2SkinExpires) {
		defer cs2SkinMu.Unlock()
		return cs2SkinCache, nil
	}
	cs2SkinMu.Unlock()

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", cs2SkinsURL, nil)
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
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var skins []cs2Skin
	if err := json.Unmarshal(body, &skins); err != nil {
		return nil, err
	}
	cs2SkinMu.Lock()
	cs2SkinCache = skins
	cs2SkinExpires = time.Now().Add(cs2SkinCacheTTL)
	cs2SkinMu.Unlock()
	return skins, nil
}

func cs2SkinFind(skins []cs2Skin, query string) *cs2Skin {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	for i := range skins {
		if strings.EqualFold(skins[i].Name, q) {
			return &skins[i]
		}
	}
	for i := range skins {
		if strings.ToLower(skins[i].Name) == q {
			return &skins[i]
		}
	}
	tokens := strings.Fields(q)
	var best *cs2Skin
	bestScore := -1
	for i := range skins {
		ln := strings.ToLower(skins[i].Name)
		if !strings.Contains(ln, tokens[0]) {
			continue
		}
		score := 0
		for _, t := range tokens {
			if strings.Contains(ln, t) {
				score++
			}
		}
		if score == len(tokens) && score > bestScore {
			bestScore = score
			best = &skins[i]
		}
	}
	if best != nil {
		return best
	}
	for i := range skins {
		if strings.Contains(strings.ToLower(skins[i].Name), q) {
			return &skins[i]
		}
	}
	return nil
}

func cs2SkinFormat(s *cs2Skin) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔫 <b>%s</b>\n", html.EscapeString(s.Name)))
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	if s.Weapon.Name != "" {
		b.WriteString(fmt.Sprintf("<b>Weapon:</b> <code>%s</code>\n", html.EscapeString(s.Weapon.Name)))
	}
	if s.Category.Name != "" {
		b.WriteString(fmt.Sprintf("<b>Category:</b> <code>%s</code>\n", html.EscapeString(s.Category.Name)))
	}
	if s.Pattern.Name != "" {
		b.WriteString(fmt.Sprintf("<b>Pattern:</b> <code>%s</code>\n", html.EscapeString(s.Pattern.Name)))
	}
	if s.Rarity.Name != "" {
		b.WriteString(fmt.Sprintf("<b>Rarity:</b> <code>%s</code>\n", html.EscapeString(s.Rarity.Name)))
	}
	if s.Team.Name != "" {
		b.WriteString(fmt.Sprintf("<b>Team:</b> <code>%s</code>\n", html.EscapeString(s.Team.Name)))
	}
	b.WriteString(fmt.Sprintf("<b>Float:</b> <code>%.2f – %.2f</code>\n", s.MinFloat, s.MaxFloat))
	b.WriteString(fmt.Sprintf("<b>StatTrak:</b> <code>%s</code>  <b>Souvenir:</b> <code>%s</code>\n", cs2SkinYesNo(s.StatTrak), cs2SkinYesNo(s.Souvenir)))
	if s.PaintIndex != "" {
		b.WriteString(fmt.Sprintf("<b>Paint Index:</b> <code>%s</code>\n", html.EscapeString(s.PaintIndex)))
	}
	if len(s.Wears) > 0 {
		parts := make([]string, 0, len(s.Wears))
		for _, w := range s.Wears {
			if w.Name != "" {
				parts = append(parts, html.EscapeString(w.Name))
			}
		}
		if len(parts) > 0 {
			b.WriteString(fmt.Sprintf("<b>Wears:</b> <i>%s</i>\n", strings.Join(parts, ", ")))
		}
	}
	if len(s.Collections) > 0 {
		parts := make([]string, 0, len(s.Collections))
		for _, c := range s.Collections {
			if c.Name != "" {
				parts = append(parts, html.EscapeString(c.Name))
			}
		}
		if len(parts) > 0 {
			b.WriteString(fmt.Sprintf("<b>Collection:</b> <i>%s</i>\n", strings.Join(parts, ", ")))
		}
	} else {
		b.WriteString("<b>Collection:</b> <i>None</i>\n")
	}
	if len(s.Crates) > 0 {
		parts := make([]string, 0, len(s.Crates))
		for i, c := range s.Crates {
			if i >= 3 {
				parts = append(parts, "…")
				break
			}
			if c.Name != "" {
				parts = append(parts, html.EscapeString(c.Name))
			}
		}
		if len(parts) > 0 {
			b.WriteString(fmt.Sprintf("<b>Crates:</b> <i>%s</i>\n", strings.Join(parts, ", ")))
		}
	}
	return b.String()
}

func cs2SkinYesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func CS2SkinHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/cs2skin &lt;name&gt;</code>\n<b>Example:</b> <code>/cs2skin AK-47 Redline</code>")
		return nil
	}

	status, _ := m.Reply("<i>Searching CS2 skin database...</i>")

	skins, err := cs2SkinsLoad()
	if err != nil {
		if status != nil {
			status.Edit("<b>Failed to fetch skin database.</b>")
		}
		return nil
	}

	skin := cs2SkinFind(skins, arg)
	if skin == nil {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>No skin found matching:</b> <code>%s</code>", html.EscapeString(arg)))
		}
		return nil
	}

	caption := cs2SkinFormat(skin)
	if len(caption) > 1024 {
		caption = caption[:1020] + "..."
	}

	if skin.Image != "" {
		if _, err := m.ReplyMedia(skin.Image, &tg.MediaOptions{Caption: caption}); err == nil {
			if status != nil {
				status.Delete()
			}
			return nil
		}
	}

	if status != nil {
		status.Edit(caption)
	} else {
		m.Reply(caption)
	}
	return nil
}

func registerCS2SkinHandlers() {
	c := Client
	c.On("cmd:cs2skin", CS2SkinHandler)
}

func init() {
	QueueHandlerRegistration(registerCS2SkinHandlers)
}
