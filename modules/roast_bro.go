package modules

import (
	"fmt"
	"html"
	"math/rand"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var roastBroList = []string{
	"You're not stupid; you just have bad luck thinking.",
	"You bring everyone so much joy when you leave the room.",
	"If I had a face like yours, I'd sue my parents.",
	"You're proof that even evolution takes breaks.",
	"Your secrets are always safe with me. I never even listen.",
	"You have the perfect face for radio.",
	"You're like a cloud. When you disappear, it's a beautiful day.",
	"I'd agree with you, but then we'd both be wrong.",
	"You're the human equivalent of a participation trophy.",
	"Some people just need a high-five. In the face. With a chair.",
	"You're not the dumbest person on the planet, but you sure better hope they don't die.",
	"You're like a software update. Whenever I see you, I think 'not now'.",
	"I'd explain it to you, but I left my crayons at home.",
	"Your face makes onions cry.",
	"You're the reason the gene pool needs a lifeguard.",
	"You have something on your chin. No, the third one down.",
	"If laughter is the best medicine, your face must be curing the world.",
	"You're like a penny: two-faced and not worth much.",
	"You're proof that God has a sense of humor.",
	"You're the reason shampoo has instructions.",
	"I'm jealous of people who don't know you.",
	"You're not even worth the first round of insults.",
	"If you were any slower, you'd be going backwards.",
	"You bring everyone so much joy. Especially when you leave.",
	"You're like a broken pencil. Pointless.",
	"You're a grey sprinkle on a rainbow cupcake.",
	"You're as bright as a black hole and twice as dense.",
	"You're the reason instructions on shampoo bottles exist.",
	"You're like Monday mornings. Nobody likes you.",
	"You're not completely useless. You can always serve as a bad example.",
	"You look like something I'd draw with my left hand.",
	"You have the personality of damp cardboard.",
	"You're like a cloud. When you go away, it's a beautiful day.",
	"You must have been born on a highway, because that's where most accidents happen.",
	"You're the human version of a participation award.",
	"Your face could make a glass eye cry.",
	"I'd call you stupid, but that would be an insult to stupid people.",
	"You're the reason mirrors were invented and then immediately regretted.",
	"If brains were dynamite, you couldn't blow your nose.",
	"You're so dense, light bends around you.",
	"Roses are red, violets are blue, I have five fingers, the middle one's for you.",
	"You're the kind of person who'd trip over a cordless phone.",
	"You're like a candle in the wind. Unstable and quickly extinguished.",
	"You're the reason aliens won't visit us.",
	"You're not a complete idiot. Some parts are missing.",
	"You bring the meaning of 'less is more' to a whole new level.",
	"You're as useless as the 'g' in lasagna.",
	"You're like an iPhone without WiFi. Limited functionality.",
	"You have a face that could stop a clock.",
	"You're the human equivalent of a typo.",
	"If you were a spice, you'd be flour.",
	"You're so boring, you make watching paint dry exciting.",
	"You're like a cloud of dust: pointless, annoying, and everywhere.",
	"You're so slow, it takes you an hour and a half to watch 60 Minutes.",
	"You're proof that nature has a recycling problem.",
	"You're the reason the warning labels say 'do not eat'.",
	"You're like a software bug. Annoying and hard to get rid of.",
	"You have the charisma of a wet sock.",
	"You're the reason we can't have nice things.",
	"You're the human equivalent of a 404 error.",
	"You're so basic, autocorrect finishes your sentences.",
	"You're like a screen door on a submarine. Useless.",
	"You're not even mediocre, you're below average at being average.",
	"You bring everyone joy when you cancel plans.",
	"You're so forgettable, even your reflection ghosts you.",
	"You're the kind of person who claps when the plane lands.",
	"You're like a broken vending machine. Takes everything, gives nothing.",
	"You're the reason every group chat has a mute button.",
	"You're so slow, snails finish marathons before you tie your shoes.",
	"You have the depth of a puddle and the wit of a paperclip.",
	"You're the human equivalent of a dropped call.",
	"You're like decaf coffee. What's even the point?",
	"You're so unoriginal, your shadow tries to ditch you.",
	"You're the kind of person who'd microwave a salad.",
	"You're like a free trial. Limited features, expires fast.",
	"You're so basic, mayonnaise calls you bland.",
	"You're the reason elevators have mirrors. To distract from you.",
	"You're like a flat tire. Always slowing everyone down.",
	"You're so dull, you make C-SPAN look like a thriller.",
	"You're the human version of a participation trophy nobody wanted.",
	"You're like a low battery warning. Inconvenient and always there.",
}

var roastReplyList = []string{
	"%s, you're not stupid; you just have bad luck thinking.",
	"%s, you bring everyone so much joy when you leave the room.",
	"%s, you're proof that even evolution takes breaks.",
	"%s, your secrets are always safe with me. I never even listen.",
	"%s, you have the perfect face for radio.",
	"%s, you're like a cloud. When you disappear, it's a beautiful day.",
	"%s, I'd agree with you, but then we'd both be wrong.",
	"%s, you're the human equivalent of a participation trophy.",
	"%s, you're like a software update. Whenever I see you, I think 'not now'.",
	"%s, I'd explain it to you, but I left my crayons at home.",
	"%s, your face makes onions cry.",
	"%s, you're the reason the gene pool needs a lifeguard.",
	"%s, if laughter is the best medicine, your face must be curing the world.",
	"%s, you're like a penny: two-faced and not worth much.",
	"%s, you're proof that God has a sense of humor.",
	"%s, you're the reason shampoo has instructions.",
	"%s, I'm jealous of people who don't know you.",
	"%s, if you were any slower, you'd be going backwards.",
	"%s, you're like a broken pencil. Pointless.",
	"%s, you're a grey sprinkle on a rainbow cupcake.",
	"%s, you're as bright as a black hole and twice as dense.",
	"%s, you're like Monday mornings. Nobody likes you.",
	"%s, you're not completely useless. You can always serve as a bad example.",
	"%s, you look like something I'd draw with my left hand.",
	"%s, you have the personality of damp cardboard.",
	"%s, you must have been born on a highway, because that's where most accidents happen.",
	"%s, your face could make a glass eye cry.",
	"%s, I'd call you stupid, but that would be an insult to stupid people.",
	"%s, if brains were dynamite, you couldn't blow your nose.",
	"%s, you're so dense, light bends around you.",
	"%s, you're the kind of person who'd trip over a cordless phone.",
	"%s, you're like a candle in the wind. Unstable and quickly extinguished.",
	"%s, you're the reason aliens won't visit us.",
	"%s, you're not a complete idiot. Some parts are missing.",
	"%s, you're as useless as the 'g' in lasagna.",
	"%s, you have a face that could stop a clock.",
	"%s, you're the human equivalent of a typo.",
	"%s, you're so boring, you make watching paint dry exciting.",
	"%s, you're so slow, it takes you an hour and a half to watch 60 Minutes.",
	"%s, you're proof that nature has a recycling problem.",
	"%s, you have the charisma of a wet sock.",
	"%s, you're the human equivalent of a 404 error.",
	"%s, you're like a screen door on a submarine. Useless.",
	"%s, you're the kind of person who claps when the plane lands.",
	"%s, you're so forgettable, even your reflection ghosts you.",
	"%s, you're like a broken vending machine. Takes everything, gives nothing.",
	"%s, you're so slow, snails finish marathons before you tie your shoes.",
	"%s, you have the depth of a puddle and the wit of a paperclip.",
	"%s, you're the human equivalent of a dropped call.",
	"%s, you're like decaf coffee. What's even the point?",
}

var complimentList = []string{
	"You light up every room you enter.",
	"Your smile is contagious in the best way.",
	"You're the human equivalent of a warm cup of cocoa.",
	"Your kindness makes the world softer.",
	"You have a wonderful way of making people feel heard.",
	"You're stronger than you know.",
	"You make ordinary moments magical.",
	"Your laugh is one of the best sounds in the universe.",
	"You bring out the best in everyone around you.",
	"You're a walking masterpiece.",
	"Your energy is genuinely magnetic.",
	"The world is more interesting because you're in it.",
	"You have a heart of pure gold.",
	"You're proof that good people still exist.",
	"You inspire people without even trying.",
	"You're the friend everyone wishes they had.",
	"Your perspective on life is refreshing.",
	"You make hard things look easy.",
	"You're an absolute treasure.",
	"You have an incredible sense of humor.",
	"You handle tough situations with grace.",
	"Your creativity is boundless.",
	"You make every conversation better.",
	"You radiate positive vibes.",
	"You're braver than you believe.",
	"Your dedication is admirable.",
	"You make the impossible feel possible.",
	"Your patience is a superpower.",
	"You have a beautiful soul.",
	"You're an unforgettable kind of person.",
	"Your authenticity is rare and precious.",
	"You give the best advice.",
	"You're a quiet kind of extraordinary.",
	"You make people feel safe.",
	"Your presence is a gift.",
	"You're stronger than any storm you've faced.",
	"You're the reason someone smiled today.",
	"Your kindness ripples further than you know.",
	"You're a lighthouse in a foggy world.",
	"You're absolutely amazing, just as you are.",
}

var motivateList = []string{
	"You are stronger than your strongest excuse.",
	"Small steps every day still get you there.",
	"Done is better than perfect. Ship it.",
	"The hardest part is starting. Start.",
	"Your future self is watching. Don't disappoint them.",
	"Discipline beats motivation. Show up anyway.",
	"You don't need to be ready. You need to begin.",
	"Doubt kills more dreams than failure ever will.",
	"Progress, no matter how small, is still progress.",
	"You are capable of more than you think.",
	"The pain of discipline weighs less than the pain of regret.",
	"Don't count the days. Make the days count.",
	"Be the person your past self needed.",
	"The grind is quiet. The results are loud.",
	"You didn't come this far to only come this far.",
	"Comfort and growth cannot coexist.",
	"Hard times build hard people. Lean in.",
	"Action cures fear.",
	"Stop waiting for permission to be great.",
	"One day or day one. Your choice.",
	"You are one decision away from a totally different life.",
	"Energy flows where focus goes.",
	"The dream is free. The hustle is sold separately.",
	"Make peace with not being understood.",
	"Don't lower the goal. Raise the action.",
	"You are not behind. You are exactly where you need to start.",
	"Outwork your insecurities.",
	"Stay patient. Trust the process.",
	"The version of you that you're becoming is rooting for you.",
	"Showing up tired still counts. Keep going.",
	"Be obsessed or be average.",
	"Burn the bridge to the easy way out.",
	"You can rest. But don't quit.",
	"Greatness is forged in the unseen hours.",
	"You become what you repeatedly do. Choose wisely.",
	"Stop comparing your beginning to someone's middle.",
	"Hard work in silence. Let success make the noise.",
	"Don't break the chain. One more day.",
	"You're not stuck. You're recalibrating.",
	"Keep going. The view from the top is worth it.",
}

var dissList = []string{
	"%s, you're the human equivalent of a pop-up ad.",
	"%s, even your shadow tries to leave you on read.",
	"%s, you're so dull, sleep aids take notes from you.",
	"%s, you're the reason muting group chats exists.",
	"%s, your personality has a buffering icon.",
	"%s, you're like expired milk: nobody asked, but here you are.",
	"%s, if disappointment had a face, it would still upgrade from yours.",
	"%s, you've got the charisma of unsalted crackers.",
	"%s, your vibe is 'do not resuscitate'.",
	"%s, you're proof that some NPCs forgot to load.",
	"%s, your opinions belong in the recycling bin, unsorted.",
	"%s, you bring the room down faster than a bad WiFi signal.",
	"%s, you've got the depth of a sticker and the impact of a sneeze in a hurricane.",
	"%s, you're the side quest nobody picks up.",
	"%s, your existence is a typo in the universe's draft.",
	"%s, you're so forgettable, my memory garbage-collects you in real time.",
	"%s, your comebacks expire before they leave your mouth.",
	"%s, you're the reason elevators stay silent.",
	"%s, you have main-character syndrome with background-actor energy.",
	"%s, you're a permanent loading screen with no payoff.",
	"%s, if mediocrity had a mascot, you'd be the runner-up.",
	"%s, your aura is dial-up internet in a 5G world.",
	"%s, you're the human version of an unread terms of service.",
	"%s, your relevance peaked the day you were born and went downhill from there.",
	"%s, you're the spam folder of social interactions.",
}

func RoastMeHandler(m *tg.NewMessage) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	name := html.EscapeString(m.Sender.FirstName)
	if name == "" {
		name = "you"
	}
	line := roastBroList[r.Intn(len(roastBroList))]
	m.Reply("<b>" + name + "</b>, " + html.EscapeString(line))
	return nil
}

