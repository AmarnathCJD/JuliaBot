package modules

import (
	"context"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

const previewMaxBytes = 5 * 1024 * 1024
const previewMaxRedirects = 5

var (
	ogTitleRe       = regexp.MustCompile(`(?is)<meta[^>]+property\s*=\s*["']og:title["'][^>]+content\s*=\s*["']([^"']+)["']`)
	ogTitleRe2      = regexp.MustCompile(`(?is)<meta[^>]+content\s*=\s*["']([^"']+)["'][^>]+property\s*=\s*["']og:title["']`)
	ogDescRe        = regexp.MustCompile(`(?is)<meta[^>]+property\s*=\s*["']og:description["'][^>]+content\s*=\s*["']([^"']+)["']`)
	ogDescRe2       = regexp.MustCompile(`(?is)<meta[^>]+content\s*=\s*["']([^"']+)["'][^>]+property\s*=\s*["']og:description["']`)
	ogImageRe       = regexp.MustCompile(`(?is)<meta[^>]+property\s*=\s*["']og:image["'][^>]+content\s*=\s*["']([^"']+)["']`)
	ogImageRe2      = regexp.MustCompile(`(?is)<meta[^>]+content\s*=\s*["']([^"']+)["'][^>]+property\s*=\s*["']og:image["']`)
	ogSiteNameRe    = regexp.MustCompile(`(?is)<meta[^>]+property\s*=\s*["']og:site_name["'][^>]+content\s*=\s*["']([^"']+)["']`)
	ogSiteNameRe2   = regexp.MustCompile(`(?is)<meta[^>]+content\s*=\s*["']([^"']+)["'][^>]+property\s*=\s*["']og:site_name["']`)
	titleTagRe      = regexp.MustCompile(`(?is)<title[^>]*>([^<]+)</title>`)
	metaDescRe      = regexp.MustCompile(`(?is)<meta[^>]+name\s*=\s*["']description["'][^>]+content\s*=\s*["']([^"']+)["']`)
	metaDescRe2     = regexp.MustCompile(`(?is)<meta[^>]+content\s*=\s*["']([^"']+)["'][^>]+name\s*=\s*["']description["']`)
	htmlEntityAmpRe = regexp.MustCompile(`&amp;`)
)

type previewData struct {
	URL         string
	Title       string
	Description string
	Image       string
	SiteName    string
}

func isPrivateHostIP(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return false
		}
		for _, a := range ips {
			if isPrivateOrReservedIP(a) {
				return true
			}
		}
		return false
	}
	return isPrivateOrReservedIP(ip)
}

func previewSafeDialer() *net.Dialer {
	return &net.Dialer{Timeout: 10 * time.Second}
}

func previewSafeTransport() *http.Transport {
	d := previewSafeDialer()
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, fmt.Errorf("unused")
		},
		ResponseHeaderTimeout: 15 * time.Second,
		MaxIdleConns:          5,
		Dial: func(network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.LookupIP(host)
			if err != nil {
				return nil, err
			}
			for _, ip := range ips {
				if isPrivateOrReservedIP(ip) {
					return nil, fmt.Errorf("refusing to connect to private/reserved IP")
				}
			}
			return d.Dial(network, net.JoinHostPort(ips[0].String(), port))
		},
	}
}

func extractOG(body string) previewData {
	pd := previewData{}
	getFirst := func(res ...[]string) string {
		for _, r := range res {
			if len(r) >= 2 {
				v := strings.TrimSpace(r[1])
				v = htmlEntityAmpRe.ReplaceAllString(v, "&")
				if v != "" {
					return v
				}
			}
		}
		return ""
	}
	pd.Title = getFirst(ogTitleRe.FindStringSubmatch(body), ogTitleRe2.FindStringSubmatch(body))
	if pd.Title == "" {
		pd.Title = getFirst(titleTagRe.FindStringSubmatch(body))
	}
	pd.Description = getFirst(ogDescRe.FindStringSubmatch(body), ogDescRe2.FindStringSubmatch(body))
	if pd.Description == "" {
		pd.Description = getFirst(metaDescRe.FindStringSubmatch(body), metaDescRe2.FindStringSubmatch(body))
	}
	pd.Image = getFirst(ogImageRe.FindStringSubmatch(body), ogImageRe2.FindStringSubmatch(body))
	pd.SiteName = getFirst(ogSiteNameRe.FindStringSubmatch(body), ogSiteNameRe2.FindStringSubmatch(body))
	return pd
}

