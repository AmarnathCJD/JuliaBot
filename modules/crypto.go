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

const cryptoCacheTTL = 60 * time.Second

var cryptoSymbolMap = map[string]string{
	"btc":  "bitcoin",
	"eth":  "ethereum",
	"sol":  "solana",
	"doge": "dogecoin",
	"ada":  "cardano",
	"bnb":  "binancecoin",
	"xrp":  "ripple",
	"ltc":  "litecoin",
	"link": "chainlink",
	"dot":  "polkadot",
}

type cryptoPrice struct {
	ID           string
	USD          float64
	INR          float64
	EUR          float64
	USDChange24h float64
	MarketCapUSD float64
}

type cryptoTopCoin struct {
	ID                 string  `json:"id"`
	Symbol             string  `json:"symbol"`
	Name               string  `json:"name"`
	CurrentPrice       float64 `json:"current_price"`
	MarketCap          float64 `json:"market_cap"`
	MarketCapRank      int     `json:"market_cap_rank"`
	PriceChangePct24h  float64 `json:"price_change_percentage_24h"`
}

type cryptoPriceCacheEntry struct {
	data      cryptoPrice
	timestamp time.Time
}

type cryptoTopCacheEntry struct {
	data      []cryptoTopCoin
	timestamp time.Time
}

var (
	cryptoPriceCache   = make(map[string]cryptoPriceCacheEntry)
	cryptoPriceCacheMu sync.Mutex

	cryptoTopCache   cryptoTopCacheEntry
	cryptoTopCacheMu sync.Mutex
)

func resolveCryptoID(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	if mapped, ok := cryptoSymbolMap[lower]; ok {
		return mapped
	}
	return lower
}

func fetchCryptoPrice(coinID string) (cryptoPrice, error) {
	key := strings.ToLower(coinID)

	cryptoPriceCacheMu.Lock()
	if entry, ok := cryptoPriceCache[key]; ok {
		if time.Since(entry.timestamp) < cryptoCacheTTL {
			cryptoPriceCacheMu.Unlock()
			return entry.data, nil
		}
	}
	cryptoPriceCacheMu.Unlock()

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd,inr,eur&include_24hr_change=true&include_market_cap=true", key)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return cryptoPrice{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return cryptoPrice{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cryptoPrice{}, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	var raw map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return cryptoPrice{}, err
	}

	inner, ok := raw[key]
	if !ok || len(inner) == 0 {
		return cryptoPrice{}, fmt.Errorf("coin not found")
	}

	p := cryptoPrice{
		ID:           key,
		USD:          inner["usd"],
		INR:          inner["inr"],
		EUR:          inner["eur"],
		USDChange24h: inner["usd_24h_change"],
		MarketCapUSD: inner["usd_market_cap"],
	}

	cryptoPriceCacheMu.Lock()
	cryptoPriceCache[key] = cryptoPriceCacheEntry{data: p, timestamp: time.Now()}
	cryptoPriceCacheMu.Unlock()

	return p, nil
}

func fetchCryptoTop() ([]cryptoTopCoin, error) {
	cryptoTopCacheMu.Lock()
	if cryptoTopCache.data != nil && time.Since(cryptoTopCache.timestamp) < cryptoCacheTTL {
		data := cryptoTopCache.data
		cryptoTopCacheMu.Unlock()
		return data, nil
	}
	cryptoTopCacheMu.Unlock()

	url := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=10&page=1&sparkline=false&price_change_percentage=24h"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
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

	var coins []cryptoTopCoin
	if err := json.NewDecoder(resp.Body).Decode(&coins); err != nil {
		return nil, err
	}

	if len(coins) == 0 {
		return nil, fmt.Errorf("no coins returned")
	}

	cryptoTopCacheMu.Lock()
	cryptoTopCache = cryptoTopCacheEntry{data: coins, timestamp: time.Now()}
	cryptoTopCacheMu.Unlock()

	return coins, nil
}

