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

type pokemonAPIResp struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Height int    `json:"height"`
	Weight int    `json:"weight"`
	Types  []struct {
		Slot int `json:"slot"`
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
	} `json:"types"`
	Stats []struct {
		BaseStat int `json:"base_stat"`
		Stat     struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	Sprites struct {
		FrontDefault string `json:"front_default"`
	} `json:"sprites"`
}

type pokemonCacheEntry struct {
	data    *pokemonAPIResp
	expires time.Time
}

var (
	pokemonCacheMu sync.Mutex
	pokemonCache   = map[string]pokemonCacheEntry{}
)

const pokemonCacheTTL = 1 * time.Hour

func pokemonFetch(query string) (*pokemonAPIResp, int, error) {
	key := strings.ToLower(strings.TrimSpace(query))
	pokemonCacheMu.Lock()
	if e, ok := pokemonCache[key]; ok && time.Now().Before(e.expires) {
		pokemonCacheMu.Unlock()
		return e.data, 200, nil
	}
	pokemonCacheMu.Unlock()

	endpoint := "https://pokeapi.co/api/v2/pokemon/" + key
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
	var p pokemonAPIResp
	if jerr := json.Unmarshal(body, &p); jerr != nil {
		return nil, resp.StatusCode, jerr
	}
	pokemonCacheMu.Lock()
	pokemonCache[key] = pokemonCacheEntry{data: &p, expires: time.Now().Add(pokemonCacheTTL)}
	pokemonCacheMu.Unlock()
	return &p, resp.StatusCode, nil
}

func pokemonDownloadSprite(spriteURL string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", spriteURL, nil)
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
		return nil, fmt.Errorf("sprite http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func pokemonTitleCase(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func pokemonStatLabel(name string) string {
	switch name {
	case "hp":
		return "HP"
	case "attack":
		return "Attack"
	case "defense":
		return "Defense"
	case "special-attack":
		return "Sp. Atk"
	case "special-defense":
		return "Sp. Def"
	case "speed":
		return "Speed"
	}
	return pokemonTitleCase(name)
}

func PokemonHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/pokemon &lt;name_or_id&gt;</code>\n<b>Example:</b> <code>/pokemon pikachu</code>")
		return nil
	}

	if strings.ContainsAny(arg, " \t\n/?#") {
		arg = strings.Fields(arg)[0]
	}

	status, _ := m.Reply("<i>Fetching Pokemon data...</i>")

	p, code, err := pokemonFetch(arg)
	if err != nil {
		if status != nil {
			status.Edit("<b>Request failed.</b> Could not reach pokeapi.co.")
		}
		return nil
	}
	if code == 404 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>Pokemon not found:</b> <code>%s</code>", html.EscapeString(arg)))
		}
		return nil
	}
	if code != 200 || p == nil {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", code))
		}
		return nil
	}

	var typeNames []string
	for _, t := range p.Types {
		typeNames = append(typeNames, pokemonTitleCase(t.Type.Name))
	}
	typesStr := strings.Join(typeNames, ", ")
	if typesStr == "" {
		typesStr = "—"
	}

	heightM := float64(p.Height) / 10.0
	weightKg := float64(p.Weight) / 10.0

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\U0001F4D8 <b>%s</b>  <code>#%d</code>\n", html.EscapeString(pokemonTitleCase(p.Name)), p.ID))
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("<b>Types:</b> <code>%s</code>\n", html.EscapeString(typesStr)))
	b.WriteString(fmt.Sprintf("<b>Height:</b> <code>%.1f m</code>\n", heightM))
	b.WriteString(fmt.Sprintf("<b>Weight:</b> <code>%.1f kg</code>\n", weightKg))
	b.WriteString("\n<b>Base Stats:</b>\n")
	for _, s := range p.Stats {
		b.WriteString(fmt.Sprintf("• <b>%s:</b> <code>%d</code>\n", html.EscapeString(pokemonStatLabel(s.Stat.Name)), s.BaseStat))
	}
	b.WriteString("\n<i>Source: pokeapi.co</i>")

	caption := b.String()

	if p.Sprites.FrontDefault != "" {
		imgBytes, ierr := pokemonDownloadSprite(p.Sprites.FrontDefault)
		if ierr == nil && len(imgBytes) > 0 {
			if status != nil {
				status.Delete()
			}
			if _, merr := m.ReplyMedia(imgBytes, &tg.MediaOptions{
				Caption:  caption,
				FileName: fmt.Sprintf("%s.png", strings.ToLower(p.Name)),
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

func registerPokemonHandlers() {
	c := Client
	c.On("cmd:pokemon", PokemonHandler)
}

func init() {
	QueueHandlerRegistration(registerPokemonHandlers)
}
