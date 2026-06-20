package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type issPass struct {
	Start        string  `json:"start"`
	TCA          string  `json:"tca"`
	End          string  `json:"end"`
	AOSAzimuth   float64 `json:"aos_azimuth"`
	LOSAzimuth   float64 `json:"los_azimuth"`
	MaxElevation float64 `json:"max_elevation"`
}

type issPassResponse struct {
	APIStatus        string    `json:"api_status"`
	RequestTimestamp string    `json:"request_timestamp"`
	NoradID          int       `json:"norad_id"`
	SatelliteName    string    `json:"satellite_name"`
	Lat              float64   `json:"lat"`
	Lon              float64   `json:"lon"`
	Hours            int       `json:"hours"`
	MinElevation     float64   `json:"min_elevation"`
	Passes           []issPass `json:"passes"`
}

func issAzimuthToCompass(deg float64) string {
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int((deg+11.25)/22.5) % 16
	if idx < 0 {
		idx += 16
	}
	return dirs[idx]
}

func issFormatDuration(start, end time.Time) string {
	d := end.Sub(start)
	if d < 0 {
		return "0s"
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	if mins > 0 {
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

func IssPassHandler(m *tg.NewMessage) error {
	args := strings.Fields(strings.TrimSpace(m.Args()))
	if len(args) < 2 {
		m.Reply("<b>Usage:</b> <code>/isspass &lt;lat&gt; &lt;lon&gt;</code>\n<b>Example:</b> <code>/isspass 40.7 -74.0</code>")
		return nil
	}

	lat, err1 := strconv.ParseFloat(args[0], 64)
	lon, err2 := strconv.ParseFloat(args[1], 64)
	if err1 != nil || err2 != nil {
		m.Reply("<b>Invalid coordinates.</b> Latitude and longitude must be numbers.")
		return nil
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		m.Reply("<b>Out of range.</b> Lat must be -90..90, lon must be -180..180.")
		return nil
	}

	status, _ := m.Reply("<i>Calculating ISS passes...</i>")

	endpoint := fmt.Sprintf("https://api.g7vrd.co.uk/v1/satellite-passes/25544/%g/%g.json", lat, lon)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		if status != nil {
			status.Edit("<b>Failed to build request.</b>")
		}
		return nil
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if status != nil {
			status.Edit("<b>Request failed.</b> Could not reach g7vrd.co.uk.")
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>API error.</b> Status: <code>%d</code>", resp.StatusCode))
		}
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if status != nil {
			status.Edit("<b>Failed to read response.</b>")
		}
		return nil
	}

	var data issPassResponse
	if err := json.Unmarshal(body, &data); err != nil {
		if status != nil {
			status.Edit("<b>Failed to parse response.</b>")
		}
		return nil
	}

	if len(data.Passes) == 0 {
		if status != nil {
			status.Edit(fmt.Sprintf("<b>\U0001F6F0 No upcoming ISS passes</b> for <code>%g, %g</code> in the next %dh (min %.0f° elevation).", lat, lon, data.Hours, data.MinElevation))
		}
		return nil
	}

	limit := 5
	if len(data.Passes) < limit {
		limit = len(data.Passes)
	}

	var b strings.Builder
	b.WriteString("\U0001F6F0 <b>ISS Pass Predictions</b>\n")
	b.WriteString(fmt.Sprintf("<b>Location:</b> <code>%g, %g</code>\n", lat, lon))
	b.WriteString(fmt.Sprintf("<b>Satellite:</b> %s (NORAD %d)\n", html.EscapeString(data.SatelliteName), data.NoradID))
	b.WriteString(fmt.Sprintf("<b>Min elevation:</b> <code>%.0f°</code>\n", data.MinElevation))
	b.WriteString("━━━━━━━━━━━━━━━━\n")

	for i := 0; i < limit; i++ {
		p := data.Passes[i]
		start, errS := time.Parse(time.RFC3339, p.Start)
		end, errE := time.Parse(time.RFC3339, p.End)

		b.WriteString(fmt.Sprintf("<b>Pass #%d</b>\n", i+1))
		if errS == nil {
			b.WriteString(fmt.Sprintf("  <b>Start:</b> <code>%s</code> UTC\n", html.EscapeString(start.UTC().Format("2006-01-02 15:04:05"))))
		} else {
			b.WriteString(fmt.Sprintf("  <b>Start:</b> <code>%s</code>\n", html.EscapeString(p.Start)))
		}
		if errE == nil {
			b.WriteString(fmt.Sprintf("  <b>End:</b> <code>%s</code> UTC\n", html.EscapeString(end.UTC().Format("2006-01-02 15:04:05"))))
		}
		if errS == nil && errE == nil {
			b.WriteString(fmt.Sprintf("  <b>Duration:</b> <code>%s</code>\n", issFormatDuration(start, end)))
		}
		b.WriteString(fmt.Sprintf("  <b>Max elevation:</b> <code>%.1f°</code>\n", p.MaxElevation))
		b.WriteString(fmt.Sprintf("  <b>AOS:</b> <code>%.0f° %s</code> → <b>LOS:</b> <code>%.0f° %s</code>\n",
			p.AOSAzimuth, issAzimuthToCompass(p.AOSAzimuth),
			p.LOSAzimuth, issAzimuthToCompass(p.LOSAzimuth)))
		if i < limit-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n<i>Source: g7vrd.co.uk</i>")

	if status != nil {
		status.Edit(b.String())
	} else {
		m.Reply(b.String())
	}
	return nil
}

func registerIssPassHandlers() {
	c := Client
	c.On("cmd:isspass", IssPassHandler)
}

func init() {
	QueueHandlerRegistration(registerIssPassHandlers)
}
