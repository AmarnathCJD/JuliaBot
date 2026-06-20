package modules

import (
	"fmt"
	"html"
	"math/rand"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var emojifyDict = map[string]string{
	"love":      "❤️",
	"heart":     "💖",
	"fire":      "🔥",
	"hot":       "🥵",
	"cold":      "🥶",
	"money":     "💰",
	"cash":      "💵",
	"dollar":    "💵",
	"cat":       "🐱",
	"dog":       "🐶",
	"code":      "💻",
	"coding":    "👨‍💻",
	"computer":  "🖥️",
	"phone":     "📱",
	"laptop":    "💻",
	"food":      "🍔",
	"pizza":     "🍕",
	"burger":    "🍔",
	"coffee":    "☕",
	"tea":       "🍵",
	"beer":      "🍺",
	"wine":      "🍷",
	"water":     "💧",
	"sun":       "☀️",
	"moon":      "🌙",
	"star":      "⭐",
	"stars":     "✨",
	"rain":      "🌧️",
	"snow":      "❄️",
	"cloud":     "☁️",
	"wind":      "🌬️",
	"earth":     "🌍",
	"world":     "🌎",
	"car":       "🚗",
	"bike":      "🚲",
	"plane":     "✈️",
	"train":     "🚆",
	"boat":      "🛥️",
	"rocket":    "🚀",
	"book":      "📚",
	"music":     "🎵",
	"song":      "🎶",
	"movie":     "🎬",
	"camera":    "📷",
	"game":      "🎮",
	"party":     "🎉",
	"birthday":  "🎂",
	"gift":      "🎁",
	"happy":     "😊",
	"sad":       "😢",
	"cry":       "😭",
	"angry":     "😡",
	"laugh":     "😂",
	"smile":     "😄",
	"wink":      "😉",
	"cool":      "😎",
	"sleep":     "😴",
	"sick":      "🤒",
	"think":     "🤔",
	"love_you":  "😘",
	"king":      "👑",
	"queen":     "👸",
	"work":      "💼",
	"home":      "🏠",
	"school":    "🏫",
	"hospital":  "🏥",
	"church":    "⛪",
	"time":      "⏰",
	"clock":     "🕐",
	"flower":    "🌸",
	"rose":      "🌹",
	"tree":      "🌳",
	"leaf":      "🍃",
	"apple":     "🍎",
	"banana":    "🍌",
	"diamond":   "💎",
	"crown":     "👑",
	"medal":     "🏅",
	"trophy":    "🏆",
	"win":       "🏆",
	"lose":      "💔",
	"bomb":      "💣",
	"gun":       "🔫",
	"sword":     "⚔️",
	"shield":    "🛡️",
	"check":     "✅",
	"cross":     "❌",
	"warning":   "⚠️",
	"hundred":   "💯",
	"poop":      "💩",
	"ghost":     "👻",
	"skull":     "💀",
	"alien":     "👽",
	"robot":     "🤖",
	"devil":     "😈",
	"angel":     "😇",
	"hello":     "👋",
	"bye":       "👋",
	"ok":        "👌",
	"yes":       "✅",
	"no":        "❌",
	"thanks":    "🙏",
	"pray":      "🙏",
	"clap":      "👏",
	"thumbs":    "👍",
	"peace":     "✌️",
	"strong":    "💪",
	"brain":     "🧠",
	"eye":       "👁️",
	"eyes":      "👀",
	"speak":     "🗣️",
}

var randomEmojiPool = []string{
	"😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😂", "🙂", "🙃",
	"😉", "😊", "😇", "🥰", "😍", "🤩", "😘", "😗", "☺️", "😚",
	"😙", "🥲", "😋", "😛", "😜", "🤪", "😝", "🤑", "🤗", "🤭",
	"🤫", "🤔", "🤐", "🤨", "😐", "😑", "😶", "😏", "😒", "🙄",
	"😬", "🤥", "😌", "😔", "😪", "🤤", "😴", "😷", "🤒", "🤕",
	"🤢", "🤮", "🤧", "🥵", "🥶", "🥴", "😵", "🤯", "🤠", "🥳",
	"🥸", "😎", "🤓", "🧐", "😕", "😟", "🙁", "☹️", "😮", "😯",
	"😲", "😳", "🥺", "😦", "😧", "😨", "😰", "😥", "😢", "😭",
	"😱", "😖", "😣", "😞", "😓", "😩", "😫", "🥱", "😤", "😡",
	"😠", "🤬", "😈", "👿", "💀", "☠️", "💩", "🤡", "👹", "👺",
	"👻", "👽", "👾", "🤖", "😺", "😸", "😹", "😻", "😼", "😽",
	"🙀", "😿", "😾", "💋", "👋", "🤚", "🖐️", "✋", "🖖", "👌",
	"🤌", "🤏", "✌️", "🤞", "🤟", "🤘", "🤙", "👈", "👉", "👆",
	"🖕", "👇", "☝️", "👍", "👎", "✊", "👊", "🤛", "🤜", "👏",
	"🙌", "👐", "🤲", "🤝", "🙏", "✍️", "💅", "🤳", "💪", "🦾",
	"🦵", "🦿", "🦶", "👂", "🦻", "👃", "🧠", "🫀", "🫁", "🦷",
	"🦴", "👀", "👁️", "👅", "👄", "👶", "🧒", "👦", "👧", "🧑",
	"👱", "👨", "🧔", "👩", "🧓", "👴", "👵", "🙍", "🙎", "🙅",
	"🙆", "💁", "🙋", "🧏", "🙇", "🤦", "🤷", "👮", "🕵️", "💂",
	"🥷", "👷", "🤴", "👸", "👳", "👲", "🧕", "🤵", "👰", "🤰",
	"🤱", "👼", "🎅", "🤶", "🦸", "🦹", "🧙", "🧚", "🧛", "🧜",
	"🧝", "🧞", "🧟", "💆", "💇", "🚶", "🧍", "🧎", "🏃", "💃",
	"🕺", "🕴️", "👯", "🧖", "🧗", "🤺", "🏇", "⛷️", "🏂", "🏌️",
	"🏄", "🚣", "🏊", "⛹️", "🏋️", "🚴", "🚵", "🤸", "🤼", "🤽",
	"🤾", "🤹", "🧘", "🛀", "🛌", "👭", "👫", "👬", "💏", "💑",
	"👪", "🗣️", "👤", "👥", "🫂", "👣", "🐵", "🐒", "🦍", "🦧",
	"🐶", "🐕", "🦮", "🐩", "🐺", "🦊", "🦝", "🐱", "🐈", "🦁",
	"🐯", "🐅", "🐆", "🐴", "🐎", "🦄", "🦓", "🦌", "🦬", "🐮",
	"🐂", "🐃", "🐄", "🐷", "🐖", "🐗", "🐽", "🐏", "🐑", "🐐",
	"🐪", "🐫", "🦙", "🦒", "🐘", "🦣", "🦏", "🦛", "🐭", "🐁",
	"🐀", "🐹", "🐰", "🐇", "🐿️", "🦫", "🦔", "🦇", "🐻", "🐨",
	"🐼", "🦥", "🦦", "🦨", "🦘", "🦡", "🐾", "🦃", "🐔", "🐓",
	"🐣", "🐤", "🐥", "🐦", "🐧", "🕊️", "🦅", "🦆", "🦢", "🦉",
	"🦤", "🪶", "🦩", "🦚", "🦜", "🐸", "🐊", "🐢", "🦎", "🐍",
	"🐲", "🐉", "🦕", "🦖", "🐳", "🐋", "🐬", "🦭", "🐟", "🐠",
	"🐡", "🦈", "🐙", "🐚", "🐌", "🦋", "🐛", "🐜", "🐝", "🪲",
	"🐞", "🦗", "🪳", "🕷️", "🕸️", "🦂", "🦟", "🪰", "🪱", "🦠",
	"💐", "🌸", "💮", "🏵️", "🌹", "🥀", "🌺", "🌻", "🌼", "🌷",
	"🌱", "🪴", "🌲", "🌳", "🌴", "🌵", "🌾", "🌿", "☘️", "🍀",
	"🍁", "🍂", "🍃", "🍇", "🍈", "🍉", "🍊", "🍋", "🍌", "🍍",
	"🥭", "🍎", "🍏", "🍐", "🍑", "🍒", "🍓", "🫐", "🥝", "🍅",
	"🫒", "🥥", "🥑", "🍆", "🥔", "🥕", "🌽", "🌶️", "🫑", "🥒",
	"🥬", "🥦", "🧄", "🧅", "🍄", "🥜", "🌰", "🍞", "🥐", "🥖",
	"🫓", "🥨", "🥯", "🥞", "🧇", "🧀", "🍖", "🍗", "🥩", "🥓",
	"🍔", "🍟", "🍕", "🌭", "🥪", "🌮", "🌯", "🫔", "🥙", "🧆",
	"🥚", "🍳", "🥘", "🍲", "🫕", "🥣", "🥗", "🍿", "🧈", "🧂",
	"🥫", "🍱", "🍘", "🍙", "🍚", "🍛", "🍜", "🍝", "🍠", "🍢",
	"🍣", "🍤", "🍥", "🥮", "🍡", "🥟", "🥠", "🥡", "🦀", "🦞",
	"🦐", "🦑", "🦪", "🍦", "🍧", "🍨", "🍩", "🍪", "🎂", "🍰",
	"🧁", "🥧", "🍫", "🍬", "🍭", "🍮", "🍯", "🍼", "🥛", "☕",
	"🫖", "🍵", "🍶", "🍾", "🍷", "🍸", "🍹", "🍺", "🍻", "🥂",
}

func EmojifyHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/emojify &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	words := strings.Fields(text)
	var out []string
	for _, w := range words {
		lower := strings.ToLower(strings.Trim(w, ".,!?;:\"'()[]{}"))
		out = append(out, html.EscapeString(w))
		if emo, ok := emojifyDict[lower]; ok {
			out = append(out, emo)
		}
	}
	m.Reply(strings.Join(out, " "))
	return nil
}

