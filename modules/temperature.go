package modules

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func tempParseInput(m *tg.NewMessage) (float64, bool) {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	if text == "" {
		return 0, false
	}
	fields := strings.Fields(text)
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func tempReply(m *tg.NewMessage, from string, fromVal float64, to string, toVal float64) {
	msg := fmt.Sprintf("<b>%s</b> %s = <b>%s</b> %s",
		html.EscapeString(strconv.FormatFloat(fromVal, 'f', -1, 64)),
		from,
		html.EscapeString(strconv.FormatFloat(toVal, 'f', 2, 64)),
		to,
	)
	m.Reply(msg)
}

func C2FHandler(m *tg.NewMessage) error {
	v, ok := tempParseInput(m)
	if !ok {
		m.Reply("usage: /c2f &lt;celsius&gt;")
		return nil
	}
	tempReply(m, "°C", v, "°F", v*9/5+32)
	return nil
}

func F2CHandler(m *tg.NewMessage) error {
	v, ok := tempParseInput(m)
	if !ok {
		m.Reply("usage: /f2c &lt;fahrenheit&gt;")
		return nil
	}
	tempReply(m, "°F", v, "°C", (v-32)*5/9)
	return nil
}

func K2CHandler(m *tg.NewMessage) error {
	v, ok := tempParseInput(m)
	if !ok {
		m.Reply("usage: /k2c &lt;kelvin&gt;")
		return nil
	}
	tempReply(m, "K", v, "°C", v-273.15)
	return nil
}

func C2KHandler(m *tg.NewMessage) error {
	v, ok := tempParseInput(m)
	if !ok {
		m.Reply("usage: /c2k &lt;celsius&gt;")
		return nil
	}
	tempReply(m, "°C", v, "K", v+273.15)
	return nil
}

func K2FHandler(m *tg.NewMessage) error {
	v, ok := tempParseInput(m)
	if !ok {
		m.Reply("usage: /k2f &lt;kelvin&gt;")
		return nil
	}
	tempReply(m, "K", v, "°F", (v-273.15)*9/5+32)
	return nil
}

func F2KHandler(m *tg.NewMessage) error {
	v, ok := tempParseInput(m)
	if !ok {
		m.Reply("usage: /f2k &lt;fahrenheit&gt;")
		return nil
	}
	tempReply(m, "°F", v, "K", (v-32)*5/9+273.15)
	return nil
}

func init() { QueueHandlerRegistration(registerTemperatureHandlers) }
func registerTemperatureHandlers() {
	c := Client
	c.On("cmd:c2f", C2FHandler)
	c.On("cmd:f2c", F2CHandler)
	c.On("cmd:k2c", K2CHandler)
	c.On("cmd:c2k", C2KHandler)
	c.On("cmd:k2f", K2FHandler)
	c.On("cmd:f2k", F2KHandler)
}
