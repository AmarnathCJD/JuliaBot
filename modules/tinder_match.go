package modules

import (
	"fmt"
	"hash/fnv"
	"html"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

type tinderUser struct {
	UserID    int64
	FirstName string
	LastName  string
	Username  string
	Photo     tg.UserProfilePhoto
	AvatarPng string
}

func tinderDisplayName(u tinderUser) string {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" && u.Username != "" {
		name = "@" + u.Username
	}
	if name == "" {
		name = fmt.Sprintf("User%d", u.UserID)
	}
	return name
}

func tinderResolveOne(m *tg.NewMessage, token string) (tinderUser, error) {
	var info tinderUser
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "@")
	if token == "" {
		return info, fmt.Errorf("empty token")
	}
	if n, err := strconv.ParseInt(token, 10, 64); err == nil {
		u, uerr := m.Client.GetUser(n)
		if uerr == nil && u != nil {
			info.UserID = u.ID
			info.FirstName = u.FirstName
			info.LastName = u.LastName
			info.Username = u.Username
			info.Photo = u.Photo
			return info, nil
		}
		info.UserID = n
		info.FirstName = "User"
		return info, nil
	}
	peer, err := m.Client.ResolvePeer(token)
	if err != nil {
		return info, fmt.Errorf("could not resolve %s", token)
	}
	id := m.Client.GetPeerID(peer)
	u, uerr := m.Client.GetUser(id)
	if uerr == nil && u != nil {
		info.UserID = u.ID
		info.FirstName = u.FirstName
		info.LastName = u.LastName
		info.Username = u.Username
		info.Photo = u.Photo
		return info, nil
	}
	info.UserID = id
	info.FirstName = token
	return info, nil
}

func tinderGetAccessHash(m *tg.NewMessage, userID int64) int64 {
	peer, err := m.Client.ResolvePeer(userID)
	if err != nil {
		return 0
	}
	if pu, ok := peer.(*tg.InputPeerUser); ok {
		return pu.AccessHash
	}
	return 0
}

func tinderDownloadAvatar(m *tg.NewMessage, u tinderUser) (string, error) {
	if u.Photo == nil {
		return "", fmt.Errorf("no photo")
	}
	full, err := m.Client.UsersGetFullUser(&tg.InputUserObj{
		UserID:     u.UserID,
		AccessHash: tinderGetAccessHash(m, u.UserID),
	})
	if err != nil || full == nil {
		return "", fmt.Errorf("full user fail")
	}
	uf := full.FullUser
	var photo tg.Photo
	if uf.ProfilePhoto != nil {
		photo = uf.ProfilePhoto
	} else if uf.PersonalPhoto != nil {
		photo = uf.PersonalPhoto
	} else if uf.FallbackPhoto != nil {
		photo = uf.FallbackPhoto
	}
	if photo == nil {
		return "", fmt.Errorf("no photo object")
	}
	p, ok := photo.(*tg.PhotoObj)
	if !ok || p == nil {
		return "", fmt.Errorf("invalid photo")
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("tinder_av_%d_%d.jpg", u.UserID, time.Now().UnixNano()))
	_, err = m.Client.DownloadMedia(p, &tg.DownloadOptions{
		FileName: tmp,
	})
	if err != nil {
		os.Remove(tmp)
		return "", err
	}
	return tmp, nil
}

func tinderLoadImage(path string) image.Image {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

func tinderSeed(a, b int64) int64 {
	lo, hi := a, b
	if lo > hi {
		lo, hi = hi, lo
	}
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("tinder:%d:%d", lo, hi)))
	return int64(h.Sum64())
}

func tinderPercent(a, b int64) int {
	seed := tinderSeed(a, b)
	r := rand.New(rand.NewSource(seed))
	return r.Intn(101)
}

func tinderColorForID(id int64, alt int) (float64, float64, float64) {
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("col:%d:%d", id, alt)))
	v := h.Sum64()
	hue := float64(v%360) / 360.0
	return tinderHSL(hue, 0.65, 0.55)
}

