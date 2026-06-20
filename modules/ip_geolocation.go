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

type ipWhoIsFlag struct {
	Img   string `json:"img"`
	Emoji string `json:"emoji"`
}

type ipWhoIsConnection struct {
	ASN    int    `json:"asn"`
	Org    string `json:"org"`
	ISP    string `json:"isp"`
	Domain string `json:"domain"`
}

type ipWhoIsResponse struct {
	IP          string            `json:"ip"`
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	Type        string            `json:"type"`
	Continent   string            `json:"continent"`
	Country     string            `json:"country"`
	CountryCode string            `json:"country_code"`
	Region      string            `json:"region"`
	City        string            `json:"city"`
	Latitude    float64           `json:"latitude"`
	Longitude   float64           `json:"longitude"`
	Postal      string            `json:"postal"`
	Flag        ipWhoIsFlag       `json:"flag"`
	Connection  ipWhoIsConnection `json:"connection"`
}

func fetchIPWhoIs(query string) (*ipWhoIsResponse, error) {
	url := fmt.Sprintf("https://ipwho.is/%s", query)
	req, err := http.NewRequest("GET", url, nil)
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
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	var data ipWhoIsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func GeoIPHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/geoip &lt;ip&gt;</code>\n<b>Example:</b> <code>/geoip 8.8.8.8</code>")
		return err
	}

	arg = strings.Fields(arg)[0]
	arg = strings.TrimPrefix(arg, "http://")
	arg = strings.TrimPrefix(arg, "https://")
	arg = strings.TrimSuffix(arg, "/")
	if idx := strings.Index(arg, "/"); idx != -1 {
		arg = arg[:idx]
	}

	ip, err := resolveHostToIP(arg)
	if err != nil {
		_, e := m.Reply("Failed to resolve <code>" + html.EscapeString(arg) + "</code>: " + html.EscapeString(err.Error()))
		return e
	}

	if isPrivateOrReservedIP(ip) {
		_, e := m.Reply("Refusing to query private/reserved IP: <code>" + html.EscapeString(ip.String()) + "</code>")
		return e
	}

	status, _ := m.Reply("Querying ipwho.is for <code>" + html.EscapeString(ip.String()) + "</code>...")

	data, err := fetchIPWhoIs(ip.String())
	if err != nil {
		msg := "Failed to fetch geolocation: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	if !data.Success {
		reason := data.Message
		if reason == "" {
			reason = "lookup failed"
		}
		msg := "ipwho.is error: <code>" + html.EscapeString(reason) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	var sb strings.Builder
	flag := data.Flag.Emoji
	if flag == "" {
		flag = countryFlagEmoji(data.CountryCode)
	}

	header := "<b>GeoIP Lookup</b>"
	if flag != "" {
		header = flag + " <b>GeoIP Lookup</b>"
	}
	sb.WriteString(header + "\n\n")

	if data.IP != "" {
		sb.WriteString("<b>IP:</b> <code>" + html.EscapeString(data.IP) + "</code>")
		if data.Type != "" {
			sb.WriteString(" (" + html.EscapeString(data.Type) + ")")
		}
		sb.WriteString("\n")
	}

	var locParts []string
	if data.City != "" {
		locParts = append(locParts, data.City)
	}
	if data.Region != "" {
		locParts = append(locParts, data.Region)
	}
	if data.Country != "" {
		country := data.Country
		if data.CountryCode != "" {
			country = country + " (" + data.CountryCode + ")"
		}
		locParts = append(locParts, country)
	}
	if len(locParts) > 0 {
		sb.WriteString("<b>Location:</b> " + html.EscapeString(strings.Join(locParts, ", ")) + "\n")
	}

	if data.Continent != "" {
		sb.WriteString("<b>Continent:</b> " + html.EscapeString(data.Continent) + "\n")
	}

	if data.Postal != "" {
		sb.WriteString("<b>Postal:</b> <code>" + html.EscapeString(data.Postal) + "</code>\n")
	}

	if data.Latitude != 0 || data.Longitude != 0 {
		coords := fmt.Sprintf("%.6f, %.6f", data.Latitude, data.Longitude)
		sb.WriteString("<b>Coords:</b> <code>" + html.EscapeString(coords) + "</code>\n")
		mapURL := fmt.Sprintf("https://maps.google.com/?q=%.6f,%.6f", data.Latitude, data.Longitude)
		sb.WriteString("<b>Map:</b> <a href=\"" + mapURL + "\">Open in Maps</a>\n")
	}

	if data.Connection.ISP != "" {
		sb.WriteString("<b>ISP:</b> " + html.EscapeString(data.Connection.ISP) + "\n")
	}
	if data.Connection.Org != "" && data.Connection.Org != data.Connection.ISP {
		sb.WriteString("<b>Org:</b> " + html.EscapeString(data.Connection.Org) + "\n")
	}
	if data.Connection.ASN != 0 {
		sb.WriteString(fmt.Sprintf("<b>ASN:</b> <code>AS%d</code>\n", data.Connection.ASN))
	}
	if data.Connection.Domain != "" {
		sb.WriteString("<b>Domain:</b> <code>" + html.EscapeString(data.Connection.Domain) + "</code>\n")
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
		return nil
	}
	_, err = m.Reply(out)
	return err
}

func registerGeoIPHandlers() {
	c := Client
	c.On("cmd:geoip", GeoIPHandler)
}

func init() {
	QueueHandlerRegistration(registerGeoIPHandlers)
}
