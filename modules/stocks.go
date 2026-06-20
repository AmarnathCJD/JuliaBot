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

const stockCacheTTL = 60 * time.Second

type stockQuote struct {
	Symbol         string
	ShortName      string
	LongName       string
	Currency       string
	Exchange       string
	Price          float64
	PrevClose      float64
	DayHigh        float64
	DayLow         float64
	Volume         int64
	FiftyTwoWkHigh float64
	FiftyTwoWkLow  float64
	MarketTime     int64
}

type stockCacheEntry struct {
	data      stockQuote
	timestamp time.Time
}

var (
	stockCache   = make(map[string]stockCacheEntry)
	stockCacheMu sync.Mutex
)

type stockYahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency             string  `json:"currency"`
				Symbol               string  `json:"symbol"`
				ExchangeName         string  `json:"exchangeName"`
				FullExchangeName     string  `json:"fullExchangeName"`
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				RegularMarketTime    int64   `json:"regularMarketTime"`
				RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
				RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
				RegularMarketVolume  int64   `json:"regularMarketVolume"`
				ChartPreviousClose   float64 `json:"chartPreviousClose"`
				PreviousClose        float64 `json:"previousClose"`
				FiftyTwoWeekHigh     float64 `json:"fiftyTwoWeekHigh"`
				FiftyTwoWeekLow      float64 `json:"fiftyTwoWeekLow"`
				ShortName            string  `json:"shortName"`
				LongName             string  `json:"longName"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

func fetchStockQuote(ticker string) (stockQuote, error) {
	key := strings.ToUpper(ticker)

	stockCacheMu.Lock()
	if entry, ok := stockCache[key]; ok {
		if time.Since(entry.timestamp) < stockCacheTTL {
			stockCacheMu.Unlock()
			return entry.data, nil
		}
	}
	stockCacheMu.Unlock()

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", key)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return stockQuote{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return stockQuote{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return stockQuote{}, fmt.Errorf("ticker not found")
	}
	if resp.StatusCode != http.StatusOK {
		return stockQuote{}, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	var parsed stockYahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return stockQuote{}, err
	}

	if parsed.Chart.Error != nil {
		return stockQuote{}, fmt.Errorf("%s", parsed.Chart.Error.Description)
	}

	if len(parsed.Chart.Result) == 0 {
		return stockQuote{}, fmt.Errorf("no data for ticker")
	}

	meta := parsed.Chart.Result[0].Meta
	prev := meta.ChartPreviousClose
	if prev == 0 {
		prev = meta.PreviousClose
	}

	q := stockQuote{
		Symbol:         meta.Symbol,
		ShortName:      meta.ShortName,
		LongName:       meta.LongName,
		Currency:       meta.Currency,
		Exchange:       meta.FullExchangeName,
		Price:          meta.RegularMarketPrice,
		PrevClose:      prev,
		DayHigh:        meta.RegularMarketDayHigh,
		DayLow:         meta.RegularMarketDayLow,
		Volume:         meta.RegularMarketVolume,
		FiftyTwoWkHigh: meta.FiftyTwoWeekHigh,
		FiftyTwoWkLow:  meta.FiftyTwoWeekLow,
		MarketTime:     meta.RegularMarketTime,
	}

	stockCacheMu.Lock()
	stockCache[key] = stockCacheEntry{data: q, timestamp: time.Now()}
	stockCacheMu.Unlock()

	return q, nil
}

func stockCurrencySymbol(code string) string {
	switch strings.ToUpper(code) {
	case "USD":
		return "$"
	case "INR":
		return "₹"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "JPY":
		return "¥"
	case "CNY":
		return "¥"
	case "KRW":
		return "₩"
	case "RUB":
		return "₽"
	case "AUD":
		return "A$"
	case "CAD":
		return "C$"
	case "HKD":
		return "HK$"
	case "SGD":
		return "S$"
	case "":
		return ""
	default:
		return code + " "
	}
}

func formatStockNumber(v float64) string {
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

func formatStockVolume(v int64) string {
	abs := v
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", float64(v)/1_000_000_000)
	case abs >= 1_000_000:
		return fmt.Sprintf("%.2fM", float64(v)/1_000_000)
	case abs >= 1_000:
		return fmt.Sprintf("%.2fK", float64(v)/1_000)
	default:
		return fmt.Sprintf("%d", v)
	}
}

func stockChangeArrow(change, pct float64) string {
	sign := "+"
	arrow := "↑"
	if change < 0 {
		sign = "-"
		arrow = "↓"
		change = -change
		pct = -pct
	}
	return fmt.Sprintf("%s %s%s (%s%.2f%%)", arrow, sign, formatStockNumber(change), sign, pct)
}

func StockHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/stock &lt;ticker&gt;</code>\n\n<b>Examples:</b> <code>/stock AAPL</code>, <code>/stock TSLA</code>, <code>/stock RELIANCE.NS</code>")
		return err
	}

	fields := strings.Fields(arg)
	ticker := strings.ToUpper(fields[0])

	if len(ticker) > 20 {
		_, err := m.Reply("Invalid ticker.")
		return err
	}

	status, _ := m.Reply("Fetching <b>" + html.EscapeString(ticker) + "</b>...")

	q, err := fetchStockQuote(ticker)
	if err != nil {
		msg := "Failed to fetch stock: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if q.Price == 0 {
		msg := "No price data available for <code>" + html.EscapeString(ticker) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	sym := stockCurrencySymbol(q.Currency)
	change := q.Price - q.PrevClose
	var pct float64
	if q.PrevClose != 0 {
		pct = (change / q.PrevClose) * 100
	}

	name := q.LongName
	if name == "" {
		name = q.ShortName
	}
	if name == "" {
		name = q.Symbol
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>%s</b> <code>(%s)</code>\n", html.EscapeString(name), html.EscapeString(q.Symbol)))
	if q.Exchange != "" {
		sb.WriteString(fmt.Sprintf("<i>%s</i>\n", html.EscapeString(q.Exchange)))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("<b>Price:</b> %s%s\n", sym, formatStockNumber(q.Price)))
	sb.WriteString(fmt.Sprintf("<b>Change:</b> %s\n", stockChangeArrow(change, pct)))
	sb.WriteString(fmt.Sprintf("<b>Prev Close:</b> %s%s\n", sym, formatStockNumber(q.PrevClose)))
	sb.WriteString(fmt.Sprintf("<b>Day High:</b> %s%s\n", sym, formatStockNumber(q.DayHigh)))
	sb.WriteString(fmt.Sprintf("<b>Day Low:</b> %s%s\n", sym, formatStockNumber(q.DayLow)))
	if q.FiftyTwoWkHigh > 0 || q.FiftyTwoWkLow > 0 {
		sb.WriteString(fmt.Sprintf("<b>52W Range:</b> %s%s - %s%s\n", sym, formatStockNumber(q.FiftyTwoWkLow), sym, formatStockNumber(q.FiftyTwoWkHigh)))
	}
	sb.WriteString(fmt.Sprintf("<b>Volume:</b> %s\n", formatStockVolume(q.Volume)))
	if q.MarketTime > 0 {
		sb.WriteString(fmt.Sprintf("\n<i>As of %s UTC</i>", time.Unix(q.MarketTime, 0).UTC().Format("2006-01-02 15:04")))
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
	} else {
		_, err := m.Reply(out)
		return err
	}
	return nil
}

func registerStocksHandlers() {
	c := Client
	c.On("cmd:stock", StockHandler)
}

func init() {
	QueueHandlerRegistration(registerStocksHandlers)
}
