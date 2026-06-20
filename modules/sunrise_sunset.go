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

func sunriseFetch(rawURL string) ([]byte, int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (JuliaBot)")
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
	return body, resp.StatusCode, nil
}

func sunriseFormatDayLength(seconds float64) string {
	if seconds <= 0 {
		return "—"
	}
	total := int(seconds)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func sunriseFormatTime(iso string, loc *time.Location) string {
	iso = strings.TrimSpace(iso)
	if iso == "" {
		return "—"
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	if t.Year() <= 1970 {
		return "N/A"
	}
	if loc != nil {
		t = t.In(loc)
	}
	return t.Format("15:04:05 MST")
}

func sunriseGeocode(city string) (float64, float64, string, string, string, error) {
	q := url.QueryEscape(city)
	apiURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json", q)
	body, code, err := sunriseFetch(apiURL)
	if err != nil {
		return 0, 0, "", "", "", err
	}
	if code != 200 {
		return 0, 0, "", "", "", fmt.Errorf("geocoding API status %d", code)
	}
	var resp struct {
		Results []struct {
			Name        string  `json:"name"`
			Latitude    float64 `json:"latitude"`
			Longitude   float64 `json:"longitude"`
			Country     string  `json:"country"`
			Admin1      string  `json:"admin1"`
			Timezone    string  `json:"timezone"`
			CountryCode string  `json:"country_code"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, "", "", "", err
	}
	if len(resp.Results) == 0 {
		return 0, 0, "", "", "", fmt.Errorf("no results")
	}
	r := resp.Results[0]
	display := r.Name
	if r.Admin1 != "" && r.Admin1 != r.Name {
		display += ", " + r.Admin1
	}
	if r.Country != "" {
		display += ", " + r.Country
	}
	return r.Latitude, r.Longitude, display, r.Timezone, r.CountryCode, nil
}

func SunriseHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/sunrise &lt;city&gt;</code>\n<b>Example:</b> <code>/sunrise Tokyo</code>")
		return nil
	}

	status, _ := m.Reply(fmt.Sprintf("<i>Looking up %s...</i>", html.EscapeString(arg)))

	lat, lon, display, tzid, _, err := sunriseGeocode(arg)
	if err != nil {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>Could not find location:</b> <code>%s</code>", html.EscapeString(arg)))
		}
		return nil
	}

	loc, errTz := time.LoadLocation(tzid)
	if errTz != nil || loc == nil {
		loc = time.UTC
		tzid = "UTC"
	}

	if status != nil {
		status.Edit(fmt.Sprintf("<i>Fetching sun times for %s...</i>", html.EscapeString(display)))
	}

	apiURL := fmt.Sprintf("https://api.sunrise-sunset.org/json?lat=%f&lng=%f&formatted=0", lat, lon)
	body, code, err := sunriseFetch(apiURL)
	if err != nil {
		if status != nil {
			status.Edit("<b>Request failed.</b> Could not reach sunrise-sunset.org.")
		}
		return nil
	}
	if code != 200 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", code))
		}
		return nil
	}

	var resp struct {
		Results struct {
			Sunrise                   string  `json:"sunrise"`
			Sunset                    string  `json:"sunset"`
			SolarNoon                 string  `json:"solar_noon"`
			DayLength                 float64 `json:"day_length"`
			CivilTwilightBegin        string  `json:"civil_twilight_begin"`
			CivilTwilightEnd          string  `json:"civil_twilight_end"`
			NauticalTwilightBegin     string  `json:"nautical_twilight_begin"`
			NauticalTwilightEnd       string  `json:"nautical_twilight_end"`
			AstronomicalTwilightBegin string  `json:"astronomical_twilight_begin"`
			AstronomicalTwilightEnd   string  `json:"astronomical_twilight_end"`
		} `json:"results"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		if status != nil {
			status.Edit("<b>Failed to parse API response.</b>")
		}
		return nil
	}
	if resp.Status != "OK" {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API returned status:</b> <code>%s</code>", html.EscapeString(resp.Status)))
		}
		return nil
	}

	r := resp.Results

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\U0001F305 <b>Sunrise &amp; Sunset — %s</b>\n", html.EscapeString(display)))
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("\U0001F4CD <b>Coords:</b> <code>%.4f, %.4f</code>\n", lat, lon))
	b.WriteString(fmt.Sprintf("\U0001F551 <b>Timezone:</b> <code>%s</code>\n", html.EscapeString(tzid)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("\U0001F305 <b>Sunrise:</b> <code>%s</code>\n", html.EscapeString(sunriseFormatTime(r.Sunrise, loc))))
	b.WriteString(fmt.Sprintf("\U0001F307 <b>Sunset:</b> <code>%s</code>\n", html.EscapeString(sunriseFormatTime(r.Sunset, loc))))
	b.WriteString(fmt.Sprintf("☀️ <b>Solar Noon:</b> <code>%s</code>\n", html.EscapeString(sunriseFormatTime(r.SolarNoon, loc))))
	b.WriteString(fmt.Sprintf("⏳ <b>Day Length:</b> <code>%s</code>\n", html.EscapeString(sunriseFormatDayLength(r.DayLength))))
	b.WriteString("\n")
	b.WriteString("\U0001F306 <b>Civil Twilight</b>\n")
	b.WriteString(fmt.Sprintf("  • Begin: <code>%s</code>\n", html.EscapeString(sunriseFormatTime(r.CivilTwilightBegin, loc))))
	b.WriteString(fmt.Sprintf("  • End:   <code>%s</code>\n", html.EscapeString(sunriseFormatTime(r.CivilTwilightEnd, loc))))
	b.WriteString("\n\U0001F30A <b>Nautical Twilight</b>\n")
	b.WriteString(fmt.Sprintf("  • Begin: <code>%s</code>\n", html.EscapeString(sunriseFormatTime(r.NauticalTwilightBegin, loc))))
	b.WriteString(fmt.Sprintf("  • End:   <code>%s</code>\n", html.EscapeString(sunriseFormatTime(r.NauticalTwilightEnd, loc))))
	b.WriteString("\n\U0001F52D <b>Astronomical Twilight</b>\n")
	b.WriteString(fmt.Sprintf("  • Begin: <code>%s</code>\n", html.EscapeString(sunriseFormatTime(r.AstronomicalTwilightBegin, loc))))
	b.WriteString(fmt.Sprintf("  • End:   <code>%s</code>\n", html.EscapeString(sunriseFormatTime(r.AstronomicalTwilightEnd, loc))))
	b.WriteString("\n<i>Source: sunrise-sunset.org &amp; open-meteo</i>")

	if status != nil {
		status.Edit(b.String())
	} else {
		m.Reply(b.String())
	}
	return nil
}

func registerSunriseHandlers() {
	c := Client
	c.On("cmd:sunrise", SunriseHandler)
}

func init() {
	QueueHandlerRegistration(registerSunriseHandlers)
}
