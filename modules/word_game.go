package modules

import (
	"fmt"
	"html"
	"sort"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var scrabbleLetterValues = map[rune]int{
	'a': 1, 'b': 3, 'c': 3, 'd': 2, 'e': 1, 'f': 4, 'g': 2, 'h': 4,
	'i': 1, 'j': 8, 'k': 5, 'l': 1, 'm': 3, 'n': 1, 'o': 1, 'p': 3,
	'q': 10, 'r': 1, 's': 1, 't': 1, 'u': 1, 'v': 4, 'w': 4, 'x': 8,
	'y': 4, 'z': 10,
}

var scrabbleDict = []string{
	"ace", "act", "add", "age", "ago", "aid", "aim", "air", "ale", "all", "and", "ant", "any", "ape", "are", "arm", "art", "ash", "ask", "ate",
	"awe", "axe", "bad", "bag", "ban", "bar", "bat", "bay", "bed", "bee", "beg", "bet", "bid", "big", "bin", "bit", "boa", "bob", "bog", "bow",
	"box", "boy", "bra", "bud", "bug", "bun", "bus", "but", "buy", "bye", "cab", "can", "cap", "car", "cat", "cob", "cod", "cog", "con", "cop",
	"cot", "cow", "coy", "cry", "cub", "cue", "cup", "cur", "cut", "dab", "dad", "dam", "day", "den", "dew", "did", "die", "dig", "dim", "dip",
	"doe", "dog", "don", "dot", "dry", "dub", "due", "dug", "duo", "dye", "ear", "eat", "ebb", "eel", "egg", "ego", "elf", "elk", "elm", "end",
	"era", "ewe", "eye", "fad", "fan", "far", "fat", "fax", "fed", "fee", "few", "fig", "fin", "fir", "fit", "fix", "flu", "fly", "foe", "fog",
	"for", "fox", "fry", "fun", "fur", "gab", "gag", "gal", "gap", "gas", "gel", "gem", "get", "gig", "gin", "god", "got", "gum", "gun", "gut",
	"guy", "gym", "had", "hag", "ham", "has", "hat", "hay", "hem", "hen", "her", "hex", "hey", "hid", "him", "hip", "his", "hit", "hob", "hoe",
	"hog", "hop", "hot", "how", "hub", "hue", "hug", "hum", "hut", "ice", "icy", "ill", "imp", "ink", "inn", "ion", "ire", "ivy", "jab", "jag",
	"jam", "jar", "jaw", "jay", "jet", "jig", "job", "jog", "jot", "joy", "jug", "jut", "keg", "key", "kid", "kin", "kit", "lab", "lad", "lag",
	"lap", "law", "lax", "lay", "led", "leg", "let", "lid", "lie", "lip", "lit", "lob", "log", "lop", "lot", "low", "mad", "man", "map", "mar",
	"mat", "may", "mob", "mom", "mop", "mud", "mug", "mum", "nab", "nag", "nap", "net", "new", "nil", "nip", "nob", "nod", "nor", "not", "now",
	"nub", "nun", "nut", "oak", "oar", "oat", "odd", "off", "oil", "old", "one", "opt", "orb", "ore", "our", "out", "owe", "owl", "own", "pad",
	"pal", "pan", "par", "pat", "paw", "pay", "pea", "pen", "per", "pet", "pew", "pie", "pig", "pin", "pit", "ply", "pod", "pop", "pot", "pow",
	"pro", "pry", "pub", "pun", "pup", "put", "rag", "ram", "ran", "rap", "rat", "raw", "ray", "red", "rib", "rid", "rig", "rim", "rip", "rob",
	"rod", "rot", "row", "rub", "rug", "rum", "run", "rut", "rye", "sad", "sag", "sap", "sat", "saw", "say", "sea", "see", "set", "sew", "she",
	"shy", "sib", "sin", "sip", "sir", "sis", "sit", "six", "ski", "sky", "sly", "sob", "son", "sop", "sow", "soy", "spa", "spy", "sty", "sub",
	"sue", "sum", "sun", "tab", "tag", "tan", "tap", "tar", "tax", "tea", "ten", "the", "thy", "tic", "tie", "tin", "tip", "toe", "tom", "ton",
	"too", "top", "tot", "tow", "toy", "try", "tub", "tug", "two", "use", "van", "vat", "vet", "via", "vie", "vow", "wad", "wag", "war", "was",
	"wax", "way", "web", "wed", "wee", "wet", "who", "why", "wig", "win", "wit", "woe", "wok", "won", "woo", "wow", "yam", "yap", "yes", "yet",
	"able", "acid", "acne", "aged", "ahem", "ahoy", "aide", "aids", "aims", "ajar", "akin", "alas", "alga", "alms", "aloe", "also", "alto", "ambo", "amen", "amid",
	"ammo", "amps", "anew", "ankh", "ante", "anti", "ants", "apex", "aqua", "arch", "arcs", "area", "argh", "aria", "arid", "arms", "army", "arts", "arty", "ashy",
	"asks", "atom", "atop", "aunt", "aura", "auto", "aver", "avid", "avow", "away", "awed", "axes", "axis", "axle", "babe", "baby", "back", "bade", "bags", "bail",
	"bait", "bake", "bald", "bale", "balk", "ball", "balm", "band", "bang", "bank", "bans", "barb", "bard", "bare", "bark", "barn", "bars", "base", "bash", "bask",
	"bass", "bath", "bats", "batt", "baud", "bawd", "bawl", "bays", "bead", "beak", "beam", "bean", "bear", "beat", "beau", "beck", "beds", "beef", "been", "beep",
	"beer", "bees", "begs", "bell", "belt", "bend", "bent", "berm", "best", "beta", "bets", "bevy", "bias", "bibs", "bide", "bids", "bike", "bile", "bilk", "bill",
	"bind", "bins", "bird", "bite", "bits", "blab", "blah", "bleb", "bled", "blew", "blip", "blob", "bloc", "blot", "blow", "blue", "blur", "boar", "boas", "boat",
	"bobs", "bode", "body", "bogs", "boil", "bold", "bole", "boll", "bolt", "bomb", "bond", "bone", "bong", "bony", "book", "boom", "boon", "boor", "boos", "boot",
	"bops", "bore", "born", "bosh", "boss", "both", "bots", "bout", "bowl", "bows", "boxy", "boys", "bozo", "brag", "bran", "bras", "bray", "bred", "brew", "brig",
	"brim", "brow", "buck", "buds", "buff", "bugs", "bulb", "bulk", "bull", "bump", "buns", "buoy", "burl", "burn", "burp", "burr", "bury", "bush", "bust", "busy",
	"buts", "butt", "buys", "buzz", "byes", "byte", "cabs", "cads", "cafe", "cage", "cake", "calf", "call", "calm", "came", "camp", "cane", "cans", "cape", "caps",
	"card", "care", "carp", "cars", "cart", "case", "cash", "cask", "cast", "cats", "cave", "caws", "cede", "cell", "cent", "chap", "char", "chat", "chef", "chew",
	"chic", "chin", "chip", "chop", "chow", "chub", "chug", "chum", "cite", "city", "clad", "clam", "clan", "clap", "claw", "clay", "clef", "clip", "clod", "clog",
	"clop", "clot", "cloy", "club", "clue", "coal", "coat", "coax", "cobs", "cock", "coco", "coda", "code", "cods", "coed", "cogs", "coil", "coin", "coke", "cola",
	"cold", "cole", "colt", "coma", "comb", "come", "cons", "cony", "cook", "cool", "coop", "cope", "cops", "cord", "core", "cork", "corn", "cost", "cosy", "cots",
	"couch", "could", "count", "court", "cover", "crowd", "crown", "cycle", "daily", "dance", "death", "depth", "doubt", "dozen", "draft", "drama", "dream", "dress", "drink", "drive",
	"earth", "eight", "elite", "empty", "enemy", "enjoy", "enter", "entry", "equal", "error", "event", "exact", "exist", "extra", "faith", "false", "fault", "field", "fifth", "fifty",
	"fight", "final", "first", "fixed", "flash", "fleet", "floor", "fluid", "focus", "force", "forth", "forty", "forum", "found", "frame", "frank", "fraud", "fresh", "front", "fruit",
	"fully", "funny", "giant", "given", "glass", "globe", "going", "grace", "grade", "grand", "grant", "grass", "grave", "great", "green", "gross", "group", "grown", "guard", "guess",
	"guest", "guide", "happy", "harry", "heart", "heavy", "hence", "henry", "horse", "hotel", "house", "human", "ideal", "image", "index", "inner", "input", "issue", "japan", "jimmy",
	"joint", "jones", "judge", "knife", "known", "label", "large", "laser", "later", "laugh", "layer", "learn", "lease", "least", "leave", "legal", "level", "light", "limit", "links",
	"lives", "local", "logic", "loose", "lower", "lucky", "lunch", "lying", "magic", "major", "maker", "march", "maria", "match", "maybe", "mayor", "meant", "media", "metal", "might",
	"minor", "minus", "mixed", "model", "money", "month", "moral", "motor", "mount", "mouse", "mouth", "movie", "music", "needs", "never", "newly", "night", "noise", "north", "noted",
	"novel", "nurse", "occur", "ocean", "offer", "often", "order", "other", "ought", "paint", "paper", "party", "peace", "peter", "phase", "phone", "photo", "piece", "pilot", "pitch",
	"place", "plain", "plane", "plant", "plate", "point", "pound", "power", "press", "price", "pride", "prime", "print", "prior", "prize", "proof", "proud", "prove", "queen", "quick",
	"quiet", "quite", "radio", "raise", "range", "rapid", "ratio", "reach", "ready", "refer", "right", "rival", "river", "rough", "round", "route", "royal", "rural", "scale", "scene",
	"scope", "score", "sense", "serve", "seven", "shall", "shape", "share", "sharp", "sheet", "shelf", "shell", "shift", "shirt", "shock", "shoot", "short", "shown", "sight", "silly",
	"sixth", "sixty", "sized", "skill", "sleep", "slide", "small", "smart", "smile", "smith", "smoke", "solid", "solve", "sorry", "sound", "south", "space", "spare", "speak", "speed",
	"spend", "spent", "split", "spoke", "sport", "staff", "stage", "stake", "stand", "start", "state", "steam", "steel", "stick", "still", "stock", "stone", "stood", "store", "storm",
	"story", "strip", "stuck", "study", "stuff", "style", "sugar", "suite", "super", "sweet", "table", "taken", "taste", "taxes", "teach", "teeth", "texas", "thank", "theft", "their",
	"theme", "there", "these", "thick", "thing", "think", "third", "those", "three", "threw", "throw", "tight", "times", "tired", "title", "today", "topic", "total", "touch", "tough",
	"tower", "track", "trade", "train", "treat", "trend", "trial", "tried", "tries", "truck", "truly", "trust", "truth", "twice", "under", "undue", "union", "unity", "until", "upper",
	"upset", "urban", "usage", "usual", "valid", "value", "video", "virus", "visit", "vital", "voice", "waste", "watch", "water", "wheel", "where", "which", "while", "white", "whole",
	"whose", "woman", "women", "world", "worry", "worse", "worst", "worth", "would", "wound", "write", "wrong", "wrote", "yield", "young", "youth", "zebra",
	"jazz", "buzz", "quiz", "fuzz", "fizz", "jinx", "lynx", "onyx", "oryx", "waxy", "wavy", "ivory", "joker", "fixer", "vixen", "zilch", "zings", "zonal", "zoned", "zoner",
	"crazy", "fuzzy", "jazzy", "pizza", "vexed", "boxer", "epoxy", "extol", "extra", "judges", "joking", "exhale", "expand", "exodus", "buffer", "muffin", "puzzle", "junkie", "monkey", "jacket",
}

var unscrambleDict = []string{}

func init() {
	seen := make(map[string]bool, len(scrabbleDict))
	for _, w := range scrabbleDict {
		w = strings.ToLower(w)
		if !seen[w] {
			seen[w] = true
			unscrambleDict = append(unscrambleDict, w)
		}
	}
	for _, w := range extraUnscrambleWords {
		w = strings.ToLower(w)
		if !seen[w] {
			seen[w] = true
			unscrambleDict = append(unscrambleDict, w)
		}
	}
}

var extraUnscrambleWords = []string{
	"abate", "abbey", "abbot", "abode", "abort", "about", "above", "abuse", "actor", "acute", "adapt", "adept", "admit", "adobe", "adopt", "adore", "adult", "afoot", "after", "again",
	"agent", "agile", "aglow", "agony", "agree", "ahead", "aided", "aider", "aides", "ailed", "aimed", "aimer", "aired", "aisle", "alarm", "album", "alert", "algae", "alibi", "alien",
	"align", "alike", "alive", "allay", "alley", "allot", "allow", "alloy", "aloft", "alone", "along", "aloof", "aloud", "alpha", "altar", "alter", "amass", "amaze", "amber", "amble",
	"amend", "among", "ample", "amuse", "anew", "angel", "anger", "angle", "angry", "angst", "ankle", "annex", "annoy", "annul", "anode", "antic", "anvil", "apart", "apple", "apply",
	"april", "apron", "arena", "argue", "arise", "armed", "armor", "aroma", "arose", "array", "arrow", "arson", "artsy", "ashen", "ashes", "aside", "asked", "asker", "asleep", "aspen",
	"assay", "asset", "atlas", "atone", "attic", "audio", "audit", "augur", "avail", "avert", "avoid", "await", "awake", "award", "aware", "awful", "awoke", "axing", "axles", "azure",
	"baby", "back", "bade", "bait", "bake", "bald", "ball", "balm", "band", "bang", "bank", "bare", "bark", "barn", "base", "bash", "bask", "bath", "bawl", "bays",
	"bead", "beam", "bean", "bear", "beat", "beef", "been", "beep", "beer", "bell", "belt", "bend", "best", "bias", "bide", "bike", "bile", "bill", "bind", "bird",
	"bite", "blab", "blah", "bled", "blew", "blip", "blob", "blot", "blow", "blue", "blur", "boat", "body", "boil", "bold", "bolt", "bomb", "bond", "bone", "book",
	"boom", "boon", "boot", "bore", "born", "boss", "both", "bowl", "boys", "brag", "bran", "bray", "bred", "brew", "brim", "brow", "buck", "buds", "buff", "bulb",
	"bulk", "bull", "bump", "buns", "burn", "burp", "bury", "bush", "bust", "busy", "butt", "buzz", "byte",
	"cable", "cadet", "cafes", "cages", "cake", "calf", "calm", "came", "camp", "cane", "cans", "cape", "card", "care", "carp", "cart", "case", "cash", "cask", "cast",
	"chair", "chalk", "champ", "chant", "chaos", "chart", "chase", "cheap", "cheat", "check", "cheek", "cheer", "chest", "chief", "child", "chili", "chime", "chips", "chirp", "choke",
	"chord", "chose", "chuck", "chunk", "churn", "cider", "cigar", "cinch", "cite", "civic", "civil", "claim", "clamp", "clang", "clank", "clash", "clasp", "class", "clean", "clear",
	"cleft", "clerk", "click", "cliff", "climb", "cling", "clink", "cloak", "clock", "close", "cloth", "cloud", "clout", "clown", "cluck", "clued", "clues", "clump", "clung", "coach",
	"coast", "cobra", "cocoa", "color", "comet", "comic", "comma", "coral", "corny", "couch", "cough", "could", "count", "court", "cover", "covet", "crack", "craft", "cramp", "crane",
	"crank", "crash", "crate", "crave", "crawl", "craze", "crazy", "creak", "cream", "credo", "creed", "creek", "creep", "crepe", "crept", "cress", "crest", "crew", "crib", "cried",
	"cries", "crime", "crimp", "cripe", "crisp", "croak", "crock", "crone", "crony", "crook", "croon", "cross", "crowd", "crown", "crude", "cruel", "crumb", "crush", "crust", "crypt",
	"cubic", "cubit", "curio", "curly", "curry", "curse", "curve", "curvy", "cushy", "cutie",
	"daddy", "daily", "dairy", "daisy", "dally", "dance", "dared", "dares", "darts", "dated", "dates", "datum", "deals", "dealt", "deans", "dears", "death", "debit", "debts", "decal",
	"decay", "decks", "decoy", "decry", "deeds", "deems", "deeps", "defer", "deify", "deign", "deity", "delay", "delve", "demos", "dense", "depth", "derby", "desks", "deter", "devil",
	"diary", "dicey", "diets", "digit", "dimly", "diner", "dines", "dingo", "dingy", "dinky", "diode", "dirge", "dirty", "ditch", "ditto", "ditty", "divan", "diver", "dives", "dixie",
	"dizzy", "docks", "dodge", "dodgy", "doily", "doing", "dolls", "dolly", "donor", "donut", "doors", "dopey", "doted", "dotes", "doubt", "dough", "doused", "dowel", "downs", "dowry", "dozed",
	"dozen", "dozes", "draft", "drags", "drain", "drake", "drama", "drank", "drape", "drawl", "drawn", "draws", "dread", "dream", "drear", "dregs", "dress", "dried", "drier", "dries", "drift",
	"drill", "drink", "drips", "drive", "drone", "drool", "droop", "drops", "dross", "drove", "drown", "drugs", "druid", "drums", "drunk", "dryad", "dryer", "dryly", "duchy", "ducks",
	"early", "earth", "easel", "eaten", "eater", "eaves", "ebony", "edged", "edges", "edict", "edify", "eerie", "egret", "eight", "eject", "elate", "elbow", "elder", "elect", "elite",
	"elope", "elude", "elves", "embed", "ember", "emcee", "empty", "enact", "ended", "endow", "enema", "enemy", "enjoy", "ennui", "ensue", "enter", "entry", "envoy", "epoch", "epoxy",
	"equal", "equip", "erase", "erect", "erode", "erupt", "essay", "ester", "ethic", "ethos", "event", "every", "evict", "evoke", "exact", "exalt", "excel", "exert", "exile", "exist",
	"extol", "extra", "exude", "exult", "eying",
}

func computeWordScore(w string) int {
	w = strings.ToLower(w)
	score := 0
	for _, r := range w {
		if v, ok := scrabbleLetterValues[r]; ok {
			score += v
		}
	}
	return score
}

func makeLetterCount(s string) map[rune]int {
	m := make(map[rune]int)
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' {
			m[r]++
		}
	}
	return m
}

