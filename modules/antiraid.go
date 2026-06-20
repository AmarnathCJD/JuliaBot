package modules

import (
	"encoding/json"
	"fmt"
	"main/modules/db"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
)

type AntiraidSettings struct {
	Enabled      bool  `json:"enabled"`
	Threshold    int   `json:"threshold"`
	WindowSec    int   `json:"window_sec"`
	CooldownMin  int   `json:"cooldown_min"`
	LockedUntil  int64 `json:"locked_until"`
}

var (
	antiraidJoinsMu sync.Mutex
	antiraidJoins   = make(map[int64][]time.Time)
)

func defaultAntiraidSettings() *AntiraidSettings {
	return &AntiraidSettings{
		Enabled:     false,
		Threshold:   5,
		WindowSec:   10,
		CooldownMin: 5,
	}
}

func getAntiraidSettings(chatID int64) *AntiraidSettings {
	s := defaultAntiraidSettings()
	database, err := db.GetDB()
	if err != nil || database == nil {
		return s
	}
	_ = database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("antiraid"))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(strconv.FormatInt(chatID, 10)))
		if data == nil {
			return nil
		}
		_ = json.Unmarshal(data, s)
		return nil
	})
	return s
}

func saveAntiraidSettings(chatID int64, s *AntiraidSettings) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db unavailable")
	}
	return database.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("antiraid"))
		if err != nil {
			return err
		}
		data, err := json.Marshal(s)
		if err != nil {
			return err
		}
		return b.Put([]byte(strconv.FormatInt(chatID, 10)), data)
	})
}

func AntiraidHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Antiraid is only available in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		m.Reply("You need Ban Users permission to manage antiraid")
		return nil
	}

	args := strings.Fields(m.Args())
	settings := getAntiraidSettings(m.ChatID())

	if len(args) == 0 {
		status := "off"
		if settings.Enabled {
			status = "on"
		}
		lockMsg := ""
		if settings.LockedUntil > time.Now().Unix() {
			remain := time.Until(time.Unix(settings.LockedUntil, 0)).Round(time.Second)
			lockMsg = fmt.Sprintf("\nLockdown active for: %s", remain)
		}
		m.Reply(fmt.Sprintf(`<b>Antiraid Settings</b>

Status: <b>%s</b>
Threshold: <b>%d joins / %d seconds</b>
Lockdown cooldown: <b>%d minutes</b>%s

<b>Usage:</b>
/antiraid on|off|status
/antiraid threshold &lt;joins&gt; &lt;seconds&gt;
/antiraid cooldown &lt;minutes&gt;`,
			status, settings.Threshold, settings.WindowSec, settings.CooldownMin, lockMsg))
		return nil
	}

	switch strings.ToLower(args[0]) {
	case "on", "enable", "yes", "1":
		settings.Enabled = true
		if err := saveAntiraidSettings(m.ChatID(), settings); err != nil {
			m.Reply("Failed to enable antiraid")
			return nil
		}
		m.Reply(fmt.Sprintf("Antiraid <b>enabled</b>\nThreshold: %d joins in %d seconds\nCooldown: %d minutes",
			settings.Threshold, settings.WindowSec, settings.CooldownMin))
	case "off", "disable", "no", "0":
		settings.Enabled = false
		settings.LockedUntil = 0
		if err := saveAntiraidSettings(m.ChatID(), settings); err != nil {
			m.Reply("Failed to disable antiraid")
			return nil
		}
		antiraidJoinsMu.Lock()
		delete(antiraidJoins, m.ChatID())
		antiraidJoinsMu.Unlock()
		m.Reply("Antiraid <b>disabled</b>")
	case "status":
		status := "off"
		if settings.Enabled {
			status = "on"
		}
		lockMsg := "Lockdown: <b>inactive</b>"
		if settings.LockedUntil > time.Now().Unix() {
			remain := time.Until(time.Unix(settings.LockedUntil, 0)).Round(time.Second)
			lockMsg = fmt.Sprintf("Lockdown: <b>active</b> (%s left)", remain)
		}
		m.Reply(fmt.Sprintf(`<b>Antiraid Status</b>

State: <b>%s</b>
Threshold: <b>%d joins / %d seconds</b>
Cooldown: <b>%d minutes</b>
%s`, status, settings.Threshold, settings.WindowSec, settings.CooldownMin, lockMsg))
	case "threshold":
		if len(args) < 3 {
			m.Reply("Usage: /antiraid threshold &lt;joins&gt; &lt;seconds&gt;")
			return nil
		}
		joins, err1 := strconv.Atoi(args[1])
		secs, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil || joins < 2 || joins > 100 || secs < 1 || secs > 600 {
			m.Reply("Invalid threshold. joins: 2-100, seconds: 1-600")
			return nil
		}
		settings.Threshold = joins
		settings.WindowSec = secs
		if err := saveAntiraidSettings(m.ChatID(), settings); err != nil {
			m.Reply("Failed to update threshold")
			return nil
		}
		m.Reply(fmt.Sprintf("Antiraid threshold set to <b>%d joins / %d seconds</b>", joins, secs))
	case "cooldown":
		if len(args) < 2 {
			m.Reply("Usage: /antiraid cooldown &lt;minutes&gt;")
			return nil
		}
		mins, err := strconv.Atoi(args[1])
		if err != nil || mins < 1 || mins > 1440 {
			m.Reply("Invalid cooldown. Must be between 1 and 1440 minutes")
			return nil
		}
		settings.CooldownMin = mins
		if err := saveAntiraidSettings(m.ChatID(), settings); err != nil {
			m.Reply("Failed to update cooldown")
			return nil
		}
		m.Reply(fmt.Sprintf("Antiraid cooldown set to <b>%d minutes</b>", mins))
	default:
		m.Reply("Usage: /antiraid on|off|status | threshold &lt;joins&gt; &lt;seconds&gt; | cooldown &lt;minutes&gt;")
	}
	return nil
}

