package modules

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"strings"
	"sync"

	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
)

type autotrConfig struct {
	Enabled bool   `json:"e"`
	Lang    string `json:"l"`
	Min     int    `json:"m"`
}

var (
	autotrBucket = []byte("autotr")
	autotrCache  = make(map[int64]*autotrConfig)
	autotrMu     sync.RWMutex
)

func autotrChatKey(chatID int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(chatID))
	return b
}

func autotrLoad(chatID int64) *autotrConfig {
	autotrMu.RLock()
	if c, ok := autotrCache[chatID]; ok {
		autotrMu.RUnlock()
		return c
	}
	autotrMu.RUnlock()

	cfg := &autotrConfig{Enabled: false, Lang: "en", Min: 4}
	database, err := db.GetDB()
	if err != nil || database == nil {
		return cfg
	}
	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(autotrBucket)
		if b == nil {
			return nil
		}
		raw := b.Get(autotrChatKey(chatID))
		if raw == nil {
			return nil
		}
		var c autotrConfig
		if err := json.Unmarshal(raw, &c); err == nil {
			if c.Lang == "" {
				c.Lang = "en"
			}
			if c.Min <= 0 {
				c.Min = 4
			}
			cfg = &c
		}
		return nil
	})
	autotrMu.Lock()
	autotrCache[chatID] = cfg
	autotrMu.Unlock()
	return cfg
}

func autotrSave(chatID int64, cfg *autotrConfig) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db unavailable")
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	err = database.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(autotrBucket)
		if err != nil {
			return err
		}
		return b.Put(autotrChatKey(chatID), data)
	})
	if err == nil {
		autotrMu.Lock()
		autotrCache[chatID] = cfg
		autotrMu.Unlock()
	}
	return err
}

func AutoTrHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Auto-translate works in groups only.</b>")
		return nil
	}
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		return nil
	}

	args := strings.TrimSpace(m.Args())
	cfg := autotrLoad(m.ChatID())

	if args == "" || args == "status" {
		state := "off"
		if cfg.Enabled {
			state = "on"
		}
		m.Reply(fmt.Sprintf("<b>Auto-Translate</b>\n • State: <code>%s</code>\n • Lang: <code>%s</code>\n • Min chars: <code>%d</code>\n\n<i>Usage:</i>\n <code>/autotr on|off</code>\n <code>/autotr lang &lt;iso&gt;</code>\n <code>/autotr min &lt;chars&gt;</code>",
			state, html.EscapeString(cfg.Lang), cfg.Min))
		return nil
	}

	parts := strings.Fields(args)
	sub := strings.ToLower(parts[0])

	switch sub {
	case "on", "enable":
		cfg.Enabled = true
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Auto-translate enabled.</b> Target: <code>%s</code>", html.EscapeString(cfg.Lang)))
	case "off", "disable":
		cfg.Enabled = false
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply("<b>Auto-translate disabled.</b>")
	case "lang", "language":
		if len(parts) < 2 {
			m.Reply("<b>Usage:</b> <code>/autotr lang &lt;iso&gt;</code>")
			return nil
		}
		lang := strings.ToLower(strings.TrimSpace(parts[1]))
		if len(lang) < 2 || len(lang) > 8 {
			m.Reply("<b>Invalid language code.</b>")
			return nil
		}
		cfg.Lang = lang
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Target language set to</b> <code>%s</code>", html.EscapeString(lang)))
	case "min":
		if len(parts) < 2 {
			m.Reply("<b>Usage:</b> <code>/autotr min &lt;chars&gt;</code>")
			return nil
		}
		var n int
		_, err := fmt.Sscanf(parts[1], "%d", &n)
		if err != nil || n < 1 || n > 4096 {
			m.Reply("<b>Invalid number.</b> Must be between 1 and 4096.")
			return nil
		}
		cfg.Min = n
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Minimum length set to</b> <code>%d</code>", n))
	default:
		m.Reply("<b>Unknown subcommand.</b> Use <code>on</code>, <code>off</code>, <code>lang</code>, <code>min</code>, or <code>status</code>.")
	}
	return nil
}

func AutoTrWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	if m.Sender != nil && m.Sender.Bot {
		return nil
	}
	if m.Message != nil && m.Message.ViaBotID != 0 {
		return nil
	}
	text := strings.TrimSpace(m.Text())
	if text == "" {
		return nil
	}
	if t := text; len(t) > 0 && (t[0] == '/' || t[0] == '!' || t[0] == '.') {
		return nil
	}

	cfg := autotrLoad(m.ChatID())
	if !cfg.Enabled {
		return nil
	}
	if len([]rune(text)) < cfg.Min {
		return nil
	}

	translated, src, err := googleTranslate(text, cfg.Lang)
	if err != nil || translated == "" {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(src), strings.TrimSpace(cfg.Lang)) {
		return nil
	}
	if strings.TrimSpace(translated) == strings.TrimSpace(text) {
		return nil
	}

	m.Reply(fmt.Sprintf("<blockquote><i>%s→%s</i> %s</blockquote>",
		html.EscapeString(src), html.EscapeString(cfg.Lang), html.EscapeString(translated)))
	return nil
}

func registerAutoTranslateHandlers() {
	c := Client
	c.On("cmd:autotr", AutoTrHandler)
	c.On(tg.OnNewMessage, AutoTrWatcher)

	Mods.AddModule("AutoTranslate", `<b>Auto-Translate Module</b>

<b>Commands:</b>
 • /autotr on|off - Toggle auto-translation for this chat
 • /autotr lang &lt;iso&gt; - Set target language (default: en)
 • /autotr min &lt;chars&gt; - Set minimum message length (default: 4)
 • /autotr status - Show current settings

<i>Admin only. Skips bots, commands, and short messages.</i>`)
}

func init() {
	QueueHandlerRegistration(registerAutoTranslateHandlers)
}
