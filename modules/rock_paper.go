package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
)

type RPSStats struct {
	UserID int64  `json:"uid"`
	Name   string `json:"n"`
	Wins   int64  `json:"w"`
	Losses int64  `json:"l"`
	Draws  int64  `json:"d"`
}

var (
	rpsBucket = []byte("rps_stats")
	rpsRng    = rand.New(rand.NewSource(time.Now().UnixNano()))
	rpsMu     sync.Mutex
)

var rpsChoices = []string{"rock", "paper", "scissors"}

var rpsEmoji = map[string]string{
	"rock":     "Rock",
	"paper":    "Paper",
	"scissors": "Scissors",
}

func rpsEnsureBucket() error {
	d, err := db.GetDB()
	if err != nil {
		return err
	}
	return d.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(rpsBucket)
		return e
	})
}

func rpsLoadStats(uid int64) *RPSStats {
	d, err := db.GetDB()
	if err != nil {
		return &RPSStats{UserID: uid}
	}
	if err := rpsEnsureBucket(); err != nil {
		return &RPSStats{UserID: uid}
	}
	var out *RPSStats
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(rpsBucket)
		if b == nil {
			return nil
		}
		raw := b.Get([]byte(strconv.FormatInt(uid, 10)))
		if raw == nil {
			return nil
		}
		var s RPSStats
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil
		}
		out = &s
		return nil
	})
	if out == nil {
		out = &RPSStats{UserID: uid}
	}
	return out
}

func rpsSaveStats(s *RPSStats) error {
	d, err := db.GetDB()
	if err != nil {
		return err
	}
	if err := rpsEnsureBucket(); err != nil {
		return err
	}
	return d.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(rpsBucket)
		if err != nil {
			return err
		}
		data, err := json.Marshal(s)
		if err != nil {
			return err
		}
		return b.Put([]byte(strconv.FormatInt(s.UserID, 10)), data)
	})
}

func rpsAllStats() []*RPSStats {
	d, err := db.GetDB()
	if err != nil {
		return nil
	}
	if err := rpsEnsureBucket(); err != nil {
		return nil
	}
	var out []*RPSStats
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(rpsBucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var s RPSStats
			if err := json.Unmarshal(v, &s); err != nil {
				return nil
			}
			sc := s
			out = append(out, &sc)
			return nil
		})
	})
	return out
}

func rpsSenderName(m *tg.NewMessage) string {
	if m.Sender == nil {
		return strconv.FormatInt(m.SenderID(), 10)
	}
	name := strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
	if name == "" && m.Sender.Username != "" {
		name = m.Sender.Username
	}
	if name == "" {
		name = strconv.FormatInt(m.SenderID(), 10)
	}
	return name
}

func rpsOutcome(user, bot string) string {
	if user == bot {
		return "draw"
	}
	switch user {
	case "rock":
		if bot == "scissors" {
			return "win"
		}
		return "loss"
	case "paper":
		if bot == "rock" {
			return "win"
		}
		return "loss"
	case "scissors":
		if bot == "paper" {
			return "win"
		}
		return "loss"
	}
	return "loss"
}

func RPSHandler(m *tg.NewMessage) error {
	arg := strings.ToLower(strings.TrimSpace(m.Args()))
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/rps &lt;rock|paper|scissors&gt;</code>")
		return nil
	}
	switch arg {
	case "r":
		arg = "rock"
	case "p":
		arg = "paper"
	case "s":
		arg = "scissors"
	}
	if arg != "rock" && arg != "paper" && arg != "scissors" {
		m.Reply("<b>Error:</b> choose one of <code>rock</code>, <code>paper</code>, <code>scissors</code>.")
		return nil
	}

	rpsMu.Lock()
	botPick := rpsChoices[rpsRng.Intn(3)]
	rpsMu.Unlock()

	outcome := rpsOutcome(arg, botPick)

	uid := m.SenderID()
	if uid != 0 {
		rpsMu.Lock()
		s := rpsLoadStats(uid)
		s.UserID = uid
		s.Name = rpsSenderName(m)
		switch outcome {
		case "win":
			s.Wins++
		case "loss":
			s.Losses++
		case "draw":
			s.Draws++
		}
		_ = rpsSaveStats(s)
		rpsMu.Unlock()
	}

	var result string
	switch outcome {
	case "win":
		result = "<b>You win!</b>"
	case "loss":
		result = "<b>You lose!</b>"
	default:
		result = "<b>It's a draw!</b>"
	}

	out := fmt.Sprintf("<b>Rock Paper Scissors</b>\nYou: <code>%s</code>\nBot: <code>%s</code>\n%s",
		rpsEmoji[arg], rpsEmoji[botPick], result)
	m.Reply(out)
	return nil
}

