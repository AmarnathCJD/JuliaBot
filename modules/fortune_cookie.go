package modules

import (
	"fmt"
	"html"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

var fortuneCookieRng = rand.New(rand.NewSource(time.Now().UnixNano()))

var fortuneCookieList = []string{
	"A beautiful, smart, and loving person will be coming into your life.",
	"A dubious friend may be an enemy in camouflage.",
	"A faithful friend is a strong defense.",
	"A feather in the hand is better than a bird in the air.",
	"A fresh start will put you on your way.",
	"A friend asks only for your time, not your money.",
	"A golden egg of opportunity falls into your lap this month.",
	"A good time to finish up old tasks.",
	"A hunch is creativity trying to tell you something.",
	"A lifetime of happiness awaits you.",
	"A light heart carries you through all the hard times.",
	"A new perspective will come with the new year.",
	"A pleasant surprise is waiting for you.",
	"A short stranger will soon enter your life with blessings to share.",
	"A small house will hold as much happiness as a big one.",
	"A smile is your personal welcome mat.",
	"A soft voice may be awfully persuasive.",
	"A truly rich life contains love and art in abundance.",
	"Adventure can be real happiness.",
	"All the effort you are making will ultimately pay off.",
	"All your hard work will soon pay off.",
	"An exciting opportunity lies ahead of you.",
	"An inch of time cannot be bought by an inch of gold.",
	"Any decision you have to make tomorrow is a good decision.",
	"Be careful or you could fall for some tricks today.",
	"Be true to your work, your word, and your friend.",
	"Beauty in its various forms appeals to you.",
	"Because you demand more from yourself, others respect you deeply.",
	"Believe it can be done.",
	"Better ask twice than lose yourself once.",
	"Bide your time, for success is near.",
	"Bloom where you are planted.",
	"Carve your name on your heart and not on marble.",
	"Change is happening in your life, so go with the flow.",
	"Curiosity kills boredom. Nothing can kill curiosity.",
	"Distance yourself from the vain.",
	"Do not be afraid of competition.",
	"Do not let ambitions overshadow small success.",
	"Don't be discouraged, every wrong attempt is a step forward.",
	"Don't pursue happiness, create it.",
	"Each day, compel yourself to do something you would rather not do.",
	"Embrace this love relationship you have.",
	"Even a broken clock is right two times a day.",
	"Every flower blooms in its own sweet time.",
	"Every wise man started out by asking many questions.",
	"Failure is the chance to do better next time.",
	"Fear and desire are two sides of the same coin.",
	"Fortune favors the brave.",
	"Friends long absent are coming back to you.",
	"From listening comes wisdom and from speaking, repentance.",
	"Generosity and perfection are your everlasting goals.",
	"Get your mind set, confidence will lead you on.",
	"Goodness is the only investment that never fails.",
	"Happiness begins with facing life with a smile and a wink.",
	"Hard words break no bones, fine words butter no parsnips.",
	"He who expects no gratitude shall never be disappointed.",
	"Hidden in a valley beside an open stream you will find your dream.",
	"If you continually give, you will continually have.",
	"If you have something good in your life, don't let it go.",
	"If you look in the right places, you can find some good offerings.",
	"If you think you can, you're right.",
	"In order to take, one must first give.",
	"It is honorable to stand up for what is right.",
	"It takes courage to admit fault.",
	"Land is always on the mind of a flying bird.",
	"Let the deeds speak.",
	"Listen to everyone. Ideas come from everywhere.",
	"Living with a positive mind set will bring you good fortune.",
	"Love is a warm fire to keep the soul warm.",
	"Make yourself necessary to someone.",
	"Many a false step is made by standing still.",
	"Materialism is a distraction from true bliss.",
	"Meeting adversity well is the source of your strength.",
	"Never give up. You're not a failure if you don't give up.",
	"New ideas could be profitable.",
	"No one can drive us crazy unless we give them the keys.",
	"Now is the time to try something new.",
	"One who admires you greatly is hidden before your eyes.",
	"Patience is your ally at the moment.",
	"People are naturally attracted to you.",
	"Plan for many pleasures ahead.",
	"Please continue with the good work, you will be rewarded.",
	"Practice makes perfect.",
	"Sell your ideas, they have exceptional merit.",
	"Show your affection to people you care for.",
	"Smile, big things are coming your way.",
	"Someone is speaking well of you.",
	"Stay healthy. Walk a mile.",
	"The early bird gets the worm.",
	"The greatest risk is not taking one.",
	"The harder you work, the luckier you get.",
	"The one you love is closer than you think.",
	"The secret to good friends is no secret to you.",
	"The world may be your oyster.",
	"Today is the conquest of the unknown.",
	"You will travel to many exotic places in your lifetime.",
	"Your shoes will make you happy today.",
	"Your dreams are never silly, depend on them to guide you.",
	"You will inherit a large sum of money.",
	"You create your own stage and the audience is waiting.",
	"You are the master of every situation.",
	"A journey of a thousand miles begins with a single step.",
}

func fortuneCookieLoadFont(dc *gg.Context, size float64) {
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

func fortuneCookieWrap(text string, maxChars int) []string {
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

func fortuneCookieDrawHalf(dc *gg.Context, cx, cy, radius float64, leftHalf bool, tiltDeg float64) {
	dc.Push()
	dc.RotateAbout(gg.Radians(tiltDeg), cx, cy)

	startAngle := -math.Pi / 2
	endAngle := math.Pi / 2
	if leftHalf {
		startAngle = math.Pi / 2
		endAngle = 3 * math.Pi / 2
	}

	dc.NewSubPath()
	steps := 60
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		a := startAngle + (endAngle-startAngle)*t
		px := cx + radius*math.Cos(a)
		py := cy + radius*math.Sin(a)
		if i == 0 {
			dc.MoveTo(px, py)
		} else {
			dc.LineTo(px, py)
		}
	}

	jagSteps := 8
	for i := 0; i <= jagSteps; i++ {
		t := float64(i) / float64(jagSteps)
		y := cy + radius - 2*radius*t
		off := 0.0
		if i%2 == 0 {
			off = -8
		} else {
			off = 8
		}
		if leftHalf {
			off = -off
		}
		dc.LineTo(cx+off, y)
	}
	dc.ClosePath()

	dc.SetRGB(0.87, 0.66, 0.32)
	dc.FillPreserve()

	dc.SetRGB(0.55, 0.36, 0.14)
	dc.SetLineWidth(3)
	dc.Stroke()

	for s := 0; s < 30; s++ {
		sx := cx + (radius-20)*(fortuneCookieRng.Float64()*2-1)
		sy := cy + (radius-20)*(fortuneCookieRng.Float64()*2-1)
		if leftHalf && sx > cx-5 {
			continue
		}
		if !leftHalf && sx < cx+5 {
			continue
		}
		dc.SetRGBA(0.45, 0.27, 0.10, 0.35)
		dc.DrawCircle(sx, sy, 1.2+fortuneCookieRng.Float64()*1.6)
		dc.Fill()
	}

	if leftHalf {
		dc.SetRGBA(0.65, 0.45, 0.18, 0.6)
		dc.SetLineWidth(2)
		dc.DrawArc(cx-radius*0.3, cy, radius*0.6, -math.Pi/2.2, math.Pi/2.2)
		dc.Stroke()
	} else {
		dc.SetRGBA(0.65, 0.45, 0.18, 0.6)
		dc.SetLineWidth(2)
		dc.DrawArc(cx+radius*0.3, cy, radius*0.6, math.Pi-math.Pi/2.2, math.Pi+math.Pi/2.2)
		dc.Stroke()
	}

	dc.Pop()
}

func fortuneCookieRender(fortune string) (string, error) {
	const W, H = 600, 400
	dc := gg.NewContext(W, H)

	for y := 0; y < H; y++ {
		t := float64(y) / float64(H)
		r := 0.10 + 0.05*t
		g := 0.05 + 0.04*t
		b := 0.18 + 0.10*t
		dc.SetRGB(r, g, b)
		dc.DrawRectangle(0, float64(y), float64(W), 1)
		dc.Fill()
	}

	for i := 0; i < 60; i++ {
		sx := fortuneCookieRng.Float64() * float64(W)
		sy := fortuneCookieRng.Float64() * float64(H)
		dc.SetRGBA(1, 1, 1, 0.05+fortuneCookieRng.Float64()*0.15)
		dc.DrawCircle(sx, sy, 0.6+fortuneCookieRng.Float64()*1.2)
		dc.Fill()
	}

	slipW := 360.0
	slipH := 90.0
	slipX := (float64(W) - slipW) / 2
	slipY := (float64(H)-slipH)/2 - 10

	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawRoundedRectangle(slipX+6, slipY+10, slipW, slipH, 4)
	dc.Fill()

	dc.Push()
	dc.RotateAbout(gg.Radians(-2.5), slipX+slipW/2, slipY+slipH/2)
	dc.SetRGB(0.98, 0.96, 0.88)
	dc.DrawRoundedRectangle(slipX, slipY, slipW, slipH, 4)
	dc.FillPreserve()
	dc.SetRGB(0.72, 0.62, 0.40)
	dc.SetLineWidth(1.5)
	dc.Stroke()

	for i := 0; i < 25; i++ {
		fx := slipX + fortuneCookieRng.Float64()*slipW
		fy := slipY + fortuneCookieRng.Float64()*slipH
		dc.SetRGBA(0.80, 0.70, 0.45, 0.18)
		dc.DrawCircle(fx, fy, 0.6+fortuneCookieRng.Float64()*1.2)
		dc.Fill()
	}

	dc.SetRGB(0.85, 0.74, 0.48)
	dc.SetLineWidth(1)
	dc.DrawLine(slipX+12, slipY+slipH/2, slipX+22, slipY+slipH/2)
	dc.Stroke()
	dc.DrawLine(slipX+slipW-22, slipY+slipH/2, slipX+slipW-12, slipY+slipH/2)
	dc.Stroke()

	fortuneCookieLoadFont(dc, 16)
	dc.SetRGB(0.20, 0.14, 0.08)
	lines := fortuneCookieWrap(fortune, 38)
	if len(lines) > 3 {
		lines = lines[:3]
		last := lines[2]
		if len(last) > 35 {
			last = last[:35] + "..."
		}
		lines[2] = last
	}
	lineH := 22.0
	totalH := float64(len(lines)) * lineH
	startY := slipY + (slipH-totalH)/2 + lineH/2
	for i, ln := range lines {
		dc.DrawStringAnchored(ln, slipX+slipW/2, startY+float64(i)*lineH, 0.5, 0.5)
	}
	dc.Pop()

	cx := float64(W) / 2
	cy := float64(H)/2 + 30
	radius := 140.0

	fortuneCookieDrawHalf(dc, cx-radius-10, cy, radius, true, -8)
	fortuneCookieDrawHalf(dc, cx+radius+10, cy, radius, false, 6)

	for i := 0; i < 14; i++ {
		px := cx + (fortuneCookieRng.Float64()*60 - 30)
		py := cy + radius + 20 + fortuneCookieRng.Float64()*20
		sz := 2.0 + fortuneCookieRng.Float64()*4
		dc.SetRGB(0.78, 0.55, 0.22)
		dc.DrawRectangle(px, py, sz, sz)
		dc.Fill()
	}

	fortuneCookieLoadFont(dc, 26)
	dc.SetRGBA(0, 0, 0, 0.45)
	dc.DrawStringAnchored("FORTUNE COOKIE", float64(W)/2+2, 36, 0.5, 0.5)
	dc.SetRGB(1, 0.92, 0.70)
	dc.DrawStringAnchored("FORTUNE COOKIE", float64(W)/2, 34, 0.5, 0.5)

	fortuneCookieLoadFont(dc, 12)
	dc.SetRGBA(1, 1, 1, 0.55)
	dc.DrawStringAnchored("crack open your destiny", float64(W)/2, 58, 0.5, 0.5)

	out := filepath.Join(os.TempDir(), fmt.Sprintf("fortune_cookie_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func FortuneCookieHandler(m *tg.NewMessage) error {
	pick := fortuneCookieList[fortuneCookieRng.Intn(len(fortuneCookieList))]

	status, _ := m.Reply("<code>cracking your fortune cookie...</code>")

	path, err := fortuneCookieRender(pick)
	if err != nil {
		msg := "<b>Failed to render fortune cookie.</b>"
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

	caption := fmt.Sprintf("<b>Fortune Cookie</b>\n\n<i>%s</i>", html.EscapeString(pick))
	_, merr := m.ReplyMedia(path, &tg.MediaOptions{
		Caption:  caption,
		FileName: "fortune_cookie.png",
		MimeType: "image/png",
	})
	os.Remove(path)
	if merr != nil {
		m.Reply("<b>Upload failed:</b> " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerFortuneCookieHandlers() {
	c := Client
	c.On("cmd:fortunecookie", FortuneCookieHandler)

	Mods.AddModule("FortuneCookie", `<b>Fortune Cookie Module</b>

<b>Commands:</b>
 • /fortunecookie - Generate a fortune cookie image with a random fortune slip

<i>Cracks open a digital cookie and reveals your fortune on a paper slip.</i>`)
}

func init() {
	QueueHandlerRegistration(registerFortuneCookieHandlers)
}
