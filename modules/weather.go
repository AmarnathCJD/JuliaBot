package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type weatherGeocodeResult struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Country   string  `json:"country"`
	Admin1    string  `json:"admin1"`
	Timezone  string  `json:"timezone"`
}

type weatherGeocodeResponse struct {
	Results []weatherGeocodeResult `json:"results"`
}

type weatherCurrent struct {
	Time             string  `json:"time"`
	Temperature2m    float64 `json:"temperature_2m"`
	RelativeHumidity int     `json:"relative_humidity_2m"`
	WeatherCode      int     `json:"weather_code"`
	WindSpeed10m     float64 `json:"wind_speed_10m"`
	IsDay            int     `json:"is_day"`
}

type weatherForecastResponse struct {
	Latitude  float64        `json:"latitude"`
	Longitude float64        `json:"longitude"`
	Timezone  string         `json:"timezone"`
	Current   weatherCurrent `json:"current"`
}

func weatherCodeEmoji(code int, isDay int) string {
	switch {
	case code >= 0 && code <= 3:
		if isDay == 1 {
			if code == 0 {
				return "☀️"
			}
			return "⛅"
		}
		return "\U0001f319"
	case code >= 45 && code <= 48:
		return "\U0001f32b️"
	case code >= 51 && code <= 67:
		return "\U0001f327️"
	case code >= 71 && code <= 77:
		return "❄️"
	case code >= 80 && code <= 82:
		return "\U0001f326️"
	case code >= 95 && code <= 99:
		return "⛈️"
	}
	return "\U0001f324️"
}

func weatherCodeDescription(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45:
		return "Fog"
	case 48:
		return "Depositing rime fog"
	case 51:
		return "Light drizzle"
	case 53:
		return "Moderate drizzle"
	case 55:
		return "Dense drizzle"
	case 56:
		return "Light freezing drizzle"
	case 57:
		return "Dense freezing drizzle"
	case 61:
		return "Slight rain"
	case 63:
		return "Moderate rain"
	case 65:
		return "Heavy rain"
	case 66:
		return "Light freezing rain"
	case 67:
		return "Heavy freezing rain"
	case 71:
		return "Slight snow fall"
	case 73:
		return "Moderate snow fall"
	case 75:
		return "Heavy snow fall"
	case 77:
		return "Snow grains"
	case 80:
		return "Slight rain showers"
	case 81:
		return "Moderate rain showers"
	case 82:
		return "Violent rain showers"
	case 85:
		return "Slight snow showers"
	case 86:
		return "Heavy snow showers"
	case 95:
		return "Thunderstorm"
	case 96:
		return "Thunderstorm with slight hail"
	case 99:
		return "Thunderstorm with heavy hail"
	}
	return "Unknown"
}

func fetchWeatherGeocode(city string) (*weatherGeocodeResult, error) {
	endpoint := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1", url.QueryEscape(city))
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
		return nil, fmt.Errorf("geocode api returned status %d", resp.StatusCode)
	}

	var data weatherGeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Results) == 0 {
		return nil, fmt.Errorf("no results found")
	}
	return &data.Results[0], nil
}

func fetchWeatherForecast(lat, lon float64) (*weatherForecastResponse, error) {
	endpoint := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m,is_day&timezone=auto", lat, lon)
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
		return nil, fmt.Errorf("forecast api returned status %d", resp.StatusCode)
	}

	var data weatherForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func WeatherHandler(m *tg.NewMessage) error {
	city := strings.TrimSpace(m.Args())
	if city == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/weather &lt;city&gt;</code>\n<b>Example:</b> <code>/weather London</code>")
		return err
	}

	status, _ := m.Reply("Fetching weather for <code>" + html.EscapeString(city) + "</code>...")

	geo, err := fetchWeatherGeocode(city)
	if err != nil {
		msg := "Failed to find city <code>" + html.EscapeString(city) + "</code>: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	forecast, err := fetchWeatherForecast(geo.Latitude, geo.Longitude)
	if err != nil {
		msg := "Failed to fetch weather: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	emoji := weatherCodeEmoji(forecast.Current.WeatherCode, forecast.Current.IsDay)
	desc := weatherCodeDescription(forecast.Current.WeatherCode)

	var locParts []string
	if geo.Name != "" {
		locParts = append(locParts, geo.Name)
	}
	if geo.Admin1 != "" && geo.Admin1 != geo.Name {
		locParts = append(locParts, geo.Admin1)
	}
	if geo.Country != "" {
		locParts = append(locParts, geo.Country)
	}
	locationStr := strings.Join(locParts, ", ")

	var sb strings.Builder
	sb.WriteString(emoji + " <b>Weather</b>\n\n")
	sb.WriteString("<b>Location:</b> " + html.EscapeString(locationStr) + "\n")
	sb.WriteString("<b>Condition:</b> " + html.EscapeString(desc) + "\n")
	sb.WriteString(fmt.Sprintf("<b>Temperature:</b> <code>%.1f°C</code>\n", forecast.Current.Temperature2m))
	sb.WriteString(fmt.Sprintf("<b>Humidity:</b> <code>%d%%</code>\n", forecast.Current.RelativeHumidity))
	sb.WriteString(fmt.Sprintf("<b>Wind:</b> <code>%.1f km/h</code>\n", forecast.Current.WindSpeed10m))
	sb.WriteString(fmt.Sprintf("<b>Coords:</b> <code>%.4f, %.4f</code>\n", geo.Latitude, geo.Longitude))
	if forecast.Timezone != "" {
		sb.WriteString("<b>Timezone:</b> <code>" + html.EscapeString(forecast.Timezone) + "</code>\n")
	}
	if forecast.Current.Time != "" {
		sb.WriteString("<b>Updated:</b> <code>" + html.EscapeString(forecast.Current.Time) + "</code>\n")
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
		return nil
	}
	_, err = m.Reply(out)
	return err
}

func registerWeatherHandlers() {
	c := Client
	c.On("cmd:weather", WeatherHandler)
}

func init() {
	QueueHandlerRegistration(registerWeatherHandlers)
}