func RoastReplyHandler(m *tg.NewMessage) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if !m.IsReply() {
		line := roastBroList[r.Intn(len(roastBroList))]
		m.Reply("<i>Reply to someone to roast them. Have one on me:</i>\n\n" + html.EscapeString(line))
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil || reply == nil || reply.Sender == nil {
		m.Reply("Couldn't find that user to roast.")
		return nil
	}
	name := html.EscapeString(reply.Sender.FirstName)
	if name == "" {
		name = "this one"
	}
	tmpl := roastReplyList[r.Intn(len(roastReplyList))]
	parts := strings.SplitN(tmpl, "%s", 2)
	if len(parts) == 2 {
		m.Reply(html.EscapeString(parts[0]) + "<b>" + name + "</b>" + html.EscapeString(parts[1]))
	} else {
		m.Reply(html.EscapeString(fmt.Sprintf(tmpl, name)))
	}
	return nil
}

func ComplimentHandler(m *tg.NewMessage) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	line := complimentList[r.Intn(len(complimentList))]
	name := html.EscapeString(m.Sender.FirstName)
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply != nil && reply.Sender != nil && reply.Sender.FirstName != "" {
			name = html.EscapeString(reply.Sender.FirstName)
		}
	}
	if name == "" {
		name = "Friend"
	}
	m.Reply("<b>" + name + "</b>, " + html.EscapeString(line))
	return nil
}

