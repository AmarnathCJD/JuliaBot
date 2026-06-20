package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

var quoteGenRng = rand.New(rand.NewSource(time.Now().UnixNano()))

type quoteGenItem struct {
	Quote  string
	Author string
}

var quoteGenFallback = []quoteGenItem{
	{"Knowing yourself is the beginning of all wisdom.", "Aristotle"},
	{"It is during our darkest moments that we must focus to see the light.", "Aristotle"},
	{"The whole is greater than the sum of its parts.", "Aristotle"},
	{"Quality is not an act, it is a habit.", "Aristotle"},
	{"Happiness depends upon ourselves.", "Aristotle"},
	{"Educating the mind without educating the heart is no education at all.", "Aristotle"},
	{"Patience is bitter, but its fruit is sweet.", "Aristotle"},
	{"Our life is what our thoughts make it.", "Marcus Aurelius"},
	{"The happiness of your life depends upon the quality of your thoughts.", "Marcus Aurelius"},
	{"Waste no more time arguing what a good man should be. Be one.", "Marcus Aurelius"},
	{"You have power over your mind, not outside events. Realize this, and you will find strength.", "Marcus Aurelius"},
	{"It is not death that a man should fear, but he should fear never beginning to live.", "Marcus Aurelius"},
	{"Confine yourself to the present.", "Marcus Aurelius"},
	{"It does not matter how slowly you go as long as you do not stop.", "Confucius"},
	{"Real knowledge is to know the extent of one's ignorance.", "Confucius"},
	{"Our greatest glory is not in never falling, but in rising every time we fall.", "Confucius"},
	{"The man who moves a mountain begins by carrying away small stones.", "Confucius"},
	{"Wherever you go, go with all your heart.", "Confucius"},
	{"By three methods we may learn wisdom: first, by reflection, which is noblest; second, by imitation, which is easiest; and third by experience, which is the bitterest.", "Confucius"},
	{"Choose a job you love, and you will never have to work a day in your life.", "Confucius"},
	{"The two most important days in your life are the day you are born and the day you find out why.", "Mark Twain"},
	{"Whenever you find yourself on the side of the majority, it is time to pause and reflect.", "Mark Twain"},
	{"Kindness is the language which the deaf can hear and the blind can see.", "Mark Twain"},
	{"Get your facts first, then you can distort them as you please.", "Mark Twain"},
	{"The secret of getting ahead is getting started.", "Mark Twain"},
	{"Twenty years from now you will be more disappointed by the things you didn't do than by the ones you did do.", "Mark Twain"},
	{"Courage is resistance to fear, mastery of fear, not absence of fear.", "Mark Twain"},
	{"Be yourself; everyone else is already taken.", "Oscar Wilde"},
	{"To live is the rarest thing in the world. Most people exist, that is all.", "Oscar Wilde"},
	{"We are all in the gutter, but some of us are looking at the stars.", "Oscar Wilde"},
	{"Always forgive your enemies; nothing annoys them so much.", "Oscar Wilde"},
	{"I can resist everything except temptation.", "Oscar Wilde"},
	{"Experience is simply the name we give our mistakes.", "Oscar Wilde"},
	{"The only way to get rid of a temptation is to yield to it.", "Oscar Wilde"},
	{"In the middle of difficulty lies opportunity.", "Albert Einstein"},
	{"Imagination is more important than knowledge.", "Albert Einstein"},
	{"Life is like riding a bicycle. To keep your balance, you must keep moving.", "Albert Einstein"},
	{"A person who never made a mistake never tried anything new.", "Albert Einstein"},
	{"Try not to become a man of success, but rather try to become a man of value.", "Albert Einstein"},
	{"The important thing is not to stop questioning. Curiosity has its own reason for existing.", "Albert Einstein"},
	{"Logic will get you from A to B. Imagination will take you everywhere.", "Albert Einstein"},
	{"That which does not kill us makes us stronger.", "Friedrich Nietzsche"},
	{"He who has a why to live for can bear almost any how.", "Friedrich Nietzsche"},
	{"Without music, life would be a mistake.", "Friedrich Nietzsche"},
	{"You must have chaos within you to give birth to a dancing star.", "Friedrich Nietzsche"},
	{"In heaven all the interesting people are missing.", "Friedrich Nietzsche"},
	{"The unexamined life is not worth living.", "Socrates"},
	{"I know that I know nothing.", "Socrates"},
	{"The only true wisdom is in knowing you know nothing.", "Socrates"},
	{"Wonder is the beginning of wisdom.", "Socrates"},
	{"To find yourself, think for yourself.", "Socrates"},
	{"To be is to do.", "Socrates"},
	{"There is only one good, knowledge, and one evil, ignorance.", "Socrates"},
	{"Be the change that you wish to see in the world.", "Mahatma Gandhi"},
	{"The weak can never forgive. Forgiveness is the attribute of the strong.", "Mahatma Gandhi"},
	{"Live as if you were to die tomorrow. Learn as if you were to live forever.", "Mahatma Gandhi"},
	{"An eye for an eye only ends up making the whole world blind.", "Mahatma Gandhi"},
	{"Strength does not come from physical capacity. It comes from an indomitable will.", "Mahatma Gandhi"},
	{"The future depends on what you do today.", "Mahatma Gandhi"},
	{"Happiness is when what you think, what you say, and what you do are in harmony.", "Mahatma Gandhi"},
	{"All that we are is the result of what we have thought.", "Buddha"},
	{"Three things cannot be long hidden: the sun, the moon, and the truth.", "Buddha"},
	{"Peace comes from within. Do not seek it without.", "Buddha"},
	{"The mind is everything. What you think you become.", "Buddha"},
	{"There is no path to happiness: happiness is the path.", "Buddha"},
	{"Do not dwell in the past, do not dream of the future, concentrate the mind on the present moment.", "Buddha"},
	{"In the end, only three things matter: how much you loved, how gently you lived, and how gracefully you let go of things not meant for you.", "Buddha"},
	{"To be yourself in a world that is constantly trying to make you something else is the greatest accomplishment.", "Ralph Waldo Emerson"},
	{"What lies behind us and what lies before us are tiny matters compared to what lies within us.", "Ralph Waldo Emerson"},
	{"Do not go where the path may lead, go instead where there is no path and leave a trail.", "Ralph Waldo Emerson"},
	{"Write it on your heart that every day is the best day in the year.", "Ralph Waldo Emerson"},
	{"The only person you are destined to become is the person you decide to be.", "Ralph Waldo Emerson"},
	{"Two roads diverged in a wood, and I took the one less traveled by, and that has made all the difference.", "Robert Frost"},
	{"In three words I can sum up everything I've learned about life: it goes on.", "Robert Frost"},
	{"The best way out is always through.", "Robert Frost"},
	{"Don't ever take a fence down until you know why it was put up.", "Robert Frost"},
	{"I have not failed. I've just found 10,000 ways that won't work.", "Thomas Edison"},
	{"Genius is one percent inspiration and ninety-nine percent perspiration.", "Thomas Edison"},
	{"Many of life's failures are people who did not realize how close they were to success when they gave up.", "Thomas Edison"},
	{"Our greatest weakness lies in giving up. The most certain way to succeed is always to try just one more time.", "Thomas Edison"},
	{"All our dreams can come true, if we have the courage to pursue them.", "Walt Disney"},
	{"The way to get started is to quit talking and begin doing.", "Walt Disney"},
	{"If you can dream it, you can do it.", "Walt Disney"},
	{"It is better to remain silent at the risk of being thought a fool, than to talk and remove all doubt of it.", "Abraham Lincoln"},
	{"Whatever you are, be a good one.", "Abraham Lincoln"},
	{"In the end, it's not the years in your life that count. It's the life in your years.", "Abraham Lincoln"},
	{"The best way to predict your future is to create it.", "Abraham Lincoln"},
}

