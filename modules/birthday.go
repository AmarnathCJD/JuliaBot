package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"main/modules/db"

	"go.etcd.io/bbolt"
)

type birthdayEntry struct {
	UserID     int64  `json:"user_id"`
	Day        int    `json:"day"`
	Month      int    `json:"month"`
	Year       int    `json:"year"`
	LastChatID int64  `json:"last_chat_id"`
	Name       string `json:"name"`
	LastSent   string `json:"last_sent"`
}

const birthdayBucket = "birthdays"

func parseBirthdayDate(s string) (int, int, int, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "-")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid format")
	}
	day, err := strconv.Atoi(parts[0])
	if err != nil || day < 1 || day > 31 {
		return 0, 0, 0, fmt.Errorf("invalid day")
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return 0, 0, 0, fmt.Errorf("invalid month")
	}
	year := 0
	if len(parts) == 3 {
		year, err = strconv.Atoi(parts[2])
		if err != nil || year < 1900 || year > 2100 {
			return 0, 0, 0, fmt.Errorf("invalid year")
		}
	}
	if _, err := time.Parse("2-1-2006", fmt.Sprintf("%d-%d-%d", day, month, 2000)); err != nil {
		return 0, 0, 0, fmt.Errorf("invalid date")
	}
	return day, month, year, nil
}

func saveBirthday(entry *birthdayEntry) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db error")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(birthdayBucket))
		if err != nil {
			return err
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return b.Put([]byte(strconv.FormatInt(entry.UserID, 10)), data)
	})
}

func loadBirthday(userID int64) (*birthdayEntry, error) {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil, fmt.Errorf("db error")
	}
	var entry *birthdayEntry
	err = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(birthdayBucket))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(strconv.FormatInt(userID, 10)))
		if v == nil {
			return nil
		}
		var e birthdayEntry
		if err := json.Unmarshal(v, &e); err != nil {
			return err
		}
		entry = &e
		return nil
	})
	return entry, err
}

func deleteBirthday(userID int64) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db error")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(birthdayBucket))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(strconv.FormatInt(userID, 10)))
	})
}

func listAllBirthdays() ([]*birthdayEntry, error) {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil, fmt.Errorf("db error")
	}
	var entries []*birthdayEntry
	err = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(birthdayBucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var e birthdayEntry
			if err := json.Unmarshal(v, &e); err != nil {
				return nil
			}
			entries = append(entries, &e)
			return nil
		})
	})
	return entries, err
}

func updateLastActiveChat(userID, chatID int64, name string) {
	entry, err := loadBirthday(userID)
	if err != nil || entry == nil {
		return
	}
	entry.LastChatID = chatID
	if name != "" {
		entry.Name = name
	}
	_ = saveBirthday(entry)
}

func daysUntilBirthday(day, month int, now time.Time) int {
	year := now.Year()
	next := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if next.Before(today) {
		next = time.Date(year+1, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}
	return int(next.Sub(today).Hours() / 24)
}

func formatBirthdayDate(day, month, year int) string {
	monthName := time.Month(month).String()
	if year > 0 {
		return fmt.Sprintf("%d %s %d", day, monthName, year)
	}
	return fmt.Sprintf("%d %s", day, monthName)
}

func BirthdayHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		entry, _ := loadBirthday(m.SenderID())
		if entry == nil {
			m.Reply("<b>Usage:</b>\n" +
				"<code>/bday set DD-MM</code> or <code>/bday set DD-MM-YYYY</code>\n" +
				"<code>/bday clear</code>\n" +
				"<code>/bday check [@user]</code>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Your birthday:</b> %s", formatBirthdayDate(entry.Day, entry.Month, entry.Year)))
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	sub := strings.ToLower(parts[0])
	rest := ""
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}

	switch sub {
	case "set":
		if rest == "" {
			m.Reply("<b>Usage:</b> <code>/bday set DD-MM</code> or <code>/bday set DD-MM-YYYY</code>")
			return nil
		}
		day, month, year, err := parseBirthdayDate(rest)
		if err != nil {
			m.Reply("<b>Invalid date.</b> Use <code>DD-MM</code> or <code>DD-MM-YYYY</code>.")
			return nil
		}
		entry := &birthdayEntry{
			UserID:     m.SenderID(),
			Day:        day,
			Month:      month,
			Year:       year,
			LastChatID: m.ChatID(),
			Name:       m.Sender.FirstName,
		}
		if err := saveBirthday(entry); err != nil {
			m.Reply("<b>Failed to save birthday.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Birthday saved:</b> %s", formatBirthdayDate(day, month, year)))
		return nil

	case "clear", "remove", "delete":
		entry, _ := loadBirthday(m.SenderID())
		if entry == nil {
			m.Reply("<b>No birthday set.</b>")
			return nil
		}
		if err := deleteBirthday(m.SenderID()); err != nil {
			m.Reply("<b>Failed to clear birthday.</b>")
			return nil
		}
		m.Reply("<b>Birthday cleared.</b>")
		return nil

	case "check":
		targetID := m.SenderID()
		targetName := m.Sender.FirstName
		if m.IsReply() {
			r, err := m.GetReplyMessage()
			if err == nil {
				targetID = r.SenderID()
				if r.Sender != nil {
					targetName = r.Sender.FirstName
				}
			}
		} else if rest != "" {
			peer, err := m.Client.ResolvePeer(rest)
			if err == nil {
				targetID = m.Client.GetPeerID(peer)
				if u, _ := m.Client.GetUser(targetID); u != nil {
					targetName = u.FirstName
				}
			}
		}
		entry, _ := loadBirthday(targetID)
		if entry == nil {
			m.Reply(fmt.Sprintf("<b>%s</b> has not set a birthday.", html.EscapeString(targetName)))
			return nil
		}
		days := daysUntilBirthday(entry.Day, entry.Month, time.Now().UTC())
		dateStr := formatBirthdayDate(entry.Day, entry.Month, entry.Year)
		if days == 0 {
			m.Reply(fmt.Sprintf("<b>Today is %s's birthday!</b> (%s)", html.EscapeString(targetName), dateStr))
		} else {
			m.Reply(fmt.Sprintf("<b>%s's birthday:</b> %s\n<b>In:</b> %d day(s)", html.EscapeString(targetName), dateStr, days))
		}
		return nil

	default:
		day, month, year, err := parseBirthdayDate(args)
		if err == nil {
			entry := &birthdayEntry{
				UserID:     m.SenderID(),
				Day:        day,
				Month:      month,
				Year:       year,
				LastChatID: m.ChatID(),
				Name:       m.Sender.FirstName,
			}
			if err := saveBirthday(entry); err != nil {
				m.Reply("<b>Failed to save birthday.</b>")
				return nil
			}
			m.Reply(fmt.Sprintf("<b>Birthday saved:</b> %s", formatBirthdayDate(day, month, year)))
			return nil
		}
		m.Reply("<b>Unknown subcommand.</b>\n" +
			"Use <code>/bday set DD-MM</code>, <code>/bday clear</code>, or <code>/bday check [@user]</code>")
		return nil
	}
}