func MotivateHandler(m *tg.NewMessage) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	line := motivateList[r.Intn(len(motivateList))]
	m.Reply("<b>Motivation</b>\n\n" + html.EscapeString(line))
	return nil
}

func DissHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a user to /diss them. No self-disses allowed.")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil || reply == nil || reply.Sender == nil {
		m.Reply("Couldn't find that user to diss.")
		return nil
	}
	if reply.Sender.ID == m.SenderID() {
		m.Reply("You can't diss yourself. Try /roastme if you want self-punishment.")
		return nil
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	name := html.EscapeString(reply.Sender.FirstName)
	if name == "" {
		name = "this one"
	}
	tmpl := dissList[r.Intn(len(dissList))]
	parts := strings.SplitN(tmpl, "%s", 2)
	if len(parts) == 2 {
		m.Reply(html.EscapeString(parts[0]) + "<b>" + name + "</b>" + html.EscapeString(parts[1]))
	} else {
		m.Reply(html.EscapeString(fmt.Sprintf(tmpl, name)))
	}
	return nil
}

func init() { QueueHandlerRegistration(registerRoastBroHandlers) }
func registerRoastBroHandlers() {
	c := Client
	c.On("cmd:roastme", RoastMeHandler)
	c.On("cmd:roast", RoastReplyHandler)
	c.On("cmd:compliment", ComplimentHandler)
	c.On("cmd:motivate", MotivateHandler)
	c.On("cmd:diss", DissHandler)
}
