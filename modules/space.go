package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type issNowResponse struct {
	Message       string `json:"message"`
	Timestamp     int64  `json:"timestamp"`
	IssPosition struct {
		Latitude  string `json:"latitude"`
		Longitude string `json:"longitude"`
	} `json:"iss_position"`
}

type astrosResponse struct {
	Message string `json:"message"`
	Number  int    `json:"number"`
	People  []struct {
		Name  string `json:"name"`
		Craft string `json:"craft"`
	} `json:"people"`
}

func issRegionGuess(lat, lon float64) string {
	if lat > 66.5 {
		return "Arctic region"
	}
	if lat < -66.5 {
		return "Antarctic region"
	}
	if lat > 23.5 {
		switch {
		case lon >= -10 && lon <= 60:
			return "over Europe / North Africa / Middle East"
		case lon > 60 && lon <= 150:
			return "over Asia"
		case lon > 150 || lon <= -130:
			return "over the North Pacific Ocean"
		case lon > -130 && lon <= -50:
			return "over North America"
		case lon > -50 && lon < -10:
			return "over the North Atlantic Ocean"
		}
	}
	if lat < -23.5 {
		switch {
		case lon >= -75 && lon <= -30:
			return "over South America"
		case lon > -30 && lon <= 50:
			return "over Southern Africa / South Atlantic"
		case lon > 50 && lon <= 160:
			return "over the Indian Ocean / Australia"
		default:
			return "over the South Pacific Ocean"
		}
	}
	switch {
	case lon >= -20 && lon <= 50:
		return "over Africa"
	case lon > 50 && lon <= 100:
		return "over the Indian Ocean / South Asia"
	case lon > 100 && lon <= 160:
		return "over Southeast Asia"
	case lon > 160 || lon <= -90:
		return "over the Pacific Ocean"
	case lon > -90 && lon <= -30:
		return "over South America / Atlantic"
	}
	return "over the equatorial region"
}

func issAsciiMap(lat, lon float64) string {
	const width = 36
	const height = 13
	col := int(math.Round((lon + 180) / 360 * float64(width-1)))
	row := int(math.Round((90 - lat) / 180 * float64(height-1)))
	if col < 0 {
		col = 0
	}
	if col >= width {
		col = width - 1
	}
	if row < 0 {
		row = 0
	}
	if row >= height {
		row = height - 1
	}
	var b strings.Builder
	b.WriteString("+" + strings.Repeat("-", width) + "+\n")
	for r := 0; r < height; r++ {
		b.WriteString("|")
		for c := 0; c < width; c++ {
			if r == row && c == col {
				b.WriteString("X")
			} else if r == height/2 {
				b.WriteString("-")
			} else if c == width/2 {
				b.WriteString("|")
			} else {
				b.WriteString(" ")
			}
		}
		b.WriteString("|\n")
	}
	b.WriteString("+" + strings.Repeat("-", width) + "+")
	return b.String()
}

