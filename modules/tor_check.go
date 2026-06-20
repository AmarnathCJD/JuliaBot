package modules

import (
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func isCurlValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("empty host")
	}
	h := host
	if strings.Contains(h, ":") {
		hh, _, err := net.SplitHostPort(h)
		if err == nil {
			h = hh
		}
	}
	if strings.EqualFold(h, "localhost") {
		return fmt.Errorf("localhost is blocked")
	}
	if ip := net.ParseIP(h); ip != nil {
		if isPrivateOrReservedIP(ip) {
			return fmt.Errorf("private/reserved IP blocked: %s", ip.String())
		}
		return nil
	}
	ips, err := net.LookupIP(h)
	if err != nil {
		return fmt.Errorf("dns lookup failed: %v", err)
	}
	for _, ip := range ips {
		if isPrivateOrReservedIP(ip) {
			return fmt.Errorf("resolved IP is private/reserved: %s", ip.String())
		}
	}
	return nil
}

func isCurlPerform(target string) (string, int, http.Header, []string, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return "", 0, nil, nil, err
	}
	if parsed.Scheme == "" {
		target = "http://" + target
		parsed, err = url.Parse(target)
		if err != nil {
			return "", 0, nil, nil, err
		}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", 0, nil, nil, fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
	}
	if err := isCurlValidateHost(parsed.Host); err != nil {
		return "", 0, nil, nil, err
	}

	var redirectChain []string
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			if err := isCurlValidateHost(req.URL.Host); err != nil {
				return err
			}
			redirectChain = append(redirectChain, req.URL.String())
			return nil
		},
	}

	req, err := http.NewRequest("HEAD", target, nil)
	if err != nil {
		return "", 0, nil, nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0 iscurl")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		req2, e2 := http.NewRequest("GET", target, nil)
		if e2 != nil {
			return "", 0, nil, nil, err
		}
		req2.Header.Set("User-Agent", "JuliaBot/1.0 iscurl")
		req2.Header.Set("Accept", "*/*")
		resp, err = client.Do(req2)
		if err != nil {
			return "", 0, nil, nil, err
		}
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	return finalURL, resp.StatusCode, resp.Header, redirectChain, nil
}

func IsCurlHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/iscurl &lt;url&gt;</code>\n<b>Example:</b> <code>/iscurl https://example.com</code>")
		return err
	}
	target := strings.Fields(arg)[0]

	status, _ := m.Reply("Checking <code>" + html.EscapeString(target) + "</code>...")

	finalURL, code, headers, chain, err := isCurlPerform(target)
	if err != nil {
		msg := "Request failed: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	var sb strings.Builder
	sb.WriteString("<b>iscurl</b>\n\n")
	sb.WriteString("<b>Target:</b> <code>" + html.EscapeString(target) + "</code>\n")
	sb.WriteString("<b>Final URL:</b> <code>" + html.EscapeString(finalURL) + "</code>\n")
	sb.WriteString("<b>Status:</b> <code>" + strconv.Itoa(code) + " " + html.EscapeString(http.StatusText(code)) + "</code>\n")

	if len(chain) > 0 {
		sb.WriteString("<b>Redirects:</b> <code>" + strconv.Itoa(len(chain)) + "</code>\n")
	}

	server := headers.Get("Server")
	ctype := headers.Get("Content-Type")
	clen := headers.Get("Content-Length")

	if server != "" {
		sb.WriteString("<b>Server:</b> <code>" + html.EscapeString(server) + "</code>\n")
	}
	if ctype != "" {
		sb.WriteString("<b>Content-Type:</b> <code>" + html.EscapeString(ctype) + "</code>\n")
	}
	if clen != "" {
		sb.WriteString("<b>Content-Length:</b> <code>" + html.EscapeString(clen) + "</code>\n")
	}

	if loc := headers.Get("Location"); loc != "" {
		sb.WriteString("<b>Location:</b> <code>" + html.EscapeString(loc) + "</code>\n")
	}
	if poweredBy := headers.Get("X-Powered-By"); poweredBy != "" {
		sb.WriteString("<b>X-Powered-By:</b> <code>" + html.EscapeString(poweredBy) + "</code>\n")
	}
	if cf := headers.Get("CF-Ray"); cf != "" {
		sb.WriteString("<b>CF-Ray:</b> <code>" + html.EscapeString(cf) + "</code>\n")
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
		return nil
	}
	_, err = m.Reply(out)
	return err
}

func registerIsCurlHandlers() {
	c := Client
	c.On("cmd:iscurl", IsCurlHandler)
}

func init() {
	QueueHandlerRegistration(registerIsCurlHandlers)
}
