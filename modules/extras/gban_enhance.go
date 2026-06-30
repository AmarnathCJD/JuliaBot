package extras

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"main/modules/db"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
	modules "main/modules"
)

const (
	reportCfgBucket = "report_cfg"
	reportsBucket   = "reports"
	reportCooldown  = 5 * time.Minute
	reportMaxStore  = 20
)

type reportEntry struct {
	ChatID     int64  `json:"chat_id"`
	MessageID  int64  `json:"message_id"`
	ReporterID int64  `json:"reporter_id"`
	OffenderID int64  `json:"offender_id"`
	Reason     string `json:"reason"`
	Timestamp  int64  `json:"timestamp"`
	Link       string `json:"link"`
}

var (
	reportRateMu sync.Mutex
	reportRate   = make(map[string]time.Time)
)

func reportRateKey(chatID, userID int64) string {
	return strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(userID, 10)
}

func reportRateAllowed(chatID, userID int64) (bool, time.Duration) {
	reportRateMu.Lock()
	defer reportRateMu.Unlock()
	k := reportRateKey(chatID, userID)
	last, ok := reportRate[k]
	if ok {
		elapsed := time.Since(last)
		if elapsed < reportCooldown {
			return false, reportCooldown - elapsed
		}
	}
	reportRate[k] = time.Now()
	return true, 0
}

func ensureReportBuckets(database *bolt.DB) error {
	return database.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(reportCfgBucket)); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists([]byte(reportsBucket))
		return err
	})
}

func getReportCfg(chatID int64) bool {
	database, err := db.GetDB()
	if err != nil {
		return true
	}
	if err := ensureReportBuckets(database); err != nil {
		return true
	}
	enabled := true
	database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(reportCfgBucket))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(strconv.FormatInt(chatID, 10)))
		if v == nil {
			return nil
		}
		if string(v) == "off" {
			enabled = false
		}
		return nil
	})
	return enabled
}

func setReportCfg(chatID int64, enabled bool) error {
	database, err := db.GetDB()
	if err != nil {
		return err
	}
	if err := ensureReportBuckets(database); err != nil {
		return err
	}
	return database.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(reportCfgBucket))
		val := []byte("on")
		if !enabled {
			val = []byte("off")
		}
		return b.Put([]byte(strconv.FormatInt(chatID, 10)), val)
	})
}

func saveReport(chatID int64, entry *reportEntry) error {
	database, err := db.GetDB()
	if err != nil {
		return err
	}
	if err := ensureReportBuckets(database); err != nil {
		return err
	}
	return database.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket([]byte(reportsBucket))
		cb, err := root.CreateBucketIfNotExists([]byte(strconv.FormatInt(chatID, 10)))
		if err != nil {
			return err
		}
		seq, err := cb.NextSequence()
		if err != nil {
			return err
		}
		entry.Timestamp = time.Now().Unix()
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("%020d", seq)
		if err := cb.Put([]byte(key), data); err != nil {
			return err
		}
		c := cb.Cursor()
		var keys [][]byte
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			cp := make([]byte, len(k))
			copy(cp, k)
			keys = append(keys, cp)
		}
		if len(keys) > reportMaxStore {
			sort.Slice(keys, func(i, j int) bool { return string(keys[i]) < string(keys[j]) })
			excess := len(keys) - reportMaxStore
			for i := 0; i < excess; i++ {
				cb.Delete(keys[i])
			}
		}
		return nil
	})
}

func getRecentReports(chatID int64, limit int) ([]*reportEntry, error) {
	database, err := db.GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureReportBuckets(database); err != nil {
		return nil, err
	}
	var out []*reportEntry
	err = database.View(func(tx *bolt.Tx) error {
		root := tx.Bucket([]byte(reportsBucket))
		if root == nil {
			return nil
		}
		cb := root.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if cb == nil {
			return nil
		}
		c := cb.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var e reportEntry
			if err := json.Unmarshal(v, &e); err != nil {
				continue
			}
			out = append(out, &e)
			if len(out) >= limit {
				break
			}
		}
		return nil
	})
	return out, err
}

func reportMessageLink(m *tg.NewMessage, msgID int32) string {
	if m.Channel != nil && m.Channel.Username != "" {
		return fmt.Sprintf("https://t.me/%s/%d", m.Channel.Username, msgID)
	}
	return fmt.Sprintf("https://t.me/c/%d/%d", m.ChatID(), msgID)
}

func ReportHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Reports can only be used in groups")
		return nil
	}
	if !getReportCfg(m.ChatID()) {
		m.Reply("Reporting is disabled in this chat")
		return nil
	}
	if !m.IsReply() {
		m.Reply("Reply to a message with /report [reason] to flag it for admins")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil || reply == nil {
		m.Reply("Unable to read the replied message")
		return nil
	}

	offenderID := reply.SenderID()
	if offenderID == m.SenderID() {
		m.Reply("You cannot report yourself")
		return nil
	}
	if modules.IsUserAdmin(m.Client, offenderID, m.ChatID(), "") {
		m.Reply("Admins cannot be reported")
		return nil
	}
	if reply.Sender != nil && reply.Sender.Bot {
		m.Reply("Bots cannot be reported")
		return nil
	}

	ok, wait := reportRateAllowed(m.ChatID(), m.SenderID())
	if !ok {
		m.Reply(fmt.Sprintf("Slow down. Try again in %s", wait.Round(time.Second)))
		return nil
	}

	reason := strings.TrimSpace(m.Args())
	if reason == "" {
		reason = "No reason given"
	}
	if len(reason) > 256 {
		reason = reason[:256]
	}

	chat, err := m.GetChat()
	if err != nil || chat == nil {
		m.Reply("Unable to fetch chat info")
		return nil
	}

	participants, _, err := m.Client.GetChatMembers(chat, &tg.ParticipantOptions{
		Filter: &tg.ChannelParticipantsAdmins{},
		Limit:  100,
	})
	if err != nil || len(participants) == 0 {
		m.Reply("Unable to resolve admins for this chat")
		return nil
	}

	link := reportMessageLink(m, int32(reply.ID))

	reporterName := "User"
	if m.Sender != nil && m.Sender.FirstName != "" {
		reporterName = m.Sender.FirstName
	}
	reporterMention := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>",
		m.SenderID(), html.EscapeString(reporterName))

	offenderName := "User"
	if reply.Sender != nil && reply.Sender.FirstName != "" {
		offenderName = reply.Sender.FirstName
	}
	offenderMention := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>",
		offenderID, html.EscapeString(offenderName))

	var mentions []string
	seen := make(map[int64]bool)
	for _, p := range participants {
		if p == nil || p.User == nil {
			continue
		}
		u := p.User
		if u.Bot || u.Deleted {
			continue
		}
		if u.ID == m.SenderID() || u.ID == offenderID {
			continue
		}
		if seen[u.ID] {
			continue
		}
		seen[u.ID] = true
		mentions = append(mentions, fmt.Sprintf("<a href=\"tg://user?id=%d\">⁣</a>", u.ID))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Report</b> from %s\n", reporterMention))
	sb.WriteString(fmt.Sprintf("Offender: %s (<code>%d</code>)\n", offenderMention, offenderID))
	sb.WriteString(fmt.Sprintf("Reason: %s\n", html.EscapeString(reason)))
	sb.WriteString(fmt.Sprintf("Message: <a href=\"%s\">view</a>\n", link))
	if len(mentions) > 0 {
		sb.WriteString(strings.Join(mentions, ""))
	} else {
		sb.WriteString("(no admins available to notify)")
	}

	m.Reply(sb.String(), &tg.SendOptions{LinkPreview: false})

	saveReport(m.ChatID(), &reportEntry{
		ChatID:     m.ChatID(),
		MessageID:  int64(reply.ID),
		ReporterID: m.SenderID(),
		OffenderID: offenderID,
		Reason:     reason,
		Link:       link,
	})

	return nil
}

func ReportCfgHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Report config is per-group")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to toggle reports")
		return nil
	}
	arg := strings.ToLower(strings.TrimSpace(m.Args()))
	current := getReportCfg(m.ChatID())
	if arg == "" {
		state := "on"
		if !current {
			state = "off"
		}
		m.Reply(fmt.Sprintf("Reports are currently <b>%s</b>\nUsage: /reportcfg on|off", state))
		return nil
	}
	switch arg {
	case "on", "enable", "yes", "true":
		if err := setReportCfg(m.ChatID(), true); err != nil {
			m.Reply("Failed to update setting")
			return nil
		}
		m.Reply("Reports enabled")
	case "off", "disable", "no", "false":
		if err := setReportCfg(m.ChatID(), false); err != nil {
			m.Reply("Failed to update setting")
			return nil
		}
		m.Reply("Reports disabled")
	default:
		m.Reply("Usage: /reportcfg on|off")
	}
	return nil
}

func ReportsListHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Reports are tracked per-group")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		m.Reply("You need Ban Users permission to view reports")
		return nil
	}
	entries, err := getRecentReports(m.ChatID(), reportMaxStore)
	if err != nil {
		m.Reply("Failed to read reports")
		return nil
	}
	if len(entries) == 0 {
		m.Reply("No reports recorded yet")
		return nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Recent reports</b> (%d)\n\n", len(entries)))
	for i, e := range entries {
		ts := time.Unix(e.Timestamp, 0).UTC().Format("02 Jan 06 15:04")
		reason := e.Reason
		if reason == "" {
			reason = "No reason given"
		}
		sb.WriteString(fmt.Sprintf(
			"%d. <a href=\"%s\">msg</a> | offender <code>%d</code> | by <code>%d</code>\n   %s | %s\n",
			i+1, e.Link, e.OffenderID, e.ReporterID, html.EscapeString(reason), ts))
	}
	m.Reply(sb.String(), &tg.SendOptions{LinkPreview: false})
	return nil
}

func registerReportHandlers() {
	c := modules.Client
	c.On("cmd:report", ReportHandler)
	c.On("cmd:reportcfg", ReportCfgHandler)
	c.On("cmd:reports", ReportsListHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerReportHandlers)

	modules.Mods.AddModule("Reports", `<b>Report System</b>

Reply to a message and use /report [reason] to flag it for admins. The bot silently pings every admin with a link to the offending message.

<b>Commands:</b>
/report [reason] - report a replied message (1 per 5 min per user)
/reportcfg on|off - admin: toggle the report system in this chat
/reports - admin: view the last 20 reports`)
}