func AntiraidParticipantHandler(p *tg.ParticipantUpdate) error {
	if !p.IsJoined() && !p.IsAdded() {
		return nil
	}
	if p.User == nil {
		return nil
	}

	chatID := p.ChatID()
	settings := getAntiraidSettings(chatID)
	if !settings.Enabled {
		return nil
	}

	now := time.Now()

	if settings.LockedUntil > now.Unix() {
		user, err := p.Client.ResolvePeer(p.User.ID)
		if err == nil && user != nil {
			_, _ = p.Client.EditBanned(chatID, user, &tg.BannedOptions{Ban: true})
			_, _ = p.Client.EditBanned(chatID, user, &tg.BannedOptions{Unban: true})
		}
		return nil
	}

	antiraidJoinsMu.Lock()
	window := time.Duration(settings.WindowSec) * time.Second
	cutoff := now.Add(-window)
	joins := antiraidJoins[chatID]
	pruned := joins[:0]
	for _, t := range joins {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	pruned = append(pruned, now)
	antiraidJoins[chatID] = pruned
	tripped := len(pruned) >= settings.Threshold
	if tripped {
		antiraidJoins[chatID] = nil
	}
	antiraidJoinsMu.Unlock()

	if tripped {
		settings.LockedUntil = now.Add(time.Duration(settings.CooldownMin) * time.Minute).Unix()
		_ = saveAntiraidSettings(chatID, settings)

		_, _ = p.Client.SendMessage(chatID, fmt.Sprintf(
			"<b>Antiraid triggered</b>\nDetected %d joins within %d seconds. Lockdown engaged for %d minute(s). New joins will be removed.",
			settings.Threshold, settings.WindowSec, settings.CooldownMin))

		user, err := p.Client.ResolvePeer(p.User.ID)
		if err == nil && user != nil {
			_, _ = p.Client.EditBanned(chatID, user, &tg.BannedOptions{Ban: true})
			_, _ = p.Client.EditBanned(chatID, user, &tg.BannedOptions{Unban: true})
		}
	}

	return nil
}

func registerAntiraidHandlers() {
	c := Client
	c.On("cmd:antiraid", AntiraidHandler)
	c.On(tg.OnParticipant, AntiraidParticipantHandler)
}

func init() {
	QueueHandlerRegistration(registerAntiraidHandlers)

	Mods.AddModule("Antiraid", `<b>Antiraid Module</b>

Detects join floods and auto-locks the group, kicking new joins during cooldown.

<b>Commands:</b>
 /antiraid on|off|status - Toggle or view antiraid
 /antiraid threshold &lt;joins&gt; &lt;seconds&gt; - Set flood threshold
 /antiraid cooldown &lt;minutes&gt; - Lockdown duration after trip

<b>Defaults:</b>
 Threshold: 5 joins in 10 seconds
 Cooldown: 5 minutes`)
}
