package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type ipInfoResponse struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	Bogon    bool   `json:"bogon"`
}

func countryFlagEmoji(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 2 {
		return ""
	}
	r1 := rune(code[0])
	r2 := rune(code[1])
	if r1 < 'A' || r1 > 'Z' || r2 < 'A' || r2 > 'Z' {
		return ""
	}
	return string(rune(0x1F1E6+(r1-'A'))) + string(rune(0x1F1E6+(r2-'A')))
}

func isPrivateOrReservedIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		if ip4[0] >= 240 {
			return true
		}
	}
	return false
}

func resolveHostToIP(host string) (net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return ip, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip, nil
		}
	}
	if len(ips) > 0 {
		return ips[0], nil
	}
	return nil, fmt.Errorf("no IP found")
}

func fetchIPInfo(query string) (*ipInfoResponse, error) {
	url := fmt.Sprintf("https://ipinfo.io/%s/json", query)
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

	var data ipInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func IPInfoHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/ip &lt;ip_or_hostname&gt;</code>\n<b>Example:</b> <code>/ip 8.8.8.8</code>\n<code>/ip google.com</code>")
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

	status, _ := m.Reply("Querying ipinfo.io for <code>" + html.EscapeString(ip.String()) + "</code>...")

	data, err := fetchIPInfo(ip.String())
	if err != nil {
		msg := "Failed to fetch IP info: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	if data.Bogon {
		msg := "IP <code>" + html.EscapeString(ip.String()) + "</code> is a bogon address."
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	var sb strings.Builder
	flag := countryFlagEmoji(data.Country)
	header := "<b>IP Info</b>"
	if flag != "" {
		header = flag + " <b>IP Info</b>"
	}
	sb.WriteString(header + "\n\n")

	queryDisplay := arg
	if arg != ip.String() {
		sb.WriteString("<b>Query:</b> <code>" + html.EscapeString(queryDisplay) + "</code>\n")
	}

	if data.IP != "" {
		sb.WriteString("<b>IP:</b> <code>" + html.EscapeString(data.IP) + "</code>\n")
	} else {
		sb.WriteString("<b>IP:</b> <code>" + html.EscapeString(ip.String()) + "</code>\n")
	}

	if data.Hostname != "" {
		sb.WriteString("<b>Hostname:</b> <code>" + html.EscapeString(data.Hostname) + "</code>\n")
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
		if flag != "" {
			country = flag + " " + country
		}
		locParts = append(locParts, country)
	}
	if len(locParts) > 0 {
		sb.WriteString("<b>Location:</b> " + html.EscapeString(strings.Join(locParts, ", ")) + "\n")
	}

	if data.Loc != "" {
		sb.WriteString("<b>Coords:</b> <code>" + html.EscapeString(data.Loc) + "</code>\n")
	}

	if data.Org != "" {
		org := data.Org
		var asn string
		if strings.HasPrefix(org, "AS") {
			parts := strings.SplitN(org, " ", 2)
			if len(parts) == 2 {
				asn = parts[0]
				org = parts[1]
			}
		}
		if asn != "" {
			sb.WriteString("<b>ASN:</b> <code>" + html.EscapeString(asn) + "</code>\n")
		}
		sb.WriteString("<b>Org:</b> " + html.EscapeString(org) + "\n")
	}

	if data.Postal != "" {
		sb.WriteString("<b>Postal:</b> <code>" + html.EscapeString(data.Postal) + "</code>\n")
	}

	if data.Timezone != "" {
		sb.WriteString("<b>Timezone:</b> <code>" + html.EscapeString(data.Timezone) + "</code>\n")
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
		return nil
	}
	_, err = m.Reply(out)
	return err
}

func registerIPInfoHandlers() {
	c := Client
	c.On("cmd:ip", IPInfoHandler)
}

func init() {
	QueueHandlerRegistration(registerIPInfoHandlers)
}
