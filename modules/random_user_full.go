package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

type agentCardName struct {
	Title string `json:"title"`
	First string `json:"first"`
	Last  string `json:"last"`
}

type agentCardStreet struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
}

type agentCardLocation struct {
	Street   agentCardStreet `json:"street"`
	City     string          `json:"city"`
	State    string          `json:"state"`
	Country  string          `json:"country"`
	Postcode any             `json:"postcode"`
}

type agentCardDOB struct {
	Date string `json:"date"`
	Age  int    `json:"age"`
}

type agentCardID struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type agentCardPicture struct {
	Large     string `json:"large"`
	Medium    string `json:"medium"`
	Thumbnail string `json:"thumbnail"`
}

type agentCardLogin struct {
	UUID     string `json:"uuid"`
	Username string `json:"username"`
}

type agentCardUser struct {
	Gender   string            `json:"gender"`
	Name     agentCardName     `json:"name"`
	Location agentCardLocation `json:"location"`
	Email    string            `json:"email"`
	Login    agentCardLogin    `json:"login"`
	DOB      agentCardDOB      `json:"dob"`
	Phone    string            `json:"phone"`
	Cell     string            `json:"cell"`
	ID       agentCardID       `json:"id"`
	Picture  agentCardPicture  `json:"picture"`
	Nat      string            `json:"nat"`
}

type agentCardResponse struct {
	Results []agentCardUser `json:"results"`
}

func agentCardFetchUser() (*agentCardUser, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://randomuser.me/api/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var data agentCardResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Results) == 0 {
		return nil, fmt.Errorf("empty results")
	}
	return &data.Results[0], nil
}

func agentCardDownloadImage(url string) (image.Image, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("agentcard_pic_%d.jpg", time.Now().UnixNano()))
	f, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return nil, err
	}
	f.Close()
	defer os.Remove(tmp)
	img, err := gg.LoadImage(tmp)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func agentCardLoadFont(dc *gg.Context, name string, size float64) bool {
	p := memeFontPath(name)
	if p == "" {
		dc.SetFontFace(basicfont.Face7x13)
		return false
	}
	if err := dc.LoadFontFace(p, size); err != nil {
		dc.SetFontFace(basicfont.Face7x13)
		return false
	}
	return true
}

func agentCardFmtPostcode(p any) string {
	switch v := p.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%d", int(v))
	case int:
		return fmt.Sprintf("%d", v)
	}
	return ""
}

func agentCardTrunc(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "..."
}

func agentCardClearance(uuid string) string {
	if uuid == "" {
		return "ALPHA-1"
	}
	levels := []string{"ALPHA", "BRAVO", "DELTA", "ECHO", "OMEGA", "SIGMA"}
	sum := 0
	for _, c := range uuid {
		sum += int(c)
	}
	lvl := levels[sum%len(levels)]
	rank := (sum % 9) + 1
	return fmt.Sprintf("%s-%d", lvl, rank)
}

func agentCardDrawGradient(dc *gg.Context, x, y, w, h float64, r1, g1, b1, r2, g2, b2 float64) {
	steps := int(h)
	if steps < 1 {
		steps = 1
	}
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		if steps == 1 {
			t = 0
		}
		r := r1*(1-t) + r2*t
		g := g1*(1-t) + g2*t
		b := b1*(1-t) + b2*t
		dc.SetRGB(r, g, b)
		dc.DrawRectangle(x, y+float64(i), w, 1)
		dc.Fill()
	}
}

