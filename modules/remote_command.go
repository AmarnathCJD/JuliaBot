package modules

import (
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var ip2flagCountryNames = map[string]string{
	"AF": "Afghanistan", "AX": "Aland Islands", "AL": "Albania", "DZ": "Algeria",
	"AS": "American Samoa", "AD": "Andorra", "AO": "Angola", "AI": "Anguilla",
	"AQ": "Antarctica", "AG": "Antigua and Barbuda", "AR": "Argentina", "AM": "Armenia",
	"AW": "Aruba", "AU": "Australia", "AT": "Austria", "AZ": "Azerbaijan",
	"BS": "Bahamas", "BH": "Bahrain", "BD": "Bangladesh", "BB": "Barbados",
	"BY": "Belarus", "BE": "Belgium", "BZ": "Belize", "BJ": "Benin",
	"BM": "Bermuda", "BT": "Bhutan", "BO": "Bolivia", "BQ": "Bonaire",
	"BA": "Bosnia and Herzegovina", "BW": "Botswana", "BV": "Bouvet Island", "BR": "Brazil",
	"IO": "British Indian Ocean Territory", "BN": "Brunei", "BG": "Bulgaria", "BF": "Burkina Faso",
	"BI": "Burundi", "CV": "Cabo Verde", "KH": "Cambodia", "CM": "Cameroon",
	"CA": "Canada", "KY": "Cayman Islands", "CF": "Central African Republic", "TD": "Chad",
	"CL": "Chile", "CN": "China", "CX": "Christmas Island", "CC": "Cocos Islands",
	"CO": "Colombia", "KM": "Comoros", "CG": "Congo", "CD": "Congo (DRC)",
	"CK": "Cook Islands", "CR": "Costa Rica", "CI": "Cote d'Ivoire", "HR": "Croatia",
	"CU": "Cuba", "CW": "Curacao", "CY": "Cyprus", "CZ": "Czechia",
	"DK": "Denmark", "DJ": "Djibouti", "DM": "Dominica", "DO": "Dominican Republic",
	"EC": "Ecuador", "EG": "Egypt", "SV": "El Salvador", "GQ": "Equatorial Guinea",
	"ER": "Eritrea", "EE": "Estonia", "SZ": "Eswatini", "ET": "Ethiopia",
	"FK": "Falkland Islands", "FO": "Faroe Islands", "FJ": "Fiji", "FI": "Finland",
	"FR": "France", "GF": "French Guiana", "PF": "French Polynesia", "TF": "French Southern Territories",
	"GA": "Gabon", "GM": "Gambia", "GE": "Georgia", "DE": "Germany",
	"GH": "Ghana", "GI": "Gibraltar", "GR": "Greece", "GL": "Greenland",
	"GD": "Grenada", "GP": "Guadeloupe", "GU": "Guam", "GT": "Guatemala",
	"GG": "Guernsey", "GN": "Guinea", "GW": "Guinea-Bissau", "GY": "Guyana",
	"HT": "Haiti", "HM": "Heard Island", "VA": "Vatican City", "HN": "Honduras",
	"HK": "Hong Kong", "HU": "Hungary", "IS": "Iceland", "IN": "India",
	"ID": "Indonesia", "IR": "Iran", "IQ": "Iraq", "IE": "Ireland",
	"IM": "Isle of Man", "IL": "Israel", "IT": "Italy", "JM": "Jamaica",
	"JP": "Japan", "JE": "Jersey", "JO": "Jordan", "KZ": "Kazakhstan",
	"KE": "Kenya", "KI": "Kiribati", "KP": "North Korea", "KR": "South Korea",
	"KW": "Kuwait", "KG": "Kyrgyzstan", "LA": "Laos", "LV": "Latvia",
	"LB": "Lebanon", "LS": "Lesotho", "LR": "Liberia", "LY": "Libya",
	"LI": "Liechtenstein", "LT": "Lithuania", "LU": "Luxembourg", "MO": "Macao",
	"MG": "Madagascar", "MW": "Malawi", "MY": "Malaysia", "MV": "Maldives",
	"ML": "Mali", "MT": "Malta", "MH": "Marshall Islands", "MQ": "Martinique",
	"MR": "Mauritania", "MU": "Mauritius", "YT": "Mayotte", "MX": "Mexico",
	"FM": "Micronesia", "MD": "Moldova", "MC": "Monaco", "MN": "Mongolia",
	"ME": "Montenegro", "MS": "Montserrat", "MA": "Morocco", "MZ": "Mozambique",
	"MM": "Myanmar", "NA": "Namibia", "NR": "Nauru", "NP": "Nepal",
	"NL": "Netherlands", "NC": "New Caledonia", "NZ": "New Zealand", "NI": "Nicaragua",
	"NE": "Niger", "NG": "Nigeria", "NU": "Niue", "NF": "Norfolk Island",
	"MK": "North Macedonia", "MP": "Northern Mariana Islands", "NO": "Norway", "OM": "Oman",
	"PK": "Pakistan", "PW": "Palau", "PS": "Palestine", "PA": "Panama",
	"PG": "Papua New Guinea", "PY": "Paraguay", "PE": "Peru", "PH": "Philippines",
	"PN": "Pitcairn", "PL": "Poland", "PT": "Portugal", "PR": "Puerto Rico",
	"QA": "Qatar", "RE": "Reunion", "RO": "Romania", "RU": "Russia",
	"RW": "Rwanda", "BL": "Saint Barthelemy", "SH": "Saint Helena", "KN": "Saint Kitts and Nevis",
	"LC": "Saint Lucia", "MF": "Saint Martin", "PM": "Saint Pierre and Miquelon", "VC": "Saint Vincent",
	"WS": "Samoa", "SM": "San Marino", "ST": "Sao Tome and Principe", "SA": "Saudi Arabia",
	"SN": "Senegal", "RS": "Serbia", "SC": "Seychelles", "SL": "Sierra Leone",
	"SG": "Singapore", "SX": "Sint Maarten", "SK": "Slovakia", "SI": "Slovenia",
	"SB": "Solomon Islands", "SO": "Somalia", "ZA": "South Africa", "GS": "South Georgia",
	"SS": "South Sudan", "ES": "Spain", "LK": "Sri Lanka", "SD": "Sudan",
	"SR": "Suriname", "SJ": "Svalbard and Jan Mayen", "SE": "Sweden", "CH": "Switzerland",
	"SY": "Syria", "TW": "Taiwan", "TJ": "Tajikistan", "TZ": "Tanzania",
	"TH": "Thailand", "TL": "Timor-Leste", "TG": "Togo", "TK": "Tokelau",
	"TO": "Tonga", "TT": "Trinidad and Tobago", "TN": "Tunisia", "TR": "Turkey",
	"TM": "Turkmenistan", "TC": "Turks and Caicos Islands", "TV": "Tuvalu", "UG": "Uganda",
	"UA": "Ukraine", "AE": "United Arab Emirates", "GB": "United Kingdom", "US": "United States",
	"UM": "United States Minor Outlying Islands", "UY": "Uruguay", "UZ": "Uzbekistan", "VU": "Vanuatu",
	"VE": "Venezuela", "VN": "Vietnam", "VG": "British Virgin Islands", "VI": "US Virgin Islands",
	"WF": "Wallis and Futuna", "EH": "Western Sahara", "YE": "Yemen", "ZM": "Zambia",
	"ZW": "Zimbabwe",
}

func ip2flagEmojiFromCode(code string) string {
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

func ip2flagLookupCountryByIP(ip string) (string, error) {
	url := "https://ipapi.co/" + ip + "/country/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	code := strings.TrimSpace(string(body))
	if len(code) != 2 {
		return "", fmt.Errorf("unexpected response: %s", code)
	}
	return strings.ToUpper(code), nil
}

func IP2FlagHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/ip2flag &lt;country_code|ip&gt;</code>\n<b>Examples:</b>\n<code>/ip2flag US</code>\n<code>/ip2flag 8.8.8.8</code>")
		return err
	}

	arg = strings.Fields(arg)[0]

	if len(arg) == 2 {
		code := strings.ToUpper(arg)
		flag := ip2flagEmojiFromCode(code)
		if flag == "" {
			_, err := m.Reply("Invalid country code: <code>" + html.EscapeString(arg) + "</code>")
			return err
		}
		name, ok := ip2flagCountryNames[code]
		out := flag + " <b>" + html.EscapeString(code) + "</b>"
		if ok {
			out += " — " + html.EscapeString(name)
		}
		_, err := m.Reply(out)
		return err
	}

	ip := net.ParseIP(arg)
	if ip == nil {
		_, err := m.Reply("Not a valid country code or IP: <code>" + html.EscapeString(arg) + "</code>")
		return err
	}

	if isPrivateOrReservedIP(ip) {
		_, err := m.Reply("Refusing to query private/reserved IP: <code>" + html.EscapeString(ip.String()) + "</code>")
		return err
	}

	status, _ := m.Reply("Looking up <code>" + html.EscapeString(ip.String()) + "</code>...")

	code, err := ip2flagLookupCountryByIP(ip.String())
	if err != nil {
		msg := "Lookup failed: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	flag := ip2flagEmojiFromCode(code)
	if flag == "" {
		msg := "Got country <code>" + html.EscapeString(code) + "</code> but flag conversion failed."
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	name := ip2flagCountryNames[code]
	out := flag + " <b>" + html.EscapeString(code) + "</b>"
	if name != "" {
		out += " — " + html.EscapeString(name)
	}
	out += "\n<b>IP:</b> <code>" + html.EscapeString(ip.String()) + "</code>"

	if status != nil {
		status.Edit(out)
		return nil
	}
	_, err = m.Reply(out)
	return err
}

func registerIP2FlagHandlers() {
	c := Client
	c.On("cmd:ip2flag", IP2FlagHandler)
}

func init() {
	QueueHandlerRegistration(registerIP2FlagHandlers)
}