type quoteGenAPIItem struct {
	Q string `json:"q"`
	A string `json:"a"`
}

func quoteGenFetchAPI() (quoteGenItem, bool) {
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", "https://zenquotes.io/api/random", nil)
	if err != nil {
		return quoteGenItem{}, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return quoteGenItem{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return quoteGenItem{}, false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return quoteGenItem{}, false
	}
	var items []quoteGenAPIItem
	if err := json.Unmarshal(body, &items); err != nil {
		return quoteGenItem{}, false
	}
	if len(items) == 0 {
		return quoteGenItem{}, false
	}
	q := strings.TrimSpace(items[0].Q)
	a := strings.TrimSpace(items[0].A)
	if q == "" {
		return quoteGenItem{}, false
	}
	if a == "" {
		a = "Unknown"
	}
	return quoteGenItem{Quote: q, Author: a}, true
}

func quoteGenPick() quoteGenItem {
	if item, ok := quoteGenFetchAPI(); ok {
		return item
	}
	return quoteGenFallback[quoteGenRng.Intn(len(quoteGenFallback))]
}

func quoteGenLoadFont(dc *gg.Context, size float64) {
	name := getRandomFont()
	candidates := []string{
		"./assets/" + name,
		"assets/" + name,
		"../assets/" + name,
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "assets", name))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "assets", name),
			filepath.Join(dir, "..", "assets", name),
		)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if err := dc.LoadFontFace(p, size); err == nil {
				return
			}
		}
	}
}

