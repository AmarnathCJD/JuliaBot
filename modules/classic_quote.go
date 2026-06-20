package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type classicQuoteAPIResp struct {
	ID      string   `json:"_id"`
	Content string   `json:"content"`
	Author  string   `json:"author"`
	Tags    []string `json:"tags"`
	Length  int      `json:"length"`
}

type classicQuoteCacheEntry struct {
	data    classicQuoteAPIResp
	expires time.Time
}

var (
	classicQuoteCacheMu sync.Mutex
	classicQuoteCache   = map[string]classicQuoteCacheEntry{}
	classicQuoteRng     = rand.New(rand.NewSource(time.Now().UnixNano()))
)

const classicQuoteCacheTTL = 1 * time.Hour

var classicQuoteFallbackList = []classicQuoteAPIResp{
	{Content: "Be yourself; everyone else is already taken.", Author: "Oscar Wilde"},
	{Content: "Two things are infinite: the universe and human stupidity; and I'm not sure about the universe.", Author: "Albert Einstein"},
	{Content: "So many books, so little time.", Author: "Frank Zappa"},
	{Content: "A room without books is like a body without a soul.", Author: "Marcus Tullius Cicero"},
	{Content: "You only live once, but if you do it right, once is enough.", Author: "Mae West"},
	{Content: "Be the change that you wish to see in the world.", Author: "Mahatma Gandhi"},
	{Content: "In three words I can sum up everything I've learned about life: it goes on.", Author: "Robert Frost"},
	{Content: "If you tell the truth, you don't have to remember anything.", Author: "Mark Twain"},
	{Content: "A friend is someone who knows all about you and still loves you.", Author: "Elbert Hubbard"},
	{Content: "Always forgive your enemies; nothing annoys them so much.", Author: "Oscar Wilde"},
	{Content: "To live is the rarest thing in the world. Most people exist, that is all.", Author: "Oscar Wilde"},
	{Content: "Without music, life would be a mistake.", Author: "Friedrich Nietzsche"},
	{Content: "We accept the love we think we deserve.", Author: "Stephen Chbosky"},
	{Content: "It is better to be hated for what you are than to be loved for what you are not.", Author: "Andre Gide"},
	{Content: "The only way to do great work is to love what you do.", Author: "Steve Jobs"},
	{Content: "In the end, we will remember not the words of our enemies, but the silence of our friends.", Author: "Martin Luther King Jr."},
	{Content: "The future belongs to those who believe in the beauty of their dreams.", Author: "Eleanor Roosevelt"},
	{Content: "Tell me and I forget. Teach me and I remember. Involve me and I learn.", Author: "Benjamin Franklin"},
	{Content: "The best and most beautiful things in the world cannot be seen or even touched - they must be felt with the heart.", Author: "Helen Keller"},
	{Content: "It is during our darkest moments that we must focus to see the light.", Author: "Aristotle"},
	{Content: "Whoever is happy will make others happy too.", Author: "Anne Frank"},
	{Content: "Do not go where the path may lead, go instead where there is no path and leave a trail.", Author: "Ralph Waldo Emerson"},
	{Content: "You will face many defeats in life, but never let yourself be defeated.", Author: "Maya Angelou"},
	{Content: "The greatest glory in living lies not in never falling, but in rising every time we fall.", Author: "Nelson Mandela"},
	{Content: "In the middle of difficulty lies opportunity.", Author: "Albert Einstein"},
	{Content: "Life is what happens when you're busy making other plans.", Author: "John Lennon"},
	{Content: "Spread love everywhere you go. Let no one ever come to you without leaving happier.", Author: "Mother Teresa"},
	{Content: "When you reach the end of your rope, tie a knot in it and hang on.", Author: "Franklin D. Roosevelt"},
	{Content: "The only impossible journey is the one you never begin.", Author: "Tony Robbins"},
	{Content: "In this life we cannot do great things. We can only do small things with great love.", Author: "Mother Teresa"},
	{Content: "Only a life lived for others is a life worthwhile.", Author: "Albert Einstein"},
	{Content: "The purpose of our lives is to be happy.", Author: "Dalai Lama"},
	{Content: "You only live once, but if you do it right, once is enough.", Author: "Mae West"},
	{Content: "Life is either a daring adventure or nothing at all.", Author: "Helen Keller"},
	{Content: "Many of life's failures are people who did not realize how close they were to success when they gave up.", Author: "Thomas A. Edison"},
	{Content: "If life were predictable it would cease to be life, and be without flavor.", Author: "Eleanor Roosevelt"},
	{Content: "The whole secret of a successful life is to find out what is one's destiny to do, and then do it.", Author: "Henry Ford"},
	{Content: "In order to write about life first you must live it.", Author: "Ernest Hemingway"},
	{Content: "The big lesson in life, baby, is never be scared of anyone or anything.", Author: "Frank Sinatra"},
	{Content: "Sing like no one's listening, love like you've never been hurt, dance like nobody's watching, and live like it's heaven on earth.", Author: "Mark Twain"},
	{Content: "Curiosity about life in all of its aspects, I think, is still the secret of great creative people.", Author: "Leo Burnett"},
	{Content: "Life is not a problem to be solved, but a reality to be experienced.", Author: "Soren Kierkegaard"},
	{Content: "The unexamined life is not worth living.", Author: "Socrates"},
	{Content: "Turn your wounds into wisdom.", Author: "Oprah Winfrey"},
	{Content: "The way I see it, if you want the rainbow, you gotta put up with the rain.", Author: "Dolly Parton"},
	{Content: "Do all the good you can, for all the people you can, in all the ways you can, as long as you can.", Author: "Hillary Clinton"},
	{Content: "Don't cry because it's over, smile because it happened.", Author: "Dr. Seuss"},
	{Content: "You must be the change you wish to see in the world.", Author: "Mahatma Gandhi"},
	{Content: "Live as if you were to die tomorrow. Learn as if you were to live forever.", Author: "Mahatma Gandhi"},
	{Content: "That which does not kill us makes us stronger.", Author: "Friedrich Nietzsche"},
	{Content: "I have not failed. I've just found 10,000 ways that won't work.", Author: "Thomas A. Edison"},
	{Content: "A person who never made a mistake never tried anything new.", Author: "Albert Einstein"},
	{Content: "The person who says it cannot be done should not interrupt the person who is doing it.", Author: "Chinese Proverb"},
	{Content: "There are no traffic jams along the extra mile.", Author: "Roger Staubach"},
	{Content: "It is never too late to be what you might have been.", Author: "George Eliot"},
	{Content: "You become what you believe.", Author: "Oprah Winfrey"},
	{Content: "I would rather die of passion than of boredom.", Author: "Vincent van Gogh"},
	{Content: "A truly rich man is one whose children run into his arms when his hands are empty.", Author: "Unknown"},
	{Content: "It is not what you do for your children, but what you have taught them to do for themselves, that will make them successful human beings.", Author: "Ann Landers"},
	{Content: "If you want your children to turn out well, spend twice as much time with them, and half as much money.", Author: "Abigail Van Buren"},
	{Content: "Build your own dreams, or someone else will hire you to build theirs.", Author: "Farrah Gray"},
	{Content: "The battles that count aren't the ones for gold medals. The struggles within yourself are the invisible battles that count.", Author: "Jesse Owens"},
	{Content: "Education costs money. But then so does ignorance.", Author: "Claus Moser"},
	{Content: "I have learned over the years that when one's mind is made up, this diminishes fear.", Author: "Rosa Parks"},
	{Content: "I alone cannot change the world, but I can cast a stone across the waters to create many ripples.", Author: "Mother Teresa"},
	{Content: "What we achieve inwardly will change outer reality.", Author: "Plutarch"},
	{Content: "Whatever you are, be a good one.", Author: "Abraham Lincoln"},
	{Content: "I attribute my success to this: I never gave or took any excuse.", Author: "Florence Nightingale"},
	{Content: "You miss 100% of the shots you don't take.", Author: "Wayne Gretzky"},
	{Content: "I've missed more than 9000 shots in my career. I've lost almost 300 games. I've failed over and over and over again in my life. And that is why I succeed.", Author: "Michael Jordan"},
	{Content: "The most difficult thing is the decision to act, the rest is merely tenacity.", Author: "Amelia Earhart"},
	{Content: "Every strike brings me closer to the next home run.", Author: "Babe Ruth"},
	{Content: "Definiteness of purpose is the starting point of all achievement.", Author: "W. Clement Stone"},
	{Content: "Life isn't about getting and having, it's about giving and being.", Author: "Kevin Kruse"},
	{Content: "Life is what we make it, always has been, always will be.", Author: "Grandma Moses"},
	{Content: "The question isn't who is going to let me; it's who is going to stop me.", Author: "Ayn Rand"},
	{Content: "When everything seems to be going against you, remember that the airplane takes off against the wind, not with it.", Author: "Henry Ford"},
	{Content: "It's not the years in your life that count. It's the life in your years.", Author: "Abraham Lincoln"},
	{Content: "Change your thoughts and you change your world.", Author: "Norman Vincent Peale"},
	{Content: "Either write something worth reading or do something worth writing.", Author: "Benjamin Franklin"},
	{Content: "Nothing is impossible, the word itself says, I'm possible!", Author: "Audrey Hepburn"},
	{Content: "The only way to do great work is to love what you do. If you haven't found it yet, keep looking. Don't settle.", Author: "Steve Jobs"},
	{Content: "If you can dream it, you can achieve it.", Author: "Zig Ziglar"},
	{Content: "Twenty years from now you will be more disappointed by the things that you didn't do than by the ones you did do.", Author: "Mark Twain"},
	{Content: "Don't judge each day by the harvest you reap but by the seeds that you plant.", Author: "Robert Louis Stevenson"},
	{Content: "The future depends on what you do today.", Author: "Mahatma Gandhi"},
	{Content: "Don't watch the clock; do what it does. Keep going.", Author: "Sam Levenson"},
	{Content: "Believe you can and you're halfway there.", Author: "Theodore Roosevelt"},
	{Content: "It does not matter how slowly you go as long as you do not stop.", Author: "Confucius"},
	{Content: "Our greatest weakness lies in giving up. The most certain way to succeed is always to try just one more time.", Author: "Thomas A. Edison"},
	{Content: "Start where you are. Use what you have. Do what you can.", Author: "Arthur Ashe"},
	{Content: "Fall seven times and stand up eight.", Author: "Japanese Proverb"},
	{Content: "When you have a dream, you've got to grab it and never let go.", Author: "Carol Burnett"},
	{Content: "Nothing is impossible. The word itself says I'm possible!", Author: "Audrey Hepburn"},
	{Content: "There is nothing impossible to they who will try.", Author: "Alexander the Great"},
	{Content: "The bad news is time flies. The good news is you're the pilot.", Author: "Michael Altshuler"},
	{Content: "Life has got all those twists and turns. You've got to hold on tight and off you go.", Author: "Nicole Kidman"},
	{Content: "Keep your face always toward the sunshine, and shadows will fall behind you.", Author: "Walt Whitman"},
	{Content: "If opportunity doesn't knock, build a door.", Author: "Milton Berle"},
	{Content: "Wherever you go, no matter what the weather, always bring your own sunshine.", Author: "Anthony J. D'Angelo"},
	{Content: "If you want to lift yourself up, lift up someone else.", Author: "Booker T. Washington"},
	{Content: "The best revenge is massive success.", Author: "Frank Sinatra"},
	{Content: "Try to be a rainbow in someone's cloud.", Author: "Maya Angelou"},
	{Content: "What you get by achieving your goals is not as important as what you become by achieving your goals.", Author: "Zig Ziglar"},
}

