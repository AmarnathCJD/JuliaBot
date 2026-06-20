package modules

import (
	"fmt"
	"html"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type yahtzeeGame struct {
	Dice      [5]int
	Kept      [5]bool
	RollsUsed int
	Scored    map[string]int
	Total     int
	mu        sync.Mutex
}

var (
	yahtzeeGames sync.Map
	yahtzeeRng   = rand.New(rand.NewSource(time.Now().UnixNano()))
	yahtzeeRngMu sync.Mutex
)

var yahtzeeCategories = []string{
	"ones", "twos", "threes", "fours", "fives", "sixes",
	"3kind", "4kind", "fullhouse", "smstraight", "lgstraight", "yahtzee", "chance",
}

func yahtzeeKey(chatID, userID int64) string {
	return strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(userID, 10)
}

func yahtzeeRollOne() int {
	yahtzeeRngMu.Lock()
	defer yahtzeeRngMu.Unlock()
	return yahtzeeRng.Intn(6) + 1
}

func yahtzeeRollDice(g *yahtzeeGame, first bool) {
	for i := 0; i < 5; i++ {
		if first || !g.Kept[i] {
			g.Dice[i] = yahtzeeRollOne()
		}
	}
	g.RollsUsed++
}

func yahtzeeDiceFace(v int) string {
	faces := []string{"", "⚀", "⚁", "⚂", "⚃", "⚄", "⚅"}
	if v >= 1 && v <= 6 {
		return faces[v]
	}
	return "?"
}

func yahtzeeRenderDice(g *yahtzeeGame) string {
	var sb strings.Builder
	sb.WriteString("<b>Dice:</b> ")
	for i := 0; i < 5; i++ {
		if g.Kept[i] {
			sb.WriteString(fmt.Sprintf("<u>%s</u>", yahtzeeDiceFace(g.Dice[i])))
		} else {
			sb.WriteString(yahtzeeDiceFace(g.Dice[i]))
		}
		sb.WriteString(" ")
	}
	sb.WriteString("\n<b>Values:</b> <code>")
	vals := make([]string, 5)
	for i, v := range g.Dice {
		vals[i] = fmt.Sprintf("[%d]%d", i+1, v)
	}
	sb.WriteString(strings.Join(vals, " "))
	sb.WriteString("</code>")
	return sb.String()
}

func yahtzeeCounts(dice [5]int) [7]int {
	var counts [7]int
	for _, d := range dice {
		counts[d]++
	}
	return counts
}

func yahtzeeSum(dice [5]int) int {
	s := 0
	for _, d := range dice {
		s += d
	}
	return s
}

func yahtzeeScoreCategory(dice [5]int, cat string) (int, bool) {
	counts := yahtzeeCounts(dice)
	switch cat {
	case "ones":
		return counts[1] * 1, true
	case "twos":
		return counts[2] * 2, true
	case "threes":
		return counts[3] * 3, true
	case "fours":
		return counts[4] * 4, true
	case "fives":
		return counts[5] * 5, true
	case "sixes":
		return counts[6] * 6, true
	case "3kind":
		for i := 1; i <= 6; i++ {
			if counts[i] >= 3 {
				return yahtzeeSum(dice), true
			}
		}
		return 0, true
	case "4kind":
		for i := 1; i <= 6; i++ {
			if counts[i] >= 4 {
				return yahtzeeSum(dice), true
			}
		}
		return 0, true
	case "fullhouse":
		hasThree := false
		hasTwo := false
		for i := 1; i <= 6; i++ {
			if counts[i] == 3 {
				hasThree = true
			} else if counts[i] == 2 {
				hasTwo = true
			}
		}
		if hasThree && hasTwo {
			return 25, true
		}
		return 0, true
	case "smstraight":
		sorted := make([]int, 0, 6)
		for i := 1; i <= 6; i++ {
			if counts[i] > 0 {
				sorted = append(sorted, i)
			}
		}
		sort.Ints(sorted)
		run := 1
		for i := 1; i < len(sorted); i++ {
			if sorted[i] == sorted[i-1]+1 {
				run++
				if run >= 4 {
					return 30, true
				}
			} else {
				run = 1
			}
		}
		return 0, true
	case "lgstraight":
		sorted := make([]int, 0, 6)
		for i := 1; i <= 6; i++ {
			if counts[i] > 0 {
				sorted = append(sorted, i)
			}
		}
		if len(sorted) < 5 {
			return 0, true
		}
		run := 1
		for i := 1; i < len(sorted); i++ {
			if sorted[i] == sorted[i-1]+1 {
				run++
				if run >= 5 {
					return 40, true
				}
			} else {
				run = 1
			}
		}
		return 0, true
	case "yahtzee":
		for i := 1; i <= 6; i++ {
			if counts[i] == 5 {
				return 50, true
			}
		}
		return 0, true
	case "chance":
		return yahtzeeSum(dice), true
	}
	return 0, false
}

func yahtzeeCategoryList() string {
	return "ones, twos, threes, fours, fives, sixes, 3kind, 4kind, fullhouse, smstraight, lgstraight, yahtzee, chance"
}

func yahtzeeScoreCard(g *yahtzeeGame) string {
	var sb strings.Builder
	sb.WriteString("<b>Scorecard:</b>\n<pre>")
	for _, cat := range yahtzeeCategories {
		if v, ok := g.Scored[cat]; ok {
			sb.WriteString(fmt.Sprintf("%-12s %3d\n", cat, v))
		} else {
			sb.WriteString(fmt.Sprintf("%-12s   -\n", cat))
		}
	}
	sb.WriteString(fmt.Sprintf("%-12s %3d", "TOTAL", g.Total))
	sb.WriteString("</pre>")
	return sb.String()
}

func YahtzeeHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(strings.ToLower(m.Args()))
	chatID := m.ChatID()
	userID := m.SenderID()
	key := yahtzeeKey(chatID, userID)

	fields := strings.Fields(args)
	sub := ""
	if len(fields) > 0 {
		sub = fields[0]
	}

	switch sub {
	case "", "help":
		m.Reply("<b>Yahtzee</b>\n" +
			"<code>/yahtzee start</code> - begin a new turn (rolls 5 dice)\n" +
			"<code>/yahtzee keep 1 3 5</code> - keep dice at those indexes and reroll the rest\n" +
			"<code>/yahtzee score &lt;category&gt;</code> - score current dice into a category\n" +
			"<code>/yahtzee show</code> - show current dice and scorecard\n" +
			"<code>/yahtzee end</code> - abort the current game\n\n" +
			"<b>Categories:</b> <code>" + yahtzeeCategoryList() + "</code>\n" +
			"You get up to 3 rolls per turn before you must score.")
		return nil

	case "start":
		v, exists := yahtzeeGames.Load(key)
		if exists {
			g := v.(*yahtzeeGame)
			g.mu.Lock()
			finished := len(g.Scored) >= len(yahtzeeCategories)
			g.mu.Unlock()
			if !finished {
				m.Reply("You already have an active turn. Use <code>/yahtzee keep</code>, <code>/yahtzee score</code>, or <code>/yahtzee end</code>.")
				return nil
			}
			yahtzeeGames.Delete(key)
		}
		g := &yahtzeeGame{
			Scored: make(map[string]int),
		}
		yahtzeeRollDice(g, true)
		yahtzeeGames.Store(key, g)
		m.Reply(fmt.Sprintf("<b>Yahtzee started!</b>\n%s\nRolls used: <b>%d/3</b>\nUse <code>/yahtzee keep &lt;idxs&gt;</code> or <code>/yahtzee score &lt;cat&gt;</code>.", yahtzeeRenderDice(g), g.RollsUsed))
		return nil

	case "keep":
		v, ok := yahtzeeGames.Load(key)
		if !ok {
			m.Reply("No active turn. Use <code>/yahtzee start</code>.")
			return nil
		}
		g := v.(*yahtzeeGame)
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.RollsUsed >= 3 {
			m.Reply("You have used all 3 rolls. Use <code>/yahtzee score &lt;cat&gt;</code>.")
			return nil
		}
		if g.RollsUsed == 0 {
			m.Reply("Roll first with <code>/yahtzee start</code>.")
			return nil
		}
		for i := 0; i < 5; i++ {
			g.Kept[i] = false
		}
		idxs := fields[1:]
		if len(idxs) == 0 {
			m.Reply("Usage: <code>/yahtzee keep 1 3 5</code> (1-5 indexes to keep).")
			return nil
		}
		for _, s := range idxs {
			n, err := strconv.Atoi(s)
			if err != nil || n < 1 || n > 5 {
				m.Reply("Indexes must be integers from 1 to 5.")
				return nil
			}
			g.Kept[n-1] = true
		}
		yahtzeeRollDice(g, false)
		m.Reply(fmt.Sprintf("<b>Rerolled.</b>\n%s\nRolls used: <b>%d/3</b>", yahtzeeRenderDice(g), g.RollsUsed))
		return nil

	case "score":
		v, ok := yahtzeeGames.Load(key)
		if !ok {
			m.Reply("No active turn. Use <code>/yahtzee start</code>.")
			return nil
		}
		g := v.(*yahtzeeGame)
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.RollsUsed == 0 {
			m.Reply("Roll first with <code>/yahtzee start</code>.")
			return nil
		}
		if len(fields) < 2 {
			m.Reply("Usage: <code>/yahtzee score &lt;category&gt;</code>\nCategories: <code>" + yahtzeeCategoryList() + "</code>")
			return nil
		}
		cat := fields[1]
		score, valid := yahtzeeScoreCategory(g.Dice, cat)
		if !valid {
			m.Reply("Unknown category. Choose from: <code>" + yahtzeeCategoryList() + "</code>")
			return nil
		}
		if _, used := g.Scored[cat]; used {
			m.Reply("You already scored <code>" + html.EscapeString(cat) + "</code> this game.")
			return nil
		}
		g.Scored[cat] = score
		g.Total += score
		reply := fmt.Sprintf("<b>Scored %d</b> in <code>%s</code>.\n%s", score, html.EscapeString(cat), yahtzeeScoreCard(g))
		if len(g.Scored) >= len(yahtzeeCategories) {
			reply += fmt.Sprintf("\n\n<b>Game over!</b> Final total: <b>%d</b>", g.Total)
			yahtzeeGames.Delete(key)
		} else {
			g.RollsUsed = 0
			for i := 0; i < 5; i++ {
				g.Kept[i] = false
				g.Dice[i] = 0
			}
			reply += "\n\nUse <code>/yahtzee start</code> to roll the next turn."
		}
		m.Reply(reply)
		return nil

	case "show":
		v, ok := yahtzeeGames.Load(key)
		if !ok {
			m.Reply("No active game. Use <code>/yahtzee start</code>.")
			return nil
		}
		g := v.(*yahtzeeGame)
		g.mu.Lock()
		defer g.mu.Unlock()
		out := yahtzeeScoreCard(g)
		if g.RollsUsed > 0 {
			out = yahtzeeRenderDice(g) + "\nRolls used: <b>" + strconv.Itoa(g.RollsUsed) + "/3</b>\n\n" + out
		} else {
			out = "<i>No active roll. Use </i><code>/yahtzee start</code><i> to roll.</i>\n\n" + out
		}
		m.Reply(out)
		return nil

	case "end", "stop", "abort":
		v, ok := yahtzeeGames.LoadAndDelete(key)
		if !ok {
			m.Reply("No active Yahtzee game.")
			return nil
		}
		g := v.(*yahtzeeGame)
		m.Reply(fmt.Sprintf("<b>Yahtzee aborted.</b> Final total: <b>%d</b>\n%s", g.Total, yahtzeeScoreCard(g)))
		return nil

	default:
		m.Reply("Unknown subcommand. Use <code>/yahtzee help</code>.")
		return nil
	}
}

func registerYahtzeeHandlers() {
	c := Client
	c.On("cmd:yahtzee", YahtzeeHandler)
}

func init() {
	QueueHandlerRegistration(registerYahtzeeHandlers)
}
