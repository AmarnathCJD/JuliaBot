package modules

import (
	"context"
	"fmt"
	"html"
	"net"
	"sort"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func dnsCleanDomain(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "https://")
	if i := strings.IndexAny(raw, "/?#"); i >= 0 {
		raw = raw[:i]
	}
	return strings.ToLower(strings.TrimSuffix(raw, "."))
}

func dnsValidDomain(d string) bool {
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

func dnsResolver() *net.Resolver {
	return &net.Resolver{PreferGo: false}
}

func dnsLookupA(domain string) ([]string, []string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ips, err := dnsResolver().LookupIPAddr(ctx, domain)
	if err != nil {
		return nil, nil, err
	}
	var v4, v6 []string
	for _, ip := range ips {
		if ip.IP.To4() != nil {
			v4 = append(v4, ip.IP.String())
		} else {
			v6 = append(v6, ip.IP.String())
		}
	}
	sort.Strings(v4)
	sort.Strings(v6)
	return v4, v6, nil
}

func dnsLookupMX(domain string) ([]*net.MX, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return dnsResolver().LookupMX(ctx, domain)
}

func dnsLookupNS(domain string) ([]*net.NS, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return dnsResolver().LookupNS(ctx, domain)
}

func dnsLookupTXT(domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return dnsResolver().LookupTXT(ctx, domain)
}

func dnsLookupCNAME(domain string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return dnsResolver().LookupCNAME(ctx, domain)
}

func dnsFormatA(domain string) string {
	v4, v6, err := dnsLookupA(domain)
	if err != nil {
		return "<i>No A/AAAA records found: <code>" + html.EscapeString(err.Error()) + "</code></i>"
	}
	if len(v4) == 0 && len(v6) == 0 {
		return "<i>No A/AAAA records found.</i>"
	}
	var sb strings.Builder
	if len(v4) > 0 {
		sb.WriteString("<b>A records:</b>")
		for _, ip := range v4 {
			sb.WriteString("\n  - <code>")
			sb.WriteString(html.EscapeString(ip))
			sb.WriteString("</code>")
		}
	}
	if len(v6) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString("<b>AAAA records:</b>")
		for _, ip := range v6 {
			sb.WriteString("\n  - <code>")
			sb.WriteString(html.EscapeString(ip))
			sb.WriteString("</code>")
		}
	}
	return sb.String()
}

func dnsFormatAOnly(domain string) string {
	v4, _, err := dnsLookupA(domain)
	if err != nil {
		return "<i>No A records found: <code>" + html.EscapeString(err.Error()) + "</code></i>"
	}
	if len(v4) == 0 {
		return "<i>No A records found.</i>"
	}
	var sb strings.Builder
	sb.WriteString("<b>A records:</b>")
	for _, ip := range v4 {
		sb.WriteString("\n  - <code>")
		sb.WriteString(html.EscapeString(ip))
		sb.WriteString("</code>")
	}
	return sb.String()
}

func dnsFormatAAAAOnly(domain string) string {
	_, v6, err := dnsLookupA(domain)
	if err != nil {
		return "<i>No AAAA records found: <code>" + html.EscapeString(err.Error()) + "</code></i>"
	}
	if len(v6) == 0 {
		return "<i>No AAAA records found.</i>"
	}
	var sb strings.Builder
	sb.WriteString("<b>AAAA records:</b>")
	for _, ip := range v6 {
		sb.WriteString("\n  - <code>")
		sb.WriteString(html.EscapeString(ip))
		sb.WriteString("</code>")
	}
	return sb.String()
}

func dnsFormatMX(domain string) string {
	mxs, err := dnsLookupMX(domain)
	if err != nil {
		return "<i>No MX records found: <code>" + html.EscapeString(err.Error()) + "</code></i>"
	}
	if len(mxs) == 0 {
		return "<i>No MX records found.</i>"
	}
	sort.Slice(mxs, func(i, j int) bool { return mxs[i].Pref < mxs[j].Pref })
	var sb strings.Builder
	sb.WriteString("<b>MX records:</b>")
	for _, mx := range mxs {
		host := strings.TrimSuffix(mx.Host, ".")
		sb.WriteString(fmt.Sprintf("\n  - <code>%d</code> <code>%s</code>", mx.Pref, html.EscapeString(host)))
	}
	return sb.String()
}