func classicQuoteFetch() (classicQuoteAPIResp, bool) {
	classicQuoteCacheMu.Lock()
	if e, ok := classicQuoteCache["last"]; ok && time.Now().Before(e.expires) {
		classicQuoteCacheMu.Unlock()
		return e.data, true
	}
	classicQuoteCacheMu.Unlock()

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", "https://api.quotable.io/random", nil)
	if err != nil {
		return classicQuoteAPIResp{}, false
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return classicQuoteAPIResp{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return classicQuoteAPIResp{}, false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return classicQuoteAPIResp{}, false
	}
	var q classicQuoteAPIResp
	if jerr := json.Unmarshal(body, &q); jerr != nil {
		return classicQuoteAPIResp{}, false
	}
	if strings.TrimSpace(q.Content) == "" {
		return classicQuoteAPIResp{}, false
	}
	classicQuoteCacheMu.Lock()
	classicQuoteCache["last"] = classicQuoteCacheEntry{data: q, expires: time.Now().Add(classicQuoteCacheTTL)}
	classicQuoteCacheMu.Unlock()
	return q, true
}

func classicQuoteFallback() classicQuoteAPIResp {
	return classicQuoteFallbackList[classicQuoteRng.Intn(len(classicQuoteFallbackList))]
}

func formatClassicQuote(q classicQuoteAPIResp, source string) string {
	out := "<b>Classic Quote</b>\n\n"
	out += "<i>" + html.EscapeString(q.Content) + "</i>\n\n"
	if strings.TrimSpace(q.Author) != "" {
		out += "<b>Author:</b> " + html.EscapeString(q.Author) + "\n"
	}
	if len(q.Tags) > 0 {
		out += "<b>Tags:</b> " + html.EscapeString(strings.Join(q.Tags, ", ")) + "\n"
	}
	out += fmt.Sprintf("\n<i>Source: %s</i>", source)
	return out
}

func RandomQuote2Handler(m *tg.NewMessage) error {
	q, ok := classicQuoteFetch()
	if ok {
		m.Reply(formatClassicQuote(q, "quotable.io"))
		return nil
	}
	m.Reply(formatClassicQuote(classicQuoteFallback(), "offline list"))
	return nil
}

func registerClassicQuoteHandlers() {
	c := Client
	c.On("cmd:randomquote2", RandomQuote2Handler)
}

func init() {
	QueueHandlerRegistration(registerClassicQuoteHandlers)
}
