package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var movieQuoteClient = &http.Client{Timeout: 30 * time.Second}

type stoicQuoteResp struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

var buddhaQuotes = []string{
	"Three things cannot be long hidden: the sun, the moon, and the truth.",
	"Peace comes from within. Do not seek it without.",
	"You yourself, as much as anybody in the entire universe, deserve your love and affection.",
	"Holding onto anger is like drinking poison and expecting the other person to die.",
	"The mind is everything. What you think you become.",
	"Better than a thousand hollow words is one word that brings peace.",
	"Do not dwell in the past, do not dream of the future, concentrate the mind on the present moment.",
	"Thousands of candles can be lit from a single candle, and the life of the candle will not be shortened.",
	"In the end only three things matter: how much you loved, how gently you lived, and how gracefully you let go.",
	"To understand everything is to forgive everything.",
	"There is no path to happiness; happiness is the path.",
	"What you think, you become. What you feel, you attract. What you imagine, you create.",
	"If you light a lamp for somebody, it will also brighten your path.",
	"The trouble is, you think you have time.",
	"Health is the greatest gift, contentment the greatest wealth, faithfulness the best relationship.",
	"No one saves us but ourselves. No one can and no one may. We ourselves must walk the path.",
	"Every morning we are born again. What we do today is what matters most.",
	"Pain is certain, suffering is optional.",
	"It is better to travel well than to arrive.",
	"The way is not in the sky. The way is in the heart.",
	"Doubt everything. Find your own light.",
	"The root of suffering is attachment.",
	"Be where you are, otherwise you will miss your life.",
	"Have compassion for all beings, rich and poor alike; each has their suffering.",
	"You will not be punished for your anger, you will be punished by your anger.",
	"A jug fills drop by drop.",
	"The tongue like a sharp knife kills without drawing blood.",
	"Drop by drop is the water pot filled. Likewise, the wise man, gathering it little by little, fills himself with good.",
	"There is nothing more dreadful than the habit of doubt.",
	"Conquer anger with non-anger. Conquer badness with goodness. Conquer meanness with generosity. Conquer dishonesty with truth.",
	"Just as a candle cannot burn without fire, men cannot live without a spiritual life.",
	"All wrong-doing arises because of mind. If mind is transformed can wrong-doing remain?",
	"An idea that is developed and put into action is more important than an idea that exists only as an idea.",
	"Even death is not to be feared by one who has lived wisely.",
	"Work out your own salvation. Do not depend on others.",
	"The whole secret of existence is to have no fear.",
	"Set your heart on doing good. Do it over and over again, and you will be filled with joy.",
	"Hatred does not cease through hatred at any time. Hatred ceases through love.",
	"A man is not called wise because he talks and talks again; but if he is peaceful, loving, and fearless then he is in truth called wise.",
	"You only lose what you cling to.",
	"To keep the body in good health is a duty, otherwise we shall not be able to keep our mind strong and clear.",
	"Speak only endearing speech, speech that is welcomed.",
	"Just as treasures are uncovered from the earth, so virtue appears from good deeds.",
	"What we think, we become.",
	"There has to be evil so that good can prove its purity above it.",
	"When you realize how perfect everything is you will tilt your head back and laugh at the sky.",
	"It is a man's own mind, not his enemy or foe, that lures him to evil ways.",
	"To enjoy good health, to bring true happiness to one's family, to bring peace to all, one must first discipline and control one's own mind.",
	"Believe nothing, no matter where you read it, or who said it, unless it agrees with your own reason and your own common sense.",
	"All that we are is the result of what we have thought.",
}

