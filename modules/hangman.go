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

var hangmanWords = []string{
	"apple", "actor", "above", "agree", "alarm", "album", "alert", "alive", "alone", "amber",
	"angel", "angle", "angry", "ankle", "apart", "april", "arena", "argue", "armor", "arrow",
	"asset", "audio", "audit", "avoid", "awake", "award", "aware", "awful", "baker", "balance",
	"ballot", "banana", "banner", "barrel", "basic", "basket", "battery", "battle", "beach", "beard",
	"beast", "beauty", "begin", "below", "bench", "berry", "between", "beyond", "bicycle", "binary",
	"birth", "bishop", "black", "blade", "blame", "blank", "blast", "blend", "bless", "blind",
	"block", "blood", "bloom", "blues", "board", "boast", "bonus", "border", "bottle", "bottom",
	"bound", "boxer", "brain", "branch", "brand", "brass", "brave", "bread", "break", "breath",
	"breeze", "brick", "bride", "bridge", "brief", "bright", "bring", "broken", "bronze", "brown",
	"brush", "bubble", "bucket", "buddy", "budget", "buffalo", "bullet", "bundle", "burden", "butter",
	"button", "buyer", "cabin", "cable", "calendar", "camera", "campaign", "candle", "candy", "canvas",
	"canyon", "carbon", "career", "carpet", "carrot", "castle", "casual", "cattle", "center", "cereal",
	"chair", "chalk", "champion", "chance", "change", "chapter", "charge", "charity", "charm", "chart",
	"chase", "cheap", "check", "cheese", "chemistry", "cherry", "chest", "chicken", "chief", "child",
	"choice", "choose", "chronic", "circle", "citizen", "clarify", "class", "classic", "clean", "clear",
	"clever", "client", "cliff", "climate", "climb", "clinic", "clock", "close", "cloth", "cloud",
	"clover", "clown", "cluster", "coach", "coast", "coconut", "coffee", "cognitive", "coil", "color",
	"column", "combine", "comedy", "comfort", "comic", "command", "common", "company", "compass", "compete",
	"compile", "complex", "concept", "concert", "conduct", "confirm", "connect", "consider", "constant", "contact",
	"contain", "content", "contest", "context", "control", "convince", "cookie", "copper", "coral", "corner",
	"correct", "cosmic", "cotton", "couch", "council", "country", "couple", "courage", "course", "cousin",
	"cover", "coyote", "crack", "cradle", "craft", "crane", "crash", "crater", "crawl", "crazy",
	"cream", "credit", "creek", "crime", "crisp", "critic", "crop", "cross", "crowd", "crown",
	"crucial", "cruise", "crumble", "crush", "crystal", "cube", "cucumber", "cultural", "curious", "current",
	"curtain", "curve", "cushion", "custom", "cycle", "daily", "damage", "dance", "danger", "daring",
	"dawn", "deadline", "debate", "decade", "decide", "decline", "decode", "deer", "defense", "define",
	"degree", "delay", "deliver", "demand", "demise", "denial", "dentist", "depart", "depend", "deposit",
	"depth", "deputy", "derive", "describe", "desert", "design", "desire", "desk", "detail", "detect",
	"develop", "device", "devote", "diagram", "dial", "diamond", "diary", "dice", "diesel", "diet",
	"differ", "digital", "dignity", "dilemma", "dinner", "diploma", "direct", "dirt", "disagree", "discover",
	"disease", "dish", "dismiss", "disorder", "display", "distance", "divert", "divide", "divine", "doctor",
	"document", "donate", "donkey", "donor", "double", "doubt", "dough", "dove", "draft", "dragon",
	"drama", "drastic", "draw", "dream", "dress", "drift", "drill", "drink", "drive", "drop",
	"drum", "duck", "dumb", "dune", "duration", "during", "dust", "dutch", "duty", "dwarf",
	"dynamic", "eager", "eagle", "early", "earn", "earth", "easily", "east", "easy", "echo",
	"ecology", "economy", "edge", "edit", "educate", "effort", "egg", "eight", "either", "elbow",
	"elder", "electric", "elegant", "element", "elephant", "elevator", "elite", "embark", "embody", "embrace",
	"emerge", "emotion", "employ", "empower", "empty", "enable", "enact", "endless", "endorse", "enemy",
	"energy", "enforce", "engage", "engine", "enhance", "enjoy", "enlist", "enough", "enrich", "enroll",
	"ensure", "enter", "entire", "entry", "envelope", "episode", "equal", "equip", "erase", "erode",
	"erosion", "error", "erupt", "escape", "essay", "essence", "estate", "eternal", "ethics", "evidence",
	"evil", "evoke", "evolve", "exact", "example", "excess", "exchange", "excite", "exclude", "excuse",
	"execute", "exercise", "exhaust", "exhibit", "exile", "exist", "exit", "exotic", "expand", "expect",
	"expire", "explain", "expose", "express", "extend", "extra", "eyebrow", "fabric", "face", "faculty",
	"fade", "faint", "faith", "false", "fame", "family", "famous", "fancy", "fantasy", "farm",
	"fashion", "fat", "fatal", "father", "fatigue", "fault", "favor", "feature", "federal", "fee",
	"feed", "feel", "female", "fence", "festival", "fetch", "fever", "few", "fiber", "fiction",
	"field", "figure", "file", "film", "filter", "final", "find", "fine", "finger", "finish",
	"fire", "firm", "first", "fiscal", "fish", "fit", "fitness", "fix", "flag", "flame",
	"flash", "flat", "flavor", "flee", "flight", "flip", "float", "flock", "floor", "flower",
	"fluid", "flush", "fly", "foam", "focus", "fog", "foil", "fold", "follow", "food",
}

