package modules

import (
	"encoding/binary"
	"fmt"
	"html"
	"main/modules/db"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
)

const announceBucket = "announce_last"

func AnnounceHandler(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "pin") {
		m.Reply("<b>Permission denied.</b> You need Pin Messages permission.")
		return nil
	}

	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil {
			text = reply.RawText()
		}
	}

	if text == "" {
		m.Reply("<b>Usage:</b> <code>/announce &lt;text&gt;</code> or reply to a message.")
		return nil
	}

	escaped := html.EscapeString(text)
	timestamp := time.Now().Format("Jan 02, 2006 15:04 MST")
	sender := html.EscapeString(m.Sender.FirstName)
	if sender == "" {
		sender = fmt.Sprintf("%d", m.SenderID())
	}

	body := fmt.Sprintf(
		"<b>ANNOUNCEMENT</b>\n"+
			"<b>━━━━━━━━━━━━━━━━</b>\n\n"+
			"%s\n\n"+
			"<b>━━━━━━━━━━━━━━━━</b>\n"+
			"<i>Posted by</i> <b>%s</b>\n"+
			"<i>%s</i>",
		escaped, sender, timestamp,
	)

	sent, err := m.Reply(body)
	if err != nil || sent == nil {
		m.Reply("<b>Failed to post announcement.</b>")
		return nil
	}

	if CanBot(m.Client, m.Channel, "pin") {
		_, perr := m.Client.PinMessage(m.ChatID(), sent.ID, &tg.PinOptions{Silent: false})
		if perr != nil {
			m.Reply("<b>Announcement posted.</b> <i>Could not pin: missing permission.</i>")
		}
	} else {
		m.Reply("<b>Announcement posted.</b> <i>I need pin permission to pin it.</i>")
	}

	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil
	}

	chatID := m.ChatID()
	_ = database.Update(func(tx *bolt.Tx) error {
		b, berr := tx.CreateBucketIfNotExists([]byte(announceBucket))
		if berr != nil {
			return berr
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(chatID))
		val := make([]byte, 4)
		binary.BigEndian.PutUint32(val, uint32(sent.ID))
		return b.Put(key, val)
	})

	return nil
}

func UnannounceHandler(m *tg.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "pin") {
		m.Reply("<b>Permission denied.</b> You need Pin Messages permission.")
		return nil
	}

	database, err := db.GetDB()
	if err != nil || database == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	chatID := m.ChatID()
	var msgID int32

	_ = database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(announceBucket))
		if b == nil {
			return nil
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(chatID))
		v := b.Get(key)
		if len(v) == 4 {
			msgID = int32(binary.BigEndian.Uint32(v))
		}
		return nil
	})

	if msgID == 0 {
		m.Reply("<b>No recent announcement found.</b>")
		return nil
	}

	if CanBot(m.Client, m.Channel, "pin") {
		_, _ = m.Client.UnpinMessage(chatID, msgID)
	}

	_, derr := m.Client.DeleteMessages(chatID, []int32{msgID})
	if derr != nil {
		m.Reply("<b>Failed to remove announcement.</b> It may already be gone.")
	} else {
		m.Reply("<b>Announcement removed.</b>")
	}

	_ = database.Update(func(tx *bolt.Tx) error {
		b, berr := tx.CreateBucketIfNotExists([]byte(announceBucket))
		if berr != nil {
			return berr
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(chatID))
		return b.Delete(key)
	})

	return nil
}

func registerAnnounceHandlers() {
	c := Client
	c.On("cmd:announce", AnnounceHandler)
	c.On("cmd:unannounce", UnannounceHandler)
}

func init() {
	QueueHandlerRegistration(registerAnnounceHandlers)
}
