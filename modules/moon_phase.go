package modules

import (
	"fmt"
	"math"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func moonConwayPhase(t time.Time) (int, float64, float64) {
	y := t.Year()
	m := int(t.Month())
	d := t.Day()
	if m < 3 {
		y--
		m += 12
	}
	m++
	r := float64(y)*11.0 + 14.0 + float64(m)*0.597 + float64(d)
	r = math.Mod(r, 30.0)
	if r < 0 {
		r += 30.0
	}
	age := r
	frac := age / 29.530588853
	illum := (1 - math.Cos(2*math.Pi*frac)) / 2 * 100
	idx := int(math.Floor(frac*8+0.5)) % 8
	return idx, age, illum
}

func moonPhaseInfo(idx int) (string, string) {
	emojis := []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}
	names := []string{
		"New Moon",
		"Waxing Crescent",
		"First Quarter",
		"Waxing Gibbous",
		"Full Moon",
		"Waning Gibbous",
		"Last Quarter",
		"Waning Crescent",
	}
	if idx < 0 || idx >= 8 {
		idx = 0
	}
	return emojis[idx], names[idx]
}

func moonProgressBar(p float64) string {
	const width = 16
	filled := int(math.Round(p / 100.0 * float64(width)))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func MoonPhaseHandler(m *tg.NewMessage) error {
	now := time.Now().UTC()
	idx, age, illum := moonConwayPhase(now)
	emoji, name := moonPhaseInfo(idx)
	bar := moonProgressBar(illum)

	var sb strings.Builder
	sb.WriteString("<b>Moon Phase</b>\n\n")
	sb.WriteString(fmt.Sprintf("%s  <b>%s</b>\n\n", emoji, name))
	sb.WriteString(fmt.Sprintf("Illumination: <code>%s</code> <b>%.1f%%</b>\n", bar, illum))
	sb.WriteString(fmt.Sprintf("Lunar age: <code>%.1f</code> days\n", age))
	sb.WriteString(fmt.Sprintf("Cycle: <code>29.53</code> day synodic month\n\n"))
	sb.WriteString(fmt.Sprintf("<i>UTC: %s</i>", now.Format("2006-01-02 15:04")))

	_, err := m.Reply(sb.String())
	return err
}

func registerMoonPhaseHandlers() {
	c := Client
	c.On("cmd:moon", MoonPhaseHandler)
}

func init() { QueueHandlerRegistration(registerMoonPhaseHandlers) }
