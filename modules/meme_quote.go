package modules

import (
	"fmt"
	"html"
	"math/rand"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var memeQuoteRng = rand.New(rand.NewSource(time.Now().UnixNano()))

var iQuoteList = []string{
	"git push --force, then run.",
	"There are only two hard problems: cache invalidation, naming things, and off-by-one errors.",
	"It works on my machine, ship the machine.",
	"Why fix the bug when you can rename it a feature?",
	"Premature optimization is the root of all promotions.",
	"My code does not have bugs. It develops random unexpected features.",
	"If at first you don't succeed, blame the intern.",
	"A senior engineer is just a junior who has Stack Overflow bookmarked.",
	"Real programmers count from zero, especially their salary.",
	"The cloud is just someone else's broken laptop.",
	"AI will not replace programmers, but programmers who use AI will replace those who don't, and then AI replaces them anyway.",
	"Documentation is a love letter you write to your future self, then never read.",
	"There is no place like 127.0.0.1.",
	"Pair programming is two engineers wondering why nothing compiles.",
	"Refactor: the act of breaking things that already worked.",
	"You haven't truly lived until you've debugged production at 3 AM.",
	"Legacy code is code without tests written by a past version of yourself.",
	"Move fast and break things, then move slowly and explain things to legal.",
	"Microservices: now your monolith fails in 47 places at once.",
	"Kubernetes is just YAML cosplay.",
	"In theory, theory and practice are the same. In practice, your linter disagrees.",
	"Don't comment bad code, rewrite it. Then don't comment that either.",
	"A clever programmer is a future maintenance nightmare.",
	"Standups are just shared trauma rituals.",
	"If it compiles, ship it. If it doesn't, ship it anyway with a feature flag.",
	"The best error message is the one that never logs.",
	"Stack traces are just love notes from the runtime.",
	"Code reviews are organized arguments with extra steps.",
	"Every regex you write today is a war crime against your future self.",
	"Logging is just print debugging that grew up and got a job.",
	"Software is never finished, only abandoned with confidence.",
	"The user is always wrong, but louder.",
	"A good README is 90% screenshots and 10% lies.",
	"The roadmap is a horoscope for product managers.",
	"There are 10 types of people: those who debug binary and those who give up.",
	"Just rewrite it in Rust. That's the whole quote.",
	"Tabs vs spaces is the only war that never ends, because both sides are wrong.",
	"Continuous integration means continuously discovering you were wrong.",
	"Your test suite is only as fast as the slowest engineer who wrote it.",
	"Open source: free as in puppy.",
	"Hotfix Friday is a lifestyle.",
	"Senior engineers don't fix bugs, they negotiate with them.",
	"The deployment that never gets rolled back is the deployment that never happened.",
	"DevOps is just developers who learned to fear YAML.",
	"Every TODO comment is a promise made by a stranger.",
	"A monorepo is one mistake in 400 directories.",
	"You don't deploy on Friday because the weekend is also production.",
	"The shortest path between two bugs is a one-line patch.",
	"Behind every working feature is a config file nobody understands.",
	"Programming is just typing until the red squiggles go away.",
}

var devJokeList = []string{
	"Why do programmers prefer dark mode? Because light attracts bugs.",
	"How many programmers does it take to change a light bulb? None, that's a hardware problem.",
	"Why do Java developers wear glasses? Because they don't C#.",
	"A SQL query walks into a bar, sees two tables, and asks: 'May I join you?'",
	"Why did the developer go broke? Because he used up all his cache.",
	"Why was the JavaScript developer sad? Because he didn't Node how to Express himself.",
	"What is a programmer's favorite hangout place? The Foo Bar.",
	"Why do programmers always mix up Halloween and Christmas? Because Oct 31 == Dec 25.",
	"There are 10 kinds of people in this world: those who understand binary and those who don't.",
	"Why did the programmer quit his job? Because he didn't get arrays.",
	"What did the router say to the doctor? 'It hurts when IP.'",
	"Why do Python programmers have low self-esteem? Because they're constantly comparing their self to others.",
	"What is a programmer's favorite snack? Microchips.",
	"Why did the database administrator leave his wife? She had one-to-many relationships.",
	"Why was the function feeling down? It didn't get called.",
	"Knock knock. Race condition. Who's there?",
	"What is the object-oriented way to become wealthy? Inheritance.",
	"Why did the developer go to therapy? He had too many unresolved dependencies.",
	"Why don't programmers like nature? It has too many bugs.",
	"How do you comfort a JavaScript bug? You console it.",
	"What is a programmer's favorite type of music? Algo-rhythms.",
	"Why did the programmer's wife leave him? She said he had too many issues he couldn't close.",
	"What is a developer's favorite kind of tea? URL grey.",
	"Why do programmers hate the outdoors? Too many trees and not enough nodes.",
	"What is the most used language in programming? Profanity.",
	"Why was the JavaScript file so anxious? It had too many callbacks.",
	"How does a programmer open a jar? They use a Java key.",
	"Why did the developer get stuck in the shower? The instructions said: lather, rinse, repeat.",
	"What is a Sith Lord's favorite programming language? SQL, because they always use ORDER BY.",
	"Why do front-end developers eat lunch alone? Because they don't know how to join tables.",
	"My code doesn't work, I have no idea why. My code works, I have no idea why.",
	"Why did the C++ programmer get evicted? He kept dereferencing his landlord.",
	"A QA engineer walks into a bar. Orders a beer. Orders 0 beers. Orders 99999999 beers. Orders a lizard. Orders -1 beers. Orders a NULL.",
	"What is a programmer's favorite cereal? Semicolon Crunch.",
	"Why was the developer cold? He left his Windows open.",
	"How do you tell an introverted developer from an extroverted one? The extrovert looks at YOUR shoes.",
	"Why did the for loop break up with the while loop? It had commitment issues, it kept iterating.",
	"What do you call a programmer from Finland? Nerdic.",
	"Why don't bachelors like Git? Because they're afraid of commits.",
	"What is a recursive function's favorite song? 'Stuck in the Middle With You.'",
	"Why was the linked list always tired? It was up all night chasing pointers.",
	"How do you generate a random string? Put a junior developer in front of vim and ask them to exit.",
	"Why did the developer become a baker? He kneaded the dough.",
	"What is a hacker's favorite season? Phishing season.",
	"Why was the smartphone wearing glasses? It lost its contacts.",
	"How many software engineers does it take to fix a hardware bug? None. We'll document it as a feature.",
	"Why don't programmers go to the beach? Cats keep burying them in sand.",
	"What is a developer's favorite horror movie? The Ring Buffer.",
	"Why did the assembly programmer drown? Because he didn't C.",
	"What does a programmer wear on Halloween? A null costume, because they cannot reference it.",
	"Why was the computer cold at the office? It left its Windows open and the Linux ran out.",
	"What do you call eight hobbits? A hobbyte.",
	"Why do developers prefer iOS? Because Android has too many issues with intents.",
	"A programmer's wife says: 'Go to the store, get a loaf of bread. If they have eggs, get a dozen.' He comes back with 12 loaves of bread.",
	"Why did the developer cross the road? To refactor the chicken on the other side.",
	"What is a developer's favorite drink? Java.",
	"Why was the function so polite? It always returned promptly.",
	"How do you know a developer is an extrovert? They debug out loud.",
	"What is the difference between a junior and senior developer? About 200 lines of unnecessary code.",
	"Why did the programmer stay in the shower so long? He misread the shampoo bottle: 'while (true) { lather; rinse; }'.",
	"What is a programmer's favorite plant? An IF tree.",
	"Why are Assembly programmers always wet? They work below C level.",
	"What did the array say at the party? Don't make me push you.",
	"Why did the developer get hired immediately? He had great references.",
	"What is the difference between Git and GitHub? About 8 billion dollars.",
	"Why did the programmer name his dog Bash? Because he kept executing commands.",
	"How does a programmer fix their car? Turn it off and on again.",
	"Why did the variable break up with the constant? It needed space to change.",
	"What is the most patient profession? A compiler. It waits forever for you to fix one missing semicolon.",
	"Why did the programmer hate the forest? Too many recursive trees.",
	"What is a developer's favorite exercise? Push, pull, commit.",
	"Why was the regex always alone? Nobody could match its expectations.",
	"How do you cheer up a sad container? Give it a hug, or at least a Docker compose.",
	"Why did the programmer fail his driving test? He kept hitting the brakes recursively.",
	"What is a hacker's favorite music genre? Hash metal.",
	"Why don't programmers tell jokes in octal? Because 7, 10, 11.",
	"What is the difference between a programmer and a non-programmer? The non-programmer thinks a kilobyte is 1000 bytes.",
	"Why did the keyboard go to therapy? It had too many issues with shift.",
	"What is the difference between a tester and a developer? A tester opens 99 tabs to break the app. A developer opens 99 tabs because they forgot to close them.",
	"Why was the cloud sad? It had too many dropped packets.",
	"What did the closure say to the variable? I'll remember you forever.",
	"Why are pirates great programmers? They love the C.",
}

var aphorismList = []string{
	"The best code is no code at all.",
	"Make it work, make it right, make it fast, in that order.",
	"Programs must be written for people to read, and only incidentally for machines to execute.",
	"Simplicity is prerequisite for reliability.",
	"Premature optimization is the root of all evil.",
	"There are two ways of constructing a software design: simple, or complicated.",
	"Walking on water and developing software from a specification are easy if both are frozen.",
	"Any fool can write code that a computer can understand. Good programmers write code that humans can understand.",
	"Talk is cheap. Show me the code.",
	"Weeks of coding can save you hours of planning.",
	"Code is read more often than it is written.",
	"Deleted code is debugged code.",
	"If you can't explain it simply, you don't understand it well enough.",
	"The function of good software is to make the complex appear simple.",
	"Controlling complexity is the essence of computer programming.",
	"The most disastrous thing you can ever learn is your first programming language.",
	"Without requirements or design, programming is the art of adding bugs to an empty text file.",
	"First, solve the problem. Then, write the code.",
	"Experience is the name everyone gives to their mistakes.",
	"The best performance improvement is the transition from the nonworking state to the working state.",
	"Programming is the art of telling another human what one wants the computer to do.",
	"A good programmer is someone who always looks both ways before crossing a one-way street.",
	"Software and cathedrals are much the same; first we build them, then we pray.",
	"The cheapest, fastest, and most reliable components are those that aren't there.",
	"Optimism is an occupational hazard of programming; feedback is the treatment.",
	"Good code is its own best documentation.",
	"It is easier to write an incorrect program than to understand a correct one.",
	"Programs are meant to be read by humans and only incidentally for computers to execute.",
	"There is no programming language, no matter how structured, that will prevent programmers from making bad programs.",
	"The most important property of a program is whether it accomplishes the intention of its user.",
	"The competent programmer is fully aware of the limited size of his own skull.",
	"A language that doesn't affect the way you think about programming is not worth knowing.",
	"Computers are good at following instructions, but not at reading your mind.",
	"You can't have great software without a great team.",
	"Code never lies, comments sometimes do.",
	"Programming is breaking of one big impossible task into smaller possible tasks.",
	"Quality is never an accident; it is always the result of intelligent effort.",
	"It's not a bug, it's an undocumented feature.",
	"Adding manpower to a late software project makes it later.",
	"There is no silver bullet.",
	"Beware of bugs in the above code; I have only proved it correct, not tried it.",
	"Premature abstraction is as harmful as premature optimization.",
	"Make the easy things easy, and the hard things possible.",
	"When in doubt, use brute force.",
	"Inside every large program is a small program struggling to get out.",
	"The art of programming is the skill of controlling complexity.",
	"Software is a great combination of artistry and engineering.",
	"Sometimes it pays to stay in bed on Monday rather than spending the rest of the week debugging Monday's code.",
	"To iterate is human, to recurse divine.",
	"Truth can only be found in one place: the code.",
}

func IQuoteHandler(m *tg.NewMessage) error {
	pick := iQuoteList[memeQuoteRng.Intn(len(iQuoteList))]
	out := fmt.Sprintf("<b>Inspirational Tech Quote</b>\n\n<i>%s</i>", html.EscapeString(pick))
	m.Reply(out)
	return nil
}

func DevJokeHandler(m *tg.NewMessage) error {
	pick := devJokeList[memeQuoteRng.Intn(len(devJokeList))]
	out := fmt.Sprintf("<b>Dev Joke</b>\n\n<i>%s</i>", html.EscapeString(pick))
	m.Reply(out)
	return nil
}

func AphorismHandler(m *tg.NewMessage) error {
	pick := aphorismList[memeQuoteRng.Intn(len(aphorismList))]
	out := fmt.Sprintf("<b>Aphorism</b>\n\n<i>%s</i>", html.EscapeString(pick))
	m.Reply(out)
	return nil
}

func registerMemeQuoteHandlers() {
	c := Client
	c.On("cmd:iquote", IQuoteHandler)
	c.On("cmd:devjoke", DevJokeHandler)
	c.On("cmd:aphorism", AphorismHandler)
}

func init() {
	QueueHandlerRegistration(registerMemeQuoteHandlers)
}
