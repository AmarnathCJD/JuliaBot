package modules

import (
	"fmt"
	"hash/fnv"
	"html"
	"math/rand"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var fortuneRng = rand.New(rand.NewSource(time.Now().UnixNano()))

var fortuneList = []string{
	"A new opportunity will knock on your door soon.",
	"A gentle word turns away wrath; a kind smile melts hearts.",
	"A pleasant surprise is in store for you tonight.",
	"A small act of kindness today will return to you tenfold.",
	"A stranger will become a trusted friend within the year.",
	"Adventure can be real happiness, embrace the unknown.",
	"All the effort you are making will ultimately pay off.",
	"An exciting opportunity lies ahead of you this week.",
	"An old friend will reach out unexpectedly soon.",
	"Believe in yourself and others will believe in you too.",
	"Better days are coming; they are called Saturday and Sunday.",
	"Big changes are on the horizon, prepare to adapt.",
	"Bloom where you are planted and grow strong.",
	"Carry a positive attitude and you will carry the day.",
	"Change is the only constant; embrace it bravely.",
	"Cherish the friends who walk with you in silence.",
	"Confidence is not 'they will like me'; it's 'I'll be fine if they don't'.",
	"Creativity is intelligence having fun, keep playing.",
	"Curiosity is the wick in the candle of learning.",
	"Do not be afraid of competition; be afraid of stagnation.",
	"Do not let small minds convince you your dreams are too big.",
	"Don't count the days, make the days count.",
	"Dreams are whispers from your future self.",
	"Each small step writes a chapter in your life story.",
	"Eat well, laugh often, love much, and prosper.",
	"Every flower blooms in its own sweet time.",
	"Every storm runs out of rain eventually.",
	"Failure is just success rounded down.",
	"Fortune favors the bold and the prepared.",
	"From small beginnings come great things.",
	"Friends long absent are coming back to you.",
	"Generosity will return to you in unexpected ways.",
	"Good news will come to you by mail or message.",
	"Happiness is found in the smallest moments.",
	"Hard work beats talent when talent doesn't work hard.",
	"He who laughs, lasts longer than he who frowns.",
	"Hidden treasures lie in places you've already been.",
	"Hope is a beautiful thing; never lose it.",
	"If you can dream it, you can achieve it.",
	"If you look for the good, you will find it.",
	"In every walk with nature, one receives far more than they seek.",
	"It is better to travel well than to arrive.",
	"Joy is not in things; it is in us.",
	"Keep your face to the sunshine and shadows fall behind.",
	"Kindness is a language the deaf can hear and the blind can see.",
	"Laughter is the closest distance between two people.",
	"Life is short, smile while you still have teeth.",
	"Listen to your heart; it knows the way.",
	"Love grows by giving, not by holding back.",
	"Make today so awesome that yesterday gets jealous.",
	"Money will come, but the value of time is greater.",
	"Never let yesterday use up too much of today.",
	"New friends will enrich your life this season.",
	"Now is the time to try something new and bold.",
	"Opportunities are like sunrises; if you wait too long you miss them.",
	"Patience is bitter, but its fruit is sweet.",
	"People are drawn to your warmth and good nature.",
	"Practice makes progress; perfection is a myth.",
	"Pursue what catches your heart, not what catches your eye.",
	"Remember to look up at the stars, not down at your feet.",
	"Rivers know there is no hurry; we shall get there someday.",
	"Sing as if no one is listening, dance as if no one is watching.",
	"Small things, done with great love, change the world.",
	"Smile, it confuses people who wish you harm.",
	"Someone admires you greatly from a distance.",
	"Stars cannot shine without darkness; keep going.",
	"Strong coffee, stronger resolve; today is yours.",
	"Success is a series of small wins repeated daily.",
	"The best is yet to come, hold on tight.",
	"The early bird gets the worm; the night owl gets the silence.",
	"The greatest risk is not taking one.",
	"The harder you work, the luckier you get.",
	"The journey of a thousand miles begins with a single step.",
	"The one you love is closer than you think.",
	"The road less traveled is calling your name.",
	"The secret of happiness is not in doing what one likes, but in liking what one does.",
	"The world is round so that friendship may encircle it.",
	"There is a wisdom of the head and a wisdom of the heart.",
	"Time spent laughing is time spent with the gods.",
	"Today's struggle is tomorrow's strength.",
	"Tomorrow brings a new chance to begin again.",
	"Trust the timing of your life; everything unfolds perfectly.",
	"Try a new path; the view will surprise you.",
	"Two heads are better than one, especially yours.",
	"Use your imagination; it is the preview of life's coming attractions.",
	"Walk as if you are kissing the earth with your feet.",
	"Wealth follows wisdom, not the other way around.",
	"What you seek is seeking you.",
	"When in doubt, choose the bolder option.",
	"When one door closes, another opens with better hinges.",
	"Where there is great love, there are always miracles.",
	"While there's life, there's hope; while there's hope, there's joy.",
	"You are exactly where you need to be right now.",
	"You attract what you radiate; shine bright.",
	"You will discover a hidden talent within yourself.",
	"You will find what you have been looking for.",
	"You will laugh until your sides hurt very soon.",
	"You will receive good news from a distant friend.",
	"You will travel to many exotic places in your lifetime.",
	"Your future is created by what you do today, not tomorrow.",
	"Your heart knows the answer before your mind asks.",
	"Your kindness will lead you to unexpected blessings.",
	"Your smile is your superpower; use it often.",
}

var yesNoList = []string{
	"Yes",
	"No",
	"Maybe",
	"Absolutely yes",
	"Absolutely not",
	"Definitely yes",
	"Definitely no",
	"Most likely yes",
	"Most likely no",
	"Without a doubt, yes",
	"I wouldn't count on it",
	"Ask again later",
	"Signs point to yes",
	"Signs point to no",
	"Could go either way",
	"The stars say yes",
	"The stars say no",
	"Hmm... probably yes",
	"Hmm... probably no",
	"Trust your gut",
}

var eightBallList = []string{
	"It is certain.",
	"It is decidedly so.",
	"Without a doubt.",
	"Yes definitely.",
	"You may rely on it.",
	"As I see it, yes.",
	"Most likely.",
	"Outlook good.",
	"Yes.",
	"Signs point to yes.",
	"Reply hazy, try again.",
	"Ask again later.",
	"Better not tell you now.",
	"Cannot predict now.",
	"Concentrate and ask again.",
	"Don't count on it.",
	"My reply is no.",
	"My sources say no.",
	"Outlook not so good.",
	"Very doubtful.",
}

var horoscopeSigns = map[string]string{
	"aries":       "Aries",
	"taurus":      "Taurus",
	"gemini":      "Gemini",
	"cancer":      "Cancer",
	"leo":         "Leo",
	"virgo":       "Virgo",
	"libra":       "Libra",
	"scorpio":     "Scorpio",
	"sagittarius": "Sagittarius",
	"capricorn":   "Capricorn",
	"aquarius":    "Aquarius",
	"pisces":      "Pisces",
}

var horoscopeSignEmoji = map[string]string{
	"aries":       "♈",
	"taurus":      "♉",
	"gemini":      "♊",
	"cancer":      "♋",
	"leo":         "♌",
	"virgo":       "♍",
	"libra":       "♎",
	"scorpio":     "♏",
	"sagittarius": "♐",
	"capricorn":   "♑",
	"aquarius":    "♒",
	"pisces":      "♓",
}

var horoscopeMessages = []string{
	"Today the universe nudges you toward bold action; trust the spark and move.",
	"A quiet conversation today reveals a truth you've been avoiding; lean in.",
	"Your energy is magnetic today; people will gravitate toward your ideas.",
	"Small frustrations early give way to a surprising breakthrough by evening.",
	"Focus on one task today; multitasking will drain you more than usual.",
	"A familiar face brings unfamiliar news; keep your mind open and curious.",
	"Money matters look favorable; review your finances with a calm head.",
	"Romance is in the air, even if it's quiet; notice the small gestures.",
	"Your creativity peaks today; jot down every idea, even the strange ones.",
	"Be patient with someone close; they are fighting an invisible battle.",
	"An unexpected invitation arrives; saying yes will reshape your week.",
	"Rest is productive today; pushing through will only slow you down later.",
	"Forgive an old grudge; the weight lifts the moment you let it go.",
	"A risk you've been weighing pays off; trust the data and your instincts.",
	"Communication flows easily today; speak the thing you've been holding back.",
	"Today favors planners over dreamers; sketch the next three steps.",
	"A small purchase today brings unexpectedly long-lasting joy.",
	"Family dynamics shift in your favor; a kind word goes a long way.",
	"Career-wise, someone notices your quiet excellence; be ready to speak up.",
	"Health gets a boost from one tiny habit; pick water, walk, or sleep.",
	"Travel plans gain clarity; book the thing you've been hesitating about.",
	"An old friend reaches out; reconnecting will warm you for days.",
	"Today is for finishing, not starting; close three loops and breathe.",
	"Luck favors the brave today; ask for what you actually want.",
	"Your intuition is sharper than usual; the gut feeling is the message.",
	"A creative collaboration sparks; share half-formed ideas freely.",
	"Slow mornings serve you well today; protect the first hour.",
	"Negotiations tilt in your favor; don't accept the first offer.",
	"A teacher appears in an unlikely place; listen more than you speak.",
	"Your patience is tested, but the lesson is golden; take notes.",
}

func fortuneDailySeed(sign string) int64 {
	now := time.Now()
	key := fmt.Sprintf("%s|%04d-%02d-%02d", strings.ToLower(sign), now.Year(), int(now.Month()), now.Day())
	h := fnv.New64a()
	h.Write([]byte(key))
	return int64(h.Sum64())
}

func FortuneHandler(m *tg.NewMessage) error {
	pick := fortuneList[fortuneRng.Intn(len(fortuneList))]
	msg := fmt.Sprintf("<b>Fortune</b>\n\n<i>%s</i>", html.EscapeString(pick))
	m.Reply(msg)
	return nil
}

func YesNoHandler(m *tg.NewMessage) error {
	pick := yesNoList[fortuneRng.Intn(len(yesNoList))]
	q := strings.TrimSpace(m.Args())
	var msg string
	if q != "" {
		msg = fmt.Sprintf("<b>Question:</b> %s\n<b>Answer:</b> <i>%s</i>", html.EscapeString(q), html.EscapeString(pick))
	} else {
		msg = fmt.Sprintf("<b>Answer:</b> <i>%s</i>", html.EscapeString(pick))
	}
	m.Reply(msg)
	return nil
}

func EightBallHandler(m *tg.NewMessage) error {
	q := strings.TrimSpace(m.Args())
	if q == "" {
		m.Reply("<b>Magic 8-Ball</b>\n\nUsage: <code>/8ball &lt;question&gt;</code>")
		return nil
	}
	pick := eightBallList[fortuneRng.Intn(len(eightBallList))]
	msg := fmt.Sprintf("<b>Magic 8-Ball</b>\n\n<b>Question:</b> %s\n<b>Answer:</b> <i>%s</i>", html.EscapeString(q), html.EscapeString(pick))
	m.Reply(msg)
	return nil
}

func HoroscopeHandler(m *tg.NewMessage) error {
	arg := strings.ToLower(strings.TrimSpace(m.Args()))
	if arg == "" {
		m.Reply("<b>Horoscope</b>\n\nUsage: <code>/horoscope &lt;sign&gt;</code>\n\n<b>Signs:</b> aries, taurus, gemini, cancer, leo, virgo, libra, scorpio, sagittarius, capricorn, aquarius, pisces")
		return nil
	}
	name, ok := horoscopeSigns[arg]
	if !ok {
		m.Reply("<b>Unknown sign:</b> " + html.EscapeString(arg) + "\n\nValid: aries, taurus, gemini, cancer, leo, virgo, libra, scorpio, sagittarius, capricorn, aquarius, pisces")
		return nil
	}
	seed := fortuneDailySeed(arg)
	r := rand.New(rand.NewSource(seed))
	pick := horoscopeMessages[r.Intn(len(horoscopeMessages))]
	luckyNum := r.Intn(99) + 1
	moods := []string{"Calm", "Focused", "Restless", "Joyful", "Reflective", "Energetic", "Curious", "Grounded", "Inspired", "Tender"}
	colors := []string{"Indigo", "Crimson", "Emerald", "Gold", "Silver", "Coral", "Teal", "Lavender", "Amber", "Ivory"}
	mood := moods[r.Intn(len(moods))]
	color := colors[r.Intn(len(colors))]
	emoji := horoscopeSignEmoji[arg]
	now := time.Now()
	date := fmt.Sprintf("%04d-%02d-%02d", now.Year(), int(now.Month()), now.Day())
	msg := fmt.Sprintf("<b>%s Horoscope</b> %s\n<b>Date:</b> <code>%s</code>\n\n<i>%s</i>\n\n<b>Mood:</b> %s\n<b>Lucky number:</b> <code>%d</code>\n<b>Lucky color:</b> %s",
		html.EscapeString(name), emoji, date, html.EscapeString(pick), html.EscapeString(mood), luckyNum, html.EscapeString(color))
	m.Reply(msg)
	return nil
}

func registerFortuneHandlers() {
	c := Client
	c.On("cmd:fortune", FortuneHandler)
	c.On("cmd:yesno", YesNoHandler)
	c.On("cmd:8ball", EightBallHandler)
	c.On("cmd:horoscope", HoroscopeHandler)

	Mods.AddModule("Fortune", `<b>Fortune Module</b>

<b>Commands:</b>
 • /fortune - Random fortune from a curated list
 • /yesno [question] - Quick yes/no/maybe oracle
 • /8ball &lt;question&gt; - Classic magic 8-ball with 20 responses
 • /horoscope &lt;sign&gt; - Deterministic daily horoscope per zodiac sign

<i>Daily horoscopes are seeded by date and sign, so they remain consistent for the whole day.</i>`)
}

func init() {
	QueueHandlerRegistration(registerFortuneHandlers)
}
