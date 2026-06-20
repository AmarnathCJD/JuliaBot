package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func metarFetch(rawURL string) ([]byte, int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (JuliaBot)")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
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

func metarToFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
			return f, true
		}
		return 0, false
	}
	return 0, false
}

func metarToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func metarDirToCompass(deg float64) string {
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int((deg+11.25)/22.5) % 16
	if idx < 0 {
		idx += 16
	}
	return dirs[idx]
}

func metarFormatSky(skyVal any) string {
	if skyVal == nil {
		return ""
	}
	arr, ok := skyVal.([]any)
	if !ok || len(arr) == 0 {
		return ""
	}
	covers := map[string]string{
		"CLR": "Clear", "SKC": "Sky Clear", "NSC": "No Sig. Clouds", "NCD": "No Cloud Detected",
		"FEW": "Few", "SCT": "Scattered", "BKN": "Broken", "OVC": "Overcast", "VV": "Vert. Vis.",
	}
	parts := []string{}
	for _, it := range arr {
		obj, ok := it.(map[string]any)
		if !ok {
			continue
		}
		cover := strings.ToUpper(metarToString(obj["cover"]))
		base := metarToString(obj["base"])
		name := covers[cover]
		if name == "" {
			name = cover
		}
		if base != "" && base != "0" {
			parts = append(parts, fmt.Sprintf("%s @ %sft", name, base))
		} else {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, ", ")
}

func metarFormatWxString(s string) string {
	if s == "" {
		return ""
	}
	codes := map[string]string{
		"RA": "Rain", "SN": "Snow", "DZ": "Drizzle", "SG": "Snow Grains", "IC": "Ice Crystals",
		"PL": "Ice Pellets", "GR": "Hail", "GS": "Small Hail", "UP": "Unknown Precip",
		"BR": "Mist", "FG": "Fog", "FU": "Smoke", "VA": "Volcanic Ash", "DU": "Dust",
		"SA": "Sand", "HZ": "Haze", "PY": "Spray", "PO": "Dust Whirls", "SQ": "Squalls",
		"FC": "Funnel Cloud", "SS": "Sandstorm", "DS": "Duststorm",
		"TS": "Thunderstorm", "SH": "Shower", "FZ": "Freezing", "BL": "Blowing",
		"DR": "Drifting", "MI": "Shallow", "BC": "Patches", "PR": "Partial",
	}
	tokens := strings.Fields(s)
	out := []string{}
	for _, tok := range tokens {
		raw := tok
		intensity := ""
		if strings.HasPrefix(tok, "-") {
			intensity = "Light "
			tok = tok[1:]
		} else if strings.HasPrefix(tok, "+") {
			intensity = "Heavy "
			tok = tok[1:]
		} else if strings.HasPrefix(tok, "VC") {
			intensity = "Vicinity "
			tok = tok[2:]
		}
		parts := []string{}
		for i := 0; i+2 <= len(tok); i += 2 {
			seg := tok[i : i+2]
			if name, ok := codes[seg]; ok {
				parts = append(parts, name)
			} else {
				parts = append(parts, seg)
			}
		}
		if len(parts) > 0 {
			out = append(out, intensity+strings.Join(parts, " "))
		} else {
			out = append(out, raw)
		}
	}
	return strings.Join(out, ", ")
}

func metarParseAndReply(m *tg.NewMessage, status *tg.NewMessage, icao string, obj map[string]any) {
	station := metarToString(obj["icaoId"])
	if station == "" {
		station = strings.ToUpper(icao)
	}
	name := metarToString(obj["name"])
	rawOb := metarToString(obj["rawOb"])
	reportTime := metarToString(obj["reportTime"])
	if reportTime == "" {
		reportTime = metarToString(obj["obsTime"])
	}

	wdirStr := metarToString(obj["wdir"])
	wspd, hasWspd := metarToFloat(obj["wspd"])
	wgst, hasWgst := metarToFloat(obj["wgst"])
	vis, hasVis := metarToFloat(obj["visib"])
	visStr := metarToString(obj["visib"])
	temp, hasTemp := metarToFloat(obj["temp"])
	dewp, hasDewp := metarToFloat(obj["dewp"])
	altim, hasAltim := metarToFloat(obj["altim"])
	slp, hasSlp := metarToFloat(obj["slp"])
	wx := metarToString(obj["wxString"])
	sky := metarFormatSky(obj["clouds"])
	flightCat := metarToString(obj["fltCat"])
	lat, hasLat := metarToFloat(obj["lat"])
	lon, hasLon := metarToFloat(obj["lon"])
	elev, hasElev := metarToFloat(obj["elev"])

	var b strings.Builder
	b.WriteString(fmt.Sprintf("✈️ <b>METAR — %s</b>\n", html.EscapeString(station)))
	if name != "" {
		b.WriteString(fmt.Sprintf("<i>%s</i>\n", html.EscapeString(name)))
	}
	b.WriteString("━━━━━━━━━━━━━━━━\n")

	if reportTime != "" {
		b.WriteString(fmt.Sprintf("\U0001F552 <b>Time:</b> <code>%s</code>\n", html.EscapeString(reportTime)))
	}
	if flightCat != "" {
		b.WriteString(fmt.Sprintf("\U0001F6A6 <b>Flight Cat:</b> <code>%s</code>\n", html.EscapeString(flightCat)))
	}

	windParts := []string{}
	if strings.EqualFold(strings.TrimSpace(wdirStr), "VRB") {
		windParts = append(windParts, "Variable")
	} else if wd, ok := metarToFloat(obj["wdir"]); ok {
		windParts = append(windParts, fmt.Sprintf("%.0f° (%s)", wd, metarDirToCompass(wd)))
	}
	if hasWspd {
		windParts = append(windParts, fmt.Sprintf("%.0f kt", wspd))
	}
	if hasWgst && wgst > 0 {
		windParts = append(windParts, fmt.Sprintf("gust %.0f kt", wgst))
	}
	if len(windParts) > 0 {
		b.WriteString(fmt.Sprintf("\U0001F32C️ <b>Wind:</b> <code>%s</code>\n", html.EscapeString(strings.Join(windParts, " @ "))))
	}

	if hasVis {
		b.WriteString(fmt.Sprintf("\U0001F441️ <b>Visibility:</b> <code>%s SM</code>\n", html.EscapeString(strings.TrimRight(fmt.Sprintf("%.2f", vis), "0."))))
	} else if visStr != "" {
		b.WriteString(fmt.Sprintf("\U0001F441️ <b>Visibility:</b> <code>%s</code>\n", html.EscapeString(visStr)))
	}

	if wx != "" {
		b.WriteString(fmt.Sprintf("\U0001F326️ <b>Weather:</b> <code>%s</code>\n", html.EscapeString(metarFormatWxString(wx))))
	}
	if sky != "" {
		b.WriteString(fmt.Sprintf("☁️ <b>Sky:</b> <code>%s</code>\n", html.EscapeString(sky)))
	}

	if hasTemp {
		tempF := temp*9/5 + 32
		b.WriteString(fmt.Sprintf("\U0001F321️ <b>Temp:</b> <code>%.1f°C / %.1f°F</code>\n", temp, tempF))
	}
	if hasDewp {
		dewF := dewp*9/5 + 32
		b.WriteString(fmt.Sprintf("\U0001F4A7 <b>Dew Point:</b> <code>%.1f°C / %.1f°F</code>\n", dewp, dewF))
	}
	if hasAltim {
		inHg := altim * 0.02953
		b.WriteString(fmt.Sprintf("\U0001F4CF <b>Altimeter:</b> <code>%.2f inHg / %.0f hPa</code>\n", inHg, altim))
	}
	if hasSlp && slp > 0 {
		b.WriteString(fmt.Sprintf("\U0001F30A <b>Sea Level Pres:</b> <code>%.1f hPa</code>\n", slp))
	}
	if hasLat && hasLon {
		b.WriteString(fmt.Sprintf("\U0001F4CD <b>Location:</b> <code>%.3f, %.3f</code>", lat, lon))
		if hasElev {
			b.WriteString(fmt.Sprintf(" <i>elev %.0fm</i>", elev))
		}
		b.WriteString("\n")
	}

	if rawOb != "" {
		b.WriteString(fmt.Sprintf("\n\U0001F4DD <b>Raw:</b>\n<code>%s</code>\n", html.EscapeString(rawOb)))
	}
	b.WriteString("\n<i>Source: aviationweather.gov</i>")

	if status != nil {
		status.Edit(b.String())
	} else {
		m.Reply(b.String())
	}
}

func metarReplyRaw(m *tg.NewMessage, status *tg.NewMessage, icao, raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>No METAR data found for</b> <code>%s</code>", html.EscapeString(icao)))
		}
		return
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("✈️ <b>METAR — %s</b>\n", html.EscapeString(strings.ToUpper(icao))))
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("<code>%s</code>\n", html.EscapeString(raw)))
	b.WriteString("\n<i>Source: aviationweather.gov</i>")
	if status != nil {
		status.Edit(b.String())
	} else {
		m.Reply(b.String())
	}
}

func MetarHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/metar &lt;ICAO&gt;</code>\n<b>Example:</b> <code>/metar KJFK</code>")
		return nil
	}
	icao := strings.ToUpper(strings.Fields(arg)[0])
	if len(icao) < 3 || len(icao) > 4 {
		m.Reply("<b>Invalid ICAO code.</b> Use a 3 or 4 letter station ID like <code>KJFK</code>.")
		return nil
	}

	status, _ := m.Reply(fmt.Sprintf("<i>Fetching METAR for %s...</i>", html.EscapeString(icao)))

	jsonURL := fmt.Sprintf("https://aviationweather.gov/api/data/metar?ids=%s&format=json", icao)
	body, code, err := metarFetch(jsonURL)
	if err != nil {
		if status != nil {
			status.Edit("<b>Request failed.</b> Could not reach aviationweather.gov.")
		}
		return nil
	}
	if code != 200 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", code))
		}
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	var entries []map[string]any
	parsed := false

	if strings.HasPrefix(trimmed, "[") {
		var arr []map[string]any
		if err := json.Unmarshal(body, &arr); err == nil {
			entries = arr
			parsed = true
		}
	}
	if !parsed && strings.HasPrefix(trimmed, "{") {
		var wrap map[string]any
		if err := json.Unmarshal(body, &wrap); err == nil {
			if d, ok := wrap["data"].([]any); ok {
				for _, it := range d {
					if mp, ok := it.(map[string]any); ok {
						entries = append(entries, mp)
					}
				}
				parsed = true
			} else {
				entries = append(entries, wrap)
				parsed = true
			}
		}
	}

	if parsed && len(entries) > 0 {
		metarParseAndReply(m, status, icao, entries[0])
		return nil
	}

	rawURL := fmt.Sprintf("https://aviationweather.gov/api/data/metar?ids=%s&format=raw", icao)
	rawBody, rawCode, rawErr := metarFetch(rawURL)
	if rawErr != nil || rawCode != 200 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>No METAR data found for</b> <code>%s</code>", html.EscapeString(icao)))
		}
		return nil
	}
	metarReplyRaw(m, status, icao, string(rawBody))
	return nil
}

func registerMetarHandlers() {
	c := Client
	c.On("cmd:metar", MetarHandler)
}

func init() {
	QueueHandlerRegistration(registerMetarHandlers)
}
