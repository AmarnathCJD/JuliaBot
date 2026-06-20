package modules

import (
	"fmt"
	"html"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var wordLadderDict = []string{
	"cat", "cot", "dot", "dog", "log", "bog", "bag", "bat", "bit", "big",
	"bug", "but", "hut", "hat", "hit", "hot", "lot", "lit", "fit", "fat",
	"fan", "man", "men", "ten", "tin", "win", "wig", "wag", "wad", "had",
	"hay", "bay", "ban", "bin", "pin", "pit", "pat", "pet", "pen", "ten",
	"ton", "ten", "den", "din", "dim", "rim", "ram", "raw", "row", "low",
	"law", "lay", "say", "sat", "set", "see", "bee", "beg", "bed", "red",
	"led", "let", "lot", "lop", "top", "tap", "map", "mat", "mad", "sad",
	"sap", "sip", "sin", "son", "sun", "fun", "run", "rug", "rag", "rat",
	"ear", "far", "for", "fog", "foe", "toe", "tow", "now", "new", "few",
	"fey", "key", "kit", "bit", "bid", "bad", "bud", "mud", "mug", "jug",
	"jog", "job", "mob", "rob", "rib", "ribs", "ride", "rude",
	"cold", "cord", "card", "ward", "warm", "worm", "word", "ford", "fort",
	"port", "post", "past", "pest", "test", "rest", "best", "bust", "bust",
	"head", "heat", "heal", "seal", "deal", "real", "read", "load", "road",
	"toad", "told", "tolt", "tilt", "till", "tall", "tale", "tame", "time",
	"lime", "line", "lane", "land", "band", "bond", "bend", "send", "sand",
	"hand", "hard", "harm", "warm", "ware", "wore", "wire", "fire", "five",
	"give", "live", "love", "dove", "dive", "dime", "dome", "home", "hope",
	"rope", "rose", "rise", "ride", "rode", "rude", "ruse", "muse", "mute",
	"cute", "cube", "tube", "tune", "tone", "tons", "tops", "tips", "ties",
	"like", "lake", "make", "mace", "race", "rice", "nice", "nine", "mine",
	"wine", "wind", "wand", "want", "rant", "rent", "tent", "tint", "lint",
	"link", "pink", "pine", "vine", "vane", "cane", "came", "case", "cast",
	"cost", "host", "most", "must", "bust", "busy", "bury", "burn", "barn",
	"born", "corn", "core", "bore", "bone", "bone", "gone", "gore", "gory",
	"glory",
	"heart", "heard", "beard", "bears", "bears", "years", "wears", "wares",
	"hares", "bares", "bores", "cores", "cares", "dares", "dates", "gates",
	"rates", "rites", "rides", "ridge", "bride", "pride", "price", "prize",
	"prose", "proud", "cloud", "clout", "clear", "clean", "lean", "learn",
	"earth", "earns", "yearn", "burns", "burst", "burnt", "blunt", "plant",
	"plait", "plain", "stain", "stair", "stare", "store", "stove", "stone",
	"shone", "shore", "shire", "share", "shape", "shame", "shams",
	"slime", "slim", "slip", "ship", "shop", "shot", "shut", "shun",
	"flame", "flake", "fluke", "flute", "fruit", "front", "frost", "first",
	"thirst", "thirty",
	"black", "block", "clock", "click", "stick", "stack", "shack", "shock",
	"smock", "smoke", "spoke", "spore", "spare", "scare", "score", "scope",
	"slope", "slate", "skate", "state", "stake", "snake", "shake", "shade",
	"shape", "shaft", "shift", "swift", "swirl",
	"world", "would", "could", "mould", "moult", "vault", "fault",
	"bread", "break", "creak", "creek", "cheek", "cheep", "sheep", "sleep",
	"sleek", "sleet", "fleet", "fleer", "flier", "flies", "fries", "tries",
	"trees", "frees", "freed", "greed", "green", "preen", "prone", "drone",
	"drove", "grove", "grave", "brave", "brake", "brace", "trace", "track",
	"trick", "trunk", "drunk", "drink",
}

func buildWordLadderDict(length int) map[string]bool {
	out := map[string]bool{}
	for _, w := range wordLadderDict {
		if len(w) == length {
			out[strings.ToLower(w)] = true
		}
	}
	return out
}

func wordLadderDiff(a, b string) int {
	d := 0
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			d++
		}
	}
	return d
}