func fetchPreview(rawURL string) (*previewData, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
		rawURL = u.String()
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only http/https supported")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("missing host")
	}

	hostOnly := u.Hostname()
	if isPrivateHostIP(hostOnly) {
		return nil, fmt.Errorf("refusing to fetch private/reserved host")
	}

	transport := &http.Transport{
		ResponseHeaderTimeout: 15 * time.Second,
		Dial: func(network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.LookupIP(host)
			if err != nil {
				return nil, err
			}
			var chosen net.IP
			for _, ip := range ips {
				if isPrivateOrReservedIP(ip) {
					return nil, fmt.Errorf("refusing to connect to private/reserved IP")
				}
				if chosen == nil {
					chosen = ip
				}
			}
			if chosen == nil {
				return nil, fmt.Errorf("no IP resolved")
			}
			d := &net.Dialer{Timeout: 10 * time.Second}
			return d.Dial(network, net.JoinHostPort(chosen.String(), port))
		},
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= previewMaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Host == "" {
				return fmt.Errorf("redirect missing host")
			}
			if isPrivateHostIP(req.URL.Hostname()) {
				return fmt.Errorf("redirect to private host blocked")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("redirect to non-http scheme blocked")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; JuliaBot/1.0; +https://t.me)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if ct != "" && !strings.Contains(ct, "text/html") && !strings.Contains(ct, "application/xhtml") && !strings.Contains(ct, "text/plain") && !strings.Contains(ct, "application/xml") {
		return nil, fmt.Errorf("unsupported content-type: %s", ct)
	}

	limited := io.LimitReader(resp.Body, previewMaxBytes)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	pd := extractOG(string(bodyBytes))
	pd.URL = resp.Request.URL.String()
	return &pd, nil
}

func truncatePreviewText(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "..."
}

func resolveRelativeImage(base, img string) string {
	if img == "" {
		return ""
	}
	if strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") {
		return img
	}
	bu, err := url.Parse(base)
	if err != nil {
		return img
	}
	iu, err := url.Parse(img)
	if err != nil {
		return img
	}
	return bu.ResolveReference(iu).String()
}

func PreviewHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/preview &lt;url&gt;</code>")
		return err
	}
	arg = strings.Fields(arg)[0]
	if !strings.HasPrefix(arg, "http://") && !strings.HasPrefix(arg, "https://") {
		arg = "https://" + arg
	}

	status, _ := m.Reply("Fetching preview...")

	pd, err := fetchPreview(arg)
	if err != nil {
		msg := "Failed to fetch preview: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	if pd.Title == "" && pd.Description == "" && pd.Image == "" {
		msg := "No preview metadata found for <code>" + html.EscapeString(pd.URL) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	var sb strings.Builder
	if pd.SiteName != "" {
		sb.WriteString("<i>" + html.EscapeString(pd.SiteName) + "</i>\n")
	}
	if pd.Title != "" {
		sb.WriteString("<b>" + html.EscapeString(truncatePreviewText(pd.Title, 200)) + "</b>\n")
	}
	if pd.Description != "" {
		sb.WriteString("\n" + html.EscapeString(truncatePreviewText(pd.Description, 400)) + "\n")
	}
	sb.WriteString("\n<a href=\"" + html.EscapeString(pd.URL) + "\">" + html.EscapeString(pd.URL) + "</a>")

	imgURL := resolveRelativeImage(pd.URL, pd.Image)

	if imgURL != "" {
		if status != nil {
			status.Delete()
		}
		_, err := m.ReplyMedia(imgURL, &tg.MediaOptions{
			Caption: sb.String(),
		})
		if err != nil {
			_, e := m.Reply(sb.String())
			return e
		}
		return nil
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
		return nil
	}
	_, err = m.Reply(out)
	return err
}

func registerLinksHandlers() {
	c := Client
	c.On("cmd:preview", PreviewHandler)
}

func init() {
	QueueHandlerRegistration(registerLinksHandlers)
}