var shakespeareQuotes = []struct {
	Text string
	Play string
}{
	{"To be, or not to be: that is the question.", "Hamlet"},
	{"All the world's a stage, and all the men and women merely players.", "As You Like It"},
	{"The course of true love never did run smooth.", "A Midsummer Night's Dream"},
	{"What's in a name? That which we call a rose by any other name would smell as sweet.", "Romeo and Juliet"},
	{"Cowards die many times before their deaths; the valiant never taste of death but once.", "Julius Caesar"},
	{"Some are born great, some achieve greatness, and some have greatness thrust upon them.", "Twelfth Night"},
	{"The lady doth protest too much, methinks.", "Hamlet"},
	{"Hell is empty and all the devils are here.", "The Tempest"},
	{"There is nothing either good or bad, but thinking makes it so.", "Hamlet"},
	{"We know what we are, but know not what we may be.", "Hamlet"},
	{"Love all, trust a few, do wrong to none.", "All's Well That Ends Well"},
	{"Brevity is the soul of wit.", "Hamlet"},
	{"Lord, what fools these mortals be!", "A Midsummer Night's Dream"},
	{"The fool doth think he is wise, but the wise man knows himself to be a fool.", "As You Like It"},
	{"Better three hours too soon than a minute too late.", "The Merry Wives of Windsor"},
	{"If music be the food of love, play on.", "Twelfth Night"},
	{"Now is the winter of our discontent.", "Richard III"},
	{"Et tu, Brute? Then fall, Caesar!", "Julius Caesar"},
	{"Friends, Romans, countrymen, lend me your ears.", "Julius Caesar"},
	{"Out, damned spot! Out, I say!", "Macbeth"},
	{"Something is rotten in the state of Denmark.", "Hamlet"},
	{"This above all: to thine own self be true.", "Hamlet"},
	{"A horse! A horse! My kingdom for a horse!", "Richard III"},
	{"The better part of valour is discretion.", "Henry IV, Part 1"},
	{"All that glitters is not gold.", "The Merchant of Venice"},
	{"How sharper than a serpent's tooth it is to have a thankless child!", "King Lear"},
	{"Parting is such sweet sorrow.", "Romeo and Juliet"},
	{"Uneasy lies the head that wears a crown.", "Henry IV, Part 2"},
	{"Good night, good night! Parting is such sweet sorrow, that I shall say good night till it be morrow.", "Romeo and Juliet"},
	{"Men at some time are masters of their fates: The fault, dear Brutus, is not in our stars, but in ourselves.", "Julius Caesar"},
}

func fetchStoicQuote(out interface{}) error {
	req, err := http.NewRequest("GET", "https://stoic-quotes.com/api/quote", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "JuliaBot (https://github.com/amarnathcjd)")
	resp, err := movieQuoteClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func StoicHandler(m *tg.NewMessage) error {
	var q stoicQuoteResp
	if err := fetchStoicQuote(&q); err != nil {
		m.Reply("<b>Error:</b> failed to fetch stoic quote.")
		return nil
	}
	if q.Text == "" {
		m.Reply("<b>Error:</b> empty quote received.")
		return nil
	}
	author := q.Author
	if author == "" {
		author = "Unknown"
	}
	m.Reply(fmt.Sprintf("<b>Stoic Quote:</b>\n<i>%s</i>\n\n<b>— %s</b>", html.EscapeString(q.Text), html.EscapeString(author)))
	return nil
}

func BuddhaHandler(m *tg.NewMessage) error {
	q := buddhaQuotes[rand.Intn(len(buddhaQuotes))]
	m.Reply(fmt.Sprintf("<b>Buddha:</b>\n<i>%s</i>\n\n<b>— Gautama Buddha</b>", html.EscapeString(q)))
	return nil
}

func ShakespeareHandler(m *tg.NewMessage) error {
	q := shakespeareQuotes[rand.Intn(len(shakespeareQuotes))]
	m.Reply(fmt.Sprintf("<b>Shakespeare:</b>\n<i>%s</i>\n\n<b>— William Shakespeare</b>\n<code>%s</code>", html.EscapeString(q.Text), html.EscapeString(q.Play)))
	return nil
}

func registerMovieQuoteHandlers() {
	c := Client
	c.On("cmd:stoic", StoicHandler)
	c.On("cmd:buddha", BuddhaHandler)
	c.On("cmd:shakespeare", ShakespeareHandler)
}

func init() {
	QueueHandlerRegistration(registerMovieQuoteHandlers)
}
