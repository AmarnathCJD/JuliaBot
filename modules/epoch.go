package modules

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func EpochHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		now := time.Now()
		var sb strings.Builder
		sb.WriteString("<b>current unix timestamp</b>\n<code>")
		sb.WriteString(strconv.FormatInt(now.Unix(), 10))
		sb.WriteString("</code>\n<b>ms:</b> <code>")
		sb.WriteString(strconv.FormatInt(now.UnixMilli(), 10))
		sb.WriteString("</code>\n<b>UTC:</b> <code>")
		sb.WriteString(html.EscapeString(now.UTC().Format(time.RFC3339)))
		sb.WriteString("</code>\n<b>Local:</b> <code>")
		sb.WriteString(html.EscapeString(now.Local().Format(time.RFC3339)))
		sb.WriteString("</code>")
		m.Reply(sb.String())
		return nil
	}

	n, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		m.Reply("invalid unix timestamp: <code>" + html.EscapeString(arg) + "</code>")
		return nil
	}

	var t time.Time
	switch {
	case n >= 1_000_000_000_000_000:
		t = time.Unix(0, n*1000)
	case n >= 1_000_000_000_000:
		t = time.UnixMilli(n)
	default:
		t = time.Unix(n, 0)
	}

	now := time.Now()
	diff := now.Sub(t)
	rel := "in the future"
	if diff >= 0 {
		rel = "ago"
	} else {
		diff = -diff
	}

	var sb strings.Builder
	sb.WriteString("<b>unix:</b> <code>")
	sb.WriteString(strconv.FormatInt(n, 10))
	sb.WriteString("</code>\n<b>UTC:</b> <code>")
	sb.WriteString(html.EscapeString(t.UTC().Format(time.RFC3339)))
	sb.WriteString("</code>\n<b>Local:</b> <code>")
	sb.WriteString(html.EscapeString(t.Local().Format(time.RFC3339)))
	sb.WriteString("</code>\n<b>Weekday:</b> <code>")
	sb.WriteString(t.UTC().Weekday().String())
	sb.WriteString("</code>\n<b>Relative:</b> <code>")
	sb.WriteString(html.EscapeString(formatEpochDur(diff)))
	sb.WriteString(" ")
	sb.WriteString(rel)
	sb.WriteString("</code>")
	m.Reply(sb.String())
	return nil
}

func HumanTimeHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("usage: <code>/humantime &lt;RFC3339&gt;</code>\nexample: <code>/humantime 2026-06-20T10:30:00Z</code>")
		return nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
	}

	var t time.Time
	var parsed bool
	for _, l := range layouts {
		if v, err := time.Parse(l, arg); err == nil {
			t = v
			parsed = true
			break
		}
	}
	if !parsed {
		for _, l := range layouts {
			if v, err := time.ParseInLocation(l, arg, time.Local); err == nil {
				t = v
				parsed = true
				break
			}
		}
	}
	if !parsed {
		m.Reply("could not parse time: <code>" + html.EscapeString(arg) + "</code>\ntry RFC3339 like <code>2026-06-20T10:30:00Z</code>")
		return nil
	}

	var sb strings.Builder
	sb.WriteString("<b>parsed:</b> <code>")
	sb.WriteString(html.EscapeString(t.UTC().Format(time.RFC3339)))
	sb.WriteString("</code>\n<b>unix:</b> <code>")
	sb.WriteString(strconv.FormatInt(t.Unix(), 10))
	sb.WriteString("</code>\n<b>ms:</b> <code>")
	sb.WriteString(strconv.FormatInt(t.UnixMilli(), 10))
	sb.WriteString("</code>")
	m.Reply(sb.String())
	return nil
}

func AgeHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("usage: <code>/age &lt;YYYY-MM-DD&gt;</code>")
		return nil
	}

	layouts := []string{"2006-01-02", "02-01-2006", "02/01/2006", "2006/01/02", "01-02-2006"}
	var birth time.Time
	var parsed bool
	for _, l := range layouts {
		if v, err := time.Parse(l, arg); err == nil {
			birth = v
			parsed = true
			break
		}
	}
	if !parsed {
		m.Reply("invalid date: <code>" + html.EscapeString(arg) + "</code>\nexpected <code>YYYY-MM-DD</code>")
		return nil
	}

	now := time.Now().UTC()
	if birth.After(now) {
		m.Reply("date is in the future: <code>" + html.EscapeString(arg) + "</code>")
		return nil
	}

	years, months, days := diffYMD(birth, now)
	totalDays := int64(now.Sub(birth).Hours() / 24)
	totalHours := int64(now.Sub(birth).Hours())

	var sb strings.Builder
	sb.WriteString("<b>age</b>\n<b>from:</b> <code>")
	sb.WriteString(html.EscapeString(birth.Format("2006-01-02")))
	sb.WriteString("</code>\n<b>to:</b> <code>")
	sb.WriteString(html.EscapeString(now.Format("2006-01-02")))
	sb.WriteString("</code>\n\n<code>")
	sb.WriteString(fmt.Sprintf("%d years, %d months, %d days", years, months, days))
	sb.WriteString("</code>\n<b>total days:</b> <code>")
	sb.WriteString(strconv.FormatInt(totalDays, 10))
	sb.WriteString("</code>\n<b>total hours:</b> <code>")
	sb.WriteString(strconv.FormatInt(totalHours, 10))
	sb.WriteString("</code>")
	m.Reply(sb.String())
	return nil
}

func diffYMD(from, to time.Time) (int, int, int) {
	y := to.Year() - from.Year()
	mo := int(to.Month()) - int(from.Month())
	d := to.Day() - from.Day()
	if d < 0 {
		prev := to.AddDate(0, -1, 0)
		daysInPrev := time.Date(prev.Year(), prev.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		d += daysInPrev
		mo--
	}
	if mo < 0 {
		mo += 12
		y--
	}
	return y, mo, d
}

func formatEpochDur(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	if days < 365 {
		return fmt.Sprintf("%dd %dh", days, int(d.Hours())%24)
	}
	years := days / 365
	rem := days % 365
	return fmt.Sprintf("%dy %dd", years, rem)
}

func init() { QueueHandlerRegistration(registerEpochHandlers) }

func registerEpochHandlers() {
	c := Client
	c.On("cmd:epoch", EpochHandler)
	c.On("cmd:humantime", HumanTimeHandler)
	c.On("cmd:age", AgeHandler)
}
