package modules

import (
	"crypto/tls"
	"fmt"
	"html"
	"net"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func sslCertNormalizeHost(input string) (string, string) {
	h := strings.TrimSpace(input)
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimSuffix(h, "/")
	if idx := strings.Index(h, "/"); idx != -1 {
		h = h[:idx]
	}
	host := h
	port := "443"
	if strings.HasPrefix(h, "[") {
		if end := strings.Index(h, "]"); end != -1 {
			host = h[1:end]
			rest := h[end+1:]
			if strings.HasPrefix(rest, ":") {
				port = strings.TrimPrefix(rest, ":")
			}
		}
	} else if strings.Count(h, ":") == 1 {
		hp, p, err := net.SplitHostPort(h)
		if err == nil {
			host = hp
			port = p
		}
	}
	return host, port
}

func sslCertFetch(host, port string) (*tls.ConnectionState, error) {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	state := conn.ConnectionState()
	return &state, nil
}

func SSLCertHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/sslcert &lt;host&gt;</code>\n<b>Example:</b> <code>/sslcert google.com</code>")
		return err
	}

	arg = strings.Fields(arg)[0]
	host, port := sslCertNormalizeHost(arg)
	if host == "" {
		_, err := m.Reply("Invalid host: <code>" + html.EscapeString(arg) + "</code>")
		return err
	}

	status, _ := m.Reply("Fetching certificate for <code>" + html.EscapeString(host) + ":" + html.EscapeString(port) + "</code>...")

	state, err := sslCertFetch(host, port)
	if err != nil {
		msg := "Failed to fetch certificate: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	if len(state.PeerCertificates) == 0 {
		msg := "No certificates returned by <code>" + html.EscapeString(host) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	cert := state.PeerCertificates[0]
	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
	expired := now.After(cert.NotAfter)
	notYetValid := now.Before(cert.NotBefore)

	var sb strings.Builder
	sb.WriteString("<b>SSL Certificate</b>\n\n")
	sb.WriteString("<b>Host:</b> <code>" + html.EscapeString(host) + ":" + html.EscapeString(port) + "</code>\n")

	if subject := cert.Subject.String(); subject != "" {
		sb.WriteString("<b>Subject:</b> <code>" + html.EscapeString(subject) + "</code>\n")
	}
	if cn := cert.Subject.CommonName; cn != "" {
		sb.WriteString("<b>Common Name:</b> <code>" + html.EscapeString(cn) + "</code>\n")
	}

	if issuer := cert.Issuer.String(); issuer != "" {
		sb.WriteString("<b>Issuer:</b> <code>" + html.EscapeString(issuer) + "</code>\n")
	}
	if icn := cert.Issuer.CommonName; icn != "" {
		sb.WriteString("<b>Issuer CN:</b> <code>" + html.EscapeString(icn) + "</code>\n")
	}

	sb.WriteString("<b>Not Before:</b> <code>" + html.EscapeString(cert.NotBefore.UTC().Format(time.RFC3339)) + "</code>\n")
	sb.WriteString("<b>Not After:</b> <code>" + html.EscapeString(cert.NotAfter.UTC().Format(time.RFC3339)) + "</code>\n")

	switch {
	case expired:
		sb.WriteString(fmt.Sprintf("<b>Status:</b> <code>EXPIRED %d days ago</code>\n", -daysLeft))
	case notYetValid:
		sb.WriteString("<b>Status:</b> <code>NOT YET VALID</code>\n")
	default:
		sb.WriteString(fmt.Sprintf("<b>Days Until Expiry:</b> <code>%d</code>\n", daysLeft))
	}

	if cert.SerialNumber != nil {
		sb.WriteString("<b>Serial:</b> <code>" + html.EscapeString(cert.SerialNumber.String()) + "</code>\n")
	}
	if cert.SignatureAlgorithm.String() != "" {
		sb.WriteString("<b>Signature:</b> <code>" + html.EscapeString(cert.SignatureAlgorithm.String()) + "</code>\n")
	}

	if len(cert.DNSNames) > 0 {
		dns := cert.DNSNames
		if len(dns) > 12 {
			dns = append(dns[:12:12], fmt.Sprintf("... (+%d more)", len(cert.DNSNames)-12))
		}
		sb.WriteString("<b>SANs:</b> <code>" + html.EscapeString(strings.Join(dns, ", ")) + "</code>\n")
	}

	if len(state.PeerCertificates) > 1 {
		sb.WriteString(fmt.Sprintf("<b>Chain Length:</b> <code>%d</code>\n", len(state.PeerCertificates)))
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
		return nil
	}
	_, err = m.Reply(out)
	return err
}

func registerSSLCertHandlers() {
	c := Client
	c.On("cmd:sslcert", SSLCertHandler)
}

func init() { QueueHandlerRegistration(registerSSLCertHandlers) }
