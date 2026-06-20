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

const (
	shootMaxHP      = 3
	shootHitPercent = 60
	shootAcceptTTL  = 30 * time.Second
)

type shootGame struct {
	ChallengerID   int64
	ChallengerName string
	OpponentID     int64
	OpponentName   string
	HP             map[int64]int
	Turn           int64
	Accepted       bool
	CreatedAt      time.Time
	mu             sync.Mutex
}

var (
	shootGames sync.Map
	shootRng   = rand.New(rand.NewSource(time.Now().UnixNano()))
	shootRngMu sync.Mutex
)

func shootRoll() bool {
	shootRngMu.Lock()
	defer shootRngMu.Unlock()
	return shootRng.Intn(100) < shootHitPercent
}

func shootRenderHP(g *shootGame) string {
	bar := func(hp int) string {
		if hp < 0 {
			hp = 0
		}
		return strings.Repeat("❤️", hp) + strings.Repeat("\U0001f494", shootMaxHP-hp)
	}
	return fmt.Sprintf("<b>%s</b>: %s (%d HP)\n<b>%s</b>: %s (%d HP)",
		html.EscapeString(g.ChallengerName), bar(g.HP[g.ChallengerID]), g.HP[g.ChallengerID],
		html.EscapeString(g.OpponentName), bar(g.HP[g.OpponentID]), g.HP[g.OpponentID])
}

func shootResolveTarget(m *tg.NewMessage) (int64, string, error) {
	if m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil && r != nil {
			uid := r.SenderID()
			name := "Player"
			if u, _ := m.Client.GetUser(uid); u != nil {
				name = u.FirstName
			}
			return uid, name, nil
		}
	}
	args := strings.Fields(m.Args())
	if len(args) == 0 {
		return 0, "", fmt.Errorf("missing target")
	}
	target := strings.TrimPrefix(args[0], "@")
	ent, err := m.Client.ResolveUsername(target)
	if err != nil {
		return 0, "", err
	}
	switch v := ent.(type) {
	case *tg.UserObj:
		name := v.FirstName
		if name == "" {
			name = v.Username
		}
		return int64(v.ID), name, nil
	}
	return 0, "", fmt.Errorf("not a user")
}

func ShootHandler(m *tg.NewMessage) error {
	chatID := m.ChatID()
	args := strings.TrimSpace(strings.ToLower(m.Args()))
	fields := strings.Fields(args)
	sub := ""
	if len(fields) > 0 {
		sub = fields[0]
	}

	switch sub {
	case "status":
		v, ok := shootGames.Load(chatID)
		if !ok {
			m.Reply("No active duel. Start one with <code>/shoot @user</code>.")
			return nil
		}
		g := v.(*shootGame)
		g.mu.Lock()
		defer g.mu.Unlock()
		if !g.Accepted {
			remain := shootAcceptTTL - time.Since(g.CreatedAt)
			if remain < 0 {
				remain = 0
			}
			m.Reply(fmt.Sprintf("<b>Pending duel</b>\n<b>%s</b> challenged <b>%s</b>.\n<i>Awaiting </i><code>/accept</code><i> (%ds left).</i>",
				html.EscapeString(g.ChallengerName), html.EscapeString(g.OpponentName), int(remain.Seconds())))
			return nil
		}
		turnName := g.ChallengerName
		if g.Turn == g.OpponentID {
			turnName = g.OpponentName
		}
		m.Reply(fmt.Sprintf("<b>Duel in progress</b>\n%s\n\n<b>Turn:</b> %s\nUse <code>/fire</code>.",
			shootRenderHP(g), html.EscapeString(turnName)))
		return nil

	case "quit", "end", "stop", "abort":
		v, ok := shootGames.LoadAndDelete(chatID)
		if !ok {
			m.Reply("No active duel to quit.")
			return nil
		}
		g := v.(*shootGame)
		m.Reply(fmt.Sprintf("<b>Duel cancelled.</b>\n%s vs %s",
			html.EscapeString(g.ChallengerName), html.EscapeString(g.OpponentName)))
		return nil
	}

	if _, exists := shootGames.Load(chatID); exists {
		m.Reply("A duel is already underway in this chat. Use <code>/shoot status</code>, <code>/fire</code>, or <code>/shoot quit</code>.")
		return nil
	}

	if !m.IsReply() && len(fields) == 0 {
		m.Reply("Usage: <code>/shoot @user</code> or reply to a user with <code>/shoot</code>.\nSubcommands: <code>status</code>, <code>quit</code>.")
		return nil
	}

	targetID, targetName, err := shootResolveTarget(m)
	if err != nil || targetID == 0 {
		m.Reply("Could not resolve target user. Reply to them or mention <code>@username</code>.")
		return nil
	}

	challengerID := m.SenderID()
	if targetID == challengerID {
		m.Reply("You cannot duel yourself.")
		return nil
	}

	challengerName := "Challenger"
	if u, _ := m.Client.GetUser(challengerID); u != nil {
		challengerName = u.FirstName
	}

	g := &shootGame{
		ChallengerID:   challengerID,
		ChallengerName: challengerName,
		OpponentID:     targetID,
		OpponentName:   targetName,
		HP:             map[int64]int{challengerID: shootMaxHP, targetID: shootMaxHP},
		Turn:           challengerID,
		Accepted:       false,
		CreatedAt:      time.Now(),
	}
	shootGames.Store(chatID, g)

	go func(cid int64, key int64) {
		time.Sleep(shootAcceptTTL)
		if v, ok := shootGames.Load(cid); ok {
			gg := v.(*shootGame)
			gg.mu.Lock()
			if !gg.Accepted && gg.CreatedAt.Equal(g.CreatedAt) {
				gg.mu.Unlock()
				shootGames.Delete(cid)
				return
			}
			gg.mu.Unlock()
		}
	}(chatID, challengerID)

	m.Reply(fmt.Sprintf("<b>%s</b> challenges <b>%s</b> to a duel!\nHP: <b>%d</b> each, hit rate: <b>%d%%</b>.\n<b>%s</b>, reply with <code>/accept</code> within %ds.",
		html.EscapeString(challengerName), html.EscapeString(targetName),
		shootMaxHP, shootHitPercent,
		html.EscapeString(targetName), int(shootAcceptTTL.Seconds())))
	return nil
}

