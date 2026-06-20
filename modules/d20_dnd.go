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

type dndArmorClass struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type dndProficiency struct {
	Value       int `json:"value"`
	Proficiency struct {
		Name string `json:"name"`
	} `json:"proficiency"`
}

type dndAction struct {
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	AttackBonus int    `json:"attack_bonus"`
}

type dndSpecialAbility struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type dndLegendaryAction struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type dndSenses struct {
	Blindsight        string `json:"blindsight,omitempty"`
	Darkvision        string `json:"darkvision,omitempty"`
	Tremorsense       string `json:"tremorsense,omitempty"`
	Truesight         string `json:"truesight,omitempty"`
	PassivePerception int    `json:"passive_perception"`
}

type dndMonsterResp struct {
	Index            string             `json:"index"`
	Name             string             `json:"name"`
	Size             string             `json:"size"`
	Type             string             `json:"type"`
	Subtype          string             `json:"subtype"`
	Alignment        string             `json:"alignment"`
	ArmorClass       []dndArmorClass    `json:"armor_class"`
	HitPoints        int                `json:"hit_points"`
	HitDice          string             `json:"hit_dice"`
	HitPointsRoll    string             `json:"hit_points_roll"`
	Speed            map[string]string  `json:"speed"`
	Strength         int                `json:"strength"`
	Dexterity        int                `json:"dexterity"`
	Constitution     int                `json:"constitution"`
	Intelligence     int                `json:"intelligence"`
	Wisdom           int                `json:"wisdom"`
	Charisma         int                `json:"charisma"`
	Proficiencies    []dndProficiency   `json:"proficiencies"`
	DamageVuln       []string           `json:"damage_vulnerabilities"`
	DamageRes        []string           `json:"damage_resistances"`
	DamageImm        []string           `json:"damage_immunities"`
	ConditionImm     []any              `json:"condition_immunities"`
	Senses           dndSenses          `json:"senses"`
	Languages        string             `json:"languages"`
	ChallengeRating  float64            `json:"challenge_rating"`
	ProficiencyBonus int                `json:"proficiency_bonus"`
	XP               int                `json:"xp"`
	SpecialAbilities []dndSpecialAbility `json:"special_abilities"`
	Actions          []dndAction        `json:"actions"`
	LegendaryActions []dndLegendaryAction `json:"legendary_actions"`
	Reactions        []dndAction        `json:"reactions"`
	Image            string             `json:"image"`
	URL              string             `json:"url"`
}

type dndCacheEntry struct {
	data    *dndMonsterResp
	expires time.Time
}

var (
	dndCacheMu sync.Mutex
	dndCache   = map[string]dndCacheEntry{}
)

const dndCacheTTL = 1 * time.Hour
const dndAPIBase = "https://www.dnd5eapi.co"

func dndSlugify(query string) string {
	s := strings.ToLower(strings.TrimSpace(query))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func dndFetchMonster(slug string) (*dndMonsterResp, int, error) {
	dndCacheMu.Lock()
	if e, ok := dndCache[slug]; ok && time.Now().Before(e.expires) {
		dndCacheMu.Unlock()
		return e.data, 200, nil
	}
	dndCacheMu.Unlock()

	endpoint := dndAPIBase + "/api/monsters/" + slug
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
	var d dndMonsterResp
	if jerr := json.Unmarshal(body, &d); jerr != nil {
		return nil, resp.StatusCode, jerr
	}
	dndCacheMu.Lock()
	dndCache[slug] = dndCacheEntry{data: &d, expires: time.Now().Add(dndCacheTTL)}
	dndCacheMu.Unlock()
	return &d, resp.StatusCode, nil
}

type dndSearchResult struct {
	Index string `json:"index"`
	Name  string `json:"name"`
}

type dndSearchResp struct {
	Count   int               `json:"count"`
	Results []dndSearchResult `json:"results"`
}

func dndSearchMonster(query string) (*dndSearchResp, error) {
	endpoint := dndAPIBase + "/api/monsters/?name=" + strings.ReplaceAll(strings.TrimSpace(query), " ", "+")
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search http %d", resp.StatusCode)
	}
	var s dndSearchResp
	if jerr := json.Unmarshal(body, &s); jerr != nil {
		return nil, jerr
	}
	return &s, nil
}

func dndDownloadImage(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("empty image path")
	}
	url := path
	if strings.HasPrefix(path, "/") {
		url = dndAPIBase + path
	}
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
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

func dndAbilityMod(score int) string {
	mod := (score - 10) / 2
	if score-10 < 0 && (score-10)%2 != 0 {
		mod = (score - 10 - 1) / 2
	}
	if mod >= 0 {
		return fmt.Sprintf("+%d", mod)
	}
	return fmt.Sprintf("%d", mod)
}

func dndFormatCR(cr float64) string {
	switch cr {
	case 0:
		return "0"
	case 0.125:
		return "1/8"
	case 0.25:
		return "1/4"
	case 0.5:
		return "1/2"
	}
	if cr == float64(int(cr)) {
		return fmt.Sprintf("%d", int(cr))
	}
	return fmt.Sprintf("%g", cr)
}

func dndTruncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func dndFormatSpeed(speed map[string]string) string {
	if len(speed) == 0 {
		return "—"
	}
	order := []string{"walk", "fly", "swim", "climb", "burrow"}
	var parts []string
	seen := map[string]bool{}
	for _, k := range order {
		if v, ok := speed[k]; ok {
			parts = append(parts, fmt.Sprintf("%s %s", k, v))
			seen[k] = true
		}
	}
	for k, v := range speed {
		if !seen[k] {
			parts = append(parts, fmt.Sprintf("%s %s", k, v))
		}
	}
	return strings.Join(parts, ", ")
}

func dndFormatSenses(s dndSenses) string {
	var parts []string
	if s.Blindsight != "" {
		parts = append(parts, "blindsight "+s.Blindsight)
	}
	if s.Darkvision != "" {
		parts = append(parts, "darkvision "+s.Darkvision)
	}
	if s.Tremorsense != "" {
		parts = append(parts, "tremorsense "+s.Tremorsense)
	}
	if s.Truesight != "" {
		parts = append(parts, "truesight "+s.Truesight)
	}
	if s.PassivePerception > 0 {
		parts = append(parts, fmt.Sprintf("passive Perception %d", s.PassivePerception))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, ", ")
}

func dndFormatArmorClass(ac []dndArmorClass) string {
	if len(ac) == 0 {
		return "—"
	}
	a := ac[0]
	if a.Type != "" && a.Type != "dex" {
		return fmt.Sprintf("%d (%s)", a.Value, a.Type)
	}
	return fmt.Sprintf("%d", a.Value)
}

func dndFormatStringList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func dndFormatAnyList(items []any) string {
	if len(items) == 0 {
		return "none"
	}
	var out []string
	for _, it := range items {
		switch v := it.(type) {
		case string:
			out = append(out, v)
		case map[string]any:
			if n, ok := v["name"].(string); ok {
				out = append(out, n)
			} else if idx, ok := v["index"].(string); ok {
				out = append(out, idx)
			}
		}
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ", ")
}

func dndBuildCaption(d *dndMonsterResp) string {
	var b strings.Builder
	subtype := ""
	if d.Subtype != "" {
		subtype = fmt.Sprintf(" (%s)", d.Subtype)
	}

	b.WriteString(fmt.Sprintf("\U0001F409 <b>%s</b>\n", html.EscapeString(d.Name)))
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("<i>%s %s%s, %s</i>\n\n",
		html.EscapeString(d.Size),
		html.EscapeString(d.Type),
		html.EscapeString(subtype),
		html.EscapeString(d.Alignment),
	))

	b.WriteString(fmt.Sprintf("<b>AC:</b> <code>%s</code>\n", html.EscapeString(dndFormatArmorClass(d.ArmorClass))))
	hp := fmt.Sprintf("%d", d.HitPoints)
	if d.HitPointsRoll != "" {
		hp = fmt.Sprintf("%d (%s)", d.HitPoints, d.HitPointsRoll)
	} else if d.HitDice != "" {
		hp = fmt.Sprintf("%d (%s)", d.HitPoints, d.HitDice)
	}
	b.WriteString(fmt.Sprintf("<b>HP:</b> <code>%s</code>\n", html.EscapeString(hp)))
	b.WriteString(fmt.Sprintf("<b>Speed:</b> <code>%s</code>\n", html.EscapeString(dndFormatSpeed(d.Speed))))
	b.WriteString(fmt.Sprintf("<b>CR:</b> <code>%s</code>  <b>XP:</b> <code>%d</code>\n",
		html.EscapeString(dndFormatCR(d.ChallengeRating)), d.XP))

	b.WriteString("\n<b>Ability Scores:</b>\n")
	b.WriteString(fmt.Sprintf("<code>STR %2d (%s)  DEX %2d (%s)  CON %2d (%s)</code>\n",
		d.Strength, dndAbilityMod(d.Strength),
		d.Dexterity, dndAbilityMod(d.Dexterity),
		d.Constitution, dndAbilityMod(d.Constitution),
	))
	b.WriteString(fmt.Sprintf("<code>INT %2d (%s)  WIS %2d (%s)  CHA %2d (%s)</code>\n",
		d.Intelligence, dndAbilityMod(d.Intelligence),
		d.Wisdom, dndAbilityMod(d.Wisdom),
		d.Charisma, dndAbilityMod(d.Charisma),
	))

	b.WriteString(fmt.Sprintf("\n<b>Senses:</b> <code>%s</code>\n", html.EscapeString(dndFormatSenses(d.Senses))))
	if d.Languages != "" {
		b.WriteString(fmt.Sprintf("<b>Languages:</b> <code>%s</code>\n", html.EscapeString(d.Languages)))
	}

	if len(d.DamageVuln) > 0 {
		b.WriteString(fmt.Sprintf("<b>Vulnerabilities:</b> <code>%s</code>\n", html.EscapeString(dndFormatStringList(d.DamageVuln))))
	}
	if len(d.DamageRes) > 0 {
		b.WriteString(fmt.Sprintf("<b>Resistances:</b> <code>%s</code>\n", html.EscapeString(dndFormatStringList(d.DamageRes))))
	}
	if len(d.DamageImm) > 0 {
		b.WriteString(fmt.Sprintf("<b>Damage Immunities:</b> <code>%s</code>\n", html.EscapeString(dndFormatStringList(d.DamageImm))))
	}
	if len(d.ConditionImm) > 0 {
		b.WriteString(fmt.Sprintf("<b>Condition Immunities:</b> <code>%s</code>\n", html.EscapeString(dndFormatAnyList(d.ConditionImm))))
	}

	if len(d.SpecialAbilities) > 0 {
		b.WriteString("\n<b>Special Abilities:</b>\n")
		for i, sa := range d.SpecialAbilities {
			if i >= 4 {
				b.WriteString(fmt.Sprintf("<i>... and %d more</i>\n", len(d.SpecialAbilities)-i))
				break
			}
			b.WriteString(fmt.Sprintf("• <b>%s:</b> %s\n",
				html.EscapeString(sa.Name),
				html.EscapeString(dndTruncate(sa.Desc, 220)),
			))
		}
	}

	if len(d.Actions) > 0 {
		b.WriteString("\n<b>Actions:</b>\n")
		for i, a := range d.Actions {
			if i >= 5 {
				b.WriteString(fmt.Sprintf("<i>... and %d more</i>\n", len(d.Actions)-i))
				break
			}
			b.WriteString(fmt.Sprintf("• <b>%s:</b> %s\n",
				html.EscapeString(a.Name),
				html.EscapeString(dndTruncate(a.Desc, 220)),
			))
		}
	}

	if len(d.LegendaryActions) > 0 {
		b.WriteString("\n<b>Legendary Actions:</b>\n")
		for i, la := range d.LegendaryActions {
			if i >= 3 {
				b.WriteString(fmt.Sprintf("<i>... and %d more</i>\n", len(d.LegendaryActions)-i))
				break
			}
			b.WriteString(fmt.Sprintf("• <b>%s:</b> %s\n",
				html.EscapeString(la.Name),
				html.EscapeString(dndTruncate(la.Desc, 180)),
			))
		}
	}

	b.WriteString("\n<i>Source: dnd5eapi.co</i>")
	return b.String()
}

