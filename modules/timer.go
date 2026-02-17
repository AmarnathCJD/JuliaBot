package modules

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

type timerData struct {
	chatID   int64
	userID   int64
	message  string
	media    telegram.MessageMedia
	client   *telegram.Client
	duration time.Duration
}

var (
	activeTimers   = make(map[string]*timerData)
	activeTimersMu sync.RWMutex
)

func SetTimerHandler(m *telegram.NewMessage) error {
	args := m.Args()
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/timer &lt;duration&gt; &lt;message&gt;</code>\n<b>Example:</b> <code>/timer 1h30m Take a break!</code>\n\n<i>Reply to media to include it in the reminder</i>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	message := ""
	if len(parts) >= 2 {
		message = parts[1]
	}

	duration, err := parseDuration(parts[0])
	if err != nil {
		m.Reply("<b>Error:</b> " + err.Error())
		return nil
	}

	timer := &timerData{
		chatID:   m.ChatID(),
		userID:   m.SenderID(),
		message:  message,
		client:   m.Client,
		duration: duration,
	}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply.IsMedia() {
			timer.media = reply.Media()
		}
	}

	timerID := fmt.Sprintf("%d_%d_%d", m.ChatID(), m.SenderID(), time.Now().UnixNano())

	activeTimersMu.Lock()
	activeTimers[timerID] = timer
	activeTimersMu.Unlock()

	time.AfterFunc(duration, func() {
		sendTimerNotification(timerID)
	})

	m.Reply(fmt.Sprintf("Timer set for <b>%s</b>", formatDuration(duration)))
	return nil
}

func sendTimerNotification(timerID string) {
	activeTimersMu.RLock()
	timer, exists := activeTimers[timerID]
	activeTimersMu.RUnlock()

	if !exists {
		return
	}

	text := "<b>Timer Alert!</b>"
	if timer.message != "" {
		text += "\n" + timer.message
	}

	snoozeBtn := telegram.Button.Data("Snooze 5m", "snooze_"+timerID)
	dismissBtn := telegram.Button.Data("Dismiss", "dismiss_"+timerID)
	keyboard := telegram.NewKeyboard().AddRow(snoozeBtn).AddRow(dismissBtn).Build()

	if timer.media != nil {
		timer.client.SendMedia(timer.chatID, timer.media, &telegram.MediaOptions{
			Caption:     text,
			ReplyMarkup: keyboard,
		})
	} else {
		timer.client.SendMessage(timer.chatID, text, &telegram.SendOptions{
			ReplyMarkup: keyboard,
		})
	}
}

func TimerCallbackHandler(cb *telegram.CallbackQuery) error {
	data := cb.DataString()

	var action, timerID string
	if strings.HasPrefix(data, "snooze_") {
		action = "snooze"
		timerID = strings.TrimPrefix(data, "snooze_")
	} else if strings.HasPrefix(data, "dismiss_") {
		action = "dismiss"
		timerID = strings.TrimPrefix(data, "dismiss_")
	} else {
		return nil
	}

	activeTimersMu.RLock()
	timer, exists := activeTimers[timerID]
	activeTimersMu.RUnlock()

	if !exists {
		cb.Answer("Timer expired", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	if cb.Sender.ID != timer.userID {
		cb.Answer("Only the timer setter can do this!", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	switch action {
	case "snooze":
		snoozeDuration := 5 * time.Minute
		time.AfterFunc(snoozeDuration, func() {
			sendTimerNotification(timerID)
		})
		cb.Edit("<b>Snoozed for 5 minutes</b>")
		cb.Answer("Snoozed!")

	case "dismiss":
		activeTimersMu.Lock()
		delete(activeTimers, timerID)
		activeTimersMu.Unlock()
		cb.Edit("<b>Timer dismissed</b>")
		cb.Answer("Dismissed!")
	}

	return nil
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	var total time.Duration
	var numBuf strings.Builder

	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			numBuf.WriteRune(r)
		case r == 'd' || r == 'w':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("invalid duration: missing number before '%c'", r)
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			if r == 'd' {
				total += time.Duration(n) * 24 * time.Hour
			} else {
				total += time.Duration(n) * 7 * 24 * time.Hour
			}
		case r == 'h' || r == 'm' || r == 's':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("invalid duration: missing number before '%c'", r)
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			switch r {
			case 'h':
				total += time.Duration(n) * time.Hour
			case 'm':
				total += time.Duration(n) * time.Minute
			case 's':
				total += time.Duration(n) * time.Second
			}
		default:
			return 0, fmt.Errorf("invalid character '%c' in duration", r)
		}
	}

	if numBuf.Len() > 0 {
		n, _ := strconv.Atoi(numBuf.String())
		total += time.Duration(n) * time.Second
	}

	if total <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}

	return total, nil
}

func formatDuration(d time.Duration) string {
	var parts []string

	if days := d / (24 * time.Hour); days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
		d -= days * 24 * time.Hour
	}
	if hours := d / time.Hour; hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
		d -= hours * time.Hour
	}
	if mins := d / time.Minute; mins > 0 {
		parts = append(parts, fmt.Sprintf("%dm", mins))
		d -= mins * time.Minute
	}
	if secs := d / time.Second; secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}

	return strings.Join(parts, " ")
}

func registerTimerHandlers() {
	c := Client
	c.On("command:timer", SetTimerHandler)
	c.On("callback:snooze_", TimerCallbackHandler)
	c.On("callback:dismiss_", TimerCallbackHandler)
}

func init() {
	QueueHandlerRegistration(registerTimerHandlers)
}
