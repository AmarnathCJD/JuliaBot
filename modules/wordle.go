package modules

import (
	"fmt"
	"html"
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

var wordleWords4 = []string{
	"able", "acid", "ages", "also", "area", "army", "away", "baby", "back", "ball",
	"band", "bank", "base", "bath", "bear", "beat", "been", "beer", "bell", "belt",
	"best", "bike", "bill", "bird", "blow", "blue", "boat", "body", "bomb", "bond",
	"bone", "book", "boom", "born", "boss", "both", "bowl", "bulk", "burn", "bush",
	"busy", "call", "calm", "came", "camp", "card", "care", "case", "cash", "cast",
	"cell", "chat", "chip", "city", "club", "coal", "coat", "code", "cold", "come",
	"cook", "cool", "cope", "copy", "core", "cost", "crew", "crop", "dark", "data",
	"date", "dawn", "days", "dead", "deal", "dean", "dear", "debt", "deep", "deny",
	"desk", "dial", "dick", "diet", "dirt", "disc", "disk", "does", "done", "door",
	"dose", "down", "draw", "drew", "drop", "drug", "dual", "duke", "dust", "duty",
	"each", "earn", "ease", "east", "easy", "edge", "else", "even", "ever", "evil",
	"exit", "face", "fact", "fail", "fair", "fall", "farm", "fast", "fate", "fear",
	"feed", "feel", "feet", "fell", "felt", "file", "fill", "film", "find", "fine",
	"fire", "firm", "fish", "five", "flat", "flow", "food", "foot", "ford", "form",
	"fort", "four", "free", "from", "fuel", "full", "fund", "gain", "game", "gate",
	"gave", "gear", "gene", "gift", "girl", "give", "glad", "goal", "goes", "gold",
	"gone", "good", "gray", "grew", "grey", "grow", "gulf", "hair", "half", "hall",
	"hand", "hang", "hard", "harm", "hate", "have", "head", "hear", "heat", "held",
	"help", "here", "hero", "high", "hill", "hire", "hold", "hole", "holy", "home",
	"hope", "host", "hour", "huge", "hung", "hunt", "hurt", "icon", "idea", "inch",
}

var wordleWords5 = []string{
	"about", "above", "abuse", "actor", "acute", "admit", "adopt", "adult", "after", "again",
	"agent", "agree", "ahead", "alarm", "album", "alert", "alike", "alive", "allow", "alone",
	"along", "alter", "among", "anger", "angle", "angry", "apart", "apple", "apply", "arena",
	"argue", "arise", "array", "aside", "asset", "audio", "audit", "avoid", "award", "aware",
	"badly", "baker", "bases", "basic", "basis", "beach", "began", "begin", "begun", "being",
	"below", "bench", "billy", "birth", "black", "blame", "blind", "block", "blood", "board",
	"boost", "booth", "bound", "brain", "brand", "bread", "break", "breed", "brief", "bring",
	"broad", "broke", "brown", "build", "built", "buyer", "cable", "calif", "carry", "catch",
	"cause", "chain", "chair", "chart", "chase", "cheap", "check", "chest", "chief", "child",
	"china", "chose", "civil", "claim", "class", "clean", "clear", "click", "clock", "close",
	"coach", "coast", "could", "count", "court", "cover", "craft", "crash", "cream", "crime",
	"cross", "crowd", "crown", "curve", "cycle", "daily", "dance", "dated", "dealt", "death",
	"debut", "delay", "depth", "doing", "doubt", "dozen", "draft", "drama", "drawn", "dream",
	"dress", "drill", "drink", "drive", "drove", "dying", "eager", "early", "earth", "eight",
	"elite", "empty", "enemy", "enjoy", "enter", "entry", "equal", "error", "event", "every",
	"exact", "exist", "extra", "faith", "false", "fault", "fiber", "field", "fifth", "fifty",
	"fight", "final", "first", "fixed", "flash", "fleet", "floor", "fluid", "focus", "force",
	"forth", "forty", "forum", "found", "frame", "frank", "fraud", "fresh", "front", "fruit",
	"fully", "funny", "giant", "given", "glass", "globe", "going", "grace", "grade", "grand",
	"grant", "grass", "great", "green", "gross", "group", "grown", "guard", "guess", "guest",
	"guide", "happy", "harry", "heart", "heavy", "hence", "henry", "horse", "hotel", "house",
	"human", "ideal", "image", "index", "inner", "input", "issue", "japan", "jimmy", "joint",
	"jones", "judge", "known", "label", "large", "laser", "later", "laugh", "layer", "learn",
	"lease", "least", "leave", "legal", "level", "lewis", "light", "limit", "links", "lives",
	"local", "logic", "loose", "lower", "lucky", "lunch", "lying", "magic", "major", "maker",
	"march", "maria", "match", "maybe", "mayor", "meant", "media", "metal", "might", "minor",
	"minus", "mixed", "model", "money", "month", "moral", "motor", "mount", "mouse", "mouth",
	"movie", "music", "needs", "never", "newly", "night", "noise", "north", "noted", "novel",
	"nurse", "occur", "ocean", "offer", "often", "order", "other", "ought", "paint", "panel",
	"paper", "party", "peace", "peter", "phase", "phone", "photo", "piece", "pilot", "pitch",
}

var wordleWords6 = []string{
	"abroad", "accept", "access", "across", "acting", "action", "active", "actual", "advice", "advise",
	"affect", "afford", "afraid", "agency", "agenda", "almost", "always", "amount", "anchor", "animal",
	"annual", "answer", "anyone", "anyway", "appeal", "appear", "around", "arrest", "arrive", "artist",
	"aspect", "assert", "assess", "assign", "assist", "assume", "assure", "attach", "attack", "attend",
	"august", "author", "autumn", "backup", "barely", "battle", "beauty", "became", "become", "before",
	"behalf", "behave", "behind", "belief", "belong", "berlin", "better", "beyond", "bishop", "border",
	"bottle", "bottom", "bought", "branch", "breath", "bridge", "bright", "broken", "budget", "burden",
	"bureau", "button", "camera", "cancer", "cannot", "carbon", "career", "castle", "casual", "caught",
	"center", "centre", "chance", "change", "charge", "choice", "choose", "chosen", "church", "circle",
	"client", "closed", "closer", "coffee", "column", "combat", "coming", "common", "comply", "copper",
	"corner", "costly", "county", "couple", "course", "covers", "create", "credit", "crisis", "custom",
	"damage", "danger", "dating", "dealer", "debate", "decade", "decide", "defeat", "defend", "define",
	"degree", "demand", "depend", "deputy", "desert", "design", "desire", "detail", "detect", "device",
	"differ", "dinner", "direct", "doctor", "dollar", "domain", "double", "driven", "driver", "during",
	"easily", "eating", "editor", "effect", "effort", "either", "eleven", "emerge", "empire", "employ",
	"enable", "ending", "energy", "engage", "engine", "enough", "ensure", "entire", "entity", "equity",
	"escape", "estate", "ethnic", "exceed", "except", "excess", "expand", "expect", "expert", "export",
	"extend", "extent", "fabric", "facing", "factor", "failed", "fairly", "fallen", "family", "famous",
	"father", "fellow", "female", "figure", "filing", "finger", "finish", "fiscal", "flight", "flying",
	"follow", "forced", "forest", "forget", "formal", "format", "former", "foster", "fought", "fourth",
}

type wordleGame struct {
	Word    string
	Guesses []string
	N       int
}

var (
	wordleGames sync.Map
	wordleRng   = rand.New(rand.NewSource(time.Now().UnixNano()))
	wordleMu    sync.Mutex
)

func wordlePickWord(length int) string {
	wordleMu.Lock()
	defer wordleMu.Unlock()
	var list []string
	switch length {
	case 4:
		list = wordleWords4
	case 6:
		list = wordleWords6
	default:
		list = wordleWords5
	}
	return strings.ToUpper(list[wordleRng.Intn(len(list))])
}

func wordleFontPath(name string) string {
	candidates := []string{
		"./assets/" + name,
		"assets/" + name,
		"../assets/" + name,
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "assets", name),
			filepath.Join(dir, "..", "assets", name),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "assets", name))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func wordleLoadFont(dc *gg.Context, size float64) {
	p := wordleFontPath("Inter_28pt-Bold.ttf")
	if p != "" {
		if err := dc.LoadFontFace(p, size); err == nil {
			return
		}
	}
	p = wordleFontPath("Swiss 721 Black Extended BT.ttf")
	if p != "" {
		if err := dc.LoadFontFace(p, size); err == nil {
			return
		}
	}
	dc.SetFontFace(basicfont.Face7x13)
}

