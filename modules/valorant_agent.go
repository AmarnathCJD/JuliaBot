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

type valAgentAbility struct {
	Slot        string `json:"slot"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

type valAgentRole struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

type valAgentVoiceLine struct {
	MediaList []struct {
		Wave string `json:"wave"`
		Wwav string `json:"wwise"`
	} `json:"mediaList"`
}

type valAgent struct {
	UUID                string             `json:"uuid"`
	DisplayName         string             `json:"displayName"`
	Description         string             `json:"description"`
	DeveloperName       string             `json:"developerName"`
	FullPortrait        string             `json:"fullPortrait"`
	BustPortrait        string             `json:"bustPortrait"`
	DisplayIcon         string             `json:"displayIcon"`
	Background          string             `json:"background"`
	IsPlayableCharacter bool               `json:"isPlayableCharacter"`
	Role                *valAgentRole      `json:"role"`
	Abilities           []valAgentAbility  `json:"abilities"`
	VoiceLine           *valAgentVoiceLine `json:"voiceLine"`
}

type valAgentsResp struct {
	Status int        `json:"status"`
	Data   []valAgent `json:"data"`
}

type valAgentCacheEntry struct {
	agents  []valAgent
	expires time.Time
}

var (
	valAgentCacheMu sync.Mutex
	valAgentCache   valAgentCacheEntry
)

const valAgentCacheTTL = 6 * time.Hour

func valAgentFetchAll() ([]valAgent, int, error) {
	valAgentCacheMu.Lock()
	if valAgentCache.agents != nil && time.Now().Before(valAgentCache.expires) {
		out := valAgentCache.agents
		valAgentCacheMu.Unlock()
		return out, 200, nil
	}
	valAgentCacheMu.Unlock()

	endpoint := "https://valorant-api.com/v1/agents?isPlayableCharacter=true"
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
	var r valAgentsResp
	if jerr := json.Unmarshal(body, &r); jerr != nil {
		return nil, resp.StatusCode, jerr
	}
	valAgentCacheMu.Lock()
	valAgentCache = valAgentCacheEntry{agents: r.Data, expires: time.Now().Add(valAgentCacheTTL)}
	valAgentCacheMu.Unlock()
	return r.Data, resp.StatusCode, nil
}

func valAgentFind(agents []valAgent, query string) *valAgent {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	for i := range agents {
		if strings.ToLower(agents[i].DisplayName) == q {
			return &agents[i]
		}
	}
	for i := range agents {
		if strings.Contains(strings.ToLower(agents[i].DisplayName), q) {
			return &agents[i]
		}
	}
	for i := range agents {
		if strings.Contains(strings.ToLower(agents[i].DeveloperName), q) {
			return &agents[i]
		}
	}
	return nil
}

func valAgentDownloadImage(url string) ([]byte, error) {
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
		return nil, fmt.Errorf("img http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func valAgentAbilitySlotLabel(slot string) string {
	switch strings.ToLower(slot) {
	case "ability1":
		return "C"
	case "ability2":
		return "Q"
	case "grenade":
		return "E"
	case "ultimate":
		return "X"
	case "passive":
		return "Passive"
	}
	return slot
}

func valAgentExtractVoiceLine(a *valAgent) string {
	if a == nil || a.VoiceLine == nil {
		return ""
	}
	for _, m := range a.VoiceLine.MediaList {
		if m.Wave != "" {
			return m.Wave
		}
	}
	return ""
}

func valAgentTrim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func ValorantAgentHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/valagent &lt;name&gt;</code>\n<b>Example:</b> <code>/valagent jett</code>")
		return nil
	}

	status, _ := m.Reply("<i>Fetching Valorant agent...</i>")

	agents, code, err := valAgentFetchAll()
	if err != nil {
		if status != nil {
			status.Edit("<b>Request failed.</b> Could not reach valorant-api.com.")
		}
		return nil
	}
	if code != 200 || len(agents) == 0 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", code))
		}
		return nil
	}

	a := valAgentFind(agents, arg)
	if a == nil {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>Agent not found:</b> <code>%s</code>", html.EscapeString(arg)))
		}
		return nil
	}

	roleName := "—"
	roleDesc := ""
	if a.Role != nil {
		if a.Role.DisplayName != "" {
			roleName = a.Role.DisplayName
		}
		roleDesc = a.Role.Description
	}

	voiceLine := valAgentExtractVoiceLine(a)
	voiceLineStr := "—"
	if voiceLine != "" {
		voiceLineStr = fmt.Sprintf("<a href=\"%s\">Listen</a>", html.EscapeString(voiceLine))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\U0001F3AF <b>%s</b>\n", html.EscapeString(a.DisplayName)))
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("<b>Role:</b> <code>%s</code>\n", html.EscapeString(roleName)))
	if a.DeveloperName != "" {
		b.WriteString(fmt.Sprintf("<b>Codename:</b> <code>%s</code>\n", html.EscapeString(a.DeveloperName)))
	}
	b.WriteString(fmt.Sprintf("<b>Voice Line:</b> %s\n", voiceLineStr))
	if roleDesc != "" {
		b.WriteString(fmt.Sprintf("\n<i>%s</i>\n", html.EscapeString(valAgentTrim(roleDesc, 220))))
	}
	if a.Description != "" {
		b.WriteString(fmt.Sprintf("\n<b>About:</b>\n%s\n", html.EscapeString(valAgentTrim(a.Description, 400))))
	}
	if len(a.Abilities) > 0 {
		b.WriteString("\n<b>Abilities:</b>\n")
		for _, ab := range a.Abilities {
			if ab.DisplayName == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("• <b>[%s] %s</b>\n", html.EscapeString(valAgentAbilitySlotLabel(ab.Slot)), html.EscapeString(ab.DisplayName)))
			if ab.Description != "" {
				b.WriteString(fmt.Sprintf("   <i>%s</i>\n", html.EscapeString(valAgentTrim(ab.Description, 180))))
			}
		}
	}
	b.WriteString("\n<i>Source: valorant-api.com</i>")

	caption := b.String()

	portrait := a.FullPortrait
	if portrait == "" {
		portrait = a.BustPortrait
	}
	if portrait == "" {
		portrait = a.DisplayIcon
	}

	if portrait != "" {
		imgBytes, ierr := valAgentDownloadImage(portrait)
		if ierr == nil && len(imgBytes) > 0 {
			if status != nil {
				status.Delete()
			}
			if _, merr := m.ReplyMedia(imgBytes, &tg.MediaOptions{
				Caption:  caption,
				FileName: fmt.Sprintf("%s.png", strings.ToLower(strings.ReplaceAll(a.DisplayName, "/", "_"))),
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

func registerValorantAgentHandlers() {
	c := Client
	c.On("cmd:valagent", ValorantAgentHandler)
}

func init() {
	QueueHandlerRegistration(registerValorantAgentHandlers)
}
