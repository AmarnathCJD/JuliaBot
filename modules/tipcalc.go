package modules

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func parsePositiveFloat(s string) (float64, error) {
	cleaned := strings.TrimSpace(s)
	cleaned = strings.TrimPrefix(cleaned, "$")
	cleaned = strings.TrimSuffix(cleaned, "%")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	v, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, err
	}
	if v < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return v, nil
}

func formatMoney(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func TipHandler(m *tg.NewMessage) error {
	args := strings.Fields(m.Args())
	if len(args) < 2 {
		_, err := m.Reply("<b>Usage:</b> <code>/tip &lt;amount&gt; &lt;pct&gt; [people]</code>\n\n<b>Example:</b>\n<code>/tip 87.50 18 3</code>")
		return err
	}

	amount, err := parsePositiveFloat(args[0])
	if err != nil {
		_, e := m.Reply("Invalid amount: <code>" + html.EscapeString(args[0]) + "</code>")
		return e
	}

	pct, err := parsePositiveFloat(args[1])
	if err != nil {
		_, e := m.Reply("Invalid percent: <code>" + html.EscapeString(args[1]) + "</code>")
		return e
	}

	people := 1
	if len(args) >= 3 {
		p, err := strconv.Atoi(strings.TrimSpace(args[2]))
		if err != nil || p <= 0 {
			_, e := m.Reply("Invalid people count: <code>" + html.EscapeString(args[2]) + "</code>")
			return e
		}
		people = p
	}

	tip := amount * pct / 100
	total := amount + tip
	perPerson := total / float64(people)
	tipPerPerson := tip / float64(people)
	billPerPerson := amount / float64(people)

	var sb strings.Builder
	sb.WriteString("<b>Tip Calculator</b>\n\n")
	sb.WriteString(fmt.Sprintf("Bill:   <code>%s</code>\n", formatMoney(amount)))
	sb.WriteString(fmt.Sprintf("Tip:    <code>%s</code> <i>(%s%%)</i>\n", formatMoney(tip), strconv.FormatFloat(pct, 'f', -1, 64)))
	sb.WriteString(fmt.Sprintf("Total:  <b><code>%s</code></b>\n", formatMoney(total)))

	if people > 1 {
		sb.WriteString(fmt.Sprintf("\n<b>Split between %d people:</b>\n", people))
		sb.WriteString(fmt.Sprintf("Each pays:  <b><code>%s</code></b>\n", formatMoney(perPerson)))
		sb.WriteString(fmt.Sprintf("  - bill:   <code>%s</code>\n", formatMoney(billPerPerson)))
		sb.WriteString(fmt.Sprintf("  - tip:    <code>%s</code>\n", formatMoney(tipPerPerson)))
	}

	_, err = m.Reply(sb.String())
	return err
}

func TaxHandler(m *tg.NewMessage) error {
	args := strings.Fields(m.Args())
	if len(args) < 2 {
		_, err := m.Reply("<b>Usage:</b> <code>/tax &lt;amount&gt; &lt;pct&gt;</code>\n\n<b>Example:</b>\n<code>/tax 200 7.25</code>")
		return err
	}

	amount, err := parsePositiveFloat(args[0])
	if err != nil {
		_, e := m.Reply("Invalid amount: <code>" + html.EscapeString(args[0]) + "</code>")
		return e
	}

	pct, err := parsePositiveFloat(args[1])
	if err != nil {
		_, e := m.Reply("Invalid percent: <code>" + html.EscapeString(args[1]) + "</code>")
		return e
	}

	tax := amount * pct / 100
	total := amount + tax

	reply := fmt.Sprintf("<b>Tax Calculator</b>\n\nSubtotal: <code>%s</code>\nTax:      <code>%s</code> <i>(%s%%)</i>\nTotal:    <b><code>%s</code></b>",
		formatMoney(amount),
		formatMoney(tax),
		strconv.FormatFloat(pct, 'f', -1, 64),
		formatMoney(total))

	_, err = m.Reply(reply)
	return err
}

func TipChartHandler(m *tg.NewMessage) error {
	args := strings.Fields(m.Args())
	if len(args) < 1 {
		_, err := m.Reply("<b>Usage:</b> <code>/tipchart &lt;amount&gt;</code>\n\n<b>Example:</b>\n<code>/tipchart 50</code>")
		return err
	}

	amount, err := parsePositiveFloat(args[0])
	if err != nil {
		_, e := m.Reply("Invalid amount: <code>" + html.EscapeString(args[0]) + "</code>")
		return e
	}

	percents := []float64{10, 15, 18, 20, 25}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Tip Chart for </b><code>%s</code>\n\n", formatMoney(amount)))
	sb.WriteString("<pre>")
	sb.WriteString(fmt.Sprintf("%-6s %-10s %-10s\n", "Pct", "Tip", "Total"))
	sb.WriteString("------------------------------\n")
	for _, p := range percents {
		tip := amount * p / 100
		total := amount + tip
		sb.WriteString(fmt.Sprintf("%-6s %-10s %-10s\n",
			strconv.FormatFloat(p, 'f', -1, 64)+"%",
			formatMoney(tip),
			formatMoney(total)))
	}
	sb.WriteString("</pre>")

	_, err = m.Reply(sb.String())
	return err
}

func registerTipCalcHandlers() {
	c := Client
	c.On("cmd:tip", TipHandler)
	c.On("cmd:tax", TaxHandler)
	c.On("cmd:tipchart", TipChartHandler)
}

func init() {
	QueueHandlerRegistration(registerTipCalcHandlers)
}