func tinderHSL(h, s, l float64) (float64, float64, float64) {
	var r, g, b float64
	if s == 0 {
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q
		r = tinderHueToRGB(p, q, h+1.0/3.0)
		g = tinderHueToRGB(p, q, h)
		b = tinderHueToRGB(p, q, h-1.0/3.0)
	}
	return r, g, b
}

func tinderHueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func tinderLoadFontFace(dc *gg.Context, size float64) {
	primary := memeFontPath("Swiss 721 Black Extended BT.ttf")
	if primary != "" {
		if err := dc.LoadFontFace(primary, size); err == nil {
			return
		}
	}
	fallback := memeFontPath("Inter_28pt-Bold.ttf")
	if fallback != "" {
		dc.LoadFontFace(fallback, size)
	}
}

func tinderDrawGradient(dc *gg.Context, w, h int, topR, topG, topB, botR, botG, botB float64) {
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h-1)
		r := topR*(1-t) + botR*t
		g := topG*(1-t) + botG*t
		b := topB*(1-t) + botB*t
		dc.SetRGB(r, g, b)
		dc.DrawRectangle(0, float64(y), float64(w), 1)
		dc.Fill()
	}
}

func tinderDrawInitialsAvatar(dc *gg.Context, cx, cy, radius float64, name string, id int64) {
	r1, g1, b1 := tinderColorForID(id, 0)
	r2, g2, b2 := tinderColorForID(id, 1)
	for i := 0; i < int(radius*2); i++ {
		t := float64(i) / (radius * 2)
		r := r1*(1-t) + r2*t
		g := g1*(1-t) + g2*t
		b := b1*(1-t) + b2*t
		dc.SetRGB(r, g, b)
		dc.DrawRectangle(cx-radius, cy-radius+float64(i), radius*2, 1)
		dc.Fill()
	}
	initials := tinderInitials(name)
	tinderLoadFontFace(dc, radius*0.9)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(initials, cx, cy, 0.5, 0.4)
}

func tinderInitials(name string) string {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		s := strings.TrimPrefix(parts[0], "@")
		if s == "" {
			return "?"
		}
		return strings.ToUpper(string([]rune(s)[0]))
	}
	a := strings.TrimPrefix(parts[0], "@")
	b := parts[len(parts)-1]
	if a == "" || b == "" {
		return "?"
	}
	return strings.ToUpper(string([]rune(a)[0]) + string([]rune(b)[0]))
}

func tinderDrawCard(dc *gg.Context, x, y, w, h float64, img image.Image, name string, id int64, label string, labelR, labelG, labelB float64, rotation float64) {
	dc.Push()
	dc.RotateAbout(rotation, x+w/2, y+h/2)

	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawRoundedRectangle(x+8, y+12, w, h, 28)
	dc.Fill()

	dc.SetRGB(1, 1, 1)
	dc.DrawRoundedRectangle(x, y, w, h, 28)
	dc.Fill()

	innerX := x + 14
	innerY := y + 14
	innerW := w - 28
	photoH := h - 110

	dc.Push()
	dc.DrawRoundedRectangle(innerX, innerY, innerW, photoH, 18)
	dc.Clip()
	if img != nil {
		bounds := img.Bounds()
		iw := float64(bounds.Dx())
		ih := float64(bounds.Dy())
		if iw > 0 && ih > 0 {
			scale := math.Max(innerW/iw, photoH/ih)
			drawW := iw * scale
			drawH := ih * scale
			dc.Push()
			dc.Translate(innerX+innerW/2, innerY+photoH/2)
			dc.Scale(scale, scale)
			dc.DrawImageAnchored(img, 0, 0, 0.5, 0.5)
			dc.Pop()
			_ = drawW
			_ = drawH
		}
	} else {
		cx := innerX + innerW/2
		cy := innerY + photoH/2
		rad := math.Min(innerW, photoH) * 0.42
		tinderDrawInitialsAvatar(dc, cx, cy, rad, name, id)
	}
	dc.ResetClip()
	dc.Pop()

	dc.SetRGBA(labelR, labelG, labelB, 0.95)
	dc.SetLineWidth(6)
	dc.DrawRoundedRectangle(innerX+18, innerY+18, 150, 60, 10)
	dc.Stroke()
	tinderLoadFontFace(dc, 38)
	dc.SetRGB(labelR, labelG, labelB)
	dc.DrawStringAnchored(label, innerX+18+75, innerY+18+34, 0.5, 0.5)

	nameAreaY := innerY + photoH + 18
	tinderLoadFontFace(dc, 34)
	displayName := tinderTruncate(name, 18)
	dc.SetRGB(0.13, 0.13, 0.15)
	dc.DrawStringAnchored(displayName, x+w/2, nameAreaY+18, 0.5, 0.5)

	tinderLoadFontFace(dc, 22)
	dc.SetRGB(0.45, 0.45, 0.5)
	dc.DrawStringAnchored("Telegram User", x+w/2, nameAreaY+56, 0.5, 0.5)

	dc.Pop()
}

func tinderTruncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "..."
}

func tinderDrawHeart(dc *gg.Context, cx, cy, size float64, r, g, b float64) {
	dc.SetRGB(r, g, b)
	dc.NewSubPath()
	steps := 200
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps) * 2 * math.Pi
		x := 16 * math.Pow(math.Sin(t), 3)
		y := -(13*math.Cos(t) - 5*math.Cos(2*t) - 2*math.Cos(3*t) - math.Cos(4*t))
		px := cx + x*size/16
		py := cy + y*size/16
		if i == 0 {
			dc.MoveTo(px, py)
		} else {
			dc.LineTo(px, py)
		}
	}
	dc.ClosePath()
	dc.Fill()
}

func tinderDrawSwipeArrow(dc *gg.Context, cx, cy, size float64, right bool, r, g, b float64) {
	dc.SetRGB(r, g, b)
	dc.SetLineWidth(size * 0.18)
	dc.SetLineCap(gg.LineCapRound)
	dx := size
	if !right {
		dx = -size
	}
	dc.DrawLine(cx-dx*0.6, cy, cx+dx*0.6, cy)
	dc.Stroke()
	dc.DrawLine(cx+dx*0.2, cy-size*0.5, cx+dx*0.6, cy)
	dc.Stroke()
	dc.DrawLine(cx+dx*0.2, cy+size*0.5, cx+dx*0.6, cy)
	dc.Stroke()
}

