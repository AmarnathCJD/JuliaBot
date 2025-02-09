package modules

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

type TimerService struct {
	timeToWait int
	message    string
	channelID  int64
	client     *telegram.Client
}

func (t *TimerService) startTimer() {
	timer := time.NewTimer(time.Duration(t.timeToWait) * time.Second)
	<-timer.C

	t.client.SendMessage(t.channelID, t.message)
}

func SetTimerHandler(m *telegram.NewMessage) error {
	if m.Args() == "" {
		m.Reply("Please provide the time to wait and the message")
		return nil
	}

	args := strings.Split(m.Args(), " ")
	if len(args) < 2 {
		m.Reply("Invalid arguments. Please provide the time to wait and the message")
		return nil
	}

	// time can be like 400 -> 400s, 10m, 1h etc etc 1h10m20s, etc
	timeToWait, err := parseTime(args[0])
	if err != nil {
		m.Reply("Invalid time format")
		return nil
	}

	message := strings.Join(args[1:], " ")

	timer := TimerService{
		timeToWait: timeToWait,
		message:    message,
		channelID:  m.ChatID(),
		client:     m.Client,
	}

	go timer.startTimer()
	m.Reply(fmt.Sprintf("Timer set for %s", time.Duration(timeToWait)*time.Second))

	return nil
}

func parseTime(timeStr string) (int, error) {
	var timeToWait int

	timeUnits := map[string]int{
		"s": 1,
		"m": 60,
		"h": 3600,
		"d": 24 * 3600,
		"w": 7 * 24 * 3600,
	}

	timeParts := strings.FieldsFunc(timeStr, func(r rune) bool {
		return r >= 'a' && r <= 'z'
	})

	suffixIndex := 0
	for _, part := range timeParts {
		if suffixIndex >= len(timeStr) {
			return 0, fmt.Errorf("invalid time format")
		}

		for suffixIndex < len(timeStr) && (timeStr[suffixIndex] < 'a' || timeStr[suffixIndex] > 'z') {
			suffixIndex++
		}

		if suffixIndex >= len(timeStr) {
			return 0, fmt.Errorf("invalid time format")
		}

		suffix := string(timeStr[suffixIndex])
		if multiplier, exists := timeUnits[suffix]; exists {
			value, err := strconv.Atoi(part)
			if err != nil {
				return 0, err
			}

			timeToWait += value * multiplier
			suffixIndex++
		} else {
			return 0, fmt.Errorf("invalid time unit: %s", suffix)
		}
	}

	return timeToWait, nil
}
