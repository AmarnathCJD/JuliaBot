package modules

import (
	"html"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type ccEntry struct {
	code    string
	name    string
	iso     string
	nsnMin  int
	nsnMax  int
	mobiles []string
}

var ccTable = []ccEntry{
	{"1", "United States/Canada", "US/CA", 10, 10, []string{"2", "3", "4", "5", "6", "7", "8", "9"}},
	{"7", "Russia/Kazakhstan", "RU/KZ", 10, 10, []string{"9"}},
	{"20", "Egypt", "EG", 9, 10, []string{"10", "11", "12", "15"}},
	{"27", "South Africa", "ZA", 9, 9, []string{"6", "7", "8"}},
	{"30", "Greece", "GR", 10, 10, []string{"69"}},
	{"31", "Netherlands", "NL", 9, 9, []string{"6"}},
	{"32", "Belgium", "BE", 8, 9, []string{"4"}},
	{"33", "France", "FR", 9, 9, []string{"6", "7"}},
	{"34", "Spain", "ES", 9, 9, []string{"6", "7"}},
	{"36", "Hungary", "HU", 8, 9, []string{"20", "30", "31", "50", "70"}},
	{"39", "Italy", "IT", 9, 11, []string{"3"}},
	{"40", "Romania", "RO", 9, 9, []string{"7"}},
	{"41", "Switzerland", "CH", 9, 9, []string{"7"}},
	{"43", "Austria", "AT", 10, 13, []string{"6"}},
	{"44", "United Kingdom", "GB", 10, 10, []string{"7"}},
	{"45", "Denmark", "DK", 8, 8, []string{"2", "3", "4", "5", "6", "9"}},
	{"46", "Sweden", "SE", 7, 13, []string{"7"}},
	{"47", "Norway", "NO", 8, 8, []string{"4", "9"}},
	{"48", "Poland", "PL", 9, 9, []string{"4", "5", "6", "7", "8"}},
	{"49", "Germany", "DE", 10, 11, []string{"15", "16", "17"}},
	{"51", "Peru", "PE", 8, 11, []string{"9"}},
	{"52", "Mexico", "MX", 10, 10, []string{"1"}},
	{"53", "Cuba", "CU", 8, 8, []string{"5"}},
	{"54", "Argentina", "AR", 10, 11, []string{"9", "15"}},
	{"55", "Brazil", "BR", 10, 11, []string{"9", "11", "21"}},
	{"56", "Chile", "CL", 9, 9, []string{"9"}},
	{"57", "Colombia", "CO", 10, 10, []string{"3"}},
	{"58", "Venezuela", "VE", 10, 10, []string{"4"}},
	{"60", "Malaysia", "MY", 9, 10, []string{"1"}},
	{"61", "Australia", "AU", 9, 9, []string{"4"}},
	{"62", "Indonesia", "ID", 9, 12, []string{"8"}},
	{"63", "Philippines", "PH", 10, 10, []string{"9"}},
	{"64", "New Zealand", "NZ", 8, 10, []string{"2"}},
	{"65", "Singapore", "SG", 8, 8, []string{"8", "9"}},
	{"66", "Thailand", "TH", 9, 9, []string{"6", "8", "9"}},
	{"81", "Japan", "JP", 9, 10, []string{"70", "80", "90"}},
	{"82", "South Korea", "KR", 9, 10, []string{"10", "11"}},
	{"84", "Vietnam", "VN", 9, 10, []string{"3", "5", "7", "8", "9"}},
	{"86", "China", "CN", 11, 11, []string{"13", "14", "15", "16", "17", "18", "19"}},
	{"90", "Turkey", "TR", 10, 10, []string{"5"}},
	{"91", "India", "IN", 10, 10, []string{"6", "7", "8", "9"}},
	{"92", "Pakistan", "PK", 10, 10, []string{"3"}},
	{"93", "Afghanistan", "AF", 9, 9, []string{"7"}},
	{"94", "Sri Lanka", "LK", 9, 9, []string{"7"}},
	{"95", "Myanmar", "MM", 7, 10, []string{"9"}},
	{"98", "Iran", "IR", 10, 10, []string{"9"}},
	{"211", "South Sudan", "SS", 9, 9, []string{"9"}},
	{"212", "Morocco", "MA", 9, 9, []string{"6", "7"}},
	{"213", "Algeria", "DZ", 9, 9, []string{"5", "6", "7"}},
	{"216", "Tunisia", "TN", 8, 8, []string{"2", "5", "9"}},
	{"218", "Libya", "LY", 9, 9, []string{"9"}},
	{"220", "Gambia", "GM", 7, 7, []string{"7", "9"}},
	{"221", "Senegal", "SN", 9, 9, []string{"7"}},
	{"222", "Mauritania", "MR", 8, 8, []string{"2", "3", "4"}},
	{"223", "Mali", "ML", 8, 8, []string{"6", "7", "9"}},
	{"224", "Guinea", "GN", 9, 9, []string{"6"}},
	{"225", "Ivory Coast", "CI", 10, 10, []string{"0"}},
	{"226", "Burkina Faso", "BF", 8, 8, []string{"5", "6", "7"}},
	{"227", "Niger", "NE", 8, 8, []string{"9"}},
	{"228", "Togo", "TG", 8, 8, []string{"9"}},
	{"229", "Benin", "BJ", 8, 8, []string{"9"}},
	{"230", "Mauritius", "MU", 7, 8, []string{"5"}},
	{"231", "Liberia", "LR", 7, 8, []string{"7", "8"}},
	{"232", "Sierra Leone", "SL", 8, 8, []string{"2", "3", "7", "8"}},
	{"233", "Ghana", "GH", 9, 9, []string{"2", "5"}},
	{"234", "Nigeria", "NG", 10, 10, []string{"7", "8", "9"}},
	{"235", "Chad", "TD", 8, 8, []string{"6", "9"}},
	{"236", "Central African Republic", "CF", 8, 8, []string{"7"}},
	{"237", "Cameroon", "CM", 9, 9, []string{"6"}},
	{"238", "Cape Verde", "CV", 7, 7, []string{"9"}},
	{"239", "Sao Tome and Principe", "ST", 7, 7, []string{"9"}},
	{"240", "Equatorial Guinea", "GQ", 9, 9, []string{"2"}},
	{"241", "Gabon", "GA", 7, 8, []string{"0"}},
	{"242", "Republic of the Congo", "CG", 9, 9, []string{"0"}},
	{"243", "DR Congo", "CD", 9, 9, []string{"8", "9"}},
	{"244", "Angola", "AO", 9, 9, []string{"9"}},
	{"245", "Guinea-Bissau", "GW", 7, 7, []string{"5", "6", "7"}},
	{"248", "Seychelles", "SC", 7, 7, []string{"2"}},
	{"249", "Sudan", "SD", 9, 9, []string{"9"}},
	{"250", "Rwanda", "RW", 9, 9, []string{"7"}},
	{"251", "Ethiopia", "ET", 9, 9, []string{"9"}},
	{"252", "Somalia", "SO", 7, 9, []string{"6", "7", "9"}},
	{"253", "Djibouti", "DJ", 8, 8, []string{"7", "8"}},
	{"254", "Kenya", "KE", 9, 9, []string{"7"}},
	{"255", "Tanzania", "TZ", 9, 9, []string{"6", "7"}},
	{"256", "Uganda", "UG", 9, 9, []string{"7"}},
	{"257", "Burundi", "BI", 8, 8, []string{"6", "7"}},
	{"258", "Mozambique", "MZ", 9, 9, []string{"8"}},
	{"260", "Zambia", "ZM", 9, 9, []string{"9"}},
	{"261", "Madagascar", "MG", 9, 9, []string{"3"}},
	{"263", "Zimbabwe", "ZW", 9, 9, []string{"7"}},
	{"264", "Namibia", "NA", 9, 9, []string{"8"}},
	{"265", "Malawi", "MW", 9, 9, []string{"8", "9"}},
	{"266", "Lesotho", "LS", 8, 8, []string{"5", "6"}},
	{"267", "Botswana", "BW", 8, 8, []string{"7"}},
	{"351", "Portugal", "PT", 9, 9, []string{"9"}},
	{"352", "Luxembourg", "LU", 8, 9, []string{"6"}},
	{"353", "Ireland", "IE", 9, 9, []string{"8"}},
	{"354", "Iceland", "IS", 7, 7, []string{"6", "7", "8"}},
	{"355", "Albania", "AL", 9, 9, []string{"6"}},
	{"356", "Malta", "MT", 8, 8, []string{"7", "9"}},
	{"357", "Cyprus", "CY", 8, 8, []string{"9"}},
	{"358", "Finland", "FI", 9, 10, []string{"4", "5"}},
	{"359", "Bulgaria", "BG", 9, 9, []string{"8", "9"}},
	{"370", "Lithuania", "LT", 8, 8, []string{"6"}},
	{"371", "Latvia", "LV", 8, 8, []string{"2"}},
	{"372", "Estonia", "EE", 7, 8, []string{"5"}},
	{"373", "Moldova", "MD", 8, 8, []string{"6", "7"}},
	{"374", "Armenia", "AM", 8, 8, []string{"4", "5", "7", "9"}},
	{"375", "Belarus", "BY", 9, 9, []string{"25", "29", "33", "44"}},
	{"376", "Andorra", "AD", 6, 6, []string{"3", "4", "6"}},
	{"380", "Ukraine", "UA", 9, 9, []string{"3", "5", "6", "7", "9"}},
	{"381", "Serbia", "RS", 8, 9, []string{"6"}},
	{"385", "Croatia", "HR", 8, 9, []string{"9"}},
	{"386", "Slovenia", "SI", 8, 8, []string{"3", "4", "5", "6", "7"}},
	{"420", "Czechia", "CZ", 9, 9, []string{"6", "7"}},
	{"421", "Slovakia", "SK", 9, 9, []string{"9"}},
	{"500", "Falkland Islands", "FK", 5, 5, []string{}},
	{"501", "Belize", "BZ", 7, 7, []string{"6"}},
	{"502", "Guatemala", "GT", 8, 8, []string{"3", "4", "5"}},
	{"503", "El Salvador", "SV", 8, 8, []string{"6", "7"}},
	{"504", "Honduras", "HN", 8, 8, []string{"3", "8", "9"}},
	{"505", "Nicaragua", "NI", 8, 8, []string{"5", "7", "8"}},
	{"506", "Costa Rica", "CR", 8, 8, []string{"5", "6", "7", "8"}},
	{"507", "Panama", "PA", 7, 8, []string{"6"}},
	{"509", "Haiti", "HT", 8, 8, []string{"3", "4"}},
	{"591", "Bolivia", "BO", 8, 8, []string{"6", "7"}},
	{"593", "Ecuador", "EC", 8, 9, []string{"9"}},
	{"595", "Paraguay", "PY", 9, 9, []string{"9"}},
	{"598", "Uruguay", "UY", 8, 8, []string{"9"}},
	{"673", "Brunei", "BN", 7, 7, []string{"7", "8"}},
	{"852", "Hong Kong", "HK", 8, 8, []string{"5", "6", "9"}},
	{"853", "Macau", "MO", 8, 8, []string{"6"}},
	{"855", "Cambodia", "KH", 8, 9, []string{"1", "6", "7", "8", "9"}},
	{"856", "Laos", "LA", 8, 10, []string{"20"}},
	{"880", "Bangladesh", "BD", 10, 10, []string{"1"}},
	{"886", "Taiwan", "TW", 9, 9, []string{"9"}},
	{"960", "Maldives", "MV", 7, 7, []string{"7", "9"}},
	{"961", "Lebanon", "LB", 7, 8, []string{"3", "7"}},
	{"962", "Jordan", "JO", 9, 9, []string{"7"}},
	{"963", "Syria", "SY", 9, 9, []string{"9"}},
	{"964", "Iraq", "IQ", 10, 10, []string{"7"}},
	{"965", "Kuwait", "KW", 8, 8, []string{"5", "6", "9"}},
	{"966", "Saudi Arabia", "SA", 9, 9, []string{"5"}},
	{"967", "Yemen", "YE", 9, 9, []string{"7"}},
	{"968", "Oman", "OM", 8, 8, []string{"7", "9"}},
	{"970", "Palestine", "PS", 9, 9, []string{"5"}},
	{"971", "United Arab Emirates", "AE", 8, 9, []string{"5"}},
	{"972", "Israel", "IL", 8, 9, []string{"5"}},
	{"973", "Bahrain", "BH", 8, 8, []string{"3", "6"}},
	{"974", "Qatar", "QA", 8, 8, []string{"3", "5", "6", "7"}},
	{"975", "Bhutan", "BT", 7, 8, []string{"1", "7"}},
	{"976", "Mongolia", "MN", 8, 8, []string{"5", "8", "9"}},
	{"977", "Nepal", "NP", 9, 10, []string{"9"}},
	{"992", "Tajikistan", "TJ", 9, 9, []string{"9"}},
	{"993", "Turkmenistan", "TM", 8, 8, []string{"6"}},
	{"994", "Azerbaijan", "AZ", 9, 9, []string{"4", "5", "6", "7"}},
	{"995", "Georgia", "GE", 9, 9, []string{"5"}},
	{"996", "Kyrgyzstan", "KG", 9, 9, []string{"5", "7", "9"}},
	{"998", "Uzbekistan", "UZ", 9, 9, []string{"9"}},
}

func numFlagFor(iso string) string {
	iso = strings.TrimSpace(iso)
	if idx := strings.Index(iso, "/"); idx >= 0 {
		iso = iso[:idx]
	}
	if f := countryFlagEmoji(iso); f != "" {
		return f
	}
	return "🌐"
}

func sanitizeNumber(s string) string {
	var b strings.Builder
	hasPlus := false
	for i, r := range s {
		if i == 0 && r == '+' {
			hasPlus = true
			b.WriteRune(r)
			continue
		}
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if !hasPlus && strings.HasPrefix(out, "00") {
		out = "+" + strings.TrimPrefix(out, "00")
	}
	return out
}

func matchCountry(digits string) *ccEntry {
	var best *ccEntry
	for i := range ccTable {
		e := &ccTable[i]
		if strings.HasPrefix(digits, e.code) {
			if best == nil || len(e.code) > len(best.code) {
				best = e
			}
		}
	}
	return best
}

func guessLineType(e *ccEntry, nsn string) string {
	if len(nsn) == 0 {
		return "unknown"
	}
	for _, p := range e.mobiles {
		if strings.HasPrefix(nsn, p) {
			return "mobile"
		}
	}
	return "landline/fixed"
}

func formatNational(nsn string) string {
	n := len(nsn)
	switch {
	case n <= 4:
		return nsn
	case n <= 7:
		return nsn[:3] + " " + nsn[3:]
	case n == 8:
		return nsn[:4] + " " + nsn[4:]
	case n == 9:
		return nsn[:3] + " " + nsn[3:6] + " " + nsn[6:]
	case n == 10:
		return nsn[:3] + " " + nsn[3:6] + " " + nsn[6:]
	case n == 11:
		return nsn[:2] + " " + nsn[2:5] + " " + nsn[5:8] + " " + nsn[8:]
	default:
		return nsn[:3] + " " + nsn[3:6] + " " + nsn[6:]
	}
}

func NumInfoHandler(m *tg.NewMessage) error {
	raw := strings.TrimSpace(m.Args())
	if raw == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			raw = strings.TrimSpace(r.Text())
		}
	}
	if raw == "" {
		m.Reply("usage: <code>/numinfo &lt;phone&gt;</code>\nexample: <code>/numinfo +14155552671</code>\nworks offline, no external lookup.")
		return nil
	}

	cleaned := sanitizeNumber(raw)
	if !strings.HasPrefix(cleaned, "+") {
		m.Reply("number must be in E.164 format (start with <code>+</code> and country code).\nexample: <code>/numinfo +919876543210</code>")
		return nil
	}
	digits := strings.TrimPrefix(cleaned, "+")
	if len(digits) < 6 || len(digits) > 15 {
		m.Reply("invalid length. E.164 numbers have 6-15 digits after the <code>+</code>.")
		return nil
	}

	e := matchCountry(digits)
	if e == nil {
		m.Reply("country code not in built-in table for <code>" + html.EscapeString(cleaned) + "</code>.")
		return nil
	}
	nsn := strings.TrimPrefix(digits, e.code)
	if len(nsn) < e.nsnMin || len(nsn) > e.nsnMax {
		m.Reply("number length looks wrong for <b>" + html.EscapeString(e.name) + "</b>.\nexpected national digits: <code>" + itoa(e.nsnMin) + "-" + itoa(e.nsnMax) + "</code>, got <code>" + itoa(len(nsn)) + "</code>.")
		return nil
	}

	lineType := guessLineType(e, nsn)
	natFormatted := formatNational(nsn)
	intlFormatted := "+" + e.code + " " + natFormatted
	flag := numFlagFor(e.iso)

	var sb strings.Builder
	sb.WriteString("<b>number info</b>\n\n")
	sb.WriteString("<b>input:</b> <code>")
	sb.WriteString(html.EscapeString(raw))
	sb.WriteString("</code>\n")
	sb.WriteString("<b>e.164:</b> <code>")
	sb.WriteString(cleaned)
	sb.WriteString("</code>\n")
	sb.WriteString("<b>country:</b> ")
	sb.WriteString(flag)
	sb.WriteString(" ")
	sb.WriteString(html.EscapeString(e.name))
	sb.WriteString(" (<code>")
	sb.WriteString(e.iso)
	sb.WriteString("</code>)\n")
	sb.WriteString("<b>country code:</b> <code>+")
	sb.WriteString(e.code)
	sb.WriteString("</code>\n")
	sb.WriteString("<b>national:</b> <code>")
	sb.WriteString(natFormatted)
	sb.WriteString("</code>\n")
	sb.WriteString("<b>international:</b> <code>")
	sb.WriteString(intlFormatted)
	sb.WriteString("</code>\n")
	sb.WriteString("<b>nsn length:</b> <code>")
	sb.WriteString(itoa(len(nsn)))
	sb.WriteString("</code>\n")
	sb.WriteString("<b>type guess:</b> <code>")
	sb.WriteString(lineType)
	sb.WriteString("</code>\n\n")
	sb.WriteString("<i>offline lookup — no external API queried for privacy.</i>")

	m.Reply(sb.String())
	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func init() { QueueHandlerRegistration(registerTruecallerHandlers) }

func registerTruecallerHandlers() {
	c := Client
	c.On("cmd:numinfo", NumInfoHandler)
}
