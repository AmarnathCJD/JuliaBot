package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type covidCountryResp struct {
	Country       string  `json:"country"`
	Cases         int64   `json:"cases"`
	TodayCases    int64   `json:"todayCases"`
	Deaths        int64   `json:"deaths"`
	TodayDeaths   int64   `json:"todayDeaths"`
	Recovered     int64   `json:"recovered"`
	TodayRecov    int64   `json:"todayRecovered"`
	Active        int64   `json:"active"`
	Critical      int64   `json:"critical"`
	Tests         int64   `json:"tests"`
	Population    int64   `json:"population"`
	Continent     string  `json:"continent"`
	CasesPerOneM  float64 `json:"casesPerOneMillion"`
	DeathsPerOneM float64 `json:"deathsPerOneMillion"`
	Updated       int64   `json:"updated"`
	CountryInfo   struct {
		Iso2 string `json:"iso2"`
		Flag string `json:"flag"`
	} `json:"countryInfo"`
}

type covidGlobalResp struct {
	Cases             int64   `json:"cases"`
	TodayCases        int64   `json:"todayCases"`
	Deaths            int64   `json:"deaths"`
	TodayDeaths       int64   `json:"todayDeaths"`
	Recovered         int64   `json:"recovered"`
	TodayRecov        int64   `json:"todayRecovered"`
	Active            int64   `json:"active"`
	Critical          int64   `json:"critical"`
	Tests             int64   `json:"tests"`
	Population        int64   `json:"population"`
	AffectedCountries int     `json:"affectedCountries"`
	CasesPerOneM      float64 `json:"casesPerOneMillion"`
	DeathsPerOneM     float64 `json:"deathsPerOneMillion"`
	Updated           int64   `json:"updated"`
}

type covidCacheEntry struct {
	body    []byte
	expires time.Time
}

var (
	covidCacheMu sync.Mutex
	covidCache   = map[string]covidCacheEntry{}
)

const covidCacheTTL = 30 * time.Minute

func covidFetch(rawURL string) ([]byte, int, error) {
	covidCacheMu.Lock()
	if e, ok := covidCache[rawURL]; ok && time.Now().Before(e.expires) {
		covidCacheMu.Unlock()
		return e.body, 200, nil
	}
	covidCacheMu.Unlock()

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode == 200 {
		covidCacheMu.Lock()
		covidCache[rawURL] = covidCacheEntry{body: body, expires: time.Now().Add(covidCacheTTL)}
		covidCacheMu.Unlock()
	}
	return body, resp.StatusCode, nil
}

func covidFormatDelta(n int64) string {
	if n > 0 {
		return fmt.Sprintf("+%s", covidFormatNum(n))
	}
	return covidFormatNum(n)
}

func covidFormatNum(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	var out strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(c)
	}
	if neg {
		return "-" + out.String()
	}
	return out.String()
}

func CovidHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/covid &lt;country&gt;</code>\n<b>Worldwide:</b> <code>/covid global</code>\n<b>Example:</b> <code>/covid India</code>")
		return nil
	}

	status, _ := m.Reply("<i>Fetching COVID-19 stats...</i>")

	if strings.EqualFold(arg, "global") || strings.EqualFold(arg, "world") || strings.EqualFold(arg, "all") || strings.EqualFold(arg, "worldwide") {
		body, code, err := covidFetch("https://disease.sh/v3/covid-19/all")
		if err != nil {
			if status != nil {
				status.Edit("<b>Request failed.</b> Could not reach disease.sh.")
			}
			return nil
		}
		if code != 200 {
			if status != nil {
				status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", code))
			}
			return nil
		}
		var g covidGlobalResp
		if err := json.Unmarshal(body, &g); err != nil {
			if status != nil {
				status.Edit("<b>Failed to parse response.</b>")
			}
			return nil
		}

		var b strings.Builder
		b.WriteString("\U0001F30D <b>COVID-19 Worldwide Stats</b>\n")
		b.WriteString("━━━━━━━━━━━━━━━━\n")
		b.WriteString(fmt.Sprintf("<b>Total Cases:</b> <code>%s</code> (<i>%s today</i>)\n", covidFormatNum(g.Cases), covidFormatDelta(g.TodayCases)))
		b.WriteString(fmt.Sprintf("<b>Deaths:</b> <code>%s</code> (<i>%s today</i>)\n", covidFormatNum(g.Deaths), covidFormatDelta(g.TodayDeaths)))
		b.WriteString(fmt.Sprintf("<b>Recovered:</b> <code>%s</code> (<i>%s today</i>)\n", covidFormatNum(g.Recovered), covidFormatDelta(g.TodayRecov)))
		b.WriteString(fmt.Sprintf("<b>Active:</b> <code>%s</code>\n", covidFormatNum(g.Active)))
		b.WriteString(fmt.Sprintf("<b>Critical:</b> <code>%s</code>\n", covidFormatNum(g.Critical)))
		b.WriteString(fmt.Sprintf("<b>Tests:</b> <code>%s</code>\n", covidFormatNum(g.Tests)))
		b.WriteString(fmt.Sprintf("<b>Affected Countries:</b> <code>%d</code>\n", g.AffectedCountries))
		b.WriteString(fmt.Sprintf("<b>Cases / 1M:</b> <code>%.1f</code>\n", g.CasesPerOneM))
		b.WriteString(fmt.Sprintf("<b>Deaths / 1M:</b> <code>%.1f</code>\n", g.DeathsPerOneM))
		if g.Updated > 0 {
			t := time.Unix(g.Updated/1000, 0).UTC().Format("2006-01-02 15:04 MST")
			b.WriteString(fmt.Sprintf("\n<i>Updated: %s</i>\n", html.EscapeString(t)))
		}
		b.WriteString("<i>Source: disease.sh</i>")

		if status != nil {
			status.Edit(b.String())
		} else {
			m.Reply(b.String())
		}
		return nil
	}

	endpoint := fmt.Sprintf("https://disease.sh/v3/covid-19/countries/%s?strict=false", url.PathEscape(arg))
	body, code, err := covidFetch(endpoint)
	if err != nil {
		if status != nil {
			status.Edit("<b>Request failed.</b> Could not reach disease.sh.")
		}
		return nil
	}
	if code == 404 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>Country not found:</b> <code>%s</code>", html.EscapeString(arg)))
		}
		return nil
	}
	if code != 200 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", code))
		}
		return nil
	}

	var c covidCountryResp
	if err := json.Unmarshal(body, &c); err != nil {
		if status != nil {
			status.Edit("<b>Failed to parse response.</b>")
		}
		return nil
	}

	flag := covidFlagFromISO(c.CountryInfo.Iso2)

	var b strings.Builder
	if flag != "" {
		b.WriteString(fmt.Sprintf("%s <b>COVID-19 — %s</b>\n", flag, html.EscapeString(c.Country)))
	} else {
		b.WriteString(fmt.Sprintf("\U0001F9EA <b>COVID-19 — %s</b>\n", html.EscapeString(c.Country)))
	}
	if c.Continent != "" {
		b.WriteString(fmt.Sprintf("<i>%s</i>\n", html.EscapeString(c.Continent)))
	}
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("<b>Total Cases:</b> <code>%s</code> (<i>%s today</i>)\n", covidFormatNum(c.Cases), covidFormatDelta(c.TodayCases)))
	b.WriteString(fmt.Sprintf("<b>Deaths:</b> <code>%s</code> (<i>%s today</i>)\n", covidFormatNum(c.Deaths), covidFormatDelta(c.TodayDeaths)))
	b.WriteString(fmt.Sprintf("<b>Recovered:</b> <code>%s</code> (<i>%s today</i>)\n", covidFormatNum(c.Recovered), covidFormatDelta(c.TodayRecov)))
	b.WriteString(fmt.Sprintf("<b>Active:</b> <code>%s</code>\n", covidFormatNum(c.Active)))
	b.WriteString(fmt.Sprintf("<b>Critical:</b> <code>%s</code>\n", covidFormatNum(c.Critical)))
	b.WriteString(fmt.Sprintf("<b>Tests:</b> <code>%s</code>\n", covidFormatNum(c.Tests)))
	if c.Population > 0 {
		b.WriteString(fmt.Sprintf("<b>Population:</b> <code>%s</code>\n", covidFormatNum(c.Population)))
	}
	b.WriteString(fmt.Sprintf("<b>Cases / 1M:</b> <code>%.1f</code>\n", c.CasesPerOneM))
	b.WriteString(fmt.Sprintf("<b>Deaths / 1M:</b> <code>%.1f</code>\n", c.DeathsPerOneM))
	if c.Updated > 0 {
		t := time.Unix(c.Updated/1000, 0).UTC().Format("2006-01-02 15:04 MST")
		b.WriteString(fmt.Sprintf("\n<i>Updated: %s</i>\n", html.EscapeString(t)))
	}
	b.WriteString("<i>Source: disease.sh</i>")

	if status != nil {
		status.Edit(b.String())
	} else {
		m.Reply(b.String())
	}
	return nil
}

func covidFlagFromISO(iso string) string {
	iso = strings.ToUpper(strings.TrimSpace(iso))
	if len(iso) != 2 {
		return ""
	}
	r1 := rune(iso[0])
	r2 := rune(iso[1])
	if r1 < 'A' || r1 > 'Z' || r2 < 'A' || r2 > 'Z' {
		return ""
	}
	const base = 0x1F1E6
	return string([]rune{base + (r1 - 'A'), base + (r2 - 'A')})
}

func registerCovidHandlers() {
	c := Client
	c.On("cmd:covid", CovidHandler)
}

func init() {
	QueueHandlerRegistration(registerCovidHandlers)
}
