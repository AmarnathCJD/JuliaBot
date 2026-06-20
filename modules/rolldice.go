package modules

import (
	"fmt"
	"html"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	rdRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	rdMu  sync.Mutex
)

func rdIntn(n int) int {
	if n <= 0 {
		return 0
	}
	rdMu.Lock()
	defer rdMu.Unlock()
	return rdRng.Intn(n)
}

func parseDiceSpec(spec string) (int, int, error) {
	spec = strings.ToLower(strings.TrimSpace(spec))
	if spec == "" {
		return 0, 0, fmt.Errorf("empty")
	}
	idx := strings.Index(spec, "d")
	if idx < 0 {
		return 0, 0, fmt.Errorf("missing d")
	}
	count := 1
	if idx > 0 {
		n, err := strconv.Atoi(spec[:idx])
		if err != nil || n < 1 {
			return 0, 0, fmt.Errorf("bad count")
		}
		count = n
	}
	sidesStr := spec[idx+1:]
	sides, err := strconv.Atoi(sidesStr)
	if err != nil || sides < 2 {
		return 0, 0, fmt.Errorf("bad sides")
	}
	if count > 50 {
		return 0, 0, fmt.Errorf("too many dice (max 50)")
	}
	if sides > 1000 {
		return 0, 0, fmt.Errorf("too many sides (max 1000)")
	}
	return count, sides, nil
}

func RollHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		arg = "d6"
	}
	count, sides, err := parseDiceSpec(arg)
	if err != nil {
		m.Reply("<b>Usage:</b> <code>/roll d6</code> or <code>/roll 3d20</code>\n<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}

	rolls := make([]int, count)
	total := 0
	for i := 0; i < count; i++ {
		r := rdIntn(sides) + 1
		rolls[i] = r
		total += r
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<b>Rolling %dd%d</b>\n", count, sides))
	if count == 1 {
		b.WriteString(fmt.Sprintf("Result: <code>%d</code>", rolls[0]))
	} else {
		parts := make([]string, count)
		for i, r := range rolls {
			parts[i] = strconv.Itoa(r)
		}
		b.WriteString("Rolls: <code>[")
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("]</code>\n")
		b.WriteString(fmt.Sprintf("Total: <code>%d</code>", total))
	}
	m.Reply(b.String())
	return nil
}

func FlipHandler(m *tg.NewMessage) error {
	face := "Heads"
	if rdIntn(2) == 1 {
		face = "Tails"
	}
	m.Reply(fmt.Sprintf("<b>Coin Flip</b>\nResult: <code>%s</code>", face))
	return nil
}

func ChooseHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/choose A | B | C</code>")
		return nil
	}
	raw := strings.Split(arg, "|")
	opts := make([]string, 0, len(raw))
	for _, o := range raw {
		o = strings.TrimSpace(o)
		if o != "" {
			opts = append(opts, o)
		}
	}
	if len(opts) < 2 {
		m.Reply("<b>Error:</b> need at least 2 options separated by <code>|</code>.")
		return nil
	}
	pick := opts[rdIntn(len(opts))]
	m.Reply(fmt.Sprintf("<b>Choose</b>\nOptions: <code>%d</code>\nPick: <code>%s</code>",
		len(opts), html.EscapeString(pick)))
	return nil
}

func RandomNumHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/random N</code> (returns int between 1 and N)")
		return nil
	}
	n, err := strconv.Atoi(arg)
	if err != nil || n < 1 {
		m.Reply("<b>Error:</b> provide a positive integer.")
		return nil
	}
	if n > 1000000000 {
		m.Reply("<b>Error:</b> N too large (max 1000000000).")
		return nil
	}
	v := rdIntn(n) + 1
	m.Reply(fmt.Sprintf("<b>Random 1..%d</b>\nResult: <code>%d</code>", n, v))
	return nil
}

func ShuffleHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/shuffle A,B,C,D</code>")
		return nil
	}
	raw := strings.Split(arg, ",")
	items := make([]string, 0, len(raw))
	for _, o := range raw {
		o = strings.TrimSpace(o)
		if o != "" {
			items = append(items, o)
		}
	}
	if len(items) < 2 {
		m.Reply("<b>Error:</b> need at least 2 items separated by commas.")
		return nil
	}
	if len(items) > 200 {
		m.Reply("<b>Error:</b> too many items (max 200).")
		return nil
	}

	rdMu.Lock()
	rdRng.Shuffle(len(items), func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})
	rdMu.Unlock()

	escaped := make([]string, len(items))
	for i, it := range items {
		escaped[i] = html.EscapeString(it)
	}
	m.Reply(fmt.Sprintf("<b>Shuffled (%d)</b>\n<code>%s</code>",
		len(items), strings.Join(escaped, ", ")))
	return nil
}

func registerRollDiceHandlers() {
	c := Client
	c.On("cmd:roll", RollHandler)
	c.On("cmd:flip", FlipHandler)
	c.On("cmd:choose", ChooseHandler)
	c.On("cmd:random", RandomNumHandler)
	c.On("cmd:shuffle", ShuffleHandler)
}

func init() {
	QueueHandlerRegistration(registerRollDiceHandlers)

	Mods.AddModule("RollDice", `<b>RollDice Module</b>

Random utilities: dice, coin, picker, shuffler.

<b>Commands:</b>
 • /roll [NdS] - Roll dice (e.g. <code>d6</code>, <code>3d20</code>)
 • /flip - Flip a coin
 • /choose A | B | C - Pick one option randomly
 • /random N - Random integer 1..N
 • /shuffle A,B,C,D - Shuffle a comma list`)
}
