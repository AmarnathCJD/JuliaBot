package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"math/rand"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type uselessFactResponse struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	Source     string `json:"source"`
	SourceURL  string `json:"source_url"`
	Language   string `json:"language"`
	Permalink  string `json:"permalink"`
}

type catFactResponse struct {
	Fact   string `json:"fact"`
	Length int    `json:"length"`
}

var punList = []string{
	"I'm reading a book about anti-gravity. It's impossible to put down.",
	"I used to be a banker, but I lost interest.",
	"Time flies like an arrow. Fruit flies like a banana.",
	"I told my wife she was drawing her eyebrows too high. She looked surprised.",
	"I'm on a seafood diet. I see food and I eat it.",
	"Why don't scientists trust atoms? Because they make up everything.",
	"I would tell you a chemistry joke, but I know I wouldn't get a reaction.",
	"Why did the scarecrow win an award? Because he was outstanding in his field.",
	"I used to play piano by ear, but now I use my hands.",
	"I'm reading a horror book in braille. Something bad is about to happen, I can feel it.",
	"Did you hear about the mathematician who's afraid of negative numbers? He'll stop at nothing to avoid them.",
	"I told my computer I needed a break, and it said 'No problem, I'll go to sleep.'",
	"Why don't skeletons fight each other? They don't have the guts.",
	"I'm friends with 25 letters of the alphabet. I don't know Y.",
	"What do you call fake spaghetti? An impasta.",
	"I gave all my dead batteries away today, free of charge.",
	"I don't trust stairs. They're always up to something.",
	"What did one ocean say to the other ocean? Nothing, they just waved.",
	"Why don't eggs tell jokes? They'd crack each other up.",
	"I used to hate facial hair, but then it grew on me.",
	"Why did the bicycle fall over? Because it was two-tired.",
	"I'm terrified of elevators, so I'm going to start taking steps to avoid them.",
	"What do you call a fake noodle? An impasta.",
	"Did you hear about the cheese factory that exploded? There was nothing left but de-brie.",
	"I tried to catch fog yesterday. Mist.",
	"What do you call a sleeping bull? A bulldozer.",
	"Why did the math book look so sad? Because it had too many problems.",
	"What did the grape say when it got stepped on? Nothing, it just let out a little wine.",
	"I'm so good at sleeping, I can do it with my eyes closed.",
	"What do you call a bear with no teeth? A gummy bear.",
	"Why did the cookie go to the doctor? Because it was feeling crummy.",
	"I'm reading a book on the history of glue. I just can't seem to put it down.",
	"What do you call a fish wearing a crown? A king fish.",
	"Why don't oysters donate to charity? Because they're shellfish.",
	"What did the buffalo say to his son when he left for college? Bison.",
	"I used to be addicted to soap, but I'm clean now.",
	"Why did the golfer bring two pairs of pants? In case he got a hole in one.",
	"What do you get when you cross a snowman and a vampire? Frostbite.",
	"I'm not lazy, I'm on energy-saving mode.",
	"What do you call a dinosaur with an extensive vocabulary? A thesaurus.",
	"Why was the math book stressed? Because of all the problems.",
	"What did one wall say to the other? I'll meet you at the corner.",
	"I would tell you a joke about pizza, but it's a little cheesy.",
	"Why did the coffee file a police report? It got mugged.",
	"What's orange and sounds like a parrot? A carrot.",
	"I tried to write a novel about a chicken, but I gave up halfway through.",
	"Why don't scientists trust stairs? Because they're always up to something.",
	"What do you call a pile of cats? A meowtain.",
	"I told my doctor I broke my arm in two places. He said stop going to those places.",
	"What did the ocean say to the shore? Nothing, it just waved.",
	"Why did the tomato turn red? Because it saw the salad dressing.",
	"I'm reading a book about mazes. I got lost in it.",
	"What do you call cheese that isn't yours? Nacho cheese.",
	"Why did the picture go to jail? Because it was framed.",
	"I'm writing a book about reverse psychology. Don't read it.",
	"What did the zero say to the eight? Nice belt.",
	"Why did the banana go to the doctor? Because it wasn't peeling well.",
	"I used to be a baker, but I couldn't make enough dough.",
	"What do you call a belt made of watches? A waist of time.",
	"Why did the man put his money in the blender? He wanted to make liquid assets.",
	"I'm reading a book on hot air balloons. It's a real page-turner.",
	"What did one hat say to the other? You stay here, I'll go on ahead.",
	"Why did the computer go to the doctor? Because it had a virus.",
	"I told a chemistry joke once. There was no reaction.",
	"What do you call a snowman with a six-pack? An abdominal snowman.",
	"Why did the lamp sink? Because it saw the light bulb.",
	"I'm not arguing, I'm just explaining why I'm right.",
	"What do you call a cow with no legs? Ground beef.",
	"Why don't programmers like nature? It has too many bugs.",
	"I tried to start a hot air balloon business, but it never took off.",
	"What did the traffic light say to the car? Don't look, I'm changing.",
	"Why did the cow go to space? To see the moooon.",
	"I'm so broke, I can't even pay attention.",
	"What do you call a lazy kangaroo? A pouch potato.",
	"Why did the orange stop in the middle of the road? Because it ran out of juice.",
	"I used to be a tap dancer, but I fell in the sink.",
	"What did the dad buffalo say to his kid? Bison.",
	"Why did the music teacher need a ladder? To reach the high notes.",
	"I asked the librarian if she had a book on paranoia. She whispered, it's right behind you.",
	"What do you call a fish without eyes? Fsh.",
	"Why did the smartphone need glasses? It lost its contacts.",
	"I'm not a complete idiot. Some parts are missing.",
	"What did one elevator say to the other? I think I'm coming down with something.",
	"Why did the chicken join a band? Because it had drumsticks.",
	"I have a fear of speed bumps. I'm slowly getting over it.",
	"What do you call a deer with no eyes? No idea.",
	"Why did the gym close down? It just didn't work out.",
	"I told my suitcases there will be no vacation this year. Now I'm dealing with emotional baggage.",
	"What do you call a sad strawberry? A blueberry.",
	"Why did the bicycle take a nap? It was two tired.",
	"I'm on a whiskey diet. I've lost three days already.",
	"What did the janitor say when he jumped out of the closet? Supplies.",
	"Why did the duck cross the road? To prove he wasn't chicken.",
	"I used to be a personal trainer, but then I gave my too weak notice.",
	"What do you call a bee that can't make up its mind? A maybe.",
	"Why was the broom late? It overswept.",
	"I'm a big fan of whiteboards. They're remarkable.",
	"What do you call a cow during an earthquake? A milkshake.",
	"Why don't melons get married? Because they cantaloupe.",
	"I told my plants a joke. They didn't laugh, they're stoic.",
	"What do you call a pony with a cough? A little horse.",
	"Why did the stadium get hot after the game? All the fans left.",
}

func RandomFactHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://uselessfacts.jsph.pl/api/v2/facts/random?language=en")
	if err != nil {
		m.Reply("couldn't fetch fact: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data uselessFactResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't fetch fact: " + err.Error())
		return nil
	}
	if data.Text == "" {
		m.Reply("couldn't fetch fact: empty response")
		return nil
	}
	out := "<b>Random Useless Fact</b>\n\n" + html.EscapeString(data.Text)
	if data.Source != "" {
		out += "\n\n<i>Source: " + html.EscapeString(data.Source) + "</i>"
	}
	m.Reply(out)
	return nil
}

func PunnyHandler(m *tg.NewMessage) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pun := punList[r.Intn(len(punList))]
	m.Reply("<b>Pun</b>\n\n" + html.EscapeString(pun))
	return nil
}

func CatFactHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://catfact.ninja/fact")
	if err != nil {
		m.Reply("couldn't fetch cat fact: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data catFactResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't fetch cat fact: " + err.Error())
		return nil
	}
	if data.Fact == "" {
		m.Reply("couldn't fetch cat fact: empty response")
		return nil
	}
	m.Reply("<b>Cat Fact</b>\n\n" + html.EscapeString(data.Fact))
	return nil
}

func init() { QueueHandlerRegistration(registerJokesSpecialHandlers) }
func registerJokesSpecialHandlers() {
	c := Client
	c.On("cmd:randomfact", RandomFactHandler)
	c.On("cmd:punny", PunnyHandler)
	c.On("cmd:catfact", CatFactHandler)
}