func SpaceHandler(m *tg.NewMessage) error {
	status, _ := m.Reply("<i>Locating the ISS...</i>")

	httpClient := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", "http://api.open-notify.org/iss-now.json", nil)
	if err != nil {
		if status != nil {
			status.Edit("<b>Failed to build request.</b>")
		}
		return nil
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		if status != nil {
			status.Edit("<b>Could not reach Open Notify API.</b>")
		}
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if status != nil {
			status.Edit("<b>Failed to read response.</b>")
		}
		return nil
	}

	var iss issNowResponse
	if err := json.Unmarshal(body, &iss); err != nil {
		if status != nil {
			status.Edit("<b>Failed to parse ISS response.</b>")
		}
		return nil
	}

	if iss.Message != "success" {
		if status != nil {
			status.Edit("<b>Open Notify returned an error.</b>")
		}
		return nil
	}

	lat, err := strconv.ParseFloat(iss.IssPosition.Latitude, 64)
	if err != nil {
		if status != nil {
			status.Edit("<b>Invalid latitude received.</b>")
		}
		return nil
	}
	lon, err := strconv.ParseFloat(iss.IssPosition.Longitude, 64)
	if err != nil {
		if status != nil {
			status.Edit("<b>Invalid longitude received.</b>")
		}
		return nil
	}

	region := issRegionGuess(lat, lon)
	mapArt := issAsciiMap(lat, lon)
	when := time.Unix(iss.Timestamp, 0).UTC().Format("2006-01-02 15:04:05 MST")
	mapsURL := fmt.Sprintf("https://www.google.com/maps?q=%f,%f", lat, lon)

	var b strings.Builder
	b.WriteString("\U0001F6F0 <b>International Space Station</b>\n")
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("<b>Latitude:</b> <code>%.4f</code>\n", lat))
	b.WriteString(fmt.Sprintf("<b>Longitude:</b> <code>%.4f</code>\n", lon))
	b.WriteString(fmt.Sprintf("<b>Region:</b> %s\n", html.EscapeString(region)))
	b.WriteString(fmt.Sprintf("<b>Timestamp (UTC):</b> <code>%s</code>\n", html.EscapeString(when)))
	b.WriteString(fmt.Sprintf("<b>Map:</b> <a href=\"%s\">Open in Google Maps</a>\n\n", mapsURL))
	b.WriteString("<pre>")
	b.WriteString(html.EscapeString(mapArt))
	b.WriteString("</pre>\n")
	b.WriteString("<i>X marks the approximate ISS sub-satellite point.</i>")

	if status != nil {
		status.Edit(b.String())
	} else {
		m.Reply(b.String())
	}
	return nil
}

func PeopleHandler(m *tg.NewMessage) error {
	status, _ := m.Reply("<i>Counting humans in orbit...</i>")

	httpClient := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", "http://api.open-notify.org/astros.json", nil)
	if err != nil {
		if status != nil {
			status.Edit("<b>Failed to build request.</b>")
		}
		return nil
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		if status != nil {
			status.Edit("<b>Could not reach Open Notify API.</b>")
		}
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if status != nil {
			status.Edit("<b>Failed to read response.</b>")
		}
		return nil
	}

	var astros astrosResponse
	if err := json.Unmarshal(body, &astros); err != nil {
		if status != nil {
			status.Edit("<b>Failed to parse astros response.</b>")
		}
		return nil
	}

	if astros.Message != "success" {
		if status != nil {
			status.Edit("<b>Open Notify returned an error.</b>")
		}
		return nil
	}

	crafts := make(map[string][]string)
	order := []string{}
	for _, p := range astros.People {
		if _, ok := crafts[p.Craft]; !ok {
			order = append(order, p.Craft)
		}
		crafts[p.Craft] = append(crafts[p.Craft], p.Name)
	}

	var b strings.Builder
	b.WriteString("\U0001F468‍\U0001F680 <b>People Currently in Space</b>\n")
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("<b>Total:</b> <code>%d</code>\n\n", astros.Number))

	if astros.Number == 0 || len(astros.People) == 0 {
		b.WriteString("<i>No data on crew members at the moment.</i>")
	} else {
		for _, craft := range order {
			names := crafts[craft]
			b.WriteString(fmt.Sprintf("\U0001F6F8 <b>%s</b> (<code>%d</code>)\n", html.EscapeString(craft), len(names)))
			for _, n := range names {
				b.WriteString(fmt.Sprintf("  • %s\n", html.EscapeString(n)))
			}
			b.WriteString("\n")
		}
	}

	out := strings.TrimRight(b.String(), "\n")
	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func registerSpaceHandlers() {
	c := Client
	c.On("cmd:space", SpaceHandler)
	c.On("cmd:people", PeopleHandler)
}

func init() {
	QueueHandlerRegistration(registerSpaceHandlers)
}
