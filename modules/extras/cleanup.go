package extras

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"main/modules/db"
	"strconv"
	"strings"
	"sync"
	"time"

	modules "main/modules"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
)

const cleanupBucket = "cleanup_cfg"

type cleanupCfg struct {
	Enabled bool `json:"enabled"`
	Delay   int  `json:"delay"`
	Pinned  bool `json:"pinned"`
}

var (
	cleanupCache   = make(map[int64]*cleanupCfg)
	cleanupCacheMu sync.RWMutex
)

func cleanupKey(chatID int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(chatID))
	return b
}

func defaultCleanupCfg() *cleanupCfg {
	return &cleanupCfg{Enabled: false, Delay: 2, Pinned: false}
}

func loadCleanupCfg(chatID int64) *cleanupCfg {
	cleanupCacheMu.RLock()
	if v, ok := cleanupCache[chatID]; ok {
		cp := *v
		cleanupCacheMu.RUnlock()
		return &cp
	}
	cleanupCacheMu.RUnlock()

	cfg := defaultCleanupCfg()
	d, err := db.GetDB()
	if err != nil || d == nil {
		return cfg
	}
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(cleanupBucket))
		if b == nil {
			return nil
		}
		raw := b.Get(cleanupKey(chatID))
		if len(raw) == 0 {
			return nil
		}
		_ = json.Unmarshal(raw, cfg)
		return nil
	})
	if cfg.Delay <= 0 {
		cfg.Delay = 2
	}

	cleanupCacheMu.Lock()
	cp := *cfg
	cleanupCache[chatID] = &cp
	cleanupCacheMu.Unlock()
	return cfg
}

func saveCleanupCfg(chatID int64, cfg *cleanupCfg) error {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return fmt.Errorf("db unavailable")
	}
	err = d.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(cleanupBucket))
		if err != nil {
			return err
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return b.Put(cleanupKey(chatID), raw)
	})
	if err != nil {
		return err
	}
	cleanupCacheMu.Lock()
	cp := *cfg
	cleanupCache[chatID] = &cp
	cleanupCacheMu.Unlock()
	return nil
}

func formatCleanupCfg(cfg *cleanupCfg) string {
	state := "off"
	if cfg.Enabled {
		state = "on"
	}
	pinned := "off"
	if cfg.Pinned {
		pinned = "on"
	}
	var sb strings.Builder
	sb.WriteString("<b>Service Message Cleanup</b>\n\n")
	fmt.Fprintf(&sb, " - <b>Status:</b> <code>%s</code>\n", state)
	fmt.Fprintf(&sb, " - <b>Delay:</b> <code>%d seconds</code>\n", cfg.Delay)
	fmt.Fprintf(&sb, " - <b>Pinned messages:</b> <code>%s</code>\n", pinned)
	sb.WriteString("\n<b>Usage:</b>\n")
	sb.WriteString(" - <code>/cleanup on</code> - Enable auto-delete\n")
	sb.WriteString(" - <code>/cleanup off</code> - Disable auto-delete\n")
	sb.WriteString(" - <code>/cleanup status</code> - Show current config\n")
	sb.WriteString(" - <code>/cleanup delay &lt;seconds&gt;</code> - Set delete delay (default 2)\n")
	sb.WriteString(" - <code>/cleanup pinned on|off</code> - Also clean pinned-notice service messages\n")
	return sb.String()
}

func CleanupHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Cleanup works in groups only.</b>")
		return nil
	}

	cfg := loadCleanupCfg(m.ChatID())

	args := m.ArgsList()
	if len(args) == 0 {
		m.Reply(formatCleanupCfg(cfg))
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> You need Change Info permission.")
		return nil
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "on", "enable", "yes":
		cfg.Enabled = true
		if err := saveCleanupCfg(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save config.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Cleanup enabled.</b> Service messages will be deleted after %d seconds.", cfg.Delay))
		return nil
	case "off", "disable", "no":
		cfg.Enabled = false
		if err := saveCleanupCfg(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save config.</b>")
			return nil
		}
		m.Reply("<b>Cleanup disabled.</b>")
		return nil
	case "status", "show":
		m.Reply(formatCleanupCfg(cfg))
		return nil
	case "delay":
		if len(args) < 2 {
			m.Reply("<b>Usage:</b> <code>/cleanup delay &lt;seconds&gt;</code>")
			return nil
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 0 {
			m.Reply("<b>Invalid delay.</b> Provide a non-negative integer.")
			return nil
		}
		if n > 3600 {
			m.Reply("<b>Delay too high.</b> Maximum is 3600 seconds.")
			return nil
		}
		cfg.Delay = n
		if err := saveCleanupCfg(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save config.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Delay set to %d seconds.</b>", n))
		return nil
	case "pinned":
		if len(args) < 2 {
			m.Reply("<b>Usage:</b> <code>/cleanup pinned on|off</code>")
			return nil
		}
		v := strings.ToLower(args[1])
		switch v {
		case "on", "enable", "yes":
			cfg.Pinned = true
		case "off", "disable", "no":
			cfg.Pinned = false
		default:
			m.Reply("<b>Usage:</b> <code>/cleanup pinned on|off</code>")
			return nil
		}
		if err := saveCleanupCfg(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save config.</b>")
			return nil
		}
		state := "off"
		if cfg.Pinned {
			state = "on"
		}
		m.Reply(fmt.Sprintf("<b>Pinned cleanup:</b> <code>%s</code>", state))
		return nil
	}

	m.Reply(formatCleanupCfg(cfg))
	return nil
}

func CleanupWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	if !m.IsService() || m.Action == nil {
		return nil
	}

	cfg := loadCleanupCfg(m.ChatID())
	if !cfg.Enabled {
		return nil
	}

	if _, ok := m.Action.(*tg.MessageActionPinMessage); ok {
		if !cfg.Pinned {
			return nil
		}
	}

	delay := cfg.Delay
	if delay < 0 {
		delay = 2
	}

	chatID := m.ChatID()
	peer := m.Peer
	id := m.ID
	client := m.Client

	if delay == 0 {
		go func() {
			_, _ = client.DeleteMessages(peer, []int32{id})
		}()
		return nil
	}

	time.AfterFunc(time.Duration(delay)*time.Second, func() {
		_, _ = client.DeleteMessages(peer, []int32{id})
		_ = chatID
	})
	return nil
}

func registerCleanupHandlers() {
	c := modules.Client
	c.On("cmd:cleanup", CleanupHandler)
	c.On(tg.OnAction, CleanupWatcher)
}

func init() {
	modules.QueueHandlerRegistration(registerCleanupHandlers)

	modules.Mods.AddModule("Cleanup", `<b>Service Message Cleanup</b>

Auto-delete service messages (join/leave/pin/title changes) per chat.

<b>Commands:</b>
 - /cleanup - Show current config and usage
 - /cleanup on - Enable auto-delete
 - /cleanup off - Disable auto-delete
 - /cleanup status - Show current config
 - /cleanup delay &lt;seconds&gt; - Set delete delay (default 2)
 - /cleanup pinned on|off - Toggle pinned-notice cleanup

<b>Permission:</b> Admins with Delete Messages permission.`)
}