func dnsFormatNS(domain string) string {
	nss, err := dnsLookupNS(domain)
	if err != nil {
		return "<i>No NS records found: <code>" + html.EscapeString(err.Error()) + "</code></i>"
	}
	if len(nss) == 0 {
		return "<i>No NS records found.</i>"
	}
	hosts := make([]string, 0, len(nss))
	for _, ns := range nss {
		hosts = append(hosts, strings.TrimSuffix(ns.Host, "."))
	}
	sort.Strings(hosts)
	var sb strings.Builder
	sb.WriteString("<b>NS records:</b>")
	for _, h := range hosts {
		sb.WriteString("\n  - <code>")
		sb.WriteString(html.EscapeString(h))
		sb.WriteString("</code>")
	}
	return sb.String()
}

func dnsFormatTXT(domain string) string {
	txts, err := dnsLookupTXT(domain)
	if err != nil {
		return "<i>No TXT records found: <code>" + html.EscapeString(err.Error()) + "</code></i>"
	}
	if len(txts) == 0 {
		return "<i>No TXT records found.</i>"
	}
	var sb strings.Builder
	sb.WriteString("<b>TXT records:</b>")
	for _, t := range txts {
		display := t
		if len(display) > 300 {
			display = display[:300] + "..."
		}
		sb.WriteString("\n  - <code>")
		sb.WriteString(html.EscapeString(display))
		sb.WriteString("</code>")
	}
	return sb.String()
}

func dnsFormatCNAME(domain string) string {
	cname, err := dnsLookupCNAME(domain)
	if err != nil {
		return "<i>No CNAME record found: <code>" + html.EscapeString(err.Error()) + "</code></i>"
	}
	cname = strings.TrimSuffix(cname, ".")
	if cname == "" || strings.EqualFold(cname, domain) {
		return "<i>No CNAME record found (resolves to itself).</i>"
	}
	return "<b>CNAME:</b> <code>" + html.EscapeString(cname) + "</code>"
}

func dnsFormatAll(domain string) string {
	var parts []string
	if s := dnsFormatA(domain); s != "" {
		parts = append(parts, s)
	}
	if s := dnsFormatMX(domain); s != "" {
		parts = append(parts, s)
	}
	if s := dnsFormatNS(domain); s != "" {
		parts = append(parts, s)
	}
	if s := dnsFormatTXT(domain); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n\n")
}

func DNSLookupHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/dns &lt;domain&gt; [type]</code>\n<b>Types:</b> <code>A</code>, <code>AAAA</code>, <code>MX</code>, <code>NS</code>, <code>TXT</code>, <code>CNAME</code>\n<b>Example:</b> <code>/dns google.com MX</code>")
		return nil
	}
	fields := strings.Fields(args)
	domain := dnsCleanDomain(fields[0])
	if !dnsValidDomain(domain) {
		m.Reply("<b>Invalid domain.</b> Try something like <code>example.com</code>.")
		return nil
	}
	recType := "ALL"
	if len(fields) > 1 {
		recType = strings.ToUpper(strings.TrimSpace(fields[1]))
	}
	switch recType {
	case "A", "AAAA", "MX", "NS", "TXT", "CNAME", "ALL":
	default:
		m.Reply("<b>Invalid type:</b> <code>" + html.EscapeString(recType) + "</code>\n<b>Supported:</b> <code>A</code>, <code>AAAA</code>, <code>MX</code>, <code>NS</code>, <code>TXT</code>, <code>CNAME</code>")
		return nil
	}
	status, _ := m.Reply("<i>Resolving <code>" + html.EscapeString(domain) + "</code>...</i>")
	var body string
	switch recType {
	case "A":
		body = dnsFormatAOnly(domain)
	case "AAAA":
		body = dnsFormatAAAAOnly(domain)
	case "MX":
		body = dnsFormatMX(domain)
	case "NS":
		body = dnsFormatNS(domain)
	case "TXT":
		body = dnsFormatTXT(domain)
	case "CNAME":
		body = dnsFormatCNAME(domain)
	default:
		body = dnsFormatAll(domain)
	}
	header := "<b>DNS:</b> <code>" + html.EscapeString(domain) + "</code>"
	if recType != "ALL" {
		header += " <i>(" + html.EscapeString(recType) + ")</i>"
	}
	out := header + "\n\n" + body
	if len(out) > 4000 {
		out = out[:4000] + "\n\n<i>(truncated)</i>"
	}
	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func init() { QueueHandlerRegistration(registerDNSLookupHandlers) }

func registerDNSLookupHandlers() {
	c := Client
	c.On("cmd:dns", DNSLookupHandler)
}
