package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

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
	c := Client
	c.On("cmd:cryptochart", CryptoChartHandler)
}

func init() { QueueHandlerRegistration(registerCryptoChartHandlers) }
