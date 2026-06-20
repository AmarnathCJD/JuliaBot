package modules

import (
	"html"
	"math/rand"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var truthList = []string{
	"What is the most embarrassing thing you have ever done in public?",
	"Have you ever lied to your best friend? About what?",
	"What is your biggest fear?",
	"Who was your first crush?",
	"What is the worst grade you have ever received?",
	"Have you ever stolen something? What was it?",
	"What is the most childish thing you still do?",
	"What is your most embarrassing memory from childhood?",
	"Have you ever cheated on a test?",
	"What is one secret you have never told anyone?",
	"What is the weirdest dream you have ever had?",
	"Who is the last person you stalked on social media?",
	"What is your worst habit?",
	"Have you ever pretended to be sick to skip school or work?",
	"What is the meanest thing you have ever said to someone?",
	"What is the longest you have gone without showering?",
	"Have you ever been caught lying? What about?",
	"What is the most embarrassing song on your playlist?",
	"What is something you have done that you have never told your parents?",
	"Who do you have a crush on right now?",
	"What is the strangest thing you have ever eaten?",
	"Have you ever peed in a swimming pool?",
	"What is your most irrational fear?",
	"What is the worst gift you have ever received?",
	"Have you ever talked behind a friend's back?",
	"What is the most embarrassing thing in your room right now?",
	"What is one thing you would change about yourself?",
	"What is the biggest lie you have ever told?",
	"Have you ever had a crush on a teacher?",
	"What is your guilty pleasure?",
	"Have you ever ghosted someone? Why?",
	"What is the most awkward date you have ever been on?",
	"What is something you are glad your mom does not know about?",
	"Have you ever read someone else's diary or messages?",
	"What is the dumbest thing you believed as a kid?",
	"What is the most trouble you have ever been in?",
	"Have you ever faked a laugh to make someone feel good?",
	"What is the weirdest thing you have done when alone?",
	"What is your most embarrassing nickname?",
	"Have you ever fallen asleep in class or at work?",
	"What is the longest you have stayed awake?",
	"What is something you have done that you regret?",
	"Have you ever sent a text to the wrong person? What did it say?",
	"What is the most embarrassing thing your parents have caught you doing?",
	"What is one thing you wish you could undo?",
	"Have you ever pretended to like a gift?",
	"What is your most unpopular opinion?",
	"Who in this room would you swap lives with for a day?",
	"What is the silliest thing you cried about as an adult?",
	"What is something you have done that you would never admit publicly?",
}

var dareList = []string{
	"Do 20 pushups right now.",
	"Sing the chorus of a song chosen by the group.",
	"Speak in a British accent for the next 10 minutes.",
	"Call a random contact and sing them happy birthday.",
	"Let someone draw on your face with a marker.",
	"Eat a spoonful of a condiment chosen by the group.",
	"Text your crush something embarrassing.",
	"Do a handstand against a wall for 30 seconds.",
	"Speak only in questions for the next 5 minutes.",
	"Let the group post anything they want on your social media.",
	"Imitate your favorite celebrity for 2 minutes.",
	"Dance with no music for one full minute.",
	"Try to lick your elbow.",
	"Send a screenshot of your last text conversation to the group.",
	"Wear your clothes inside out for the rest of the game.",
	"Do your best impression of a chicken.",
	"Eat a piece of food without using your hands.",
	"Let someone style your hair however they want.",
	"Talk in a baby voice until your next turn.",
	"Do 10 jumping jacks while singing the alphabet.",
	"Show the most recent photo on your phone.",
	"Let the group choose your phone wallpaper for the day.",
	"Hold an ice cube in your hand until it melts.",
	"Smell every player's armpit and rank them.",
	"Speak only in song lyrics until your next turn.",
	"Do a dramatic reading of the last text you sent.",
	"Call a friend and propose marriage.",
	"Walk like a crab for the next 3 minutes.",
	"Let someone tickle you for 30 seconds without laughing.",
	"Eat a tablespoon of hot sauce.",
	"Do a cartwheel right now.",
	"Pretend to be a news anchor for 2 minutes.",
	"Let the group choose a new nickname for you for the rest of the night.",
	"Try to do the splits.",
	"Wear socks on your hands for the next 10 minutes.",
	"Do your best evil laugh for the group.",
	"Eat a raw piece of garlic.",
	"Let the group send a text from your phone to anyone they choose.",
	"Do a runway walk down the hallway.",
	"Pretend to be a robot for the next 5 minutes.",
	"Let someone put makeup on you with their eyes closed.",
	"Howl like a wolf out the window.",
	"Do an interpretive dance to the next song that plays.",
	"Speak with your mouth full of water for 30 seconds.",
	"Let the group take an unflattering picture of you and post it.",
	"Hop on one foot until your next turn.",
	"Pretend to cry dramatically for one minute.",
	"Eat a slice of lemon without making a face.",
	"Let the group give you a new hairstyle using only their hands.",
	"Do your best impression of another player and let them guess who it is.",
}

func TruthHandler(m *tg.NewMessage) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	truth := truthList[r.Intn(len(truthList))]
	m.Reply("<b>Truth</b>\n\n" + html.EscapeString(truth))
	return nil
}

func DareHandler(m *tg.NewMessage) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	dare := dareList[r.Intn(len(dareList))]
	m.Reply("<b>Dare</b>\n\n" + html.EscapeString(dare))
	return nil
}

func TodHandler(m *tg.NewMessage) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if r.Intn(2) == 0 {
		truth := truthList[r.Intn(len(truthList))]
		m.Reply("<b>Truth</b>\n\n" + html.EscapeString(truth))
	} else {
		dare := dareList[r.Intn(len(dareList))]
		m.Reply("<b>Dare</b>\n\n" + html.EscapeString(dare))
	}
	return nil
}

func init() { QueueHandlerRegistration(registerTruthDareHandlers) }
func registerTruthDareHandlers() {
	c := Client
	c.On("cmd:truth", TruthHandler)
	c.On("cmd:dare", DareHandler)
	c.On("cmd:tod", TodHandler)
}
