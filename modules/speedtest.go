package modules

import (
	"fmt"
	"io"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func SpeedHandler(m *tg.NewMessage) error {
	msg, _ := m.Reply("<b>Running speed test...</b>")
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", "https://speed.cloudflare.com/__down?bytes=10000000", nil)
	if err != nil {
		if msg != nil {
			msg.Edit(fmt.Sprintf("<b>Speed test failed:</b> <code>%s</code>", err.Error()))
		}
		return nil
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		if msg != nil {
			msg.Edit(fmt.Sprintf("<b>Speed test failed:</b> <code>%s</code>", err.Error()))
		}
		return nil
	}
	defer resp.Body.Close()
	n, err := io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start).Seconds()
	if err != nil {
		if msg != nil {
			msg.Edit(fmt.Sprintf("<b>Speed test failed:</b> <code>%s</code>", err.Error()))
		}
		return nil
	}
	if elapsed <= 0 {
		elapsed = 0.001
	}
	mbps := (float64(n) * 8.0) / (elapsed * 1000000.0)
	mb := float64(n) / (1024.0 * 1024.0)
	out := fmt.Sprintf("<b>Speed Test</b>\nDownloaded: <code>%.2f MB</code>\nTime: <code>%.2fs</code>\nSpeed: <code>%.2f Mbps</code>", mb, elapsed, mbps)
	if msg != nil {
		msg.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func init() { QueueHandlerRegistration(registerSpeedtestHandlers) }

func registerSpeedtestHandlers() {
	c := Client
	c.On("cmd:speed", SpeedHandler, tg.CustomFilter(FilterOwner))
}
