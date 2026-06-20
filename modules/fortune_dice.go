package modules

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	fortuneDiceRng   = rand.New(rand.NewSource(time.Now().UnixNano()))
	fortuneDiceRngMu sync.Mutex
)

func fortuneDiceRoll() int {
	fortuneDiceRngMu.Lock()
	defer fortuneDiceRngMu.Unlock()
	return fortuneDiceRng.Intn(6) + 1
}

func fortuneDiceFace(v int) string {
	faces := []string{"", "⚀", "⚁", "⚂", "⚃", "⚄", "⚅"}
	if v >= 1 && v <= 6 {
		return faces[v]
	}
	return "?"
}

func fortuneDiceCounts(dice [5]int) [7]int {
	var counts [7]int
	for _, d := range dice {
		counts[d]++
	}
	return counts
}

func fortuneDiceSum(dice [5]int) int {
	s := 0
	for _, d := range dice {
		s += d
	}
	return s
}

func fortuneDiceCombo(dice [5]int) (string, int) {
	counts := fortuneDiceCounts(dice)
	sum := fortuneDiceSum(dice)

	hasFive := false
	hasFour := false
	hasThree := false
	pairs := 0
	for i := 1; i <= 6; i++ {
		switch counts[i] {
		case 5:
			hasFive = true
		case 4:
			hasFour = true
		case 3:
			hasThree = true
		case 2:
			pairs++
		}
	}

	uniq := make([]int, 0, 6)
	for i := 1; i <= 6; i++ {
		if counts[i] > 0 {
			uniq = append(uniq, i)
		}
	}
	sort.Ints(uniq)

	largeStraight := false
	if len(uniq) == 5 {
		ok := true
		for i := 1; i < len(uniq); i++ {
			if uniq[i] != uniq[i-1]+1 {
				ok = false
				break
			}
		}
		largeStraight = ok
	}

	smallStraight := false
	if !largeStraight {
		run := 1
		for i := 1; i < len(uniq); i++ {
			if uniq[i] == uniq[i-1]+1 {
				run++
				if run >= 4 {
					smallStraight = true
					break
				}
			} else {
				run = 1
			}
		}
	}

	switch {
	case hasFive:
		return "Yatzy! (5 of a Kind)", 50
	case largeStraight:
		return "Large Straight", 40
	case smallStraight:
		return "Small Straight", 30
	case hasFour:
		return "4 of a Kind", sum
	case hasThree && pairs == 1:
		return "Full House", 25
	case hasThree:
		return "3 of a Kind", sum
	case pairs == 2:
		return "Two Pair", sum
	case pairs == 1:
		return "One Pair", sum
	}
	return "Chance", sum
}

func FortuneDiceHandler(m *tg.NewMessage) error {
	var dice [5]int
	for i := 0; i < 5; i++ {
		dice[i] = fortuneDiceRoll()
	}

	combo, score := fortuneDiceCombo(dice)

	var faces strings.Builder
	var values strings.Builder
	for i, d := range dice {
		faces.WriteString(fortuneDiceFace(d))
		faces.WriteString(" ")
		if i > 0 {
			values.WriteString(" ")
		}
		values.WriteString(fmt.Sprintf("%d", d))
	}

	reply := fmt.Sprintf("<b>Fortune Dice</b>\n%s\n<b>Values:</b> <code>%s</code>\n<b>Sum:</b> <code>%d</code>\n<b>Combo:</b> <b>%s</b>\n<b>Score:</b> <code>%d</code>",
		strings.TrimRight(faces.String(), " "),
		values.String(),
		fortuneDiceSum(dice),
		combo,
		score,
	)
	m.Reply(reply)
	return nil
}

func registerFortuneDiceHandlers() {
	c := Client
	c.On("cmd:dicegame", FortuneDiceHandler)
}

func init() {
	QueueHandlerRegistration(registerFortuneDiceHandlers)
}