func agentCardDrawEmblem(dc *gg.Context, cx, cy, radius float64) {
	dc.SetRGBA(1, 0.85, 0.30, 0.95)
	dc.DrawCircle(cx, cy, radius)
	dc.Fill()

	dc.SetRGB(0.10, 0.10, 0.12)
	dc.SetLineWidth(3)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()

	dc.SetRGB(0.10, 0.10, 0.12)
	dc.SetLineWidth(2)
	dc.DrawCircle(cx, cy, radius*0.78)
	dc.Stroke()

	points := 5
	outer := radius * 0.55
	inner := radius * 0.22
	dc.NewSubPath()
	for i := 0; i < points*2; i++ {
		angle := -math.Pi/2 + float64(i)*math.Pi/float64(points)
		rr := outer
		if i%2 == 1 {
			rr = inner
		}
		px := cx + rr*math.Cos(angle)
		py := cy + rr*math.Sin(angle)
		if i == 0 {
			dc.MoveTo(px, py)
		} else {
			dc.LineTo(px, py)
		}
	}
	dc.ClosePath()
	dc.SetRGB(0.10, 0.10, 0.12)
	dc.Fill()

	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", radius*0.30)
	dc.SetRGB(0.10, 0.10, 0.12)
	dc.DrawStringAnchored("CIA", cx, cy+radius*0.62, 0.5, 0.5)
}

func agentCardDrawStripedHeader(dc *gg.Context, x, y, w, h float64) {
	dc.SetRGB(0.06, 0.07, 0.10)
	dc.DrawRectangle(x, y, w, h)
	dc.Fill()

	stripeH := 6.0
	for i := 0; float64(i)*stripeH < h; i++ {
		if i%2 == 0 {
			dc.SetRGBA(1, 0.85, 0.30, 0.10)
		} else {
			dc.SetRGBA(0.20, 0.30, 0.50, 0.10)
		}
		dc.DrawRectangle(x, y+float64(i)*stripeH, w, stripeH)
		dc.Fill()
	}
}

func agentCardDrawPhotoBox(dc *gg.Context, x, y, w, h float64, img image.Image) {
	dc.SetRGB(0.95, 0.95, 0.93)
	dc.DrawRectangle(x-6, y-6, w+12, h+12)
	dc.Fill()

	dc.SetRGB(0.85, 0.20, 0.20)
	dc.SetLineWidth(3)
	dc.DrawRectangle(x-6, y-6, w+12, h+12)
	dc.Stroke()

	dc.Push()
	dc.DrawRectangle(x, y, w, h)
	dc.Clip()
	if img != nil {
		bounds := img.Bounds()
		iw := float64(bounds.Dx())
		ih := float64(bounds.Dy())
		if iw > 0 && ih > 0 {
			scale := math.Max(w/iw, h/ih)
			dc.Push()
			dc.Translate(x+w/2, y+h/2)
			dc.Scale(scale, scale)
			dc.DrawImageAnchored(img, 0, 0, 0.5, 0.5)
			dc.Pop()
		}
	} else {
		dc.SetRGB(0.30, 0.30, 0.35)
		dc.DrawRectangle(x, y, w, h)
		dc.Fill()
		agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", 28)
		dc.SetRGB(0.85, 0.85, 0.85)
		dc.DrawStringAnchored("NO IMG", x+w/2, y+h/2, 0.5, 0.5)
	}
	dc.ResetClip()
	dc.Pop()

	dc.SetRGBA(0, 0, 0, 0.35)
	for i := 0; i < int(h); i += 4 {
		dc.DrawRectangle(x, y+float64(i), w, 1)
		dc.Fill()
	}
}

func agentCardDrawField(dc *gg.Context, x, y float64, label, value string, labelSize, valueSize float64, maxW float64) float64 {
	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", labelSize)
	dc.SetRGB(0.85, 0.20, 0.20)
	dc.DrawString(label, x, y)

	agentCardLoadFont(dc, "Inter_28pt-Bold.ttf", valueSize)
	dc.SetRGB(0.08, 0.08, 0.10)
	v := value
	if maxW > 0 {
		for {
			w, _ := dc.MeasureString(v)
			if w <= maxW || len([]rune(v)) <= 4 {
				break
			}
			r := []rune(v)
			v = string(r[:len(r)-2]) + "..."
		}
	}
	dc.DrawString(v, x, y+valueSize+4)
	return y + valueSize + 12
}

