package modules

import (
	"strings"
	"sync"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func extractText(m *tg.NewMessage) string {
	args := strings.TrimSpace(m.Args())
	if args != "" {
		return args
	}
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply != nil {
			return reply.Text()
		}
	}
	return ""
}

var chatStatsMu sync.Mutex

type chatStatsData struct {
	Users map[int64]int64
}

func chatStatsLoad(chatID int64) *chatStatsData {
	return &chatStatsData{Users: map[int64]int64{}}
}

func chatStatsSave(chatID int64, m *chatStatsData) {
}
