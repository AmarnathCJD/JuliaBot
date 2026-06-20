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

type weatherRadarGeoResp struct {
	Results []struct {
		Name      string  `json:"name"`
		Country   string  `json:"country"`
		Admin1    string  `json:"admin1"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Timezone  string  `json:"timezone"`
	} `json:"results"`
}

type weatherRadarForecastResp struct {
	Daily struct {
		Time             []string  `json:"time"`
		TempMax          []float64 `json:"temperature_2m_max"`
		TempMin          []float64 `json:"temperature_2m_min"`
		PrecipitationSum []float64 `json:"precipitation_sum"`
	} `json:"daily"`
}

func weatherRadarHTTP(u string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")
	cl := &http.Client{Timeout: 30 * time.Second}
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

func weatherRadarGeocode(city string) (float64, float64, string, string, error) {
	u := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json", url.QueryEscape(city))
	body, err := weatherRadarHTTP(u)
	if err != nil {
		return 0, 0, "", "", err
	}
	var gr weatherRadarGeoResp
	if err := json.Unmarshal(body, &gr); err != nil {
		return 0, 0, "", "", err
	}
	if len(gr.Results) == 0 {
		return 0, 0, "", "", fmt.Errorf("city not found: %s", city)
	}
	r := gr.Results[0]
	name := r.Name
	if r.Admin1 != "" {
		name += ", " + r.Admin1
	}
	return r.Latitude, r.Longitude, name, r.Country, nil
}

func weatherRadarFetch(lat, lon float64) (*weatherRadarForecastResp, error) {
	u := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&daily=temperature_2m_max,temperature_2m_min,precipitation_sum&timezone=auto&forecast_days=7", lat, lon)
	body, err := weatherRadarHTTP(u)
	if err != nil {
		return nil, err
	}
	var fr weatherRadarForecastResp
	if err := json.Unmarshal(body, &fr); err != nil {
		return nil, err
	}
	if len(fr.Daily.Time) < 2 {
		return nil, fmt.Errorf("not enough forecast data")
	}
	return &fr, nil
}

func weatherRadarRender(city, country string, f *weatherRadarForecastResp) (string, error) {
	const W, H = 900, 520
	const padL, padR, padT, padB = 70.0, 70.0, 80.0, 70.0
	dc := gg.NewContext(W, H)

	dc.SetRGB255(18, 20, 28)
	dc.Clear()

	for y := 0; y < H; y++ {
		t := float64(y) / float64(H)
		r := int(18 + t*10)
		g := int(20 + t*8)
		b := int(28 + t*15)
		dc.SetRGB255(r, g, b)
		dc.DrawLine(0, float64(y), float64(W), float64(y))
		dc.Stroke()
	}

	n := len(f.Daily.Time)
	minT := f.Daily.TempMin[0]
	maxT := f.Daily.TempMax[0]
	maxP := 0.0
	for i := 0; i < n; i++ {
		if f.Daily.TempMin[i] < minT {
			minT = f.Daily.TempMin[i]
		}
		if f.Daily.TempMax[i] > maxT {
			maxT = f.Daily.TempMax[i]
		}
		if f.Daily.PrecipitationSum[i] > maxP {
			maxP = f.Daily.PrecipitationSum[i]
		}
	}
	if maxT == minT {
		maxT = minT + 1
	}
	if maxP < 1 {
		maxP = 1
	}
	rng := maxT - minT
	minT -= rng * 0.15
	maxT += rng * 0.20

	dc.SetRGB255(230, 232, 240)
	title := fmt.Sprintf("7-Day Forecast - %s", city)
	if country != "" {
		title += fmt.Sprintf(" (%s)", country)
	}
	dc.DrawStringAnchored(title, padL, 28, 0, 0.5)

	dc.SetRGB255(150, 155, 170)
	subtitle := "Temperature (°C) and Precipitation (mm)"
	dc.DrawStringAnchored(subtitle, padL, 48, 0, 0.5)

	chartW := float64(W) - padL - padR
	chartH := float64(H) - padT - padB

	dc.SetRGB255(45, 48, 60)
	dc.SetLineWidth(1)
	for i := 0; i <= 4; i++ {
		y := padT + chartH*float64(i)/4
		dc.DrawLine(padL, y, float64(W)-padR, y)
		dc.Stroke()
	}

	dc.SetRGB255(170, 175, 190)
	for i := 0; i <= 4; i++ {
		v := maxT - (maxT-minT)*float64(i)/4
		y := padT + chartH*float64(i)/4
		dc.DrawStringAnchored(fmt.Sprintf("%.0f°", v), padL-8, y, 1, 0.4)
	}

	dc.SetRGB255(120, 180, 230)
	for i := 0; i <= 4; i++ {
		v := maxP - maxP*float64(i)/4
		y := padT + chartH*float64(i)/4
		dc.DrawStringAnchored(fmt.Sprintf("%.1fmm", v), float64(W)-padR+8, y, 0, 0.4)
	}

	xAt := func(i int) float64 {
		if n == 1 {
			return padL + chartW/2
		}
		return padL + chartW*float64(i)/float64(n-1)
	}
	yT := func(v float64) float64 {
		return padT + chartH*(1-(v-minT)/(maxT-minT))
	}
	yP := func(v float64) float64 {
		return padT + chartH*(1-v/maxP)
	}

	barW := chartW / float64(n) * 0.45
	for i := 0; i < n; i++ {
		p := f.Daily.PrecipitationSum[i]
		if p <= 0 {
			continue
		}
		x := xAt(i) - barW/2
		y := yP(p)
		dc.SetRGBA255(80, 150, 220, 180)
		dc.DrawRectangle(x, y, barW, float64(H)-padB-y)
		dc.Fill()
	}

	dc.SetRGBA255(240, 130, 80, 60)
	dc.MoveTo(xAt(0), float64(H)-padB)
	for i := 0; i < n; i++ {
		dc.LineTo(xAt(i), yT(f.Daily.TempMax[i]))
	}
	dc.LineTo(xAt(n-1), float64(H)-padB)
	dc.ClosePath()
	dc.Fill()

	dc.SetRGB255(240, 130, 80)
	dc.SetLineWidth(3)
	for i := 0; i < n; i++ {
		x, y := xAt(i), yT(f.Daily.TempMax[i])
		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.Stroke()

	dc.SetRGB255(100, 180, 240)
	dc.SetLineWidth(3)
	for i := 0; i < n; i++ {
		x, y := xAt(i), yT(f.Daily.TempMin[i])
		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.Stroke()

	for i := 0; i < n; i++ {
		dc.SetRGB255(240, 130, 80)
		dc.DrawCircle(xAt(i), yT(f.Daily.TempMax[i]), 5)
		dc.Fill()
		dc.SetRGB255(18, 20, 28)
		dc.DrawCircle(xAt(i), yT(f.Daily.TempMax[i]), 2.5)
		dc.Fill()

		dc.SetRGB255(100, 180, 240)
		dc.DrawCircle(xAt(i), yT(f.Daily.TempMin[i]), 5)
		dc.Fill()
		dc.SetRGB255(18, 20, 28)
		dc.DrawCircle(xAt(i), yT(f.Daily.TempMin[i]), 2.5)
		dc.Fill()
	}

	for i := 0; i < n; i++ {
		dc.SetRGB255(240, 130, 80)
		dc.DrawStringAnchored(fmt.Sprintf("%.0f°", f.Daily.TempMax[i]), xAt(i), yT(f.Daily.TempMax[i])-14, 0.5, 0.5)
		dc.SetRGB255(100, 180, 240)
		dc.DrawStringAnchored(fmt.Sprintf("%.0f°", f.Daily.TempMin[i]), xAt(i), yT(f.Daily.TempMin[i])+18, 0.5, 0.5)
	}

	dc.SetRGB255(180, 185, 200)
	for i := 0; i < n; i++ {
		t, err := time.Parse("2006-01-02", f.Daily.Time[i])
		if err != nil {
			continue
		}
		label := t.Format("Mon")
		date := t.Format("Jan 02")
		dc.DrawStringAnchored(label, xAt(i), float64(H)-padB+18, 0.5, 0.5)
		dc.SetRGB255(130, 135, 150)
		dc.DrawStringAnchored(date, xAt(i), float64(H)-padB+34, 0.5, 0.5)
		dc.SetRGB255(180, 185, 200)
	}

	legendY := float64(H) - 20
	dc.SetRGB255(240, 130, 80)
	dc.DrawCircle(padL+8, legendY, 5)
	dc.Fill()
	dc.SetRGB255(200, 205, 220)
	dc.DrawStringAnchored("High", padL+20, legendY, 0, 0.5)

	dc.SetRGB255(100, 180, 240)
	dc.DrawCircle(padL+80, legendY, 5)
	dc.Fill()
	dc.SetRGB255(200, 205, 220)
	dc.DrawStringAnchored("Low", padL+92, legendY, 0, 0.5)

	dc.SetRGBA255(80, 150, 220, 180)
	dc.DrawRectangle(padL+140, legendY-6, 12, 12)
	dc.Fill()
	dc.SetRGB255(200, 205, 220)
	dc.DrawStringAnchored("Precipitation", padL+158, legendY, 0, 0.5)

	dc.SetRGB255(110, 115, 130)
	dc.DrawStringAnchored("open-meteo.com", float64(W)-padR, legendY, 1, 0.5)

	out := filepath.Join(os.TempDir(), fmt.Sprintf("weatherradar_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func weatherRadarSummary(f *weatherRadarForecastResp) (float64, float64, float64) {
	n := len(f.Daily.Time)
	avgHi := 0.0
	avgLo := 0.0
	totP := 0.0
	for i := 0; i < n; i++ {
		avgHi += f.Daily.TempMax[i]
		avgLo += f.Daily.TempMin[i]
		totP += f.Daily.PrecipitationSum[i]
	}
	if n > 0 {
		avgHi /= float64(n)
		avgLo /= float64(n)
	}
	return avgHi, avgLo, math.Round(totP*10) / 10
}

func WeatherRadarHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("usage: <code>/weatherradar &lt;city&gt;</code>\nexample: <code>/weatherradar tokyo</code>")
		return nil
	}

	status, _ := m.Reply("<code>fetching forecast for " + html.EscapeString(arg) + "...</code>")

	lat, lon, name, country, err := weatherRadarGeocode(arg)
	if err != nil {
		msg := "error: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	fc, err := weatherRadarFetch(lat, lon)
	if err != nil {
		msg := "error fetching forecast: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	out, err := weatherRadarRender(name, country, fc)
	if err != nil {
		msg := "error rendering: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	avgHi, avgLo, totP := weatherRadarSummary(fc)

	caption := fmt.Sprintf("<b>7-Day Weather Forecast</b>\n<b>Location:</b> <code>%s</code>",
		html.EscapeString(name))
	if country != "" {
		caption += fmt.Sprintf(", <code>%s</code>", html.EscapeString(country))
	}
	caption += fmt.Sprintf("\n<b>Avg High:</b> <code>%.1f°C</code>\n<b>Avg Low:</b> <code>%.1f°C</code>\n<b>Total Rain:</b> <code>%.1f mm</code>",
		avgHi, avgLo, totP)

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  caption,
		FileName: "weatherradar.png",
		MimeType: "image/png",
	})
	os.Remove(out)
	if merr != nil {
		m.Reply("error sending: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerWeatherRadarHandlers() {
	c := Client
	c.On("cmd:weatherradar", WeatherRadarHandler)
}

func init() { QueueHandlerRegistration(registerWeatherRadarHandlers) }