func findWordLadder(start, end string, dict map[string]bool) []string {
	if start == end {
		return []string{start}
	}
	if !dict[start] || !dict[end] {
		return nil
	}
	type node struct {
		word string
		path []string
	}
	visited := map[string]bool{start: true}
	queue := []node{{start, []string{start}}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if len(cur.path) > 12 {
			continue
		}
		bytes := []byte(cur.word)
		for i := 0; i < len(bytes); i++ {
			orig := bytes[i]
			for c := byte('a'); c <= 'z'; c++ {
				if c == orig {
					continue
				}
				bytes[i] = c
				candidate := string(bytes)
				if dict[candidate] && !visited[candidate] {
					newPath := make([]string, len(cur.path)+1)
					copy(newPath, cur.path)
					newPath[len(cur.path)] = candidate
					if candidate == end {
						return newPath
					}
					visited[candidate] = true
					queue = append(queue, node{candidate, newPath})
				}
			}
			bytes[i] = orig
		}
	}
	return nil
}

func WordLadderHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/wordladder &lt;start&gt; &lt;end&gt;</code>\nBoth words must be the same length (3-5 letters).")
		return nil
	}
	parts := strings.Fields(args)
	if len(parts) < 2 {
		m.Reply("<b>Usage:</b> <code>/wordladder &lt;start&gt; &lt;end&gt;</code>\nProvide two words separated by a space.")
		return nil
	}
	start := strings.ToLower(parts[0])
	end := strings.ToLower(parts[1])
	if len(start) != len(end) {
		m.Reply(fmt.Sprintf("<b>Error:</b> words must be the same length. Got <code>%s</code> (%d) and <code>%s</code> (%d).",
			html.EscapeString(start), len(start), html.EscapeString(end), len(end)))
		return nil
	}
	if len(start) < 3 || len(start) > 5 {
		m.Reply("<b>Error:</b> word length must be between 3 and 5 letters.")
		return nil
	}
	for _, r := range start + end {
		if r < 'a' || r > 'z' {
			m.Reply("<b>Error:</b> only lowercase letters a-z are allowed.")
			return nil
		}
	}
	dict := buildWordLadderDict(len(start))
	if !dict[start] {
		m.Reply(fmt.Sprintf("<b>Error:</b> <code>%s</code> is not in the built-in dictionary.", html.EscapeString(start)))
		return nil
	}
	if !dict[end] {
		m.Reply(fmt.Sprintf("<b>Error:</b> <code>%s</code> is not in the built-in dictionary.", html.EscapeString(end)))
		return nil
	}
	if start == end {
		m.Reply(fmt.Sprintf("<b>Word Ladder</b>\n\n<i>Start equals End.</i>\n\n<code>%s</code>", html.EscapeString(start)))
		return nil
	}
	path := findWordLadder(start, end, dict)
	if path == nil {
		m.Reply(fmt.Sprintf("<b>No ladder found</b> between <code>%s</code> and <code>%s</code> in the built-in dictionary.",
			html.EscapeString(start), html.EscapeString(end)))
		return nil
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<b>Word Ladder</b>\n\n<b>From:</b> <code>%s</code>\n<b>To:</b> <code>%s</code>\n<b>Steps:</b> %d\n\n",
		html.EscapeString(start), html.EscapeString(end), len(path)-1))
	for i, w := range path {
		marker := "  "
		if i == 0 {
			marker = "<b>Start</b>"
		} else if i == len(path)-1 {
			marker = "<b>End</b>"
		} else {
			marker = fmt.Sprintf("Step %d", i)
		}
		diffStr := ""
		if i > 0 {
			diffStr = fmt.Sprintf(" <i>(changed %d letter)</i>", wordLadderDiff(path[i-1], w))
		}
		b.WriteString(fmt.Sprintf("%s: <code>%s</code>%s\n", marker, html.EscapeString(w), diffStr))
	}
	m.Reply(b.String())
	return nil
}

func registerWordLadderHandlers() {
	c := Client
	c.On("cmd:wordladder", WordLadderHandler)
}

func init() {
	QueueHandlerRegistration(registerWordLadderHandlers)
}