func quoteGenLoadSpecificFont(dc *gg.Context, name string, size float64) bool {
	candidates := []string{
		"./assets/" + name,
		"assets/" + name,
		"../assets/" + name,
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "assets", name))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "assets", name),
			filepath.Join(dir, "..", "assets", name),
		)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if err := dc.LoadFontFace(p, size); err == nil {
				return true
			}
		}
	}
	return false
}

func quoteGenWrap(text string, maxChars int) []string {
	words := strings.Fields(text)
	var lines []string
	var cur string
	for _, w := range words {
		if cur == "" {
			cur = w
			continue
		}
		if len(cur)+1+len(w) > maxChars {
			lines = append(lines, cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

type quoteGenPalette struct {
	BgTop    [3]float64
	BgBottom [3]float64
	Card     [3]float64
	Accent   [3]float64
	Text     [3]float64
	SubText  [3]float64
}

var quoteGenPalettes = []quoteGenPalette{
	{[3]float64{0.07, 0.09, 0.16}, [3]float64{0.18, 0.10, 0.28}, [3]float64{0.96, 0.94, 0.88}, [3]float64{0.85, 0.65, 0.30}, [3]float64{0.12, 0.10, 0.14}, [3]float64{0.45, 0.38, 0.28}},
	{[3]float64{0.05, 0.12, 0.18}, [3]float64{0.10, 0.25, 0.30}, [3]float64{0.98, 0.97, 0.93}, [3]float64{0.20, 0.65, 0.60}, [3]float64{0.08, 0.18, 0.20}, [3]float64{0.30, 0.45, 0.50}},
	{[3]float64{0.16, 0.06, 0.10}, [3]float64{0.30, 0.10, 0.18}, [3]float64{0.99, 0.95, 0.90}, [3]float64{0.85, 0.30, 0.40}, [3]float64{0.18, 0.08, 0.10}, [3]float64{0.55, 0.30, 0.32}},
	{[3]float64{0.04, 0.08, 0.06}, [3]float64{0.10, 0.20, 0.14}, [3]float64{0.97, 0.96, 0.90}, [3]float64{0.55, 0.75, 0.40}, [3]float64{0.10, 0.15, 0.10}, [3]float64{0.30, 0.40, 0.30}},
	{[3]float64{0.08, 0.08, 0.12}, [3]float64{0.22, 0.18, 0.30}, [3]float64{0.97, 0.95, 0.92}, [3]float64{0.55, 0.45, 0.85}, [3]float64{0.14, 0.10, 0.20}, [3]float64{0.40, 0.32, 0.55}},
}

func quoteGenRender(item quoteGenItem) (string, error) {
	const W, H = 900, 600
	dc := gg.NewContext(W, H)

	pal := quoteGenPalettes[quoteGenRng.Intn(len(quoteGenPalettes))]

	for y := 0; y < H; y++ {
		t := float64(y) / float64(H)
		r := pal.BgTop[0] + (pal.BgBottom[0]-pal.BgTop[0])*t
		g := pal.BgTop[1] + (pal.BgBottom[1]-pal.BgTop[1])*t
		b := pal.BgTop[2] + (pal.BgBottom[2]-pal.BgTop[2])*t
		dc.SetRGB(r, g, b)
		dc.DrawRectangle(0, float64(y), float64(W), 1)
		dc.Fill()
	}

	for i := 0; i < 120; i++ {
		sx := quoteGenRng.Float64() * float64(W)
		sy := quoteGenRng.Float64() * float64(H)
		dc.SetRGBA(1, 1, 1, 0.04+quoteGenRng.Float64()*0.12)
		dc.DrawCircle(sx, sy, 0.5+quoteGenRng.Float64()*1.4)
		dc.Fill()
	}

	for i := 0; i < 6; i++ {
		gx := quoteGenRng.Float64() * float64(W)
		gy := quoteGenRng.Float64() * float64(H)
		gr := 80.0 + quoteGenRng.Float64()*120.0
		dc.SetRGBA(pal.Accent[0], pal.Accent[1], pal.Accent[2], 0.06)
		dc.DrawCircle(gx, gy, gr)
		dc.Fill()
	}

	cardW := 740.0
	cardH := 430.0
	cardX := (float64(W) - cardW) / 2
	cardY := (float64(H) - cardH) / 2

	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawRoundedRectangle(cardX+10, cardY+14, cardW, cardH, 18)
	dc.Fill()

	dc.SetRGB(pal.Card[0], pal.Card[1], pal.Card[2])
	dc.DrawRoundedRectangle(cardX, cardY, cardW, cardH, 18)
	dc.FillPreserve()
	dc.SetRGBA(pal.Accent[0], pal.Accent[1], pal.Accent[2], 0.9)
	dc.SetLineWidth(2)
	dc.Stroke()

	for i := 0; i < 60; i++ {
		fx := cardX + quoteGenRng.Float64()*cardW
		fy := cardY + quoteGenRng.Float64()*cardH
		dc.SetRGBA(pal.Text[0], pal.Text[1], pal.Text[2], 0.04)
		dc.DrawCircle(fx, fy, 0.4+quoteGenRng.Float64()*1.0)
		dc.Fill()
	}

	dc.SetRGBA(pal.Accent[0], pal.Accent[1], pal.Accent[2], 0.8)
	dc.DrawRectangle(cardX+40, cardY+90, 50, 4)
	dc.Fill()
	dc.DrawRectangle(cardX+cardW-90, cardY+cardH-100, 50, 4)
	dc.Fill()

	quoteGenLoadSpecificFont(dc, "Inter_28pt-Bold.ttf", 140)
	dc.SetRGBA(pal.Accent[0], pal.Accent[1], pal.Accent[2], 0.35)
	dc.DrawStringAnchored("“", cardX+70, cardY+120, 0.5, 0.5)
	dc.SetRGBA(pal.Accent[0], pal.Accent[1], pal.Accent[2], 0.35)
	dc.DrawStringAnchored("”", cardX+cardW-70, cardY+cardH-80, 0.5, 0.5)

	quoteText := item.Quote
	if len(quoteText) > 320 {
		quoteText = quoteText[:317] + "..."
	}

	fontSize := 30.0
	maxChars := 38
	if len(quoteText) > 180 {
		fontSize = 24.0
		maxChars = 48
	} else if len(quoteText) > 120 {
		fontSize = 27.0
		maxChars = 42
	}

	quoteGenLoadFont(dc, fontSize)
	dc.SetRGB(pal.Text[0], pal.Text[1], pal.Text[2])
	lines := quoteGenWrap(quoteText, maxChars)
	maxLines := 8
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		last := lines[maxLines-1]
		if len(last) > maxChars-3 {
			last = last[:maxChars-3]
		}
		lines[maxLines-1] = last + "..."
	}
	lineH := fontSize * 1.4
	totalH := float64(len(lines)) * lineH
	startY := cardY + (cardH-totalH)/2 - 20 + lineH/2
	for i, ln := range lines {
		dc.DrawStringAnchored(ln, cardX+cardW/2, startY+float64(i)*lineH, 0.5, 0.5)
	}

	dividerY := cardY + cardH - 70
	dc.SetRGBA(pal.Accent[0], pal.Accent[1], pal.Accent[2], 0.6)
	dc.SetLineWidth(1.5)
	dc.DrawLine(cardX+cardW/2-40, dividerY, cardX+cardW/2+40, dividerY)
	dc.Stroke()

	dc.SetRGBA(pal.Accent[0], pal.Accent[1], pal.Accent[2], 0.9)
	dc.DrawCircle(cardX+cardW/2, dividerY, 3)
	dc.Fill()

	quoteGenLoadFont(dc, 20)
	dc.SetRGB(pal.SubText[0], pal.SubText[1], pal.SubText[2])
	authorLine := "— " + item.Author
	dc.DrawStringAnchored(authorLine, cardX+cardW/2, dividerY+30, 0.5, 0.5)

	quoteGenLoadFont(dc, 14)
	dc.SetRGBA(1, 1, 1, 0.35)
	dc.DrawStringAnchored("quotegen", float64(W)-20, float64(H)-20, 1.0, 1.0)

	out := filepath.Join(os.TempDir(), fmt.Sprintf("quotegen_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func QuoteGenHandler(m *tg.NewMessage) error {
	status, _ := m.Reply("<code>composing a quote card...</code>")

	item := quoteGenPick()
	path, err := quoteGenRender(item)
	if err != nil {
		msg := "<b>Failed to render quote card.</b>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if status != nil {
		status.Delete()
	}

	caption := fmt.Sprintf("<b>Quote of the moment</b>\n\n<i>%s</i>\n\n<b>— %s</b>",
		html.EscapeString(item.Quote), html.EscapeString(item.Author))
	_, merr := m.ReplyMedia(path, &tg.MediaOptions{
		Caption:  caption,
		FileName: "quotegen.png",
		MimeType: "image/png",
	})
	os.Remove(path)
	if merr != nil {
		m.Reply("<b>Upload failed:</b> " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerQuoteGenHandlers() {
	c := Client
	c.On("cmd:quotegen", QuoteGenHandler)

	Mods.AddModule("QuoteGen", `<b>QuoteGen Module</b>

<b>Commands:</b>
 • /quotegen - Generate a stylized quote card image with a famous quote and author

<i>Pulls a quote from a public source (with a built-in library of classic authors as fallback) and renders it on a designed card.</i>`)
}

func init() {
	QueueHandlerRegistration(registerQuoteGenHandlers)
}
