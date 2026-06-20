package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type tidesPrediction struct {
	T    string `json:"t"`
	V    string `json:"v"`
	Type string `json:"type"`
}

type tidesPredictionsResponse struct {
	Predictions []tidesPrediction `json:"predictions"`
	Error       struct {
		Message string `json:"message"`
	} `json:"error"`
}

type tidesStationMetadata struct {
	Stations []struct {
		ID    string  `json:"id"`
		Name  string  `json:"name"`
		State string  `json:"state"`
		Lat   float64 `json:"lat"`
		Lng   float64 `json:"lng"`
	} `json:"stations"`
}

func fetchTidesStationName(stationID string) string {
	endpoint := fmt.Sprintf("https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations/%s.json", stationID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var data tidesStationMetadata
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	if len(data.Stations) == 0 {
		return ""
	}
	name := data.Stations[0].Name
	if data.Stations[0].State != "" {
		name += ", " + data.Stations[0].State
	}
	return name
}

func fetchTidesPredictions(stationID string) (*tidesPredictionsResponse, error) {
	now := time.Now().UTC()
	begin := now.Format("20060102")
	end := now.Add(48 * time.Hour).Format("20060102")
	endpoint := fmt.Sprintf("https://api.tidesandcurrents.noaa.gov/api/prod/datagetter?product=predictions&application=JuliaBot&begin_date=%s&end_date=%s&datum=MLLW&station=%s&time_zone=lst_ldt&units=english&interval=hilo&format=json", begin, end, stationID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tides api returned status %d", resp.StatusCode)
	}

	var data tidesPredictionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error.Message != "" {
		return nil, fmt.Errorf("%s", strings.TrimSpace(data.Error.Message))
	}
	if len(data.Predictions) == 0 {
		return nil, fmt.Errorf("no tide predictions returned")
	}
	return &data, nil
}

func tideTypeLabel(t string) string {
	switch strings.ToUpper(t) {
	case "H":
		return "High Tide"
	case "L":
		return "Low Tide"
	}
	return "Tide"
}

func tideTypeEmoji(t string) string {
	switch strings.ToUpper(t) {
	case "H":
		return "\U0001f30a"
	case "L":
		return "\U0001f3d6️"
	}
	return "\U0001f4a7"
}

func TidesHandler(m *tg.NewMessage) error {
	stationID := strings.TrimSpace(m.Args())
	if stationID == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/tides &lt;station_id&gt;</code>\n<b>Example:</b> <code>/tides 8454000</code>\n\nFind station IDs at https://tidesandcurrents.noaa.gov/")
		return err
	}

	status, _ := m.Reply("Fetching tide predictions for station <code>" + html.EscapeString(stationID) + "</code>...")

	data, err := fetchTidesPredictions(stationID)
	if err != nil {
		msg := "Failed to fetch tides: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	stationName := fetchTidesStationName(stationID)

	now := time.Now()
	var upcoming []tidesPrediction
	for _, p := range data.Predictions {
		t, err := time.ParseInLocation("2006-01-02 15:04", p.T, time.Local)
		if err != nil {
			continue
		}
		if t.After(now) {
			upcoming = append(upcoming, p)
		}
		if len(upcoming) >= 4 {
			break
		}
	}
	if len(upcoming) == 0 {
		upcoming = data.Predictions
		if len(upcoming) > 4 {
			upcoming = upcoming[:4]
		}
	}

	var nextHigh, nextLow *tidesPrediction
	for i := range upcoming {
		p := &upcoming[i]
		if nextHigh == nil && strings.EqualFold(p.Type, "H") {
			nextHigh = p
		}
		if nextLow == nil && strings.EqualFold(p.Type, "L") {
			nextLow = p
		}
		if nextHigh != nil && nextLow != nil {
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("\U0001f30a <b>Tide Predictions</b>\n\n")
	sb.WriteString("<b>Station:</b> <code>" + html.EscapeString(stationID) + "</code>\n")
	if stationName != "" {
		sb.WriteString("<b>Name:</b> " + html.EscapeString(stationName) + "\n")
	}
	sb.WriteString("\n")

	if nextHigh != nil {
		sb.WriteString(tideTypeEmoji("H") + " <b>Next High Tide:</b>\n")
		sb.WriteString("  <b>Time:</b> <code>" + html.EscapeString(nextHigh.T) + "</code>\n")
		sb.WriteString("  <b>Height:</b> <code>" + html.EscapeString(nextHigh.V) + " ft</code>\n\n")
	}
	if nextLow != nil {
		sb.WriteString(tideTypeEmoji("L") + " <b>Next Low Tide:</b>\n")
		sb.WriteString("  <b>Time:</b> <code>" + html.EscapeString(nextLow.T) + "</code>\n")
		sb.WriteString("  <b>Height:</b> <code>" + html.EscapeString(nextLow.V) + " ft</code>\n\n")
	}

	sb.WriteString("<b>Upcoming Tides:</b>\n")
	for _, p := range upcoming {
		sb.WriteString(tideTypeEmoji(p.Type) + " <code>" + html.EscapeString(p.T) + "</code> | ")
		sb.WriteString(html.EscapeString(tideTypeLabel(p.Type)) + " | ")
		sb.WriteString("<code>" + html.EscapeString(p.V) + " ft</code>\n")
	}
	sb.WriteString("\n<i>Source: NOAA Tides &amp; Currents | Datum: MLLW | TZ: LST/LDT</i>")

	out := sb.String()
	if status != nil {
		status.Edit(out)
		return nil
	}
	_, err = m.Reply(out)
	return err
}

func registerTidesHandlers() {
	c := Client
	c.On("cmd:tides", TidesHandler)
}

func init() {
	QueueHandlerRegistration(registerTidesHandlers)
}
