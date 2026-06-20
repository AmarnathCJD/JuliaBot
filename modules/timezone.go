package modules

import (
	"fmt"
	"html"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var cityTimezoneMap = map[string]string{
	"nyc":          "America/New_York",
	"newyork":      "America/New_York",
	"new york":     "America/New_York",
	"la":           "America/Los_Angeles",
	"losangeles":   "America/Los_Angeles",
	"los angeles":  "America/Los_Angeles",
	"chicago":      "America/Chicago",
	"houston":      "America/Chicago",
	"phoenix":      "America/Phoenix",
	"denver":       "America/Denver",
	"seattle":      "America/Los_Angeles",
	"sanfrancisco": "America/Los_Angeles",
	"san francisco": "America/Los_Angeles",
	"sf":           "America/Los_Angeles",
	"boston":       "America/New_York",
	"miami":        "America/New_York",
	"atlanta":      "America/New_York",
	"dallas":       "America/Chicago",
	"toronto":      "America/Toronto",
	"vancouver":    "America/Vancouver",
	"montreal":     "America/Montreal",
	"mexico":       "America/Mexico_City",
	"mexicocity":   "America/Mexico_City",
	"mexico city":  "America/Mexico_City",
	"london":       "Europe/London",
	"manchester":   "Europe/London",
	"dublin":       "Europe/Dublin",
	"paris":        "Europe/Paris",
	"berlin":       "Europe/Berlin",
	"munich":       "Europe/Berlin",
	"frankfurt":    "Europe/Berlin",
	"hamburg":      "Europe/Berlin",
	"madrid":       "Europe/Madrid",
	"barcelona":    "Europe/Madrid",
	"rome":         "Europe/Rome",
	"milan":        "Europe/Rome",
	"amsterdam":    "Europe/Amsterdam",
	"brussels":     "Europe/Brussels",
	"vienna":       "Europe/Vienna",
	"zurich":       "Europe/Zurich",
	"geneva":       "Europe/Zurich",
	"stockholm":    "Europe/Stockholm",
	"oslo":         "Europe/Oslo",
	"copenhagen":   "Europe/Copenhagen",
	"helsinki":     "Europe/Helsinki",
	"warsaw":       "Europe/Warsaw",
	"prague":       "Europe/Prague",
	"budapest":     "Europe/Budapest",
	"athens":       "Europe/Athens",
	"istanbul":     "Europe/Istanbul",
	"moscow":       "Europe/Moscow",
	"kiev":         "Europe/Kiev",
	"kyiv":         "Europe/Kiev",
	"lisbon":       "Europe/Lisbon",
	"dubai":        "Asia/Dubai",
	"abudhabi":     "Asia/Dubai",
	"abu dhabi":    "Asia/Dubai",
	"doha":         "Asia/Qatar",
	"riyadh":       "Asia/Riyadh",
	"jeddah":       "Asia/Riyadh",
	"tehran":       "Asia/Tehran",
	"baghdad":      "Asia/Baghdad",
	"jerusalem":    "Asia/Jerusalem",
	"telaviv":      "Asia/Jerusalem",
	"tel aviv":     "Asia/Jerusalem",
	"karachi":      "Asia/Karachi",
	"lahore":       "Asia/Karachi",
	"islamabad":    "Asia/Karachi",
	"mumbai":       "Asia/Kolkata",
	"delhi":        "Asia/Kolkata",
	"newdelhi":     "Asia/Kolkata",
	"new delhi":    "Asia/Kolkata",
	"bangalore":    "Asia/Kolkata",
	"bengaluru":    "Asia/Kolkata",
	"chennai":      "Asia/Kolkata",
	"kolkata":      "Asia/Kolkata",
	"hyderabad":    "Asia/Kolkata",
	"pune":         "Asia/Kolkata",
	"ahmedabad":    "Asia/Kolkata",
	"kathmandu":    "Asia/Kathmandu",
	"dhaka":        "Asia/Dhaka",
	"colombo":      "Asia/Colombo",
	"bangkok":      "Asia/Bangkok",
	"hanoi":        "Asia/Ho_Chi_Minh",
	"hochiminh":    "Asia/Ho_Chi_Minh",
	"saigon":       "Asia/Ho_Chi_Minh",
	"jakarta":      "Asia/Jakarta",
	"kualalumpur":  "Asia/Kuala_Lumpur",
	"kuala lumpur": "Asia/Kuala_Lumpur",
	"singapore":    "Asia/Singapore",
	"manila":       "Asia/Manila",
	"hongkong":     "Asia/Hong_Kong",
	"hong kong":    "Asia/Hong_Kong",
	"hk":           "Asia/Hong_Kong",
	"taipei":       "Asia/Taipei",
	"shanghai":     "Asia/Shanghai",
	"beijing":      "Asia/Shanghai",
	"guangzhou":    "Asia/Shanghai",
	"shenzhen":     "Asia/Shanghai",
	"seoul":        "Asia/Seoul",
	"tokyo":        "Asia/Tokyo",
	"osaka":        "Asia/Tokyo",
	"kyoto":        "Asia/Tokyo",
	"sydney":       "Australia/Sydney",
	"melbourne":    "Australia/Melbourne",
	"brisbane":     "Australia/Brisbane",
	"perth":        "Australia/Perth",
	"adelaide":     "Australia/Adelaide",
	"auckland":     "Pacific/Auckland",
	"wellington":   "Pacific/Auckland",
	"fiji":         "Pacific/Fiji",
	"honolulu":     "Pacific/Honolulu",
	"hawaii":       "Pacific/Honolulu",
	"anchorage":    "America/Anchorage",
	"cairo":        "Africa/Cairo",
	"lagos":        "Africa/Lagos",
	"nairobi":      "Africa/Nairobi",
	"johannesburg": "Africa/Johannesburg",
	"capetown":     "Africa/Johannesburg",
	"cape town":    "Africa/Johannesburg",
	"casablanca":   "Africa/Casablanca",
	"saopaulo":     "America/Sao_Paulo",
	"sao paulo":    "America/Sao_Paulo",
	"rio":          "America/Sao_Paulo",
	"riodejaneiro": "America/Sao_Paulo",
	"buenosaires":  "America/Argentina/Buenos_Aires",
	"buenos aires": "America/Argentina/Buenos_Aires",
	"santiago":     "America/Santiago",
	"lima":         "America/Lima",
	"bogota":       "America/Bogota",
	"caracas":      "America/Caracas",
	"havana":       "America/Havana",
	"utc":          "UTC",
	"gmt":          "GMT",
}

func resolveTimezone(input string) (*time.Location, string, bool) {
	key := strings.ToLower(strings.TrimSpace(input))
	if key == "" {
		return nil, "", false
	}
	if tz, ok := cityTimezoneMap[key]; ok {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc, tz, true
		}
	}
	collapsed := strings.ReplaceAll(key, " ", "")
	if tz, ok := cityTimezoneMap[collapsed]; ok {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc, tz, true
		}
	}
	if loc, err := time.LoadLocation(input); err == nil {
		return loc, input, true
	}
	return nil, "", false
}

func TimeHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("usage: <code>/time &lt;city or IANA tz&gt;</code>\nexamples: <code>/time tokyo</code>, <code>/time Asia/Kolkata</code>")
		return nil
	}
	loc, tzName, ok := resolveTimezone(args)
	if !ok {
		m.Reply("unknown city or timezone: <code>" + html.EscapeString(args) + "</code>\ntry an IANA timezone like <code>Europe/London</code>")
		return nil
	}
	now := time.Now().In(loc)
	zone, offset := now.Zone()
	offHours := offset / 3600
	offMins := (offset % 3600) / 60
	if offMins < 0 {
		offMins = -offMins
	}
	sign := "+"
	if offHours < 0 {
		sign = "-"
		offHours = -offHours
	}
	offStr := fmt.Sprintf("UTC%s%02d:%02d", sign, offHours, offMins)
	var sb strings.Builder
	sb.WriteString("<b>time in </b><code>")
	sb.WriteString(html.EscapeString(tzName))
	sb.WriteString("</code>\n\n")
	sb.WriteString("<b>local:</b> <code>")
	sb.WriteString(now.Format("Mon, 02 Jan 2006 15:04:05"))
	sb.WriteString("</code>\n")
	sb.WriteString("<b>zone:</b> <code>")
	sb.WriteString(html.EscapeString(zone))
	sb.WriteString("</code> (<code>")
	sb.WriteString(offStr)
	sb.WriteString("</code>)\n")
	sb.WriteString("<b>iso:</b> <code>")
	sb.WriteString(now.Format(time.RFC3339))
	sb.WriteString("</code>")
	m.Reply(sb.String())
	return nil
}

func WorldClockHandler(m *tg.NewMessage) error {
	type wcEntry struct {
		label string
		tz    string
	}
	entries := []wcEntry{
		{"New York", "America/New_York"},
		{"Los Angeles", "America/Los_Angeles"},
		{"London", "Europe/London"},
		{"Paris", "Europe/Paris"},
		{"Mumbai", "Asia/Kolkata"},
		{"Tokyo", "Asia/Tokyo"},
		{"Sydney", "Australia/Sydney"},
		{"Dubai", "Asia/Dubai"},
	}
	now := time.Now().UTC()
	maxLabel := 0
	for _, e := range entries {
		if len(e.label) > maxLabel {
			maxLabel = len(e.label)
		}
	}
	var sb strings.Builder
	sb.WriteString("<b>world clock</b>\n")
	sb.WriteString("<pre>")
	header := fmt.Sprintf("%-*s  %-11s  %s", maxLabel, "City", "Local Time", "Offset")
	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", len(header)))
	sb.WriteString("\n")
	for _, e := range entries {
		loc, err := time.LoadLocation(e.tz)
		if err != nil {
			continue
		}
		local := now.In(loc)
		_, offset := local.Zone()
		offHours := offset / 3600
		offMins := (offset % 3600) / 60
		if offMins < 0 {
			offMins = -offMins
		}
		sign := "+"
		h := offHours
		if h < 0 {
			sign = "-"
			h = -h
		}
		offStr := fmt.Sprintf("UTC%s%02d:%02d", sign, h, offMins)
		timeStr := local.Format("Mon 15:04")
		line := fmt.Sprintf("%-*s  %-11s  %s", maxLabel, e.label, timeStr, offStr)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString("</pre>")
	sb.WriteString("\n<i>updated: ")
	sb.WriteString(now.Format("2006-01-02 15:04:05 UTC"))
	sb.WriteString("</i>")
	m.Reply(sb.String())
	return nil
}

func init() { QueueHandlerRegistration(registerTimezoneHandlers) }

func registerTimezoneHandlers() {
	c := Client
	c.On("cmd:time", TimeHandler)
	c.On("cmd:worldclock", WorldClockHandler)
}
