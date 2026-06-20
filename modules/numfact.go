package modules

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var numFactClient = &http.Client{Timeout: 30 * time.Second}

func fetchNumFact(endpoint string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := numFactClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("numbersapi returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func NumFactHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("Usage: <code>/numfact &lt;number&gt;</code>")
		return nil
	}
	if _, err := strconv.Atoi(arg); err != nil {
		m.Reply("<b>Error:</b> please provide a valid integer.")
		return nil
	}
	endpoint := "http://numbersapi.com/" + arg + "/trivia"
	fact, err := fetchNumFact(endpoint)
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch number fact.")
		return nil
	}
	if fact == "" {
		m.Reply("<b>Error:</b> no fact received.")
		return nil
	}
	m.Reply(fmt.Sprintf("<b>Number Fact (%s):</b>\n<blockquote>%s</blockquote>", html.EscapeString(arg), html.EscapeString(fact)))
	return nil
}

func DateFactHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("Usage: <code>/datefact MM/DD</code>")
		return nil
	}
	parts := strings.Split(arg, "/")
	if len(parts) != 2 {
		m.Reply("<b>Error:</b> format must be <code>MM/DD</code>.")
		return nil
	}
	month, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || month < 1 || month > 12 {
		m.Reply("<b>Error:</b> invalid month (1-12).")
		return nil
	}
	day, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || day < 1 || day > 31 {
		m.Reply("<b>Error:</b> invalid day (1-31).")
		return nil
	}
	endpoint := fmt.Sprintf("http://numbersapi.com/%d/%d/date", month, day)
	fact, err := fetchNumFact(endpoint)
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch date fact.")
		return nil
	}
	if fact == "" {
		m.Reply("<b>Error:</b> no fact received.")
		return nil
	}
	label := fmt.Sprintf("%02d/%02d", month, day)
	m.Reply(fmt.Sprintf("<b>Date Fact (%s):</b>\n<blockquote>%s</blockquote>", html.EscapeString(label), html.EscapeString(fact)))
	return nil
}

func YearFactHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("Usage: <code>/yearfact &lt;year&gt;</code>")
		return nil
	}
	year, err := strconv.Atoi(arg)
	if err != nil {
		m.Reply("<b>Error:</b> please provide a valid year.")
		return nil
	}
	endpoint := fmt.Sprintf("http://numbersapi.com/%d/year", year)
	fact, err := fetchNumFact(endpoint)
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch year fact.")
		return nil
	}
	if fact == "" {
		m.Reply("<b>Error:</b> no fact received.")
		return nil
	}
	m.Reply(fmt.Sprintf("<b>Year Fact (%d):</b>\n<blockquote>%s</blockquote>", year, html.EscapeString(fact)))
	return nil
}

func registerNumFactHandlers() {
	c := Client
	c.On("cmd:numfact", NumFactHandler)
	c.On("cmd:datefact", DateFactHandler)
	c.On("cmd:yearfact", YearFactHandler)
}

func init() {
	QueueHandlerRegistration(registerNumFactHandlers)
}