func agentCardRender(u *agentCardUser, img image.Image) (string, error) {
	const W, H = 1100, 700
	dc := gg.NewContext(W, H)

	dc.SetRGB(0.10, 0.11, 0.13)
	dc.Clear()

	margin := 40.0
	cardX := margin
	cardY := margin
	cardW := float64(W) - margin*2
	cardH := float64(H) - margin*2

	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawRoundedRectangle(cardX+8, cardY+12, cardW, cardH, 18)
	dc.Fill()

	agentCardDrawGradient(dc, cardX, cardY, cardW, cardH, 0.96, 0.94, 0.88, 0.82, 0.78, 0.68)

	dc.SetRGB(0.85, 0.20, 0.20)
	dc.SetLineWidth(4)
	dc.DrawRoundedRectangle(cardX, cardY, cardW, cardH, 18)
	dc.Stroke()

	headerH := 90.0
	agentCardDrawStripedHeader(dc, cardX, cardY, cardW, headerH)

	emblemR := 32.0
	agentCardDrawEmblem(dc, cardX+50, cardY+headerH/2, emblemR)

	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", 30)
	dc.SetRGB(1, 0.85, 0.30)
	dc.DrawString("CENTRAL INTELLIGENCE AGENCY", cardX+100, cardY+38)

	agentCardLoadFont(dc, "Inter_28pt-Bold.ttf", 16)
	dc.SetRGB(0.85, 0.85, 0.85)
	dc.DrawString("OFFICIAL IDENTIFICATION  -  CONFIDENTIAL", cardX+100, cardY+62)

	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", 14)
	dc.SetRGB(1, 0.85, 0.30)
	dc.DrawStringAnchored("USA", cardX+cardW-40, cardY+headerH/2-6, 0.5, 0.5)
	agentCardLoadFont(dc, "Inter_28pt-Bold.ttf", 10)
	dc.SetRGB(0.85, 0.85, 0.85)
	dc.DrawStringAnchored("CLEARANCE", cardX+cardW-40, cardY+headerH/2+10, 0.5, 0.5)

	photoX := cardX + 50
	photoY := cardY + headerH + 50
	photoW := 280.0
	photoH := 340.0
	agentCardDrawPhotoBox(dc, photoX, photoY, photoW, photoH, img)

	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", 13)
	dc.SetRGB(0.85, 0.20, 0.20)
	dc.DrawStringAnchored("AGENT PHOTO", photoX+photoW/2, photoY+photoH+24, 0.5, 0.5)

	infoX := photoX + photoW + 50
	infoY := photoY - 10
	infoMaxW := cardX + cardW - infoX - 40

	fullName := strings.TrimSpace(u.Name.Title + " " + u.Name.First + " " + u.Name.Last)
	codename := strings.ToUpper(u.Name.First) + " " + strings.ToUpper(string([]rune(u.Name.Last)[0])) + "."

	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", 14)
	dc.SetRGB(0.85, 0.20, 0.20)
	dc.DrawString("CODENAME", infoX, infoY+8)
	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", 30)
	dc.SetRGB(0.08, 0.08, 0.10)
	dc.DrawString(agentCardTrunc(codename, 18), infoX, infoY+44)

	y := infoY + 80
	y = agentCardDrawField(dc, infoX, y, "FULL NAME", fullName, 12, 18, infoMaxW)
	y = agentCardDrawField(dc, infoX, y, "DATE OF BIRTH", fmt.Sprintf("%s  (age %d)", agentCardFormatDate(u.DOB.Date), u.DOB.Age), 12, 16, infoMaxW)
	y = agentCardDrawField(dc, infoX, y, "GENDER", strings.ToUpper(u.Gender), 12, 16, infoMaxW)

	addr := fmt.Sprintf("%d %s, %s, %s %s",
		u.Location.Street.Number, u.Location.Street.Name,
		u.Location.City, u.Location.State, agentCardFmtPostcode(u.Location.Postcode))
	y = agentCardDrawField(dc, infoX, y, "ADDRESS", addr, 12, 14, infoMaxW)
	y = agentCardDrawField(dc, infoX, y, "COUNTRY", u.Location.Country, 12, 16, infoMaxW)
	y = agentCardDrawField(dc, infoX, y, "CONTACT", u.Phone, 12, 16, infoMaxW)

	footerY := cardY + cardH - 70
	dc.SetRGB(0.06, 0.07, 0.10)
	dc.DrawRectangle(cardX, footerY, cardW, 70)
	dc.Fill()

	stripeH := 4.0
	for i := 0; float64(i)*stripeH < 70; i++ {
		if i%2 == 0 {
			dc.SetRGBA(1, 0.85, 0.30, 0.08)
		} else {
			dc.SetRGBA(0.20, 0.30, 0.50, 0.08)
		}
		dc.DrawRectangle(cardX, footerY+float64(i)*stripeH, cardW, stripeH)
		dc.Fill()
	}

	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", 13)
	dc.SetRGB(1, 0.85, 0.30)
	dc.DrawString("ID NUMBER", cardX+30, footerY+22)
	dc.DrawString("CLEARANCE", cardX+330, footerY+22)
	dc.DrawString("ISSUED", cardX+630, footerY+22)

	agentCardLoadFont(dc, "Inter_28pt-Bold.ttf", 18)
	dc.SetRGB(0.95, 0.95, 0.95)
	idVal := u.ID.Value
	if idVal == "" {
		idVal = strings.ToUpper(u.Login.UUID[:8])
	}
	dc.DrawString(idVal, cardX+30, footerY+50)
	dc.DrawString(agentCardClearance(u.Login.UUID), cardX+330, footerY+50)
	dc.DrawString(time.Now().Format("2006-01-02"), cardX+630, footerY+50)

	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", 10)
	dc.SetRGBA(0.85, 0.20, 0.20, 0.45)
	dc.DrawStringAnchored("CONFIDENTIAL - PROPERTY OF C.I.A. - DO NOT DUPLICATE", cardX+cardW/2, cardY+cardH-8, 0.5, 0.5)

	dc.Push()
	dc.RotateAbout(-0.18, cardX+cardW-180, cardY+cardH-180)
	dc.SetRGBA(0.85, 0.20, 0.20, 0.55)
	dc.SetLineWidth(5)
	dc.DrawRoundedRectangle(cardX+cardW-280, cardY+cardH-220, 200, 70, 6)
	dc.Stroke()
	agentCardLoadFont(dc, "Swiss 721 Black Extended BT.ttf", 28)
	dc.DrawStringAnchored("CLASSIFIED", cardX+cardW-180, cardY+cardH-185, 0.5, 0.5)
	dc.Pop()

	out := filepath.Join(os.TempDir(), fmt.Sprintf("agentcard_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func agentCardFormatDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func AgentCardHandler(m *tg.NewMessage) error {
	status, _ := m.Reply("<code>generating agent dossier...</code>")

	user, err := agentCardFetchUser()
	if err != nil {
		msg := "failed to fetch user: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	var img image.Image
	if user.Picture.Large != "" {
		if im, ierr := agentCardDownloadImage(user.Picture.Large); ierr == nil {
			img = im
		}
	}

	out, err := agentCardRender(user, img)
	if err != nil {
		msg := "failed to render: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	caption := fmt.Sprintf(
		"<b>Agent Identification Card</b>\n"+
			"<b>Codename:</b> <code>%s</code>\n"+
			"<b>Full Name:</b> %s %s %s\n"+
			"<b>Age:</b> <code>%d</code>  -  <b>Gender:</b> <code>%s</code>\n"+
			"<b>Origin:</b> %s, %s\n"+
			"<b>Clearance:</b> <code>%s</code>",
		html.EscapeString(strings.ToUpper(user.Name.First)),
		html.EscapeString(user.Name.Title),
		html.EscapeString(user.Name.First),
		html.EscapeString(user.Name.Last),
		user.DOB.Age,
		html.EscapeString(strings.ToUpper(user.Gender)),
		html.EscapeString(user.Location.City),
		html.EscapeString(user.Location.Country),
		html.EscapeString(agentCardClearance(user.Login.UUID)),
	)

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  caption,
		FileName: "agentcard.png",
		MimeType: "image/png",
	})
	os.Remove(out)
	if merr != nil {
		m.Reply("upload failed: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerAgentCardHandlers() {
	c := Client
	c.On("cmd:agentcard", AgentCardHandler)
}

func init() {
	QueueHandlerRegistration(registerAgentCardHandlers)
}
