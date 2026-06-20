package modules

import (
	"fmt"
	"html"
	"math/rand"
	"sort"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var emojiComboPool = []string{
	"😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😂", "🙂", "🙃",
	"😉", "😊", "😇", "🥰", "😍", "🤩", "😘", "😗", "😚", "😙",
	"😋", "😛", "😜", "🤪", "😝", "🤑", "🤗", "🤭", "🤫", "🤔",
	"🤐", "🤨", "😐", "😑", "😶", "😏", "😒", "🙄", "😬", "🤥",
	"😌", "😔", "😪", "🤤", "😴", "😷", "🤒", "🤕", "🤢", "🤮",
	"🥵", "🥶", "🥴", "😵", "🤯", "🤠", "🥳", "😎", "🤓", "🧐",
	"😕", "😟", "🙁", "☹️", "😮", "😯", "😲", "😳", "🥺", "😦",
	"😧", "😨", "😰", "😥", "😢", "😭", "😱", "😖", "😣", "😞",
	"😓", "😩", "😫", "🥱", "😤", "😡", "😠", "🤬", "😈", "👿",
	"💀", "☠️", "💩", "🤡", "👹", "👺", "👻", "👽", "👾", "🤖",
	"🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼", "🐨", "🐯",
	"🦁", "🐮", "🐷", "🐸", "🐵", "🐔", "🐧", "🐦", "🐤", "🦆",
	"🦅", "🦉", "🦇", "🐺", "🐗", "🐴", "🦄", "🐝", "🐛", "🦋",
	"🌵", "🌲", "🌳", "🌴", "🌱", "🌿", "🍀", "🌾", "🌷", "🌹",
	"🌺", "🌸", "🌼", "🌻", "🌞", "🌝", "🌚", "🌑", "🌒", "🌓",
	"🌔", "🌕", "🌖", "🌗", "🌘", "🌙", "🌎", "🌍", "🌏", "💫",
	"⭐", "🌟", "✨", "⚡", "☄️", "💥", "🔥", "🌈", "☀️", "⛅",
	"☁️", "🌧️", "⛈️", "🌩️", "🌨️", "❄️", "☃️", "⛄", "🌬️", "💨",
	"💧", "💦", "🌊", "🍏", "🍎", "🍐", "🍊", "🍋", "🍌", "🍉",
	"🍇", "🍓", "🍈", "🍒", "🍑", "🥭", "🍍", "🥥", "🥝", "🍅",
	"🍆", "🥑", "🥦", "🥬", "🥒", "🌶️", "🌽", "🥕", "🥔", "🍠",
	"🥐", "🥯", "🍞", "🥖", "🥨", "🧀", "🥚", "🍳", "🥞", "🧇",
	"🥓", "🥩", "🍗", "🍖", "🌭", "🍔", "🍟", "🍕", "🥪", "🥙",
	"🧆", "🌮", "🌯", "🥗", "🥘", "🥫", "🍝", "🍜", "🍲", "🍛",
	"🍣", "🍱", "🥟", "🦪", "🍤", "🍙", "🍚", "🍘", "🍥", "🥮",
	"🍢", "🍡", "🍧", "🍨", "🍦", "🥧", "🧁", "🍰", "🎂", "🍮",
	"🍭", "🍬", "🍫", "🍿", "🍩", "🍪", "🌰", "🥜", "🍯", "🥛",
	"🍼", "☕", "🍵", "🧃", "🥤", "🍶", "🍺", "🍻", "🥂", "🍷",
	"🥃", "🍸", "🍹", "🧉", "🍾", "⚽", "🏀", "🏈", "⚾", "🥎",
	"🎾", "🏐", "🏉", "🥏", "🎱", "🪀", "🏓", "🏸", "🏒", "🏑",
	"🥍", "🏏", "⛳", "🪁", "🏹", "🎣", "🤿", "🥊", "🥋", "🎽",
	"🛹", "🛷", "⛸️", "🥌", "🎿", "⛷️", "🏂", "🪂", "🏋️", "🤼",
	"🤸", "⛹️", "🤺", "🤾", "🏌️", "🏇", "🧘", "🏄", "🏊", "🤽",
	"🚣", "🧗", "🚵", "🚴", "🏆", "🥇", "🥈", "🥉", "🏅", "🎖️",
	"🏵️", "🎗️", "🎫", "🎟️", "🎪", "🤹", "🎭", "🩰", "🎨", "🎬",
	"🎤", "🎧", "🎼", "🎹", "🥁", "🎷", "🎺", "🎸", "🪕", "🎻",
	"🎲", "♟️", "🎯", "🎳", "🎮", "🎰", "🧩", "❤️", "🧡", "💛",
	"💚", "💙", "💜", "🖤", "🤍", "🤎", "💔", "❣️", "💕", "💞",
	"💓", "💗", "💖", "💘", "💝", "💟", "💌", "💋", "👑", "🎩",
	"🎓", "🧢", "⛑️", "📿", "💄", "💍", "💎", "🔔", "🎵", "🎶",
}

var emojiFightActors = []string{
	"🧙", "🧝", "🧛", "🧟", "🦸", "🦹", "🥷", "🤺", "🧞", "🧜",
	"🐉", "🐲", "🦖", "🦕", "🦂", "🐍", "🦅", "🐺", "🦏", "🐅",
	"🤖", "👽", "👹", "👺", "👻", "🧌", "🎃", "👿", "🤡", "💀",
	"🧚", "🧌", "🐙", "🦈", "🦁", "🐻", "🦍", "🐊", "🦄", "🐗",
}

var emojiFightWeapons = []string{
	"⚔️", "🗡️", "🏹", "🔫", "💣", "🪓", "🔨", "🛡️", "🧨", "🪄",
	"🔱", "⚒️", "🪃", "🪚", "🥊", "🥋", "🪦", "🪙", "🧪", "🔮",
}

var emojiFightImpacts = []string{
	"💥", "🔥", "⚡", "💢", "💫", "✨", "🌪️", "☄️", "🩸", "💀",
	"🌟", "💯", "🎯", "🚀", "💨", "❄️", "🌊", "☢️", "☣️", "🕳️",
}

var emojiFightOutcomes = []string{
	"🏆", "🥇", "💀", "⚰️", "🚑", "🩹", "🏳️", "👑", "🆘", "🆗",
}

var emojiWordMap = map[rune]string{
	'A': "🍎", 'B': "🍌", 'C': "🐱", 'D': "🐶", 'E': "🥚",
	'F': "🦊", 'G': "🍇", 'H': "🏠", 'I': "🍦", 'J': "🕹️",
	'K': "🔑", 'L': "🦁", 'M': "🌙", 'N': "📒", 'O': "🐙",
	'P': "🍕", 'Q': "👸", 'R': "🌹", 'S': "⭐", 'T': "🌴",
	'U': "☂️", 'V': "🎻", 'W': "🍉", 'X': "❌", 'Y': "🪀",
	'Z': "🦓",
}

func emojiComboRng() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func emojiComboPick(rng *rand.Rand, pool []string, n int) []string {
	if n > len(pool) {
		n = len(pool)
	}
	idx := rng.Perm(len(pool))[:n]
	out := make([]string, 0, n)
	for _, i := range idx {
		out = append(out, pool[i])
	}
	return out
}

func emojiComboPickOne(rng *rand.Rand, pool []string) string {
	return pool[rng.Intn(len(pool))]
}

func EmojiComboHandler(m *tg.NewMessage) error {
	rng := emojiComboRng()
	picks := emojiComboPick(rng, emojiComboPool, 3)
	var b strings.Builder
	b.WriteString("<b>Random Emoji Combo</b>\n\n")
	b.WriteString("<code>")
	b.WriteString(strings.Join(picks, " "))
	b.WriteString("</code>\n\n")
	b.WriteString("<i>3 random emojis just for you.</i>")
	_, err := m.Reply(b.String())
	return err
}

func EmojiFightHandler(m *tg.NewMessage) error {
	rng := emojiComboRng()
	actors := emojiComboPick(rng, emojiFightActors, 2)
	if len(actors) < 2 {
		_, err := m.Reply("Not enough fighters available.")
		return err
	}
	weapon := emojiComboPickOne(rng, emojiFightWeapons)
	impact := emojiComboPickOne(rng, emojiFightImpacts)
	outcome := emojiComboPickOne(rng, emojiFightOutcomes)

	scene := fmt.Sprintf("%s  %s  %s  %s  %s", actors[0], weapon, impact, actors[1], outcome)

	winner := actors[0]
	loser := actors[1]
	if rng.Intn(2) == 0 {
		winner = actors[1]
		loser = actors[0]
	}

	var b strings.Builder
	b.WriteString("<b>Emoji Fight!</b>\n\n")
	b.WriteString("<code>")
	b.WriteString(scene)
	b.WriteString("</code>\n\n")
	b.WriteString(fmt.Sprintf("<b>Winner:</b> %s\n", winner))
	b.WriteString(fmt.Sprintf("<b>Loser:</b>  %s\n", loser))
	b.WriteString(fmt.Sprintf("<b>Weapon:</b> %s\n", weapon))
	b.WriteString(fmt.Sprintf("<b>Impact:</b> %s\n", impact))
	b.WriteString(fmt.Sprintf("<b>Outcome:</b> %s", outcome))

	_, err := m.Reply(b.String())
	return err
}

func EmojiWordHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	var b strings.Builder
	b.WriteString("<b>A-Z Emoji Map</b>\n\n")

	if arg == "" {
		letters := make([]rune, 0, len(emojiWordMap))
		for r := range emojiWordMap {
			letters = append(letters, r)
		}
		sort.Slice(letters, func(i, j int) bool { return letters[i] < letters[j] })
		b.WriteString("<pre>")
		for i, r := range letters {
			b.WriteString(fmt.Sprintf("%c %s", r, emojiWordMap[r]))
			if (i+1)%4 == 0 {
				b.WriteString("\n")
			} else {
				b.WriteString("   ")
			}
		}
		b.WriteString("</pre>\n")
		b.WriteString("<i>Tip:</i> <code>/emojiword hello</code> to translate a word.")
		_, err := m.Reply(b.String())
		return err
	}

	upper := strings.ToUpper(arg)
	var rendered strings.Builder
	var skipped []string
	for _, r := range upper {
		if r == ' ' {
			rendered.WriteString("   ")
			continue
		}
		if e, ok := emojiWordMap[r]; ok {
			rendered.WriteString(e)
			rendered.WriteString(" ")
		} else {
			skipped = append(skipped, string(r))
		}
	}

	out := strings.TrimSpace(rendered.String())
	if out == "" {
		_, err := m.Reply("No mappable letters in <code>" + html.EscapeString(arg) + "</code>. Use A-Z only.")
		return err
	}

	b.Reset()
	b.WriteString("<b>Emoji Word</b>\n\n")
	b.WriteString("<b>Input:</b> <code>")
	b.WriteString(html.EscapeString(arg))
	b.WriteString("</code>\n")
	b.WriteString("<b>Output:</b> ")
	b.WriteString(out)
	if len(skipped) > 0 {
		b.WriteString("\n\n<i>Skipped:</i> <code>")
		b.WriteString(html.EscapeString(strings.Join(skipped, " ")))
		b.WriteString("</code>")
	}
	_, err := m.Reply(b.String())
	return err
}

func registerRandomEmojiComboHandlers() {
	c := Client
	c.On("cmd:emojicombo", EmojiComboHandler)
	c.On("cmd:emojifight", EmojiFightHandler)
	c.On("cmd:emojiword", EmojiWordHandler)
}

func init() { QueueHandlerRegistration(registerRandomEmojiComboHandlers) }
