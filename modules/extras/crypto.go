package extras

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	"html"
	"io"
	modules "main/modules"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	c := modules.Client
	c.On("cmd:price", PriceHandler)
	c.On("cmd:top", TopCryptoHandler)
}

func initFromSrc_crypto_0_1() {
	modules.QueueHandlerRegistration(registerCryptoHandlers)
}
type cryptoChartMarket struct {
	Prices [][]float64 `json:"prices"`
}

type cryptoChartSearchResp struct {
	Coins []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Symbol       string `json:"symbol"`
		MarketCapRnk int    `json:"market_cap_rank"`
	} `json:"coins"`
}

func cryptoChartHTTP(u string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")
	cl := &http.Client{Timeout: 20 * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func cryptoChartResolveID(sym string) (string, string, string, error) {
	q := url.QueryEscape(sym)
	body, err := cryptoChartHTTP("https://api.coingecko.com/api/v3/search?query=" + q)
	if err != nil {
		return "", "", "", err
	}
	var sr cryptoChartSearchResp
	if err := json.Unmarshal(body, &sr); err != nil {
		return "", "", "", err
	}
	if len(sr.Coins) == 0 {
		return "", "", "", fmt.Errorf("no coin found for %q", sym)
	}
	want := strings.ToLower(strings.TrimSpace(sym))
	best := sr.Coins[0]
	for _, c := range sr.Coins {
		if strings.ToLower(c.Symbol) == want || strings.ToLower(c.ID) == want || strings.ToLower(c.Name) == want {
			best = c
			break
		}
	}
	return best.ID, best.Name, strings.ToUpper(best.Symbol), nil
}

func cryptoChartFetch(id string) ([][]float64, error) {
	u := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/market_chart?vs_currency=usd&days=7&interval=daily", url.PathEscape(id))
	body, err := cryptoChartHTTP(u)
	if err != nil {
		return nil, err
	}
	var mc cryptoChartMarket
	if err := json.Unmarshal(body, &mc); err != nil {
		return nil, err
	}
	if len(mc.Prices) < 2 {
		return nil, fmt.Errorf("not enough price points")
	}
	return mc.Prices, nil
}

func cryptoChartFmtPrice(p float64) string {
	switch {
	case p >= 1000:
		return fmt.Sprintf("$%.0f", p)
	case p >= 1:
		return fmt.Sprintf("$%.2f", p)
	case p >= 0.01:
		return fmt.Sprintf("$%.4f", p)
	default:
		return fmt.Sprintf("$%.6f", p)
	}
}

func cryptoChartRender(name, sym string, points [][]float64) (string, float64, float64, float64, error) {
	const W, H = 640, 320
	const padL, padR, padT, padB = 70.0, 24.0, 50.0, 40.0
	dc := gg.NewContext(W, H)

	dc.SetRGB255(20, 22, 30)
	dc.Clear()

	minV := points[0][1]
	maxV := points[0][1]
	for _, p := range points {
		if p[1] < minV {
			minV = p[1]
		}
		if p[1] > maxV {
			maxV = p[1]
		}
	}
	if maxV == minV {
		maxV = minV + 1
	}
	rng := maxV - minV
	minV -= rng * 0.08
	maxV += rng * 0.08

	first := points[0][1]
	last := points[len(points)-1][1]
	pct := (last - first) / first * 100

	var lineR, lineG, lineB int
	if last >= first {
		lineR, lineG, lineB = 80, 220, 130
	} else {
		lineR, lineG, lineB = 240, 90, 90
	}

	dc.SetRGB255(45, 48, 60)
	dc.SetLineWidth(1)
	for i := 0; i <= 4; i++ {
		y := padT + (float64(H)-padT-padB)*float64(i)/4
		dc.DrawLine(padL, y, float64(W)-padR, y)
		dc.Stroke()
	}

	dc.SetRGB255(170, 175, 190)
	for i := 0; i <= 4; i++ {
		v := maxV - (maxV-minV)*float64(i)/4
		y := padT + (float64(H)-padT-padB)*float64(i)/4
		dc.DrawStringAnchored(cryptoChartFmtPrice(v), padL-8, y, 1, 0.4)
	}

	n := len(points)
	xAt := func(i int) float64 {
		return padL + (float64(W)-padL-padR)*float64(i)/float64(n-1)
	}
	yAt := func(v float64) float64 {
		return padT + (float64(H)-padT-padB)*(1-(v-minV)/(maxV-minV))
	}

	dc.SetRGBA255(lineR, lineG, lineB, 60)
	dc.MoveTo(xAt(0), float64(H)-padB)
	for i, p := range points {
		dc.LineTo(xAt(i), yAt(p[1]))
	}
	dc.LineTo(xAt(n-1), float64(H)-padB)
	dc.ClosePath()
	dc.Fill()

	dc.SetRGB255(lineR, lineG, lineB)
	dc.SetLineWidth(2.5)
	for i, p := range points {
		x, y := xAt(i), yAt(p[1])
		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.Stroke()

	for i, p := range points {
		dc.DrawCircle(xAt(i), yAt(p[1]), 3)
		dc.Fill()
	}

	dc.SetRGB255(230, 232, 240)
	title := fmt.Sprintf("%s (%s) - 7d", name, sym)
	dc.DrawStringAnchored(title, padL, 22, 0, 0.5)

	pctStr := fmt.Sprintf("%+.2f%%", pct)
	dc.SetRGB255(lineR, lineG, lineB)
	dc.DrawStringAnchored(pctStr, float64(W)-padR, 22, 1, 0.5)

	dc.SetRGB255(170, 175, 190)
	priceStr := cryptoChartFmtPrice(last)
	dc.DrawStringAnchored(priceStr, padL, 42, 0, 0.5)

	dc.SetRGB255(120, 125, 140)
	now := time.Now().UTC()
	for i := 0; i < n; i += int(math.Max(1, float64(n)/7)) {
		d := now.AddDate(0, 0, -(n - 1 - i))
		label := d.Format("Jan 02")
		dc.DrawStringAnchored(label, xAt(i), float64(H)-padB+18, 0.5, 0.5)
	}

	out := filepath.Join(os.TempDir(), fmt.Sprintf("cryptochart_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", 0, 0, 0, err
	}
	return out, first, last, pct, nil
}

func CryptoChartHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("usage: <code>/cryptochart &lt;symbol&gt;</code>\nexample: <code>/cryptochart btc</code>")
		return nil
	}
	fields := strings.Fields(arg)
	sym := fields[0]

	status, _ := m.Reply("<code>fetching " + html.EscapeString(strings.ToUpper(sym)) + " chart...</code>")

	id, name, ticker, err := cryptoChartResolveID(sym)
	if err != nil {
		msg := "error: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	points, err := cryptoChartFetch(id)
	if err != nil {
		msg := "error fetching chart: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	out, first, last, pct, err := cryptoChartRender(name, ticker, points)
	if err != nil {
		msg := "error rendering: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	arrow := "+"
	if last < first {
		arrow = ""
	}
	caption := fmt.Sprintf("<b>%s</b> (<code>%s</code>) - 7d\n<b>Now:</b> <code>%s</code>\n<b>Change:</b> <code>%s%.2f%%</code>",
		html.EscapeString(name), html.EscapeString(ticker), cryptoChartFmtPrice(last), arrow, pct)

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  caption,
		FileName: "cryptochart.png",
		MimeType: "image/png",
	})
	os.Remove(out)
	if merr != nil {
		m.Reply("error sending: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerCryptoChartHandlers() {
	c := modules.Client
	c.On("cmd:cryptochart", CryptoChartHandler)
}

func initFromSrc_crypto_chart_1_1() { modules.QueueHandlerRegistration(registerCryptoChartHandlers) }
var cryptoNewsHTTPClient = &http.Client{Timeout: 30 * time.Second}

type cryptoNewsItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

type cryptoNewsRSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title string           `xml:"title"`
		Items []cryptoNewsItem `xml:"item"`
	} `xml:"channel"`
}

func cryptoNewsFetch(endpoint string) (*cryptoNewsRSS, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 JuliaBot/1.0")
	resp, err := cryptoNewsHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var feed cryptoNewsRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func cryptoNewsSource(link string) string {
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return "Unknown"
	}
	host := strings.TrimPrefix(u.Host, "www.")
	return host
}

func CryptoNewsHandler(m *tg.NewMessage) error {
	feeds := []string{
		"https://cointelegraph.com/rss",
		"https://www.coindesk.com/arc/outboundfeeds/rss/",
	}
	var items []cryptoNewsItem
	for _, f := range feeds {
		feed, err := cryptoNewsFetch(f)
		if err != nil {
			continue
		}
		items = append(items, feed.Channel.Items...)
		if len(items) >= 5 {
			break
		}
	}
	if len(items) == 0 {
		m.Reply("<b>Error:</b> failed to fetch crypto news.")
		return nil
	}
	limit := 5
	if len(items) < limit {
		limit = len(items)
	}
	items = items[:limit]

	var b strings.Builder
	b.WriteString("<b>Top Crypto News</b>\n\n")
	for i, it := range items {
		title := strings.TrimSpace(it.Title)
		link := strings.TrimSpace(it.Link)
		if title == "" || link == "" {
			continue
		}
		src := cryptoNewsSource(link)
		b.WriteString(fmt.Sprintf("<b>%d.</b> <a href=\"%s\">%s</a>\n", i+1, html.EscapeString(link), html.EscapeString(title)))
		b.WriteString(fmt.Sprintf("    <b>Source:</b> %s\n\n", html.EscapeString(src)))
	}
	m.Reply(b.String(), &tg.SendOptions{LinkPreview: false})
	return nil
}

func registerCryptoNewsHandlers() {
	c := modules.Client
	c.On("cmd:cryptonews", CryptoNewsHandler)
}

func initFromSrc_crypto_news_2_1() {
	modules.QueueHandlerRegistration(registerCryptoNewsHandlers)
}

func init() {
	initFromSrc_crypto_0_1()
	initFromSrc_crypto_chart_1_1()
	initFromSrc_crypto_news_2_1()
}