const hangmanMaxWrong = 7

type hangmanGame struct {
	Word    string
	Wrong   []rune
	Correct map[rune]bool
}

var (
	hangmanGames sync.Map
	hangmanRng   = rand.New(rand.NewSource(time.Now().UnixNano()))
	hangmanMu    sync.Mutex
)

func pickHangmanWord() string {
	hangmanMu.Lock()
	defer hangmanMu.Unlock()
	return hangmanWords[hangmanRng.Intn(len(hangmanWords))]
}

var hangmanStages = []string{
	`  +---+
  |   |
      |
      |
      |
      |
=========`,
	`  +---+
  |   |
  O   |
      |
      |
      |
=========`,
	`  +---+
  |   |
  O   |
  |   |
      |
      |
=========`,
	`  +---+
  |   |
  O   |
 /|   |
      |
      |
=========`,
	`  +---+
  |   |
  O   |
 /|\  |
      |
      |
=========`,
	`  +---+
  |   |
  O   |
 /|\  |
 /    |
      |
=========`,
	`  +---+
  |   |
  O   |
 /|\  |
 / \  |
      |
=========`,
	`  +---+
  |   |
 [O   |
 /|\  |
 / \  |
      |
=========`,
}

func hangmanMaskedWord(g *hangmanGame) string {
	var b strings.Builder
	for i, r := range g.Word {
		if i > 0 {
			b.WriteByte(' ')
		}
		if g.Correct[r] {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func hangmanRenderBoard(g *hangmanGame, reveal bool, header string) string {
	stage := len(g.Wrong)
	if stage > hangmanMaxWrong {
		stage = hangmanMaxWrong
	}
	masked := hangmanMaskedWord(g)
	if reveal {
		masked = strings.Join(strings.Split(g.Word, ""), " ")
	}
	wrongDisplay := "-"
	if len(g.Wrong) > 0 {
		parts := make([]string, len(g.Wrong))
		for i, r := range g.Wrong {
			parts[i] = strings.ToUpper(string(r))
		}
		wrongDisplay = strings.Join(parts, " ")
	}
	remaining := hangmanMaxWrong - len(g.Wrong)
	if remaining < 0 {
		remaining = 0
	}
	var sb strings.Builder
	if header != "" {
		sb.WriteString(header)
		sb.WriteString("\n")
	}
	sb.WriteString("<pre>")
	sb.WriteString(html.EscapeString(hangmanStages[stage]))
	sb.WriteString("</pre>\n")
	sb.WriteString("Word: <code>")
	sb.WriteString(html.EscapeString(strings.ToUpper(masked)))
	sb.WriteString("</code>\n")
	sb.WriteString(fmt.Sprintf("Wrong: <code>%s</code>\n", html.EscapeString(wrongDisplay)))
	sb.WriteString(fmt.Sprintf("Remaining: <b>%d</b> / %d\n", remaining, hangmanMaxWrong))
	sb.WriteString("Guess with <code>/letter &lt;a-z&gt;</code> or end with <code>/hangman end</code>.")
	return sb.String()
}

func HangmanHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(strings.ToLower(m.Args()))
	chatID := m.ChatID()

	if args == "end" || args == "stop" || args == "abort" {
		if v, ok := hangmanGames.LoadAndDelete(chatID); ok {
			g := v.(*hangmanGame)
			m.Reply(hangmanRenderBoard(g, true, fmt.Sprintf("<b>Hangman ended.</b> The word was <code>%s</code>.", html.EscapeString(strings.ToUpper(g.Word)))))
		} else {
			m.Reply("No active Hangman game in this chat.")
		}
		return nil
	}

	if args != "" {
		m.Reply("Usage: <code>/hangman</code> to start, <code>/hangman end</code> to abort.")
		return nil
	}

	if _, ok := hangmanGames.Load(chatID); ok {
		m.Reply("A Hangman game is already running. Use <code>/letter &lt;a-z&gt;</code> to guess or <code>/hangman end</code> to abort.")
		return nil
	}

	g := &hangmanGame{
		Word:    pickHangmanWord(),
		Wrong:   []rune{},
		Correct: map[rune]bool{},
	}
	hangmanGames.Store(chatID, g)

	header := fmt.Sprintf("<b>Hangman started!</b> Word length: <b>%d</b>", len(g.Word))
	m.Reply(hangmanRenderBoard(g, false, header))
	return nil
}

func LetterHandler(m *tg.NewMessage) error {
	chatID := m.ChatID()
	v, ok := hangmanGames.Load(chatID)
	if !ok {
		m.Reply("No active Hangman game. Start one with <code>/hangman</code>.")
		return nil
	}
	g := v.(*hangmanGame)

	arg := strings.TrimSpace(strings.ToLower(m.Args()))
	if arg == "" {
		m.Reply("Usage: <code>/letter &lt;a-z&gt;</code>")
		return nil
	}
	runes := []rune(arg)
	if len(runes) != 1 {
		m.Reply("Guess exactly one letter.")
		return nil
	}
	r := runes[0]
	if r < 'a' || r > 'z' {
		m.Reply("Guess a single letter a-z.")
		return nil
	}

	if g.Correct[r] {
		m.Reply(fmt.Sprintf("Letter <code>%s</code> was already guessed correctly.", html.EscapeString(strings.ToUpper(string(r)))))
		return nil
	}
	for _, w := range g.Wrong {
		if w == r {
			m.Reply(fmt.Sprintf("Letter <code>%s</code> was already guessed wrong.", html.EscapeString(strings.ToUpper(string(r)))))
			return nil
		}
	}

	hit := strings.ContainsRune(g.Word, r)
	if hit {
		g.Correct[r] = true
	} else {
		g.Wrong = append(g.Wrong, r)
	}

	won := true
	for _, wr := range g.Word {
		if !g.Correct[wr] {
			won = false
			break
		}
	}

	if won {
		hangmanGames.Delete(chatID)
		header := fmt.Sprintf("<b>You won!</b> The word was <code>%s</code>.", html.EscapeString(strings.ToUpper(g.Word)))
		m.Reply(hangmanRenderBoard(g, true, header))
		return nil
	}

	if len(g.Wrong) >= hangmanMaxWrong {
		hangmanGames.Delete(chatID)
		header := fmt.Sprintf("<b>You lost!</b> The word was <code>%s</code>.", html.EscapeString(strings.ToUpper(g.Word)))
		m.Reply(hangmanRenderBoard(g, true, header))
		return nil
	}

	var header string
	if hit {
		header = fmt.Sprintf("<b>Hit!</b> Letter <code>%s</code> is in the word.", html.EscapeString(strings.ToUpper(string(r))))
	} else {
		header = fmt.Sprintf("<b>Miss!</b> Letter <code>%s</code> is not in the word.", html.EscapeString(strings.ToUpper(string(r))))
	}
	m.Reply(hangmanRenderBoard(g, false, header))
	return nil
}

func registerHangmanHandlers() {
	c := Client
	c.On("cmd:hangman", HangmanHandler)
	c.On("cmd:letter", LetterHandler)
}

func init() {
	QueueHandlerRegistration(registerHangmanHandlers)
}
