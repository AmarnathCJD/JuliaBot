package modules

import (
	"fmt"
	"html"
	"strings"
	"unicode"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func MorseSignalHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			args = strings.TrimSpace(r.Text())
		}
	}
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/morsesignal &lt;text&gt;</code>")
		return nil
	}
	upper := strings.ToUpper(args)
	words := strings.Fields(upper)
	if len(words) == 0 {
		m.Reply("<b>Error:</b> nothing convertible (A-Z, 0-9 only).")
		return nil
	}
	signalWords := make([]string, 0, len(words))
	skipped := 0
	for _, w := range words {
		letterSignals := make([]string, 0, len(w))
		for _, r := range w {
			if unicode.IsSpace(r) {
				continue
			}
			code, ok := morseTable[r]
			if !ok {
				skipped++
				continue
			}
			var sb strings.Builder
			for _, c := range code {
				if c == '.' {
					sb.WriteString("█")
				} else if c == '-' {
					sb.WriteString("███")
				}
				sb.WriteString("░")
			}
			letterSignals = append(letterSignals, strings.TrimRight(sb.String(), "░"))
		}
		if len(letterSignals) > 0 {
			signalWords = append(signalWords, strings.Join(letterSignals, "░░░"))
		}
	}
	if len(signalWords) == 0 {
		m.Reply("<b>Error:</b> nothing convertible (A-Z, 0-9 only).")
		return nil
	}
	signal := strings.Join(signalWords, "░░░░░░░")
	if len(signal) > 3500 {
		signal = signal[:3500] + "..."
	}
	note := ""
	if skipped > 0 {
		note = fmt.Sprintf("\n\n<i>Skipped %d unsupported character(s).</i>", skipped)
	}
	out := fmt.Sprintf("<b>Morse Signal</b>\n\n<b>Input:</b> <code>%s</code>\n<b>Legend:</b> <code>█</code>=on, <code>░</code>=off\n<pre>%s</pre>%s",
		html.EscapeString(args),
		signal,
		note,
	)
	m.Reply(out)
	return nil
}

func registerMorseSignalHandlers() {
	c := Client
	c.On("cmd:morsesignal", MorseSignalHandler)
}

func init() {
	QueueHandlerRegistration(registerMorseSignalHandlers)
}