func formatCryptoNumber(v float64) string {
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

func formatCryptoMarketCap(v float64) string {
	abs := v
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1_000_000_000_000:
		return fmt.Sprintf("$%.2fT", v/1_000_000_000_000)
	case abs >= 1_000_000_000:
		return fmt.Sprintf("$%.2fB", v/1_000_000_000)
	case abs >= 1_000_000:
		return fmt.Sprintf("$%.2fM", v/1_000_000)
	case abs >= 1_000:
		return fmt.Sprintf("$%.2fK", v/1_000)
	default:
		return fmt.Sprintf("$%.2f", v)
	}
}

func formatCryptoChange(pct float64) string {
	arrow := "↑"
	sign := "+"
	val := pct
	if pct < 0 {
		arrow = "↓"
		sign = "-"
		val = -pct
	}
	return fmt.Sprintf("%s %s%.2f%%", arrow, sign, val)
}

func titleCaseCrypto(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func PriceHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/price &lt;symbol&gt;</code>\n\n<b>Examples:</b> <code>/price btc</code>, <code>/price eth</code>, <code>/price solana</code>")
		return err
	}

	fields := strings.Fields(arg)
	input := fields[0]
	if len(input) > 40 {
		_, err := m.Reply("Invalid symbol.")
		return err
	}

	coinID := resolveCryptoID(input)

	status, _ := m.Reply("Fetching <b>" + html.EscapeString(strings.ToUpper(input)) + "</b>...")

	p, err := fetchCryptoPrice(coinID)
	if err != nil {
		msg := "Failed to fetch price: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if p.USD == 0 && p.INR == 0 && p.EUR == 0 {
		msg := "No price data available for <code>" + html.EscapeString(coinID) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	name := titleCaseCrypto(p.ID)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>%s</b> <code>(%s)</code>\n\n", html.EscapeString(name), html.EscapeString(strings.ToUpper(input))))
	sb.WriteString(fmt.Sprintf("<b>USD:</b> $%s\n", formatCryptoNumber(p.USD)))
	sb.WriteString(fmt.Sprintf("<b>INR:</b> ₹%s\n", formatCryptoNumber(p.INR)))
	sb.WriteString(fmt.Sprintf("<b>EUR:</b> €%s\n", formatCryptoNumber(p.EUR)))
	sb.WriteString(fmt.Sprintf("\n<b>24h Change:</b> %s\n", formatCryptoChange(p.USDChange24h)))
	if p.MarketCapUSD > 0 {
		sb.WriteString(fmt.Sprintf("<b>Market Cap:</b> %s\n", formatCryptoMarketCap(p.MarketCapUSD)))
	}
	sb.WriteString(fmt.Sprintf("\n<i>As of %s UTC</i>", time.Now().UTC().Format("2006-01-02 15:04")))

	out := sb.String()
	if status != nil {
		status.Edit(out)
	} else {
		_, err := m.Reply(out)
		return err
	}
	return nil
}

func TopCryptoHandler(m *tg.NewMessage) error {
	status, _ := m.Reply("Fetching top 10 cryptocurrencies...")

	coins, err := fetchCryptoTop()
	if err != nil {
		msg := "Failed to fetch top coins: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	var sb strings.Builder
	sb.WriteString("<b>Top 10 Cryptocurrencies by Market Cap</b>\n\n")
	for i, c := range coins {
		if i >= 10 {
			break
		}
		sym := strings.ToUpper(c.Symbol)
		sb.WriteString(fmt.Sprintf("<b>%d. %s</b> <code>(%s)</code>\n", c.MarketCapRank, html.EscapeString(c.Name), html.EscapeString(sym)))
		sb.WriteString(fmt.Sprintf("   Price: $%s | %s\n", formatCryptoNumber(c.CurrentPrice), formatCryptoChange(c.PriceChangePct24h)))
		sb.WriteString(fmt.Sprintf("   Cap: %s\n\n", formatCryptoMarketCap(c.MarketCap)))
	}
	sb.WriteString(fmt.Sprintf("<i>As of %s UTC</i>", time.Now().UTC().Format("2006-01-02 15:04")))

	out := sb.String()
	if status != nil {
		status.Edit(out)
	} else {
		_, err := m.Reply(out)
		return err
	}
	return nil
}

func registerCryptoHandlers() {
	c := Client
	c.On("cmd:price", PriceHandler)
	c.On("cmd:top", TopCryptoHandler)
}

func init() {
	QueueHandlerRegistration(registerCryptoHandlers)
}
