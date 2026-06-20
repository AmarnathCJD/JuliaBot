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

type rdapEvent struct {
	EventAction string `json:"eventAction"`
	EventDate   string `json:"eventDate"`
}

type rdapNameserver struct {
	LdhName     string `json:"ldhName"`
	UnicodeName string `json:"unicodeName"`
}

type rdapEntity struct {
	Handle     string        `json:"handle"`
	Roles      []string      `json:"roles"`
	VcardArray []interface{} `json:"vcardArray"`
	Entities   []rdapEntity  `json:"entities"`
}

type rdapDomainResponse struct {
	LdhName     string           `json:"ldhName"`
	UnicodeName string           `json:"unicodeName"`
	Handle      string           `json:"handle"`
	Status      []string         `json:"status"`
	Events      []rdapEvent      `json:"events"`
	Nameservers []rdapNameserver `json:"nameservers"`
	Entities    []rdapEntity     `json:"entities"`
	ErrorCode   int              `json:"errorCode"`
	Title       string           `json:"title"`
	Description []string         `json:"description"`
}

func whoisHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func whoisCleanDomain(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "www.")
	if i := strings.IndexAny(raw, "/?#"); i >= 0 {
		raw = raw[:i]
	}
	return strings.ToLower(raw)
}

func whoisValidDomain(d string) bool {
	if d == "" || len(d) > 253 {
		return false
	}
	if !strings.Contains(d, ".") {
		return false
	}
	for _, r := range d {
		if !(r == '.' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z')) {
			return false
		}
	}
	if strings.HasPrefix(d, ".") || strings.HasSuffix(d, ".") || strings.HasPrefix(d, "-") || strings.HasSuffix(d, "-") {
		return false
	}
	return true
}

func whoisFormatDate(s string) string {
	if s == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC().Format("2006-01-02 15:04 UTC")
		}
	}
	return s
}

func whoisExtractRegistrar(entities []rdapEntity) string {
	for _, e := range entities {
		isRegistrar := false
		for _, r := range e.Roles {
			if strings.EqualFold(r, "registrar") {
				isRegistrar = true
				break
			}
		}
		if !isRegistrar {
			continue
		}
		if name := whoisVcardName(e.VcardArray); name != "" {
			return name
		}
		if e.Handle != "" {
			return e.Handle
		}
	}
	return ""
}

func whoisVcardName(v []interface{}) string {
	if len(v) < 2 {
		return ""
	}
	props, ok := v[1].([]interface{})
	if !ok {
		return ""
	}
	for _, p := range props {
		entry, ok := p.([]interface{})
		if !ok || len(entry) < 4 {
			continue
		}
		name, ok := entry[0].(string)
		if !ok {
			continue
		}
		if name == "fn" {
			if val, ok := entry[3].(string); ok && val != "" {
				return val
			}
		}
	}
	return ""
}

func whoisEventDate(events []rdapEvent, action string) string {
	for _, e := range events {
		if strings.EqualFold(e.EventAction, action) {
			return e.EventDate
		}
	}
	return ""
}

func whoisFetch(domain string) (*rdapDomainResponse, int, error) {
	endpoint := "https://rdap.org/domain/" + domain
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/rdap+json")
	resp, err := whoisHTTPClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var out rdapDomainResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, resp.StatusCode, err
	}
	return &out, resp.StatusCode, nil
}

func WhoisHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/whois &lt;domain&gt;</code>")
		return nil
	}
	fields := strings.Fields(args)
	domain := whoisCleanDomain(fields[0])
	if !whoisValidDomain(domain) {
		m.Reply("<b>Invalid domain.</b> Try something like <code>example.com</code>.")
		return nil
	}
	status, _ := m.Reply("<i>Looking up <code>" + html.EscapeString(domain) + "</code>...</i>")
	data, code, err := whoisFetch(domain)
	if err != nil {
		msg := "<b>WHOIS lookup failed:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if code == 404 || (data != nil && data.ErrorCode == 404) {
		msg := "<b>No WHOIS record found for</b> <code>" + html.EscapeString(domain) + "</code>.\nThe domain may be unregistered or unsupported."
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if code >= 400 {
		title := data.Title
		if title == "" && len(data.Description) > 0 {
			title = data.Description[0]
		}
		if title == "" {
			title = fmt.Sprintf("HTTP %d", code)
		}
		msg := "<b>WHOIS lookup failed:</b> <code>" + html.EscapeString(title) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	name := data.LdhName
	if name == "" {
		name = domain
	}
	registrar := whoisExtractRegistrar(data.Entities)
	created := whoisFormatDate(whoisEventDate(data.Events, "registration"))
	if created == "" {
		created = whoisFormatDate(whoisEventDate(data.Events, "last changed"))
	}
	expiry := whoisFormatDate(whoisEventDate(data.Events, "expiration"))
	updated := whoisFormatDate(whoisEventDate(data.Events, "last changed"))

	var sb strings.Builder
	sb.WriteString("<b>WHOIS:</b> <code>")
	sb.WriteString(html.EscapeString(strings.ToLower(name)))
	sb.WriteString("</code>\n")
	if registrar != "" {
		sb.WriteString("\n<b>Registrar:</b> ")
		sb.WriteString(html.EscapeString(registrar))
	}
	if created != "" {
		sb.WriteString("\n<b>Registered:</b> <code>")
		sb.WriteString(html.EscapeString(created))
		sb.WriteString("</code>")
	}
	if updated != "" && updated != created {
		sb.WriteString("\n<b>Updated:</b> <code>")
		sb.WriteString(html.EscapeString(updated))
		sb.WriteString("</code>")
	}
	if expiry != "" {
		sb.WriteString("\n<b>Expires:</b> <code>")
		sb.WriteString(html.EscapeString(expiry))
		sb.WriteString("</code>")
	}
	if len(data.Status) > 0 {
		sb.WriteString("\n<b>Status:</b> <code>")
		sb.WriteString(html.EscapeString(strings.Join(data.Status, ", ")))
		sb.WriteString("</code>")
	}
	if len(data.Nameservers) > 0 {
		sb.WriteString("\n<b>Nameservers:</b>")
		seen := map[string]bool{}
		for _, ns := range data.Nameservers {
			n := ns.LdhName
			if n == "" {
				n = ns.UnicodeName
			}
			n = strings.ToLower(strings.TrimSpace(n))
			if n == "" || seen[n] {
				continue
			}
			seen[n] = true
			sb.WriteString("\n  - <code>")
			sb.WriteString(html.EscapeString(n))
			sb.WriteString("</code>")
		}
	}
	if registrar == "" && created == "" && expiry == "" && len(data.Status) == 0 && len(data.Nameservers) == 0 {
		sb.WriteString("\n<i>No detailed records available for this domain.</i>")
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func init() { QueueHandlerRegistration(registerWhoisHandlers) }

func registerWhoisHandlers() {
	c := Client
	c.On("cmd:whois", WhoisHandler)
}
