package modules

import (
	"fmt"
	"html"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

type simDieTerm struct {
	Count int
	Sides int
}

type simExpr struct {
	Terms    []simDieTerm
	Modifier int
	Raw      string
}

var simTermRe = regexp.MustCompile(`(?i)^([+-]?)(\d*)d(\d+)$`)
var simModRe = regexp.MustCompile(`^([+-]?)(\d+)$`)

func simParseExpr(s string) (*simExpr, error) {
	clean := strings.ReplaceAll(strings.ToLower(s), " ", "")
	if clean == "" {
		return nil, fmt.Errorf("empty expression")
	}
	clean = strings.ReplaceAll(clean, "-", "+-")
	if strings.HasPrefix(clean, "+") {
		clean = clean[1:]
	}
	parts := strings.Split(clean, "+")
	expr := &simExpr{Raw: s}
	for _, p := range parts {
		if p == "" {
			continue
		}
		if mm := simTermRe.FindStringSubmatch(p); mm != nil {
			sign := 1
			if mm[1] == "-" {
				sign = -1
			}
			cnt := 1
			if mm[2] != "" {
				c, err := strconv.Atoi(mm[2])
				if err != nil {
					return nil, fmt.Errorf("bad count %q", mm[2])
				}
				cnt = c
			}
			sides, err := strconv.Atoi(mm[3])
			if err != nil {
				return nil, fmt.Errorf("bad sides %q", mm[3])
			}
			if cnt < 1 || cnt > 1000 {
				return nil, fmt.Errorf("dice count must be 1..1000")
			}
			if sides < 2 || sides > 1000 {
				return nil, fmt.Errorf("dice sides must be 2..1000")
			}
			expr.Terms = append(expr.Terms, simDieTerm{Count: sign * cnt, Sides: sides})
			continue
		}
		if mm := simModRe.FindStringSubmatch(p); mm != nil {
			sign := 1
			if mm[1] == "-" {
				sign = -1
			}
			v, err := strconv.Atoi(mm[2])
			if err != nil {
				return nil, fmt.Errorf("bad modifier %q", p)
			}
			expr.Modifier += sign * v
			continue
		}
		return nil, fmt.Errorf("unrecognized token %q", p)
	}
	if len(expr.Terms) == 0 {
		return nil, fmt.Errorf("no dice terms found (try '3d6' or '2d20+5')")
	}
	return expr, nil
}

func simRollOnce(e *simExpr, rng *rand.Rand) int {
	total := e.Modifier
	for _, t := range e.Terms {
		sign := 1
		count := t.Count
		if count < 0 {
			sign = -1
			count = -count
		}
		for i := 0; i < count; i++ {
			total += sign * (rng.Intn(t.Sides) + 1)
		}
	}
	return total
}

func simTheoreticalRange(e *simExpr) (int, int) {
	minV := e.Modifier
	maxV := e.Modifier
	for _, t := range e.Terms {
		count := t.Count
		if count >= 0 {
			minV += count * 1
			maxV += count * t.Sides
		} else {
			minV += count * t.Sides
			maxV += count * 1
		}
	}
	return minV, maxV
}

func simFormatExpr(e *simExpr) string {
	var sb strings.Builder
	for i, t := range e.Terms {
		c := t.Count
		if i == 0 {
			if c < 0 {
				sb.WriteString("-")
				c = -c
			}
		} else {
			if c < 0 {
				sb.WriteString(" - ")
				c = -c
			} else {
				sb.WriteString(" + ")
			}
		}
		sb.WriteString(fmt.Sprintf("%dd%d", c, t.Sides))
	}
	if e.Modifier > 0 {
		sb.WriteString(fmt.Sprintf(" + %d", e.Modifier))
	} else if e.Modifier < 0 {
		sb.WriteString(fmt.Sprintf(" - %d", -e.Modifier))
	}
	return sb.String()
}

func simRenderHistogram(expr *simExpr, results []int, counts map[int]int, n int, mean, stddev float64, minR, maxR, mode, modeCount int) (string, error) {
	const W, H = 760, 420
	const padL, padR, padT, padB = 64.0, 24.0, 70.0, 60.0
	dc := gg.NewContext(W, H)

	dc.SetRGB255(20, 22, 30)
	dc.Clear()

	theoMin, theoMax := simTheoreticalRange(expr)
	lo := theoMin
	hi := theoMax
	if minR < lo {
		lo = minR
	}
	if maxR > hi {
		hi = maxR
	}
	if hi <= lo {
		hi = lo + 1
	}

	maxCount := 1
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	plotW := float64(W) - padL - padR
	plotH := float64(H) - padT - padB
	span := hi - lo + 1

	maxBins := 60
	binSize := 1
	if span > maxBins {
		binSize = int(math.Ceil(float64(span) / float64(maxBins)))
	}
	nBins := (span + binSize - 1) / binSize
	if nBins < 1 {
		nBins = 1
	}

	bins := make([]int, nBins)
	for v, c := range counts {
		idx := (v - lo) / binSize
		if idx < 0 {
			idx = 0
		}
		if idx >= nBins {
			idx = nBins - 1
		}
		bins[idx] += c
	}

	maxBin := 1
	for _, b := range bins {
		if b > maxBin {
			maxBin = b
		}
	}

	dc.SetRGB255(45, 48, 60)
	dc.SetLineWidth(1)
	for i := 0; i <= 4; i++ {
		y := padT + plotH*float64(i)/4
		dc.DrawLine(padL, y, float64(W)-padR, y)
		dc.Stroke()
	}

	dc.SetRGB255(170, 175, 190)
	for i := 0; i <= 4; i++ {
		v := float64(maxBin) * (1 - float64(i)/4)
		y := padT + plotH*float64(i)/4
		label := fmt.Sprintf("%.0f", v)
		dc.DrawStringAnchored(label, padL-8, y, 1, 0.4)
	}

	barW := plotW / float64(nBins)
	gap := 0.0
	if barW > 4 {
		gap = math.Min(2, barW*0.15)
	}

	for i, b := range bins {
		h := float64(b) / float64(maxBin) * plotH
		x := padL + float64(i)*barW
		y := padT + plotH - h
		ratio := float64(b) / float64(maxBin)
		r := uint8(80 + 120*ratio)
		g := uint8(120 + 80*(1-ratio))
		bb := uint8(220 - 80*ratio)
		dc.SetRGB255(int(r), int(g), int(bb))
		dc.DrawRectangle(x+gap/2, y, barW-gap, h)
		dc.Fill()
	}

	meanX := padL + (mean-float64(lo))/float64(span)*plotW
	if meanX >= padL && meanX <= float64(W)-padR {
		dc.SetRGBA255(255, 220, 90, 220)
		dc.SetLineWidth(1.5)
		dc.SetDash(6, 4)
		dc.DrawLine(meanX, padT, meanX, padT+plotH)
		dc.Stroke()
		dc.SetDash()
		dc.DrawStringAnchored(fmt.Sprintf("mean %.2f", mean), meanX, padT-6, 0.5, 1)
	}

	dc.SetRGB255(230, 232, 240)
	title := fmt.Sprintf("Monte Carlo: %s  (n=%s)", simFormatExpr(expr), simFormatInt(n))
	dc.DrawStringAnchored(title, padL, 22, 0, 0.5)

	dc.SetRGB255(170, 175, 190)
	sub := fmt.Sprintf("min=%d  max=%d  mean=%.2f  sd=%.2f  mode=%d", minR, maxR, mean, stddev, mode)
	dc.DrawStringAnchored(sub, padL, 42, 0, 0.5)

	dc.SetRGB255(120, 125, 140)
	xTicks := 6
	if nBins < xTicks {
		xTicks = nBins
	}
	for i := 0; i <= xTicks; i++ {
		v := lo + (span-1)*i/xTicks
		x := padL + plotW*float64(i)/float64(xTicks)
		dc.DrawStringAnchored(strconv.Itoa(v), x, padT+plotH+18, 0.5, 0.5)
	}

	dc.SetRGB255(140, 145, 160)
	dc.DrawStringAnchored("outcome", padL+plotW/2, float64(H)-padB+44, 0.5, 0.5)
	dc.DrawStringAnchored(fmt.Sprintf("range [%d, %d]  bin=%d  bins=%d  modeHits=%s",
		theoMin, theoMax, binSize, nBins, simFormatInt(modeCount)),
		float64(W)-padR, float64(H)-12, 1, 0.5)

	out := filepath.Join(os.TempDir(), fmt.Sprintf("dicesim_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	_ = results
	return out, nil
}

func simFormatInt(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		return "-" + simFormatInt(-n)
	}
	var sb strings.Builder
	pre := len(s) % 3
	for i, c := range s {
		if i != 0 && (i-pre)%3 == 0 {
			sb.WriteByte(',')
		}
		sb.WriteRune(c)
	}
	return sb.String()
}

func simPercentile(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Round(p * float64(len(sorted)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func RandomDiceSimsHandler(m *tg.NewMessage) error {
	raw := strings.TrimSpace(m.Args())
	if raw == "" {
		_, err := m.Reply("<b>Monte Carlo Dice Simulator</b>\n" +
			"<code>/sim &lt;N&gt; &lt;expr&gt;</code>\n\n" +
			"<b>Examples:</b>\n" +
			"<code>/sim 1000 3d6</code>\n" +
			"<code>/sim 5000 2d20+5</code>\n" +
			"<code>/sim 2000 4d6-1d4+2</code>\n\n" +
			"Renders a histogram of outcomes plus mean, stddev, min, max, mode, and percentiles.")
		return err
	}

	fields := strings.Fields(raw)
	if len(fields) < 2 {
		_, err := m.Reply("Usage: <code>/sim &lt;N&gt; &lt;expr&gt;</code>\nExample: <code>/sim 1000 3d6</code>")
		return err
	}

	n, err := strconv.Atoi(fields[0])
	if err != nil || n < 1 {
		_, e := m.Reply("First arg must be a positive integer N (number of rolls). Got <code>" + html.EscapeString(fields[0]) + "</code>.")
		return e
	}
	if n > 1000000 {
		_, e := m.Reply("N too large. Max is <code>1,000,000</code>.")
		return e
	}

	exprStr := strings.Join(fields[1:], "")
	expr, err := simParseExpr(exprStr)
	if err != nil {
		_, e := m.Reply("Bad expression: " + html.EscapeString(err.Error()) + "\nExample: <code>3d6</code> or <code>2d20+5</code>.")
		return e
	}

	status, _ := m.Reply(fmt.Sprintf("<code>rolling %s × %s...</code>", html.EscapeString(simFormatExpr(expr)), simFormatInt(n)))

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	t0 := time.Now()
	results := make([]int, n)
	counts := make(map[int]int)
	minR := math.MaxInt
	maxR := math.MinInt
	var sum, sumSq float64
	for i := 0; i < n; i++ {
		v := simRollOnce(expr, rng)
		results[i] = v
		counts[v]++
		if v < minR {
			minR = v
		}
		if v > maxR {
			maxR = v
		}
		sum += float64(v)
		sumSq += float64(v) * float64(v)
	}
	elapsed := time.Since(t0)
	mean := sum / float64(n)
	variance := sumSq/float64(n) - mean*mean
	if variance < 0 {
		variance = 0
	}
	stddev := math.Sqrt(variance)

	mode := minR
	modeCount := 0
	for v, c := range counts {
		if c > modeCount || (c == modeCount && v < mode) {
			mode = v
			modeCount = c
		}
	}

	sorted := make([]int, len(results))
	copy(sorted, results)
	sort.Ints(sorted)
	p05 := simPercentile(sorted, 0.05)
	p25 := simPercentile(sorted, 0.25)
	p50 := simPercentile(sorted, 0.50)
	p75 := simPercentile(sorted, 0.75)
	p95 := simPercentile(sorted, 0.95)

	out, rerr := simRenderHistogram(expr, results, counts, n, mean, stddev, minR, maxR, mode, modeCount)
	if rerr != nil {
		msg := "render error: " + html.EscapeString(rerr.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	theoMin, theoMax := simTheoreticalRange(expr)
	theoMean := float64(expr.Modifier)
	for _, t := range expr.Terms {
		theoMean += float64(t.Count) * (float64(t.Sides) + 1) / 2
	}

	var sb strings.Builder
	sb.WriteString("<b>Monte Carlo Dice Simulator</b>\n")
	sb.WriteString(fmt.Sprintf("<b>Expression:</b> <code>%s</code>\n", html.EscapeString(simFormatExpr(expr))))
	sb.WriteString(fmt.Sprintf("<b>Rolls:</b> <code>%s</code>  <b>Time:</b> <code>%s</code>\n\n", simFormatInt(n), elapsed.Round(time.Microsecond)))
	sb.WriteString("<b>Stats</b>\n")
	sb.WriteString(fmt.Sprintf("<pre>min     %d\nmax     %d\nmean    %.4f\nstddev  %.4f\nmode    %d (%s hits)\n\np05     %d\np25     %d\np50     %d\np75     %d\np95     %d</pre>\n",
		minR, maxR, mean, stddev, mode, simFormatInt(modeCount),
		p05, p25, p50, p75, p95))
	sb.WriteString(fmt.Sprintf("<b>Theoretical:</b> range <code>[%d, %d]</code>, mean <code>%.4f</code>\n", theoMin, theoMax, theoMean))
	sb.WriteString(fmt.Sprintf("<b>Deviation from theory:</b> <code>%+.4f</code>", mean-theoMean))

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  sb.String(),
		FileName: "dicesim.png",
		MimeType: "image/png",
	})
	os.Remove(out)
	if merr != nil {
		m.Reply("error sending: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerRandomDiceSimsHandlers() {
	c := Client
	c.On("cmd:sim", RandomDiceSimsHandler)
}

func init() { QueueHandlerRegistration(registerRandomDiceSimsHandlers) }