func AcceptHandler(m *tg.NewMessage) error {
	chatID := m.ChatID()
	v, ok := shootGames.Load(chatID)
	if !ok {
		m.Reply("No pending duel here.")
		return nil
	}
	g := v.(*shootGame)
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Accepted {
		m.Reply("Duel already started. Use <code>/fire</code>.")
		return nil
	}
	if time.Since(g.CreatedAt) > shootAcceptTTL {
		shootGames.Delete(chatID)
		m.Reply("Challenge expired.")
		return nil
	}
	if m.SenderID() != g.OpponentID {
		m.Reply("Only the challenged player may accept.")
		return nil
	}
	g.Accepted = true
	g.Turn = g.ChallengerID
	m.Reply(fmt.Sprintf("<b>Duel accepted!</b>\n%s\n\n<b>%s</b> shoots first. Use <code>/fire</code>.",
		shootRenderHP(g), html.EscapeString(g.ChallengerName)))
	return nil
}

func FireHandler(m *tg.NewMessage) error {
	chatID := m.ChatID()
	v, ok := shootGames.Load(chatID)
	if !ok {
		m.Reply("No active duel. Start with <code>/shoot @user</code>.")
		return nil
	}
	g := v.(*shootGame)
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.Accepted {
		m.Reply("Duel not yet accepted. Waiting on <code>/accept</code>.")
		return nil
	}
	uid := m.SenderID()
	if uid != g.ChallengerID && uid != g.OpponentID {
		m.Reply("You are not in this duel.")
		return nil
	}
	if uid != g.Turn {
		m.Reply("It is not your turn.")
		return nil
	}

	var shooterName, targetName string
	var targetID int64
	if uid == g.ChallengerID {
		shooterName = g.ChallengerName
		targetID = g.OpponentID
		targetName = g.OpponentName
	} else {
		shooterName = g.OpponentName
		targetID = g.ChallengerID
		targetName = g.ChallengerName
	}

	hit := shootRoll()
	var resultLine string
	if hit {
		g.HP[targetID]--
		if g.HP[targetID] < 0 {
			g.HP[targetID] = 0
		}
		resultLine = fmt.Sprintf("\U0001f3af <b>HIT!</b> %s shoots %s for 1 damage.",
			html.EscapeString(shooterName), html.EscapeString(targetName))
	} else {
		resultLine = fmt.Sprintf("\U0001f4a8 <b>MISS!</b> %s fires at %s and misses.",
			html.EscapeString(shooterName), html.EscapeString(targetName))
	}

	if g.HP[targetID] <= 0 {
		shootGames.Delete(chatID)
		m.Reply(fmt.Sprintf("%s\n\n%s\n\n<b>%s wins the duel!</b>",
			resultLine, shootRenderHP(g), html.EscapeString(shooterName)))
		return nil
	}

	g.Turn = targetID
	nextName := targetName
	m.Reply(fmt.Sprintf("%s\n\n%s\n\n<b>Turn:</b> %s — use <code>/fire</code>.",
		resultLine, shootRenderHP(g), html.EscapeString(nextName)))
	return nil
}

func registerShootHandlers() {
	c := Client
	c.On("cmd:shoot", ShootHandler)
	c.On("cmd:accept", AcceptHandler)
	c.On("cmd:fire", FireHandler)
}

func init() {
	QueueHandlerRegistration(registerShootHandlers)
}