func UpcomingBirthdaysHandler(m *tg.NewMessage) error {
	entries, err := listAllBirthdays()
	if err != nil || len(entries) == 0 {
		m.Reply("<b>No birthdays saved yet.</b>")
		return nil
	}
	now := time.Now().UTC()
	sort.Slice(entries, func(i, j int) bool {
		return daysUntilBirthday(entries[i].Day, entries[i].Month, now) < daysUntilBirthday(entries[j].Day, entries[j].Month, now)
	})

	var sb strings.Builder
	sb.WriteString("<b>Upcoming Birthdays</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━\n")
	limit := 5
	if len(entries) < limit {
		limit = len(entries)
	}
	for i := 0; i < limit; i++ {
		e := entries[i]
		days := daysUntilBirthday(e.Day, e.Month, now)
		name := e.Name
		if name == "" {
			name = strconv.FormatInt(e.UserID, 10)
		}
		when := fmt.Sprintf("in %d day(s)", days)
		if days == 0 {
			when = "today"
		} else if days == 1 {
			when = "tomorrow"
		}
		sb.WriteString(fmt.Sprintf(" • <b>%s</b> — %s (%s)\n", html.EscapeString(name), formatBirthdayDate(e.Day, e.Month, e.Year), when))
	}
	m.Reply(sb.String())
	return nil
}

func BirthdayActivityTracker(m *tg.NewMessage) error {
	if m.SenderID() == 0 {
		return nil
	}
	name := ""
	if m.Sender != nil {
		name = m.Sender.FirstName
	}
	updateLastActiveChat(m.SenderID(), m.ChatID(), name)
	return nil
}

func birthdayTicker() {
	for {
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(next))
		runBirthdayCheck()
	}
}

func runBirthdayCheck() {
	if Client == nil {
		return
	}
	entries, err := listAllBirthdays()
	if err != nil || len(entries) == 0 {
		return
	}
	now := time.Now().UTC()
	todayKey := now.Format("2006-01-02")
	for _, e := range entries {
		if e.Day != now.Day() || e.Month != int(now.Month()) {
			continue
		}
		if e.LastSent == todayKey {
			continue
		}
		if e.LastChatID == 0 {
			continue
		}
		name := e.Name
		if name == "" {
			name = "friend"
		}
		mention := fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", e.UserID, html.EscapeString(name))
		text := fmt.Sprintf("🎂 Happy birthday %s!", mention)
		if e.Year > 0 {
			age := now.Year() - e.Year
			if age > 0 && age < 130 {
				text = fmt.Sprintf("🎂 Happy %dth birthday %s!", age, mention)
			}
		}
		_, sendErr := Client.SendMessage(e.LastChatID, text)
		if sendErr == nil {
			e.LastSent = todayKey
			_ = saveBirthday(e)
		}
	}
}

func registerBirthdayHandlers() {
	c := Client
	c.On("cmd:bday", BirthdayHandler)
	c.On("cmd:birthday", BirthdayHandler)
	c.On("cmd:upcoming", UpcomingBirthdaysHandler)
	c.On(tg.OnNewMessage, BirthdayActivityTracker)
	go birthdayTicker()

	Mods.AddModule("Birthday", `<b>Birthday Module</b>

<b>Commands:</b>
 • /bday set DD-MM or DD-MM-YYYY - Save your birthday
 • /bday clear - Remove your birthday
 • /bday check [@user] - Check a birthday
 • /upcoming - Next 5 upcoming birthdays
 • /birthday - Alias for /bday

<i>Daily wishes are sent at 09:00 UTC to your last active chat.</i>`)
}

func init() {
	QueueHandlerRegistration(registerBirthdayHandlers)
}