func canFormFromRack(word string, rack map[rune]int) bool {
	need := make(map[rune]int)
	for _, r := range strings.ToLower(word) {
		if r < 'a' || r > 'z' {
			return false
		}
		need[r]++
	}
	for r, c := range need {
		if rack[r] < c {
			return false
		}
	}
	return true
}

func ScrabbleHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b>\n<code>/scrabble &lt;letters&gt;</code> - best word from rack\n<code>/scrabble score &lt;word&gt;</code> - score a word")
		return nil
	}
	fields := strings.Fields(args)
	if strings.ToLower(fields[0]) == "score" {
		if len(fields) < 2 {
			m.Reply("<b>Usage:</b> <code>/scrabble score &lt;word&gt;</code>")
			return nil
		}
		word := strings.ToLower(fields[1])
		for _, r := range word {
			if r < 'a' || r > 'z' {
				m.Reply("Word must contain only letters A-Z.")
				return nil
			}
		}
		score := computeWordScore(word)
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("<b>Scrabble score for</b> <code>%s</code>: <b>%d</b>\n\n<blockquote>", html.EscapeString(strings.ToUpper(word)), score))
		parts := make([]string, 0, len(word))
		for _, r := range word {
			parts = append(parts, fmt.Sprintf("%c=%d", r-32, scrabbleLetterValues[r]))
		}
		sb.WriteString(strings.Join(parts, " · "))
		sb.WriteString("</blockquote>")
		m.Reply(sb.String())
		return nil
	}

	rackRaw := strings.ToLower(strings.Join(fields, ""))
	rackRaw = strings.ReplaceAll(rackRaw, " ", "")
	cleaned := make([]rune, 0, len(rackRaw))
	for _, r := range rackRaw {
		if r >= 'a' && r <= 'z' {
			cleaned = append(cleaned, r)
		}
	}
	if len(cleaned) == 0 {
		m.Reply("Provide some letters, e.g. <code>/scrabble retains</code>.")
		return nil
	}
	if len(cleaned) > 15 {
		m.Reply("Rack too large. Use up to 15 letters (Scrabble racks are 7).")
		return nil
	}
	rack := makeLetterCount(string(cleaned))

	type scored struct {
		word  string
		score int
	}
	var matches []scored
	for _, w := range scrabbleDict {
		if canFormFromRack(w, rack) {
			matches = append(matches, scored{w, computeWordScore(w)})
		}
	}
	if len(matches) == 0 {
		m.Reply(fmt.Sprintf("No valid Scrabble words found for rack <code>%s</code>.", html.EscapeString(strings.ToUpper(string(cleaned)))))
		return nil
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		if len(matches[i].word) != len(matches[j].word) {
			return len(matches[i].word) > len(matches[j].word)
		}
		return matches[i].word < matches[j].word
	})

	top := matches[0]
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Best word for rack</b> <code>%s</code>\n\n", html.EscapeString(strings.ToUpper(string(cleaned)))))
	sb.WriteString(fmt.Sprintf("Winner: <b>%s</b> — <b>%d</b> points\n", html.EscapeString(strings.ToUpper(top.word)), top.score))
	sb.WriteString(fmt.Sprintf("Length: <code>%d</code>\n\n", len(top.word)))
	limit := 10
	if len(matches) < limit {
		limit = len(matches)
	}
	sb.WriteString("<b>Top picks:</b>\n<blockquote>")
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, fmt.Sprintf("%s (%d)", strings.ToUpper(matches[i].word), matches[i].score))
	}
	sb.WriteString(html.EscapeString(strings.Join(parts, " · ")))
	sb.WriteString("</blockquote>")
	sb.WriteString(fmt.Sprintf("\n<i>%d total matches in dictionary.</i>", len(matches)))
	m.Reply(sb.String())
	return nil
}

func UnscrambleHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/unscramble &lt;letters&gt;</code>\nFinds all valid English words that can be formed from the given letters.")
		return nil
	}
	cleaned := make([]rune, 0, len(args))
	for _, r := range strings.ToLower(args) {
		if r >= 'a' && r <= 'z' {
			cleaned = append(cleaned, r)
		}
	}
	if len(cleaned) == 0 {
		m.Reply("Provide letters A-Z, e.g. <code>/unscramble listen</code>.")
		return nil
	}
	if len(cleaned) > 20 {
		m.Reply("Too many letters. Limit is 20.")
		return nil
	}
	rack := makeLetterCount(string(cleaned))
	type found struct {
		word  string
		score int
	}
	var hits []found
	for _, w := range unscrambleDict {
		if len(w) < 2 {
			continue
		}
		if canFormFromRack(w, rack) {
			hits = append(hits, found{w, computeWordScore(w)})
		}
	}
	if len(hits) == 0 {
		m.Reply(fmt.Sprintf("No valid words found in <code>%s</code>.", html.EscapeString(strings.ToUpper(string(cleaned)))))
		return nil
	}
	sort.Slice(hits, func(i, j int) bool {
		if len(hits[i].word) != len(hits[j].word) {
			return len(hits[i].word) > len(hits[j].word)
		}
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].word < hits[j].word
	})

	buckets := make(map[int][]string)
	for _, h := range hits {
		buckets[len(h.word)] = append(buckets[len(h.word)], h.word)
	}
	lengths := make([]int, 0, len(buckets))
	for l := range buckets {
		lengths = append(lengths, l)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(lengths)))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Unscramble</b> <code>%s</code>\n", html.EscapeString(strings.ToUpper(string(cleaned)))))
	sb.WriteString(fmt.Sprintf("<i>Found %d words.</i>\n\n", len(hits)))

	totalShown := 0
	maxShow := 80
	for _, l := range lengths {
		if totalShown >= maxShow {
			break
		}
		group := buckets[l]
		remain := maxShow - totalShown
		shown := group
		truncated := false
		if len(group) > remain {
			shown = group[:remain]
			truncated = true
		}
		sb.WriteString(fmt.Sprintf("<b>%d letters</b> (%d):\n<blockquote>", l, len(group)))
		parts := make([]string, 0, len(shown))
		for _, w := range shown {
			parts = append(parts, html.EscapeString(w))
		}
		sb.WriteString(strings.Join(parts, ", "))
		if truncated {
			sb.WriteString(", …")
		}
		sb.WriteString("</blockquote>\n")
		totalShown += len(shown)
	}
	if len(hits) > maxShow {
		sb.WriteString(fmt.Sprintf("<i>Showing %d of %d.</i>", maxShow, len(hits)))
	}
	m.Reply(sb.String())
	return nil
}

func registerWordGameHandlers() {
	c := Client
	c.On("cmd:scrabble", ScrabbleHandler)
	c.On("cmd:unscramble", UnscrambleHandler)
}

func init() {
	QueueHandlerRegistration(registerWordGameHandlers)
}