func wordleScore(guess, target string) []int {
	n := len(target)
	res := make([]int, n)
	tgt := []rune(target)
	gss := []rune(guess)
	used := make([]bool, n)
	for i := 0; i < n; i++ {
		if gss[i] == tgt[i] {
			res[i] = 2
			used[i] = true
		}
	}
	for i := 0; i < n; i++ {
		if res[i] == 2 {
			continue
		}
		for j := 0; j < n; j++ {
			if !used[j] && gss[i] == tgt[j] {
				res[i] = 1
				used[j] = true
				break
			}
		}
	}
	return res
}

func wordleDrawGradientBg(dc *gg.Context, w, h int) {
	a := color.RGBA{0x1a, 0x1a, 0x2e, 0xff}
	b := color.RGBA{0x16, 0x21, 0x3e, 0xff}
	diag := float64(w + h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(x+y) / diag
			t = t * t * (3 - 2*t)
			r := float64(a.R)*(1-t) + float64(b.R)*t
			g := float64(a.G)*(1-t) + float64(b.G)*t
			bb := float64(a.B)*(1-t) + float64(b.B)*t
			dc.SetRGB255(int(r), int(g), int(bb))
			dc.SetPixel(x, y)
		}
	}
}

func wordleDrawRoundedTile(dc *gg.Context, x, y, size float64, fill color.RGBA, letter string) {
	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawRoundedRectangle(x+4, y+6, size, size, 12)
	dc.Fill()

	dc.SetRGB255(int(fill.R), int(fill.G), int(fill.B))
	dc.DrawRoundedRectangle(x, y, size, size, 12)
	dc.Fill()

	dc.SetRGBA(1, 1, 1, 0.08)
	dc.DrawRoundedRectangle(x, y, size, size/3, 12)
	dc.Fill()

	if letter != "" {
		wordleLoadFont(dc, size*0.55)
		dc.SetRGBA(0, 0, 0, 0.35)
		dc.DrawStringAnchored(letter, x+size/2+2, y+size/2+2, 0.5, 0.5)
		dc.SetRGB(1, 1, 1)
		dc.DrawStringAnchored(letter, x+size/2, y+size/2, 0.5, 0.5)
	}
}