func Dnd5eHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/dnd5e &lt;monster&gt;</code>\n<b>Example:</b> <code>/dnd5e goblin</code>")
		return nil
	}

	status, _ := m.Reply("<i>Summoning monster from the Monster Manual...</i>")

	slug := dndSlugify(arg)
	if slug == "" {
		if status != nil {
			status.Edit("<b>Invalid monster name.</b>")
		}
		return nil
	}

	d, code, err := dndFetchMonster(slug)
	if err != nil {
		if status != nil {
			status.Edit("<b>Request failed.</b> Could not reach dnd5eapi.co.")
		}
		return nil
	}

	if code == 404 {
		sr, serr := dndSearchMonster(arg)
		if serr == nil && sr != nil && sr.Count > 0 {
			if sr.Count == 1 {
				d, code, err = dndFetchMonster(sr.Results[0].Index)
				if err != nil || code != 200 || d == nil {
					if status != nil {
						status.Edit(fmt.Sprintf("<b>Monster not found:</b> <code>%s</code>", html.EscapeString(arg)))
					}
					return nil
				}
			} else {
				var b strings.Builder
				b.WriteString(fmt.Sprintf("<b>Monster not found:</b> <code>%s</code>\n\n<b>Did you mean?</b>\n", html.EscapeString(arg)))
				limit := 10
				if sr.Count < limit {
					limit = sr.Count
				}
				for i := 0; i < limit; i++ {
					b.WriteString(fmt.Sprintf("• <code>%s</code>\n", html.EscapeString(sr.Results[i].Name)))
				}
				if sr.Count > limit {
					b.WriteString(fmt.Sprintf("<i>... and %d more</i>", sr.Count-limit))
				}
				if status != nil {
					status.Edit(b.String())
				}
				return nil
			}
		} else {
			if status != nil {
				status.Edit(fmt.Sprintf("<b>Monster not found:</b> <code>%s</code>", html.EscapeString(arg)))
			}
			return nil
		}
	}

	if code != 200 || d == nil {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", code))
		}
		return nil
	}

	caption := dndBuildCaption(d)

	if d.Image != "" {
		imgBytes, ierr := dndDownloadImage(d.Image)
		if ierr == nil && len(imgBytes) > 0 {
			if status != nil {
				status.Delete()
			}
			if _, merr := m.ReplyMedia(imgBytes, &tg.MediaOptions{
				Caption:  caption,
				FileName: fmt.Sprintf("%s.png", d.Index),
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

func registerDnd5eHandlers() {
	c := Client
	c.On("cmd:dnd5e", Dnd5eHandler)
}

func init() {
	QueueHandlerRegistration(registerDnd5eHandlers)
}