func RPSStatsHandler(m *tg.NewMessage) error {
	uid := m.SenderID()
	if uid == 0 {
		m.Reply("<b>Error:</b> could not identify user.")
		return nil
	}
	rpsMu.Lock()
	s := rpsLoadStats(uid)
	rpsMu.Unlock()

	total := s.Wins + s.Losses + s.Draws
	if total == 0 {
		m.Reply("<b>No RPS games played yet.</b> Try <code>/rps rock</code>.")
		return nil
	}

	winRate := float64(0)
	if total > 0 {
		winRate = float64(s.Wins) / float64(total) * 100
	}

	name := rpsSenderName(m)
	out := fmt.Sprintf("<b>RPS Stats for %s</b>\n━━━━━━━━━━━━━━━━\n", html.EscapeString(strings.TrimPrefix(name, "@")))
	out += fmt.Sprintf(" • Wins: <code>%d</code>\n", s.Wins)
	out += fmt.Sprintf(" • Losses: <code>%d</code>\n", s.Losses)
	out += fmt.Sprintf(" • Draws: <code>%d</code>\n", s.Draws)
	out += fmt.Sprintf(" • Total: <code>%d</code>\n", total)
	out += fmt.Sprintf(" • Win Rate: <code>%.1f%%</code>", winRate)
	m.Reply(out)
	return nil
}

func RPSLeaderboardHandler(m *tg.NewMessage) error {
	rpsMu.Lock()
	all := rpsAllStats()
	rpsMu.Unlock()

	if len(all) == 0 {
		m.Reply("<b>No RPS games recorded yet.</b>")
		return nil
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].Wins != all[j].Wins {
			return all[i].Wins > all[j].Wins
		}
		return all[i].Losses < all[j].Losses
	})

	limit := 10
	if len(all) < limit {
		limit = len(all)
	}

	var b strings.Builder
	b.WriteString("<b>RPS Leaderboard (Top 10 by Wins)</b>\n")
	b.WriteString("━━━━━━━━━━━━━━━━\n")
	for i := 0; i < limit; i++ {
		s := all[i]
		name := s.Name
		if name == "" {
			name = strconv.FormatInt(s.UserID, 10)
		}
		total := s.Wins + s.Losses + s.Draws
		b.WriteString(fmt.Sprintf(" %d. %s — W: <code>%d</code> L: <code>%d</code> D: <code>%d</code> (Total: <code>%d</code>)\n",
			i+1, html.EscapeString(strings.TrimPrefix(name, "@")), s.Wins, s.Losses, s.Draws, total))
	}

	m.Reply(b.String())
	return nil
}

func registerRPSHandlers() {
	c := Client
	c.On("cmd:rps", RPSHandler)
	c.On("cmd:rpsstats", RPSStatsHandler)
	c.On("cmd:rpsleaderboard", RPSLeaderboardHandler)
}

func init() {
	QueueHandlerRegistration(registerRPSHandlers)

	Mods.AddModule("RockPaperScissors", `<b>Rock Paper Scissors Module</b>

Play Rock Paper Scissors against the bot and track your stats.

<b>Commands:</b>
 • /rps &lt;rock|paper|scissors&gt; - Play a round against the bot
 • /rpsstats - Show your win/loss/draw record
 • /rpsleaderboard - Top 10 players by wins

<i>Stats are persisted per user across all chats.</i>`)
}
