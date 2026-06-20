package modules

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func birthdayCollisionProbability(n int) float64 {
	if n <= 1 {
		return 0
	}
	if n >= 366 {
		return 1
	}
	logNoMatch := 0.0
	for i := 1; i < n; i++ {
		logNoMatch += math.Log(float64(365-i)) - math.Log(365)
	}
	return 1 - math.Exp(logNoMatch)
}

func birthdayProgressBar(p float64) string {
	const width = 20
	filled := int(math.Round(p * float64(width)))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func birthdayInterpretation(n int, p float64) string {
	switch {
	case n <= 1:
		return "A single person cannot share a birthday with anyone."
	case n >= 366:
		return "By the pigeonhole principle, a collision is guaranteed with 366+ people."
	case p < 0.05:
		return "Very unlikely — most groups this size will have all unique birthdays."
	case p < 0.25:
		return "Unlikely, but possible. A small group like this rarely produces a collision."
	case p < 0.5:
		return "Coin-flip territory is approaching. The intuition that you need ~180 people is wrong."
	case p < 0.75:
		return "More likely than not. The famous 50% threshold is hit at just 23 people."
	case p < 0.95:
		return "Highly likely — a collision is the expected outcome here."
	default:
		return "Almost certain. Finding a group this size without a shared birthday would be remarkable."
	}
}

func BirthdayParadoxHandler(m *tg.NewMessage) error {
	args := strings.Fields(m.Args())
	if len(args) < 1 {
		_, err := m.Reply("<b>Usage:</b> <code>/bdayparadox &lt;N&gt;</code>\n\n<b>Example:</b>\n<code>/bdayparadox 23</code>\n\nCalculates the probability that at least two people in a group of N share a birthday.")
		return err
	}

	raw := strings.TrimSpace(args[0])
	n, err := strconv.Atoi(raw)
	if err != nil {
		_, e := m.Reply("Invalid N: <code>" + html.EscapeString(raw) + "</code>\nProvide a positive integer.")
		return e
	}
	if n < 1 {
		_, e := m.Reply("N must be at least 1.")
		return e
	}
	if n > 10000 {
		_, e := m.Reply("N too large. Try a value between 1 and 10000.")
		return e
	}

	p := birthdayCollisionProbability(n)
	pNoMatch := 1 - p
	bar := birthdayProgressBar(p)
	interp := birthdayInterpretation(n, p)

	var sb strings.Builder
	sb.WriteString("<b>Birthday Paradox</b>\n\n")
	sb.WriteString(fmt.Sprintf("Group size: <b>%d</b> %s\n", n, pluralize(n, "person", "people")))
	sb.WriteString(fmt.Sprintf("Days in year: <code>365</code> <i>(ignoring leap years)</i>\n\n"))
	sb.WriteString("<b>P(at least one shared birthday)</b>\n")
	sb.WriteString(fmt.Sprintf("<code>%s</code> <b>%.4f%%</b>\n\n", bar, p*100))
	sb.WriteString(fmt.Sprintf("P(all unique): <code>%.4f%%</code>\n", pNoMatch*100))
	sb.WriteString(fmt.Sprintf("Odds: <code>%s</code>\n\n", formatOdds(p)))
	sb.WriteString("<b>Formula</b>\n")
	sb.WriteString("<pre>P = 1 - (365! / (365-N)!) / 365^N</pre>\n")
	sb.WriteString("<b>Interpretation</b>\n")
	sb.WriteString("<i>" + html.EscapeString(interp) + "</i>")

	_, err = m.Reply(sb.String())
	return err
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func formatOdds(p float64) string {
	if p <= 0 {
		return "0 : 1"
	}
	if p >= 1 {
		return "1 : 0"
	}
	if p >= 0.5 {
		ratio := p / (1 - p)
		return fmt.Sprintf("%.2f : 1 in favor", ratio)
	}
	ratio := (1 - p) / p
	return fmt.Sprintf("1 : %.2f against", ratio)
}

func registerBirthdayParadoxHandlers() {
	c := Client
	c.On("cmd:bdayparadox", BirthdayParadoxHandler)
}

func init() {
	QueueHandlerRegistration(registerBirthdayParadoxHandlers)
}