func wordleRender(g *wordleGame, chatName string) (string, error) {
	tileSize := 80.0
	gap := 10.0
	padX := 40.0
	padTop := 110.0
	padBottom := 70.0

	cols := len(g.Word)
	rows := len(g.Guesses)
	if rows < 6 {
		rows = 6
	}

	gridW := float64(cols)*tileSize + float64(cols-1)*gap
	gridH := float64(rows)*tileSize + float64(rows-1)*gap

	w := int(gridW + padX*2)
	if w < 560 {
		w = 560
	}
	h := int(gridH + padTop + padBottom)

	dc := gg.NewContext(w, h)
	wordleDrawGradientBg(dc, w, h)

	header := fmt.Sprintf("WORDLE %d", cols)
	wordleLoadFont(dc, 38)
	dc.SetRGBA(0, 0, 0, 0.4)
	dc.DrawStringAnchored(header, float64(w)/2+2, 42, 0.5, 0.5)
	dc.SetRGB255(0xff, 0xff, 0xff)
	dc.DrawStringAnchored(header, float64(w)/2, 40, 0.5, 0.5)

	if chatName != "" {
		wordleLoadFont(dc, 18)
		dc.SetRGBA(1, 1, 1, 0.7)
		dc.DrawStringAnchored(chatName, float64(w)/2, 72, 0.5, 0.5)
	}

	startX := (float64(w) - gridW) / 2
	startY := padTop

	green := color.RGBA{0x6A, 0xAA, 0x64, 0xff}
	yellow := color.RGBA{0xC9, 0xB4, 0x58, 0xff}
	grey := color.RGBA{0x78, 0x7C, 0x7E, 0xff}
	empty := color.RGBA{0x3A, 0x3A, 0x3C, 0xff}

	for row := 0; row < rows; row++ {
		y := startY + float64(row)*(tileSize+gap)
		if row < len(g.Guesses) {
			guess := g.Guesses[row]
			score := wordleScore(guess, g.Word)
			for col := 0; col < cols; col++ {
				x := startX + float64(col)*(tileSize+gap)
				var fill color.RGBA
				switch score[col] {
				case 2:
					fill = green
				case 1:
					fill = yellow
				default:
					fill = grey
				}
				wordleDrawRoundedTile(dc, x, y, tileSize, fill, string(guess[col]))
			}
		} else {
			for col := 0; col < cols; col++ {
				x := startX + float64(col)*(tileSize+gap)
				wordleDrawRoundedTile(dc, x, y, tileSize, empty, "")
			}
		}
	}

	footer := fmt.Sprintf("Guess #%d  |  Type /guess WORD", len(g.Guesses)+1)
	wordleLoadFont(dc, 20)
	dc.SetRGBA(1, 1, 1, 0.75)
	dc.DrawStringAnchored(footer, float64(w)/2, float64(h)-30, 0.5, 0.5)

	out := filepath.Join(os.TempDir(), fmt.Sprintf("wordle_%d_%d.png", g.N, time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func wordleChatName(m *tg.NewMessage) string {
	if m.Chat != nil && m.Chat.Title != "" {
		return m.Chat.Title
	}
	return ""
}

func WordleHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(strings.ToLower(m.Args()))
	chatID := m.ChatID()

	if args == "end" || args == "stop" || args == "abort" {
		if v, ok := wordleGames.LoadAndDelete(chatID); ok {
			g := v.(*wordleGame)
			m.Reply(fmt.Sprintf("<b>Wordle ended.</b> The word was <code>%s</code>.", html.EscapeString(g.Word)))
		} else {
			m.Reply("No active Wordle game in this chat.")
		}
		return nil
	}

	length := 5
	if args == "4" {
		length = 4
	} else if args == "6" {
		length = 6
	} else if args == "5" {
		length = 5
	} else if args != "" {
		m.Reply("Usage: <code>/wordle [4|5|6]</code> to start, <code>/wordle end</code> to abort.")
		return nil
	}

	if _, ok := wordleGames.Load(chatID); ok {
		m.Reply("A Wordle game is already running. Use <code>/guess &lt;word&gt;</code> or <code>/wordle end</code>.")
		return nil
	}

	g := &wordleGame{
		Word:    wordlePickWord(length),
		Guesses: []string{},
		N:       length,
	}
	wordleGames.Store(chatID, g)

	chatName := wordleChatName(m)
	out, err := wordleRender(g, chatName)
	if err != nil {
		m.Reply("render failed: " + html.EscapeString(err.Error()))
		return nil
	}
	defer os.Remove(out)

	caption := fmt.Sprintf("<b>Wordle %d started!</b> Guess with <code>/guess &lt;word&gt;</code>.", length)
	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  caption,
		FileName: "wordle.png",
		MimeType: "image/png",
	})
	if merr != nil {
		m.Reply("upload failed: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func GuessHandler(m *tg.NewMessage) error {
	chatID := m.ChatID()
	v, ok := wordleGames.Load(chatID)
	if !ok {
		m.Reply("No active Wordle game. Start one with <code>/wordle [4|5|6]</code>.")
		return nil
	}
	g := v.(*wordleGame)

	arg := strings.TrimSpace(strings.ToUpper(m.Args()))
	if arg == "" {
		m.Reply(fmt.Sprintf("Usage: <code>/guess &lt;%d-letter word&gt;</code>", g.N))
		return nil
	}
	if len(arg) != g.N {
		m.Reply(fmt.Sprintf("Word must be exactly <b>%d</b> letters.", g.N))
		return nil
	}
	for _, r := range arg {
		if r < 'A' || r > 'Z' {
			m.Reply("Word must contain only letters a-z.")
			return nil
		}
	}

	g.Guesses = append(g.Guesses, arg)
	chatName := wordleChatName(m)

	out, err := wordleRender(g, chatName)
	if err != nil {
		m.Reply("render failed: " + html.EscapeString(err.Error()))
		return nil
	}
	defer os.Remove(out)

	won := arg == g.Word
	var caption string
	if won {
		wordleGames.Delete(chatID)
		caption = fmt.Sprintf("<b>You won!</b> Solved in <b>%d</b> guesses. The word was <code>%s</code>.", len(g.Guesses), html.EscapeString(g.Word))
	} else {
		caption = fmt.Sprintf("Guess <b>#%d</b>: <code>%s</code> — keep going!", len(g.Guesses), html.EscapeString(arg))
	}

	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  caption,
		FileName: "wordle.png",
		MimeType: "image/png",
	})
	if merr != nil {
		m.Reply("upload failed: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerWordleHandlers() {
	c := Client
	c.On("cmd:wordle", WordleHandler)
	c.On("cmd:guess", GuessHandler)
}

func init() { QueueHandlerRegistration(registerWordleHandlers) }