func EmojiArtHandler(m *tg.NewMessage) error {
	text := extractText(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/emojiart &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	var b strings.Builder
	for _, r := range strings.ToUpper(text) {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(0x1F1E6 + (r - 'A'))
			b.WriteRune(' ')
		case r >= '0' && r <= '9':
			b.WriteRune(rune('0') + (r - '0'))
			b.WriteRune(0x20E3)
			b.WriteRune(' ')
		case r == ' ':
			b.WriteString("   ")
		default:
			b.WriteRune(r)
		}
	}
	res := strings.TrimSpace(b.String())
	if res == "" {
		m.Reply("<b>Error:</b> nothing to convert.")
		return nil
	}
	m.Reply(res)
	return nil
}

func RandomEmojiHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	n := 5
	if arg != "" {
		v, err := strconv.Atoi(arg)
		if err != nil || v <= 0 {
			m.Reply("<b>Usage:</b> <code>/random_emoji [N]</code> where N is a positive integer.")
			return nil
		}
		n = v
	}
	if n > 100 {
		n = 100
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(randomEmojiPool[rand.Intn(len(randomEmojiPool))])
	}
	m.Reply(fmt.Sprintf("<b>%d random emojis:</b>\n%s", n, b.String()))
	return nil
}

func registerEmojifyHandlers() {
	c := Client
	c.On("cmd:emojify", EmojifyHandler)
	c.On("cmd:emojiart", EmojiArtHandler)
	c.On("cmd:random_emoji", RandomEmojiHandler)
}

func init() {
	QueueHandlerRegistration(registerEmojifyHandlers)
}
