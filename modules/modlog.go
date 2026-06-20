package modules

import (
	"fmt"
	"html"
	"main/modules/db"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
)

const modlogBucket = "modlog"

func ensureModlogBucket(d *bolt.DB) error {
	return d.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(modlogBucket))
		return err
	})
}

func setModlogChat(srcChat, logChat int64) error {
	d, err := db.GetDB()
	if err != nil {
		return err
	}
	if err := ensureModlogBucket(d); err != nil {
		return err
	}
	return d.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(modlogBucket))
		return b.Put([]byte(strconv.FormatInt(srcChat, 10)), []byte(strconv.FormatInt(logChat, 10)))
	})
}

func getModlogChat(srcChat int64) (int64, bool) {
	d, err := db.GetDB()
	if err != nil {
		return 0, false
	}
	if err := ensureModlogBucket(d); err != nil {
		return 0, false
	}
	var logChat int64
	var found bool
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(modlogBucket))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(strconv.FormatInt(srcChat, 10)))
		if v == nil {
			return nil
		}
		id, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return nil
		}
		logChat = id
		found = true
		return nil
	})
	return logChat, found
}

func unsetModlogChat(srcChat int64) error {
	d, err := db.GetDB()
	if err != nil {
		return err
	}
	if err := ensureModlogBucket(d); err != nil {
		return err
	}
	return d.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(modlogBucket))
		return b.Delete([]byte(strconv.FormatInt(srcChat, 10)))
	})
}

func actionEmoji(action string) string {
	switch strings.ToLower(action) {
	case "ban", "tban", "sban":
		return "\U0001F6AB"
	case "unban":
		return "♻️"
	case "kick", "punch":
		return "\U0001F462"
	case "mute", "tmute":
		return "\U0001F507"
	case "unmute":
		return "\U0001F50A"
	case "warn", "twarn":
		return "⚠️"
	case "lock":
		return "\U0001F512"
	case "unlock":
		return "\U0001F513"
	case "purge", "del", "delete":
		return "\U0001F5D1️"
	case "pin":
		return "\U0001F4CC"
	case "unpin":
		return "\U0001F4CD"
	case "promote":
		return "⬆️"
	case "demote":
		return "⬇️"
	case "blacklist":
		return "\U0001F6D1"
	default:
		return "\U0001F4DD"
	}
}

func SetLogChatHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Mod log can only be configured in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to configure the mod log")
		return nil
	}

	args := strings.TrimSpace(m.Args())
	var target int64

	if args == "" {
		target = m.ChatID()
	} else {
		id, err := strconv.ParseInt(args, 10, 64)
		if err != nil {
			m.Reply("Usage: <code>/setlogchat &lt;chat_id&gt;</code>\nOr omit the id to log into this chat.")
			return nil
		}
		target = id
	}

	if _, err := m.Client.ResolvePeer(target); err != nil {
		m.Reply(fmt.Sprintf("I cannot access chat <code>%d</code>. Make sure I am a member and can post there.", target))
		return nil
	}

	if err := setModlogChat(m.ChatID(), target); err != nil {
		m.Reply("Failed to save log chat")
		return nil
	}

	m.Reply(fmt.Sprintf("Moderation log chat set to <code>%d</code>", target))
	return nil
}

func UnsetLogChatHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Mod log can only be configured in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to configure the mod log")
		return nil
	}

	if _, ok := getModlogChat(m.ChatID()); !ok {
		m.Reply("No mod log chat configured for this group")
		return nil
	}

	if err := unsetModlogChat(m.ChatID()); err != nil {
		m.Reply("Failed to clear log chat")
		return nil
	}

	m.Reply("Moderation log chat cleared")
	return nil
}

func LogStatusHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Mod log is a group feature")
		return nil
	}

	logChat, ok := getModlogChat(m.ChatID())
	if !ok {
		m.Reply("Moderation log is <b>disabled</b> for this chat.\nUse <code>/setlogchat</code> to enable.")
		return nil
	}

	m.Reply(fmt.Sprintf("Moderation log is <b>enabled</b>.\nLog chat: <code>%d</code>", logChat))
	return nil
}

func LogModerationAction(srcChat int64, action, actor, target, reason string) {
	if Client == nil {
		return
	}

	logChat, ok := getModlogChat(srcChat)
	if !ok {
		return
	}

	emoji := actionEmoji(action)
	ts := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	var sb strings.Builder
	sb.WriteString(emoji)
	sb.WriteString(" <b>")
	sb.WriteString(html.EscapeString(strings.ToUpper(action)))
	sb.WriteString("</b>\n")
	sb.WriteString("<b>Chat:</b> <code>")
	sb.WriteString(strconv.FormatInt(srcChat, 10))
	sb.WriteString("</code>\n")

	if strings.TrimSpace(actor) != "" {
		sb.WriteString("<b>Actor:</b> ")
		sb.WriteString(html.EscapeString(actor))
		sb.WriteString("\n")
	}

	if strings.TrimSpace(target) != "" {
		sb.WriteString("<b>Target:</b> ")
		sb.WriteString(html.EscapeString(target))
		sb.WriteString("\n")
	}

	if strings.TrimSpace(reason) != "" {
		sb.WriteString("<b>Reason:</b> <i>")
		sb.WriteString(html.EscapeString(reason))
		sb.WriteString("</i>\n")
	}

	sb.WriteString("<b>Time:</b> <code>")
	sb.WriteString(ts)
	sb.WriteString("</code>")

	go func() {
		_, _ = Client.SendMessage(logChat, sb.String())
	}()
}

func registerModlogHandlers() {
	c := Client
	c.On("cmd:setlogchat", SetLogChatHandler)
	c.On("cmd:unsetlogchat", UnsetLogChatHandler)
	c.On("cmd:logstatus", LogStatusHandler)
}

func init() {
	QueueHandlerRegistration(registerModlogHandlers)

	Mods.AddModule("ModLog", `<b>Moderation Log</b>

Forward moderation events from this group into a dedicated log chat.

<b>Commands:</b>
 - /setlogchat &lt;chat_id&gt; - Configure where to send mod log entries (omit id to log into current chat)
 - /unsetlogchat - Disable moderation logging
 - /logstatus - Show whether logging is enabled and which chat receives entries

<b>Notes:</b>
 - Requires Change Info permission to configure.
 - The bot must be a member of the target log chat with permission to send messages.
 - Logged actions include warns, bans, mutes, kicks, locks and other moderation events.`)
}
