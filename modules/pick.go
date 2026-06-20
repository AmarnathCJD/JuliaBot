package modules

import (
	"fmt"
	"html"
	"math/rand"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	pickRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	pickMu  sync.Mutex
)

func pickIntn(n int) int {
	if n <= 0 {
		return 0
	}
	pickMu.Lock()
	defer pickMu.Unlock()
	return pickRng.Intn(n)
}

func pickFormatName(u *tg.UserObj) string {
	if u == nil {
		return "user"
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		if u.Username != "" {
			name = "@" + u.Username
		} else {
			name = fmt.Sprintf("user %d", u.ID)
		}
	}
	return name
}

func pickMention(u *tg.UserObj) string {
	name := pickFormatName(u)
	return fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", u.ID, html.EscapeString(name))
}

func PickOneHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/pickone a b c ...</code>")
		return nil
	}
	fields := strings.Fields(arg)
	if len(fields) < 2 {
		m.Reply("<b>Error:</b> provide at least 2 options.")
		return nil
	}
	if len(fields) > 200 {
		m.Reply("<b>Error:</b> too many options (max 200).")
		return nil
	}
	pick := fields[pickIntn(len(fields))]
	m.Reply(fmt.Sprintf("<b>Pick</b>\nOptions: <code>%d</code>\nPicked: <code>%s</code>",
		len(fields), html.EscapeString(pick)))
	return nil
}

func RandMemberHandler(m *tg.NewMessage) error {
	if !m.IsGroup() {
		m.Reply("<b>This command works in groups only.</b>")
		return nil
	}
	parts, _, err := m.Client.GetChatMembers(m.ChatID(), &tg.ParticipantOptions{
		Filter: &tg.ChannelParticipantsRecent{},
		Limit:  200,
	})
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch members.")
		return nil
	}
	candidates := make([]*tg.UserObj, 0, len(parts))
	for _, p := range parts {
		if p == nil || p.User == nil {
			continue
		}
		if p.User.Bot || p.User.Deleted {
			continue
		}
		candidates = append(candidates, p.User)
	}
	if len(candidates) == 0 {
		m.Reply("<b>Error:</b> no eligible members found.")
		return nil
	}
	pick := candidates[pickIntn(len(candidates))]
	m.Reply(fmt.Sprintf("<b>Random Member</b>\n%s", pickMention(pick)))
	return nil
}

func RandAdminHandler(m *tg.NewMessage) error {
	if !m.IsGroup() {
		m.Reply("<b>This command works in groups only.</b>")
		return nil
	}
	parts, _, err := m.Client.GetChatMembers(m.ChatID(), &tg.ParticipantOptions{
		Filter: &tg.ChannelParticipantsAdmins{},
		Limit:  200,
	})
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch admins.")
		return nil
	}
	candidates := make([]*tg.UserObj, 0, len(parts))
	for _, p := range parts {
		if p == nil || p.User == nil {
			continue
		}
		if p.User.Bot || p.User.Deleted {
			continue
		}
		if p.Status != tg.Admin && p.Status != tg.Creator {
			continue
		}
		candidates = append(candidates, p.User)
	}
	if len(candidates) == 0 {
		m.Reply("<b>Error:</b> no eligible admins found.")
		return nil
	}
	pick := candidates[pickIntn(len(candidates))]
	m.Reply(fmt.Sprintf("<b>Random Admin</b>\n%s", pickMention(pick)))
	return nil
}

func registerPickHandlers() {
	c := Client
	c.On("cmd:pickone", PickOneHandler)
	c.On("cmd:randmember", RandMemberHandler)
	c.On("cmd:randadmin", RandAdminHandler)
}

func init() {
	QueueHandlerRegistration(registerPickHandlers)

	Mods.AddModule("Pick", `<b>Pick Module</b>

Random picker utilities.

<b>Commands:</b>
 • /pickone a b c ... - Pick one item randomly from args
 • /randmember - Pick a random group member (skips bots)
 • /randadmin - Pick a random group admin (skips bots)`)
}
