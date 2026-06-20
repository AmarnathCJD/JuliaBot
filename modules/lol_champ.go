package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type lolChampImage struct {
	Full string `json:"full"`
}

type lolChampInfo struct {
	Attack     int `json:"attack"`
	Defense    int `json:"defense"`
	Magic      int `json:"magic"`
	Difficulty int `json:"difficulty"`
}

type lolChampStats struct {
	HP        float64 `json:"hp"`
	MP        float64 `json:"mp"`
	MoveSpeed float64 `json:"movespeed"`
	Armor     float64 `json:"armor"`
	AttackDmg float64 `json:"attackdamage"`
	AttackRng float64 `json:"attackrange"`
}

type lolChampSpell struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	CooldownBurn string `json:"cooldownBurn"`
	CostBurn     string `json:"costBurn"`
	RangeBurn    string `json:"rangeBurn"`
}

type lolChampPassive struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type lolChampData struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Title   string          `json:"title"`
	Lore    string          `json:"lore"`
	Tags    []string        `json:"tags"`
	Partype string          `json:"partype"`
	Image   lolChampImage   `json:"image"`
	Info    lolChampInfo    `json:"info"`
	Stats   lolChampStats   `json:"stats"`
	Spells  []lolChampSpell `json:"spells"`
	Passive lolChampPassive `json:"passive"`
}

type lolChampResponse struct {
	Data map[string]lolChampData `json:"data"`
}

var lolTagRegex = regexp.MustCompile(`<[^>]+>`)

func lolStripTags(s string) string {
	s = lolTagRegex.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.TrimSpace(s)
}

func lolFormatName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	switch lower {
	case "wukong":
		return "MonkeyKing"
	case "nunu", "nunu and willump", "nunu & willump":
		return "Nunu"
	case "renata", "renata glasc":
		return "Renata"
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '\'' || r == '.' || r == '-'
	})
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(strings.ToLower(p[1:]))
		}
	}
	return b.String()
}

func lolTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func LolHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("Usage: <code>/lol &lt;champion&gt;</code>")
		return nil
	}
	name := lolFormatName(query)
	if name == "" {
		m.Reply("Invalid champion name.")
		return nil
	}
	client := &http.Client{Timeout: 30 * time.Second}
	url := "https://ddragon.leagueoflegends.com/cdn/14.1.1/data/en_US/champion/" + name + ".json"
	resp, err := client.Get(url)
	if err != nil {
		m.Reply("Couldn't fetch champion: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == 403 || resp.StatusCode == 404 {
		m.Reply("Champion <b>" + html.EscapeString(query) + "</b> not found.")
		return nil
	}
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data lolChampResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Couldn't parse champion data.")
		return nil
	}
	if len(data.Data) == 0 {
		m.Reply("No champion data returned.")
		return nil
	}
	var champ lolChampData
	for _, v := range data.Data {
		champ = v
		break
	}
	imgURL := "https://ddragon.leagueoflegends.com/cdn/img/champion/loading/" + champ.ID + "_0.jpg"

	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(champ.Name))
	b.WriteString("</b>")
	if champ.Title != "" {
		b.WriteString(", <i>")
		b.WriteString(html.EscapeString(strings.Title(champ.Title)))
		b.WriteString("</i>")
	}
	b.WriteString("\n")
	if len(champ.Tags) > 0 {
		b.WriteString("\n<b>Role:</b> ")
		b.WriteString(html.EscapeString(strings.Join(champ.Tags, ", ")))
	}
	if champ.Partype != "" {
		b.WriteString("\n<b>Resource:</b> ")
		b.WriteString(html.EscapeString(champ.Partype))
	}
	b.WriteString(fmt.Sprintf("\n<b>Difficulty:</b> %d/10", champ.Info.Difficulty))
	b.WriteString("\n\n<b>Lore:</b>\n<i>")
	b.WriteString(html.EscapeString(lolTruncate(champ.Lore, 600)))
	b.WriteString("</i>")

	b.WriteString("\n\n<b>Base Stats:</b>")
	b.WriteString(fmt.Sprintf("\n- HP: %.0f", champ.Stats.HP))
	if champ.Stats.MP > 0 {
		b.WriteString(fmt.Sprintf("\n- %s: %.0f", html.EscapeString(champ.Partype), champ.Stats.MP))
	}
	b.WriteString(fmt.Sprintf("\n- Armor: %.0f", champ.Stats.Armor))
	b.WriteString(fmt.Sprintf("\n- Attack Damage: %.0f", champ.Stats.AttackDmg))
	b.WriteString(fmt.Sprintf("\n- Attack Range: %.0f", champ.Stats.AttackRng))
	b.WriteString(fmt.Sprintf("\n- Move Speed: %.0f", champ.Stats.MoveSpeed))

	b.WriteString("\n\n<b>Combat Profile:</b>")
	b.WriteString(fmt.Sprintf("\n- Attack: %d/10", champ.Info.Attack))
	b.WriteString(fmt.Sprintf("\n- Defense: %d/10", champ.Info.Defense))
	b.WriteString(fmt.Sprintf("\n- Magic: %d/10", champ.Info.Magic))

	if champ.Passive.Name != "" {
		b.WriteString("\n\n<b>Passive - ")
		b.WriteString(html.EscapeString(champ.Passive.Name))
		b.WriteString(":</b>\n")
		b.WriteString(html.EscapeString(lolTruncate(lolStripTags(champ.Passive.Description), 220)))
	}

	if len(champ.Spells) > 0 {
		keys := []string{"Q", "W", "E", "R"}
		b.WriteString("\n\n<b>Skills:</b>")
		for i, sp := range champ.Spells {
			if i >= 4 {
				break
			}
			b.WriteString("\n\n<b>[")
			b.WriteString(keys[i])
			b.WriteString("] ")
			b.WriteString(html.EscapeString(sp.Name))
			b.WriteString("</b>")
			if sp.CooldownBurn != "" {
				b.WriteString(" <i>(CD ")
				b.WriteString(html.EscapeString(sp.CooldownBurn))
				b.WriteString("s")
				if sp.CostBurn != "" && sp.CostBurn != "0" {
					b.WriteString(" | Cost ")
					b.WriteString(html.EscapeString(sp.CostBurn))
				}
				b.WriteString(")</i>")
			}
			b.WriteString("\n")
			b.WriteString(html.EscapeString(lolTruncate(lolStripTags(sp.Description), 200)))
		}
	}

	caption := b.String()
	if len(caption) > 1024 {
		caption = caption[:1020] + "..."
	}
	if _, err := m.ReplyMedia(imgURL, &tg.MediaOptions{Caption: caption}); err != nil {
		m.Reply(b.String())
	}
	return nil
}

func init() { QueueHandlerRegistration(registerLolHandlers) }
func registerLolHandlers() {
	c := Client
	c.On("cmd:lol", LolHandler)
}