func tinderRender(u1, u2 tinderUser, pct int) (string, error) {
	const W, H = 1100, 1400
	dc := gg.NewContext(W, H)

	tinderDrawGradient(dc, W, H, 0.07, 0.07, 0.12, 0.96, 0.30, 0.45)

	seed := tinderSeed(u1.UserID, u2.UserID)
	r := rand.New(rand.NewSource(seed))
	for i := 0; i < 60; i++ {
		x := r.Float64() * float64(W)
		y := r.Float64() * float64(H)
		rad := r.Float64()*18 + 4
		dc.SetRGBA(1, 1, 1, 0.04+r.Float64()*0.08)
		tinderDrawHeart(dc, x, y, rad, 1, 1, 1)
	}

	tinderLoadFontFace(dc, 78)
	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawStringAnchored("tinder", float64(W)/2+3, 73, 0.5, 0.5)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored("tinder", float64(W)/2, 70, 0.5, 0.5)

	var img1 image.Image
	var img2 image.Image
	if u1.AvatarPng != "" {
		img1 = tinderLoadImage(u1.AvatarPng)
	}
	if u2.AvatarPng != "" {
		img2 = tinderLoadImage(u2.AvatarPng)
	}

	cardW := 430.0
	cardH := 560.0
	cardY := 280.0

	leftX := 80.0
	rightX := float64(W) - 80.0 - cardW

	tinderDrawCard(dc, leftX, cardY, cardW, cardH, img1, tinderDisplayName(u1), u1.UserID, "LIKE", 0.18, 0.78, 0.36, -0.08)
	tinderDrawCard(dc, rightX, cardY, cardW, cardH, img2, tinderDisplayName(u2), u2.UserID, "LIKE", 0.96, 0.20, 0.40, 0.08)

	arrowY := cardY + cardH/2
	tinderDrawSwipeArrow(dc, leftX+cardW+40, arrowY, 40, true, 1, 1, 1)
	tinderDrawSwipeArrow(dc, rightX-40, arrowY, 40, false, 1, 1, 1)

	heartCX := float64(W) / 2
	heartCY := arrowY
	tinderDrawHeart(dc, heartCX+4, heartCY+6, 90, 0, 0, 0)
	heartR, heartG, heartB := 0.96, 0.20, 0.40
	if pct >= 70 {
		heartR, heartG, heartB = 1.0, 0.85, 0.30
	}
	tinderDrawHeart(dc, heartCX, heartCY, 90, heartR, heartG, heartB)

	tinderLoadFontFace(dc, 30)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%d%%", pct), heartCX, heartCY+4, 0.5, 0.5)

	bannerY := cardY + cardH + 70
	bannerH := 130.0
	if pct >= 70 {
		dc.SetRGBA(0, 0, 0, 0.45)
		dc.DrawRoundedRectangle(80, bannerY+8, float64(W)-160, bannerH, 26)
		dc.Fill()
		for i := 0; i < int(bannerH); i++ {
			t := float64(i) / bannerH
			r := 0.98*(1-t) + 0.96*t
			g := 0.30*(1-t) + 0.18*t
			b := 0.40*(1-t) + 0.55*t
			dc.SetRGB(r, g, b)
			dc.DrawRectangle(80, bannerY+float64(i), float64(W)-160, 1)
			dc.Fill()
		}
		tinderLoadFontFace(dc, 72)
		dc.SetRGBA(0, 0, 0, 0.4)
		dc.DrawStringAnchored("It's a Match!", float64(W)/2+3, bannerY+bannerH/2-12+3, 0.5, 0.5)
		dc.SetRGB(1, 1, 1)
		dc.DrawStringAnchored("It's a Match!", float64(W)/2, bannerY+bannerH/2-12, 0.5, 0.5)
		tinderLoadFontFace(dc, 30)
		dc.SetRGBA(1, 1, 1, 0.92)
		dc.DrawStringAnchored(fmt.Sprintf("%s and %s liked each other", tinderTruncate(tinderDisplayName(u1), 14), tinderTruncate(tinderDisplayName(u2), 14)), float64(W)/2, bannerY+bannerH/2+34, 0.5, 0.5)
	} else {
		dc.SetRGBA(0, 0, 0, 0.5)
		dc.DrawRoundedRectangle(80, bannerY+8, float64(W)-160, bannerH, 26)
		dc.Fill()
		dc.SetRGBA(1, 1, 1, 0.12)
		dc.DrawRoundedRectangle(80, bannerY, float64(W)-160, bannerH, 26)
		dc.Fill()
		dc.SetRGBA(1, 1, 1, 0.4)
		dc.SetLineWidth(3)
		dc.DrawRoundedRectangle(80, bannerY, float64(W)-160, bannerH, 26)
		dc.Stroke()
		tinderLoadFontFace(dc, 60)
		dc.SetRGB(1, 1, 1)
		var msg string
		switch {
		case pct < 25:
			msg = "Swipe Left"
		case pct < 50:
			msg = "Not Quite"
		default:
			msg = "Almost There"
		}
		dc.DrawStringAnchored(msg, float64(W)/2, bannerY+bannerH/2-12, 0.5, 0.5)
		tinderLoadFontFace(dc, 28)
		dc.SetRGBA(1, 1, 1, 0.85)
		dc.DrawStringAnchored(fmt.Sprintf("%s x %s", tinderTruncate(tinderDisplayName(u1), 14), tinderTruncate(tinderDisplayName(u2), 14)), float64(W)/2, bannerY+bannerH/2+34, 0.5, 0.5)
	}

	tinderLoadFontFace(dc, 24)
	dc.SetRGBA(1, 1, 1, 0.65)
	dc.DrawStringAnchored("swipe right to match", float64(W)/2, float64(H)-40, 0.5, 0.5)

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("tindermatch_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func tinderExtractTokens(m *tg.NewMessage) ([]string, error) {
	raw := strings.TrimSpace(m.Args())
	fields := strings.Fields(raw)
	var tokens []string
	for _, f := range fields {
		tokens = append(tokens, f)
		if len(tokens) == 2 {
			break
		}
	}
	if m.IsReply() && len(tokens) < 2 {
		reply, err := m.GetReplyMessage()
		if err == nil && reply != nil && reply.SenderID() != 0 {
			tokens = append([]string{strconv.FormatInt(reply.SenderID(), 10)}, tokens...)
			if len(tokens) > 2 {
				tokens = tokens[:2]
			}
		}
	}
	if len(tokens) < 2 {
		if m.Sender != nil {
			selfTok := strconv.FormatInt(m.Sender.ID, 10)
			tokens = append([]string{selfTok}, tokens...)
		}
	}
	if len(tokens) < 2 {
		return nil, fmt.Errorf("need two users")
	}
	return tokens[:2], nil
}

func TinderMatchHandler(m *tg.NewMessage) error {
	tokens, err := tinderExtractTokens(m)
	if err != nil {
		m.Reply("<b>Tinder Match</b>\n\n<b>Usage:</b> <code>/tindermatch @user1 @user2</code>\nOr reply to a user with <code>/tindermatch @user2</code>")
		return nil
	}

	status, _ := m.Reply("<i>finding a match...</i>")

	u1, err1 := tinderResolveOne(m, tokens[0])
	if err1 != nil {
		if status != nil {
			status.Edit("failed: " + html.EscapeString(err1.Error()))
		}
		return nil
	}
	u2, err2 := tinderResolveOne(m, tokens[1])
	if err2 != nil {
		if status != nil {
			status.Edit("failed: " + html.EscapeString(err2.Error()))
		}
		return nil
	}

	if u1.UserID == u2.UserID {
		if status != nil {
			status.Edit("you cannot match a user with themselves")
		}
		return nil
	}

	if p, err := tinderDownloadAvatar(m, u1); err == nil {
		u1.AvatarPng = p
		defer os.Remove(p)
	}
	if p, err := tinderDownloadAvatar(m, u2); err == nil {
		u2.AvatarPng = p
		defer os.Remove(p)
	}

	pct := tinderPercent(u1.UserID, u2.UserID)
	outPath, err := tinderRender(u1, u2, pct)
	if err != nil || outPath == "" {
		errMsg := "render failed"
		if err != nil {
			errMsg = html.EscapeString(err.Error())
		}
		if status != nil {
			status.Edit("failed: " + errMsg)
		}
		return nil
	}

	caption := fmt.Sprintf("<b>%s</b> + <b>%s</b> = <code>%d%%</code>",
		html.EscapeString(tinderDisplayName(u1)),
		html.EscapeString(tinderDisplayName(u2)),
		pct)
	if pct >= 70 {
		caption += "\n<i>It's a Match!</i>"
	}

	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		Caption:  caption,
		FileName: "tindermatch.png",
		MimeType: "image/png",
	})
	os.Remove(outPath)
	if merr != nil {
		if status != nil {
			status.Edit("upload failed: " + html.EscapeString(merr.Error()))
		}
		return nil
	}
	if status != nil {
		status.Delete()
	}
	return nil
}

func registerTinderMatchHandlers() {
	c := Client
	c.On("cmd:tindermatch", TinderMatchHandler)
}

func init() {
	QueueHandlerRegistration(registerTinderMatchHandlers)
}
