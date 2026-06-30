package extras

import (
	"main/modules/db"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
	modules "main/modules"
)

const silentBucket = "silent_cfg"

func IsSilent(chatID int64) bool {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return false
	}
	var enabled bool
	_ = database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(silentBucket))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(strconv.FormatInt(chatID, 10)))
		if v == nil {
			return nil
		}
		if string(v) == "1" {
			enabled = true
		}
		return nil
	})
	return enabled
}

func setSilent(chatID int64, enabled bool) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil
	}
	return database.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(silentBucket))
		if err != nil {
			return err
		}
		val := []byte("0")
		if enabled {
			val = []byte("1")
		}
		return b.Put([]byte(strconv.FormatInt(chatID, 10)), val)
	})
}

func SilentHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Silent mode is only available in groups")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need admin rights to manage silent mode")
		return nil
	}

	args := strings.Fields(m.Args())
	if len(args) == 0 {
		m.Reply("Usage: /silent on|off|status")
		return nil
	}

	switch strings.ToLower(args[0]) {
	case "on", "enable", "yes", "1":
		if err := setSilent(m.ChatID(), true); err != nil {
			m.Reply("Failed to enable silent mode")
			return nil
		}
		m.Reply("Silent mode <b>enabled</b>")
	case "off", "disable", "no", "0":
		if err := setSilent(m.ChatID(), false); err != nil {
			m.Reply("Failed to disable silent mode")
			return nil
		}
		m.Reply("Silent mode <b>disabled</b>")
	case "status":
		state := "off"
		if IsSilent(m.ChatID()) {
			state = "on"
		}
		m.Reply("Silent mode: <b>" + state + "</b>")
	default:
		m.Reply("Usage: /silent on|off|status")
	}
	return nil
}

func registerSilentHandlers() {
	c := modules.Client
	c.On("cmd:silent", SilentHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerSilentHandlers)
}
