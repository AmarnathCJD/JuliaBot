package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

const currencyCacheTTL = 10 * time.Minute

var currencyTopList = []string{"USD", "EUR", "GBP", "JPY", "INR", "CNY", "AUD", "CAD"}

type currencyCacheEntry struct {
	rate      float64
	timestamp time.Time
}

type currencyMultiEntry struct {
	rates     map[string]float64
	timestamp time.Time
}

var (
	currencyCache      sync.Map
	currencyMultiCache sync.Map
)

type currencyConvertResponse struct {
	Success bool `json:"success"`
	Query   struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	} `json:"query"`
	Info struct {
		Rate float64 `json:"rate"`
	} `json:"info"`
	Result float64 `json:"result"`
	Error  *struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
}

type currencyLatestResponse struct {
	Success bool               `json:"success"`
	Base    string             `json:"base"`
	Rates   map[string]float64 `json:"rates"`
	Error   *struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
}

func fetchCurrencyRate(from, to string) (float64, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	key := from + "_" + to

	if v, ok := currencyCache.Load(key); ok {
		entry := v.(currencyCacheEntry)
		if time.Since(entry.timestamp) < currencyCacheTTL {
			return entry.rate, nil
		}
	}

	url := fmt.Sprintf("https://api.exchangerate.host/convert?from=%s&to=%s&amount=1", from, to)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	var parsed currencyConvertResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, err
	}

	if parsed.Error != nil {
		return 0, fmt.Errorf("%s", parsed.Error.Info)
	}

	rate := parsed.Info.Rate
	if rate == 0 && parsed.Result != 0 {
		rate = parsed.Result
	}
	if rate == 0 {
		return 0, fmt.Errorf("invalid currency code")
	}

	currencyCache.Store(key, currencyCacheEntry{rate: rate, timestamp: time.Now()})
	return rate, nil
}

func fetchCurrencyMulti(from string, targets []string) (map[string]float64, error) {
	from = strings.ToUpper(from)
	key := from + "_MULTI"

	if v, ok := currencyMultiCache.Load(key); ok {
		entry := v.(currencyMultiEntry)
		if time.Since(entry.timestamp) < currencyCacheTTL {
			return entry.rates, nil
		}
	}

	symbols := strings.Join(targets, ",")
	url := fmt.Sprintf("https://api.exchangerate.host/latest?base=%s&symbols=%s", from, symbols)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	var parsed currencyLatestResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	if parsed.Error != nil {
		return nil, fmt.Errorf("%s", parsed.Error.Info)
	}

	if len(parsed.Rates) == 0 {
		return nil, fmt.Errorf("no rates returned")
	}

	currencyMultiCache.Store(key, currencyMultiEntry{rates: parsed.Rates, timestamp: time.Now()})
	return parsed.Rates, nil
}

func formatCurrencyNumber(v float64) string {
	abs := v
	if abs < 0 {
		abs = -abs
	}
	if abs >= 1000 {
		whole := int64(v)
		neg := ""
		if v < 0 {
			neg = "-"
			whole = -whole
		}
		s := fmt.Sprintf("%d", whole)
		n := len(s)
		var b strings.Builder
		b.WriteString(neg)
		first := n % 3
		if first == 0 {
			first = 3
		}
		b.WriteString(s[:first])
		for i := first; i < n; i += 3 {
			b.WriteString(",")
			b.WriteString(s[i : i+3])
		}
		frac := v - float64(int64(v))
		if frac < 0 {
			frac = -frac
		}
		b.WriteString(fmt.Sprintf(".%02d", int64(frac*100+0.5)))
		return b.String()
	}
	if abs >= 1 {
		return fmt.Sprintf("%.2f", v)
	}
	if abs >= 0.01 {
		return fmt.Sprintf("%.4f", v)
	}
	if abs == 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.6f", v)
}

func parseCurrencyAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", "")
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func CurrencyHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/cur &lt;amount&gt; &lt;from&gt; &lt;to&gt;</code>\n\n<b>Examples:</b>\n<code>/cur 100 USD INR</code>\n<code>/cur 50 EUR all</code>")
		return err
	}

	fields := strings.Fields(arg)
	if len(fields) < 3 {
		_, err := m.Reply("<b>Usage:</b> <code>/cur &lt;amount&gt; &lt;from&gt; &lt;to&gt;</code>")
		return err
	}

	amount, err := parseCurrencyAmount(fields[0])
	if err != nil || amount <= 0 {
		_, err := m.Reply("Invalid amount.")
		return err
	}

	from := strings.ToUpper(fields[1])
	to := strings.ToUpper(fields[2])

	if len(from) != 3 {
		_, err := m.Reply("Invalid <code>from</code> currency. Use 3-letter codes like <code>USD</code>.")
		return err
	}

	status, _ := m.Reply("Converting...")

	if to == "ALL" {
		targets := make([]string, 0, len(currencyTopList))
		for _, c := range currencyTopList {
			if c != from {
				targets = append(targets, c)
			}
		}

		rates, err := fetchCurrencyMulti(from, targets)
		if err != nil {
			msg := "Failed to fetch rates: <code>" + html.EscapeString(err.Error()) + "</code>"
			if status != nil {
				status.Edit(msg)
			} else {
				m.Reply(msg)
			}
			return nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("<b>%s %s</b> equals:\n\n", formatCurrencyNumber(amount), html.EscapeString(from)))
		for _, code := range currencyTopList {
			if code == from {
				continue
			}
			rate, ok := rates[code]
			if !ok {
				continue
			}
			sb.WriteString(fmt.Sprintf("• <code>%s</code> <b>%s</b>\n", code, formatCurrencyNumber(amount*rate)))
		}
		sb.WriteString("\n<i>Cached up to 10 min · exchangerate.host</i>")

		out := sb.String()
		if status != nil {
			status.Edit(out)
		} else {
			_, err := m.Reply(out)
			return err
		}
		return nil
	}

	if len(to) != 3 {
		msg := "Invalid <code>to</code> currency. Use 3-letter codes or <code>all</code>."
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	rate, err := fetchCurrencyRate(from, to)
	if err != nil {
		msg := "Failed to fetch rate: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	converted := amount * rate
	out := fmt.Sprintf("<b>%s %s</b> = <b>%s %s</b>\n\n<i>Rate:</i> <code>1 %s = %s %s</code>\n<i>Cached up to 10 min · exchangerate.host</i>",
		formatCurrencyNumber(amount), html.EscapeString(from),
		formatCurrencyNumber(converted), html.EscapeString(to),
		html.EscapeString(from), formatCurrencyNumber(rate), html.EscapeString(to))

	if status != nil {
		status.Edit(out)
	} else {
		_, err := m.Reply(out)
		return err
	}
	return nil
}

func registerCurrencyHandlers() {
	c := Client
	c.On("cmd:cur", CurrencyHandler)
}

func init() {
	QueueHandlerRegistration(registerCurrencyHandlers)
}
