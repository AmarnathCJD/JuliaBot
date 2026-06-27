package modules

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"main/modules/db"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/image/font/basicfont"
)

var quoteNameColorsLight = [7]color.RGBA{
	{0xFC, 0x5C, 0x51, 0xFF}, {0xFA, 0x79, 0x0F, 0xFF}, {0x89, 0x5D, 0xD5, 0xFF},
	{0x0F, 0xB2, 0x97, 0xFF}, {0x0F, 0xC9, 0xD6, 0xFF}, {0x3C, 0xA5, 0xEC, 0xFF},
	{0xD5, 0x4F, 0xAF, 0xFF},
}

var quoteNameColorsDark = [7]color.RGBA{
	{0xFF, 0x8E, 0x86, 0xFF}, {0xFF, 0xA3, 0x57, 0xFF}, {0xB1, 0x8F, 0xFF, 0xFF},
	{0x4D, 0xD6, 0xBF, 0xFF}, {0x45, 0xE8, 0xD1, 0xFF}, {0x7A, 0xC9, 0xFF, 0xFF},
	{0xFF, 0x7F, 0xD5, 0xFF},
}

var quoteAvatarColors = [7][2]color.RGBA{
	{{0xFF, 0x88, 0x5E, 0xFF}, {0xFF, 0x51, 0x6A, 0xFF}},
	{{0xFF, 0xCD, 0x6A, 0xFF}, {0xFF, 0xA8, 0x5C, 0xFF}},
	{{0xE0, 0xA2, 0xF3, 0xFF}, {0xD6, 0x69, 0xED, 0xFF}},
	{{0xA0, 0xDE, 0x7E, 0xFF}, {0x54, 0xCB, 0x68, 0xFF}},
	{{0x53, 0xED, 0xD6, 0xFF}, {0x28, 0xC9, 0xB7, 0xFF}},
	{{0x72, 0xD5, 0xFD, 0xFF}, {0x2A, 0x9E, 0xF1, 0xFF}},
	{{0xFF, 0xA8, 0xA8, 0xFF}, {0xFF, 0x71, 0x9A, 0xFF}},
}

var quoteDefaultBg = color.RGBA{0x29, 0x22, 0x32, 0xFF}

const (
	quoteScale       = 2.0
	quotePadX        = 16.0
	quotePadY        = 15.0
	quoteGap         = 9.0
	quoteHeaderGap   = 8.0
	quoteRadius      = 25.0
	quoteShadowPad   = 6.0
	quoteTailSize    = 14.0
	quoteMinWidth    = 100.0
	quoteAvatarSize  = 50.0
	quoteAvatarGap   = 10.0
	quoteBlockPadY   = 6.0
	quoteBlockPadL   = 10.0
	quoteBlockPadR   = 10.0
	quoteBlockBar    = 3.0
	quoteBlockRadius = 8.0
	quoteBlockTint   = 0.12

	quoteWidthBase = 512.0
)

var quotesBucket = []byte("quotes")

var quotesRng = rand.New(rand.NewSource(time.Now().UnixNano()))

type quoteRecord struct {
	ID          uint64 `json:"id"`
	ChatID      int64  `json:"chat_id"`
	UserID      int64  `json:"user_id"`
	UserName    string `json:"user_name"`
	UserHandle  string `json:"user_handle"`
	Text        string `json:"text"`
	SavedBy     int64  `json:"saved_by"`
	SavedByName string `json:"saved_by_name"`
	Timestamp   int64  `json:"ts"`
}

type quoteBlock struct {
	Name      string
	FirstName string
	LastName  string
	Handle    string
	Text      string
	Avatar    string
	UserID    int64
	ChatID    int64
	Date      int64
}

func quoteIsLight(c color.RGBA) bool {
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	hsp := math.Sqrt(0.299*r*r + 0.587*g*g + 0.114*b*b)
	return hsp > 127.5
}

func quoteColorLuminance(c color.RGBA, lum float64) color.RGBA {
	adjust := func(v uint8) uint8 {
		f := float64(v)
		f = math.Round(math.Min(math.Max(0, f+f*lum), 255))
		return uint8(f)
	}
	return color.RGBA{adjust(c.R), adjust(c.G), adjust(c.B), 255}
}

func quoteBrightness(c color.RGBA) float64 {
	return (float64(c.R)*299 + float64(c.G)*587 + float64(c.B)*114) / 1000
}

func quoteAdjustBrightness(c color.RGBA, amount float64) color.RGBA {
	clamp := func(v float64) uint8 {
		return uint8(math.Max(0, math.Min(255, v)))
	}
	return color.RGBA{
		clamp(float64(c.R) + amount),
		clamp(float64(c.G) + amount),
		clamp(float64(c.B) + amount),
		255,
	}
}

func quoteAdjustContrast(bg, fg color.RGBA) color.RGBA {
	const threshold = 175.0
	bb := quoteBrightness(bg)
	bf := quoteBrightness(fg)
	lightest := math.Max(bb, bf)
	darkest := math.Min(bb, bf)
	ratio := (lightest + 0.05) / (darkest + 0.05)
	if ratio >= 4.5 {
		return fg
	}
	diff := bb - bf
	if diff >= 0 {
		return quoteAdjustBrightness(fg, math.Ceil((threshold-bf)/2))
	}
	return quoteAdjustBrightness(fg, -math.Ceil((bf-threshold)/2))
}

func quoteNameColor(userID int64, bgOne, bgTwo color.RGBA) color.RGBA {
	pal := quoteNameColorsDark
	if quoteIsLight(bgOne) {
		pal = quoteNameColorsLight
	}
	idx := 1
	if userID != 0 {
		v := userID
		if v < 0 {
			v = -v
		}
		idx = int(v % 7)
	}
	nameColor := pal[idx]
	contrast := (quoteBrightness(quoteColorLuminance(bgOne, 0.55)) + 0.05) /
		(quoteBrightness(nameColor) + 0.05)
	if contrast < 1 {
		contrast = 1 / contrast
	}
	if contrast > 90 || contrast < 30 {
		nameColor = quoteAdjustContrast(quoteColorLuminance(bgTwo, 0.55), nameColor)
	}
	return nameColor
}

func quoteAvatarPair(userID int64) [2]color.RGBA {
	if userID == 0 {
		return quoteAvatarColors[quotesRng.Intn(7)]
	}
	v := userID
	if v < 0 {
		v = -v
	}
	return quoteAvatarColors[int(v%7)]
}

type quoteRadii struct{ tl, tr, br, bl float64 }

func quoteBubblePath(dc *gg.Context, w, h float64, r quoteRadii, tailSize float64) {
	cap := func(v float64) float64 { return math.Min(v, math.Min(w/2, h/2)) }
	tl, tr, br, bl := cap(r.tl), cap(r.tr), cap(r.br), cap(r.bl)

	dc.NewSubPath()
	dc.MoveTo(tl, 0)
	dc.LineTo(w-tr, 0)
	dc.DrawArc(w-tr, tr, tr, gg.Radians(-90), gg.Radians(0))
	dc.LineTo(w, h-br)
	dc.DrawArc(w-br, h-br, br, gg.Radians(0), gg.Radians(90))

	if tailSize > 0 {
		t := tailSize
		dc.LineTo(-t, h)
		// Cubic bezier — flat bottom edge curls up to the bubble's left edge.
		dc.CubicTo(-t*0.4, h, 0, h-bl*0.3, 0, h-bl)
	} else {
		dc.LineTo(bl, h)
		dc.DrawArc(bl, h-bl, bl, gg.Radians(90), gg.Radians(180))
	}
	dc.LineTo(0, tl)
	dc.DrawArc(tl, tl, tl, gg.Radians(180), gg.Radians(270))
	dc.ClosePath()
}

func quoteDrawGradientBubble(dc *gg.Context, x, y, w, h float64, c1, c2 color.RGBA, r quoteRadii, tailSize float64) {
	dc.Push()
	defer dc.Pop()
	dc.Translate(x, y)
	grad := gg.NewLinearGradient(0, 0, w, h)
	grad.AddColorStop(0, c1)
	grad.AddColorStop(1, c2)
	dc.SetFillStyle(grad)
	quoteBubblePath(dc, w, h, r, tailSize)
	dc.Fill()
}

func quoteDrawAccentBlock(dc *gg.Context, x, y, w, h float64, accent color.RGBA, s float64) {
	radius := quoteBlockRadius * s
	bar := quoteBlockBar * s

	dc.Push()
	dc.SetRGBA(float64(accent.R)/255, float64(accent.G)/255, float64(accent.B)/255, quoteBlockTint)
	dc.DrawRoundedRectangle(x, y, w, h, radius)
	dc.Fill()
	dc.Pop()

	dc.Push()
	dc.SetRGBA255(int(accent.R), int(accent.G), int(accent.B), 255)
	dc.DrawRoundedRectangle(x, y, bar, h, radius/2)
	dc.Fill()
	dc.Pop()
}

// quoteDrawShadow draws a single-pass offset silhouette of the bubble.
// NOTE: gg has no native Gaussian blur, so this is a deliberate compromise —
// a slightly soft offset instead of a true penumbra. Drawing the path twice
// produced a visible doubled outline, so we render once at alpha 0.18.
func quoteDrawShadow(dc *gg.Context, x, y, w, h float64, r quoteRadii, tailSize, s float64) {
	dc.Push()
	defer dc.Pop()
	dc.Translate(x, y+1*s)
	dc.SetRGBA(0, 0, 0, 0.18)
	quoteBubblePath(dc, w, h, r, tailSize)
	dc.Fill()
}

func quoteLoadFont(dc *gg.Context, size float64, bold bool) {
	var candidates []string
	if bold {
		candidates = []string{"NotoSans-Bold.ttf", "Inter_28pt-Bold.ttf"}
	} else {
		candidates = []string{"NotoSans-Regular.ttf"}
	}
	candidates = append(candidates, "Inter_28pt-Bold.ttf", "Swiss 721 Black Extended BT.ttf")
	for _, name := range candidates {
		if p := memeFontPath(name); p != "" {
			if err := dc.LoadFontFace(p, size); err == nil {
				return
			}
		}
	}
	dc.SetFontFace(basicfont.Face7x13)
}

var quoteHTMLTags = regexp.MustCompile(`<[^>]+>`)

func quoteSanitizeText(s string) string {
	if s == "" {
		return ""
	}
	stripped := quoteHTMLTags.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(stripped))
}

// quoteWrapLines — simple word wrap on the current dc font.
func quoteWrapLines(dc *gg.Context, text string, maxWidth float64) []string {
	if text == "" {
		return nil
	}
	var lines []string
	for paragraph := range strings.SplitSeq(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		current := ""
		for _, w := range words {
			trial := w
			if current != "" {
				trial = current + " " + w
			}
			tw, _ := dc.MeasureString(trial)
			if tw > maxWidth && current != "" {
				lines = append(lines, current)
				current = w
			} else {
				current = trial
			}
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	return lines
}

func quoteInitials(firstName, lastName string) string {
	first := strings.TrimSpace(firstName)
	last := strings.TrimSpace(lastName)
	if first != "" && last != "" {
		fr := []rune(first)
		lr := []rune(last)
		if len(fr) == 0 || len(lr) == 0 {
			return "?"
		}
		return strings.ToUpper(string(fr[0]) + string(lr[0]))
	}
	source := first
	if source == "" {
		source = last
	}
	parts := strings.Fields(source)
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		r := []rune(parts[0])
		if len(r) == 0 {
			return "?"
		}
		return strings.ToUpper(string(r[0]))
	}
	a := []rune(parts[0])
	b := []rune(parts[len(parts)-1])
	if len(a) == 0 || len(b) == 0 {
		return "?"
	}
	return strings.ToUpper(string(a[0]) + string(b[0]))
}
func quoteGetAccessHash(c *tg.Client, userID int64) int64 {
	peer, err := c.ResolvePeer(userID)
	if err != nil {
		return 0
	}
	if pu, ok := peer.(*tg.InputPeerUser); ok {
		return pu.AccessHash
	}
	return 0
}

func quoteDownloadAvatar(c *tg.Client, userID int64) string {
	if userID == 0 {
		return ""
	}
	full, err := c.UsersGetFullUser(&tg.InputUserObj{
		UserID:     userID,
		AccessHash: quoteGetAccessHash(c, userID),
	})
	if err != nil || full == nil {
		return ""
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
		return ""
	}
	p, ok := photo.(*tg.PhotoObj)
	if !ok || p == nil {
		return ""
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("qavatar_%d_%d.jpg", userID, time.Now().UnixNano()))
	_, err = c.DownloadMedia(p, &tg.DownloadOptions{FileName: tmp})
	if err != nil {
		os.Remove(tmp)
		return ""
	}
	return tmp
}

func quoteDrawAvatarCircle(dc *gg.Context, path string, cx, cy, radius float64, userID int64, firstName, lastName string) {
	if path != "" {
		if f, err := os.Open(path); err == nil {
			defer f.Close()
			if img, _, derr := image.Decode(f); derr == nil {
				b := img.Bounds()
				side := math.Min(float64(b.Dx()), float64(b.Dy()))
				// Crop to square (centered) then resize to the avatar diameter.
				sx := (float64(b.Dx()) - side) / 2
				sy := (float64(b.Dy()) - side) / 2
				diameter := int(math.Ceil(radius * 2))
				if diameter < 1 {
					diameter = 1
				}
				// Render the source into an intermediate context scaled to diameter.
				scaled := gg.NewContext(diameter, diameter)
				scale := float64(diameter) / side
				scaled.Scale(scale, scale)
				scaled.DrawImage(img, int(-sx), int(-sy))

				dc.Push()
				dc.DrawCircle(cx, cy, radius)
				dc.Clip()
				dc.DrawImageAnchored(scaled.Image(), int(cx), int(cy), 0.5, 0.5)
				dc.ResetClip()
				dc.Pop()
				return
			}
		}
	}

	pair := quoteAvatarPair(userID)
	dc.Push()
	dc.DrawCircle(cx, cy, radius)
	dc.Clip()
	grad := gg.NewLinearGradient(cx-radius, cy-radius, cx+radius, cy+radius)
	grad.AddColorStop(0, pair[0])
	grad.AddColorStop(1, pair[1])
	dc.SetFillStyle(grad)
	dc.DrawRectangle(cx-radius, cy-radius, radius*2, radius*2)
	dc.Fill()
	dc.ResetClip()
	dc.Pop()

	initials := quoteInitials(firstName, lastName)
	letterCount := len([]rune(initials))
	fontSize := radius * 2 * 0.48
	if letterCount > 1 {
		fontSize = radius * 2 * 0.38
	}
	quoteLoadFont(dc, fontSize, true)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(initials, cx, cy, 0.5, 0.5)
}

type quoteScene struct {
	main  quoteBlock
	reply *quoteBlock
}

type quoteLayout struct {
	s          float64
	nameSize   float64
	handleSize float64
	textSize   float64
	replyName  float64
	replyText  float64
	bubbleW    float64
	contentW   float64
	avatarSize float64
	avatarGap  float64

	nameH   float64
	headerH float64

	hasReply       bool
	replyAccent    color.RGBA
	replyNameLines []string
	replyLines     []string
	replyBlockH    float64

	textLines []string
	textH     float64

	bubbleH float64
}

func quoteBuildLayout(measureCtx *gg.Context, scene quoteScene, bgOne, bgTwo color.RGBA) quoteLayout {
	s := quoteScale
	L := quoteLayout{
		s:          s,
		nameSize:   22 * s,
		handleSize: 14 * s,
		textSize:   24 * s,
		replyName:  16 * s,
		replyText:  21 * s,
		bubbleW:    quoteWidthBase * s,
		avatarSize: quoteAvatarSize * s,
		avatarGap:  quoteAvatarGap * s,
	}
	L.contentW = L.bubbleW - 2*quotePadX*s

	quoteLoadFont(measureCtx, L.nameSize, true)
	_, nh := measureCtx.MeasureString(scene.main.Name)
	if nh == 0 {
		nh = L.nameSize
	}
	L.nameH = nh
	L.headerH = nh
	if scene.main.Handle != "" {
		L.headerH = nh + 4*s + L.handleSize*1.2
	}

	if scene.reply != nil {
		L.hasReply = true
		L.replyAccent = quoteNameColor(scene.reply.ChatID, bgOne, bgTwo)
		innerW := L.contentW - 2*quoteBlockPadL*s - quoteBlockBar*s - 6*s

		quoteLoadFont(measureCtx, L.replyName, true)
		L.replyNameLines = quoteWrapLines(measureCtx, scene.reply.Name, innerW)
		if len(L.replyNameLines) == 0 {
			L.replyNameLines = []string{scene.reply.Name}
		}
		nameLineH := L.replyName * 1.25

		quoteLoadFont(measureCtx, L.replyText, false)
		L.replyLines = quoteWrapLines(measureCtx, scene.reply.Text, innerW)
		textLineH := L.replyText * 1.25

		innerH := float64(len(L.replyNameLines))*nameLineH + 4*s + float64(len(L.replyLines))*textLineH
		L.replyBlockH = innerH + 2*quoteBlockPadY*s
	}

	quoteLoadFont(measureCtx, L.textSize, false)
	L.textLines = quoteWrapLines(measureCtx, scene.main.Text, L.contentW)
	lineH := L.textSize * 1.35
	L.textH = lineH * float64(len(L.textLines))

	L.bubbleH = quotePadY*s + L.headerH
	if L.hasReply {
		L.bubbleH += quoteGap*s + L.replyBlockH
	}
	if L.textH > 0 {
		L.bubbleH += quoteGap*s + L.textH
	}
	L.bubbleH += quotePadY * s
	if L.bubbleH < quoteMinWidth*s/2 {
		L.bubbleH = quoteMinWidth * s / 2
	}
	return L
}

func quoteDrawScene(dc *gg.Context, scene quoteScene, bubbleX, bubbleY, canvasH float64, L quoteLayout, bgOne, bgTwo color.RGBA) {
	s := L.s

	bubbleW := L.bubbleW
	bubbleH := L.bubbleH

	// Tail anchors against the left edge of the bubble (where it meets the
	// avatar column). Drop the bottom-left radius so the tail bezier flows
	// cleanly off the bubble.
	tailSize := quoteTailSize * s
	radii := quoteRadii{tl: quoteRadius * s, tr: quoteRadius * s, br: quoteRadius * s, bl: 0}

	// Draw avatar at canvas x=0, vertically aligned so the avatar's bottom sits
	// 2*s below the bubble's bottom edge (matches composer.js lines 271-276).
	avatarRadius := L.avatarSize / 2
	avatarCY := bubbleY + bubbleH + 2*s - avatarRadius
	avatarCX := avatarRadius
	quoteDrawAvatarCircle(dc, scene.main.Avatar, avatarCX, avatarCY, avatarRadius,
		scene.main.UserID, scene.main.FirstName, scene.main.LastName)

	quoteDrawShadow(dc, bubbleX, bubbleY, bubbleW, bubbleH, radii, tailSize, s)
	quoteDrawGradientBubble(dc, bubbleX, bubbleY, bubbleW, bubbleH, bgOne, bgTwo, radii, tailSize)

	textX := bubbleX + quotePadX*s
	nameColor := quoteNameColor(scene.main.UserID, bgOne, bgTwo)

	nameY := bubbleY + quotePadY*s + L.nameSize*0.85
	quoteLoadFont(dc, L.nameSize, true)
	dc.SetRGBA255(int(nameColor.R), int(nameColor.G), int(nameColor.B), 255)
	dc.DrawString(scene.main.Name, textX, nameY)
	if scene.main.Handle != "" {
		quoteLoadFont(dc, L.handleSize, false)
		dc.SetRGBA(0.78, 0.78, 0.86, 0.85)
		dc.DrawString("@"+scene.main.Handle, textX, nameY+4*s+L.handleSize)
	}

	cursorY := bubbleY + quotePadY*s + L.headerH
	if L.hasReply {
		cursorY += quoteGap * s
		quoteDrawAccentBlock(dc, textX, cursorY, L.contentW, L.replyBlockH, L.replyAccent, s)

		innerX := textX + quoteBlockBar*s + quoteBlockPadL*s
		innerY := cursorY + quoteBlockPadY*s

		quoteLoadFont(dc, L.replyName, true)
		dc.SetRGBA255(int(L.replyAccent.R), int(L.replyAccent.G), int(L.replyAccent.B), 255)
		nameLineH := L.replyName * 1.25
		for i, ln := range L.replyNameLines {
			dc.DrawString(ln, innerX, innerY+L.replyName*0.85+float64(i)*nameLineH)
		}
		quoteLoadFont(dc, L.replyText, false)
		if quoteIsLight(bgOne) {
			dc.SetRGB(0, 0, 0)
		} else {
			dc.SetRGB(1, 1, 1)
		}
		textLineH := L.replyText * 1.25
		textBaseY := innerY + float64(len(L.replyNameLines))*nameLineH + 4*s + L.replyText*0.85
		for i, ln := range L.replyLines {
			dc.DrawString(ln, innerX, textBaseY+float64(i)*textLineH)
		}
		cursorY += L.replyBlockH
	}

	if L.textH > 0 {
		cursorY += quoteGap * s
		quoteLoadFont(dc, L.textSize, false)
		if quoteIsLight(bgOne) {
			dc.SetRGB(0, 0, 0)
		} else {
			dc.SetRGB(1, 1, 1)
		}
		bodyY := cursorY + L.textSize*0.85
		lineH := L.textSize * 1.35
		for i, ln := range L.textLines {
			dc.DrawString(ln, textX, bodyY+float64(i)*lineH)
		}
	}
}

func quoteBuildBlock(m *tg.NewMessage, msg *tg.NewMessage, downloadAvatar bool) quoteBlock {
	text := quoteSanitizeText(msg.RawText())
	if len(text) > 600 {
		text = text[:600] + "..."
	}
	name := "User"
	firstName := ""
	lastName := ""
	handle := ""
	var userID int64
	if msg.SenderID() != 0 {
		userID = msg.SenderID()
		if u, uerr := m.Client.GetUser(userID); uerr == nil && u != nil {
			firstName = u.FirstName
			lastName = u.LastName
			name = strings.TrimSpace(firstName + " " + lastName)
			if name == "" {
				name = "User"
			}
			handle = u.Username
		}
	}
	avatar := ""
	if downloadAvatar {
		avatar = quoteDownloadAvatar(m.Client, userID)
	}
	return quoteBlock{
		Name:      name,
		FirstName: firstName,
		LastName:  lastName,
		Handle:    handle,
		Text:      text,
		Avatar:    avatar,
		UserID:    userID,
		ChatID:    msg.ChatID(),
		Date:      int64(msg.Date()),
	}
}

func quoteCollectScene(m *tg.NewMessage) (quoteScene, error) {
	main, err := m.GetReplyMessage()
	if err != nil || main == nil {
		return quoteScene{}, fmt.Errorf("no reply")
	}
	scene := quoteScene{main: quoteBuildBlock(m, main, true)}
	if main.IsReply() {
		if prev, perr := main.GetReplyMessage(); perr == nil && prev != nil {
			b := quoteBuildBlock(m, prev, false)
			if strings.TrimSpace(b.Name) != "" && strings.TrimSpace(b.Text) != "" {
				scene.reply = &b
			}
		}
	}
	return scene, nil
}

func quoteRenderImage(scene quoteScene) (string, error) {
	s := quoteScale
	bgOne := quoteColorLuminance(quoteDefaultBg, 0.35)
	bgTwo := quoteColorLuminance(quoteDefaultBg, -0.15)

	measureCtx := gg.NewContext(8, 8)
	L := quoteBuildLayout(measureCtx, scene, bgOne, bgTwo)

	// Bubble sits to the right of the avatar column. Shadow pad applied only
	// to the right and bottom edges (the shadow is offset down/right).
	shadowPad := quoteShadowPad * s
	bubblePosX := L.avatarSize + L.avatarGap
	canvasW := int(math.Ceil(bubblePosX + L.bubbleW + shadowPad))
	// Avatar may extend 2*s below the bubble; take whichever is taller.
	totalH := math.Max(L.bubbleH, L.avatarSize+2*s)
	canvasH := int(math.Ceil(totalH + shadowPad))

	rgba := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	dc := gg.NewContextForRGBA(rgba)
	quoteDrawScene(dc, scene, bubblePosX, 0, float64(canvasH), L, bgOne, bgTwo)

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("quote_%d.png", time.Now().UnixNano()))
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := png.Encode(f, rgba); err != nil {
		return "", err
	}
	return outPath, nil
}

func quotePngToWebp(pngPath string) (string, error) {
	webpPath := strings.TrimSuffix(pngPath, ".png") + ".webp"
	cmd := exec.Command("ffmpeg",
		"-loglevel", "error",
		"-y",
		"-i", pngPath,
		"-vf", "scale='if(gt(iw,ih),512,-1)':'if(gt(iw,ih),-1,512)':flags=lanczos",
		"-lossless", "1",
		"-pix_fmt", "yuva420p",
		webpPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg: %v: %s", err, string(out))
	}
	return webpPath, nil
}

func QuoteImageHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a message with <code>/q</code> to generate a quote.")
		return nil
	}

	status, _ := m.Reply("<i>painting your quote...</i>")

	scene, err := quoteCollectScene(m)
	if err != nil {
		if status != nil {
			status.Edit("could not read reply")
		}
		return nil
	}
	defer func() {
		if scene.main.Avatar != "" {
			os.Remove(scene.main.Avatar)
		}
		if scene.reply != nil && scene.reply.Avatar != "" {
			os.Remove(scene.reply.Avatar)
		}
	}()

	pngPath, rerr := quoteRenderImage(scene)
	if rerr != nil || pngPath == "" {
		errMsg := "render failed"
		if rerr != nil {
			errMsg = html.EscapeString(rerr.Error())
		}
		if status != nil {
			status.Edit("failed: " + errMsg)
		}
		return nil
	}
	defer os.Remove(pngPath)

	webpPath, werr := quotePngToWebp(pngPath)
	if werr != nil || webpPath == "" {
		if status != nil {
			status.Edit("ffmpeg failed: " + html.EscapeString(werr.Error()))
		}
		return nil
	}
	defer os.Remove(webpPath)

	_, merr := m.ReplyMedia(webpPath, &tg.MediaOptions{
		FileName: "quote.webp",
		MimeType: "image/webp",
		Attributes: []tg.DocumentAttribute{
			&tg.DocumentAttributeSticker{
				Alt:        "💬",
				Stickerset: &tg.InputStickerSetEmpty{},
			},
			&tg.DocumentAttributeFilename{FileName: "quote.webp"},
		},
	})
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

func quotesEnsureBucket() error {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return fmt.Errorf("db unavailable")
	}
	return d.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(quotesBucket)
		return e
	})
}

func quotesChatKey(chatID int64, id uint64) []byte {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[0:8], uint64(chatID))
	binary.BigEndian.PutUint64(b[8:16], id)
	return b
}

func quotesNextID(tx *bolt.Tx, chatID int64) uint64 {
	b := tx.Bucket(quotesBucket)
	if b == nil {
		return 1
	}
	prefix := make([]byte, 8)
	binary.BigEndian.PutUint64(prefix, uint64(chatID))
	c := b.Cursor()
	var maxID uint64
	for k, _ := c.Seek(prefix); len(k) >= 16; k, _ = c.Next() {
		if !quotesBytesHasPrefix(k, prefix) {
			break
		}
		id := binary.BigEndian.Uint64(k[8:16])
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1
}

func quotesBytesHasPrefix(a, b []byte) bool {
	if len(a) < len(b) {
		return false
	}
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func quotesListByChat(chatID int64) ([]quoteRecord, error) {
	if err := quotesEnsureBucket(); err != nil {
		return nil, err
	}
	d, err := db.GetDB()
	if err != nil || d == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var out []quoteRecord
	prefix := make([]byte, 8)
	binary.BigEndian.PutUint64(prefix, uint64(chatID))
	err = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(quotesBucket)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && quotesBytesHasPrefix(k, prefix); k, v = c.Next() {
			var rec quoteRecord
			if jerr := json.Unmarshal(v, &rec); jerr == nil {
				out = append(out, rec)
			}
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, err
}

func QuoteSaveHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a message with <code>/qsave</code> to save it.")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil || reply == nil {
		m.Reply("<b>Could not fetch the replied message.</b>")
		return nil
	}
	text := strings.TrimSpace(reply.RawText())
	if text == "" {
		m.Reply("<b>Nothing to save.</b> The message has no text.")
		return nil
	}
	if len(text) > 4000 {
		text = text[:4000]
	}

	var userID int64
	name := "User"
	handle := ""
	if reply.SenderID() != 0 {
		userID = reply.SenderID()
		u, uerr := m.Client.GetUser(userID)
		if uerr == nil && u != nil {
			name = strings.TrimSpace(u.FirstName + " " + u.LastName)
			if name == "" {
				name = "User"
			}
			handle = u.Username
		}
	}

	savedByName := "User"
	if m.Sender != nil {
		savedByName = strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
		if savedByName == "" {
			savedByName = "User"
		}
	}

	if err := quotesEnsureBucket(); err != nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}
	d, derr := db.GetDB()
	if derr != nil || d == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	var newID uint64
	werr := d.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(quotesBucket)
		if e != nil {
			return e
		}
		newID = quotesNextID(tx, m.ChatID())
		rec := quoteRecord{
			ID:          newID,
			ChatID:      m.ChatID(),
			UserID:      userID,
			UserName:    name,
			UserHandle:  handle,
			Text:        text,
			SavedBy:     m.SenderID(),
			SavedByName: savedByName,
			Timestamp:   time.Now().Unix(),
		}
		raw, jerr := json.Marshal(&rec)
		if jerr != nil {
			return jerr
		}
		return b.Put(quotesChatKey(m.ChatID(), newID), raw)
	})
	if werr != nil {
		m.Reply("<b>Failed to save quote.</b>")
		return nil
	}

	preview := text
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}
	m.Reply(fmt.Sprintf("<b>Quote saved.</b> <code>#%d</code>\n\n<b>%s</b>: <i>%s</i>",
		newID, html.EscapeString(name), html.EscapeString(preview)))
	return nil
}

func QuotesListHandler(m *tg.NewMessage) error {
	page := 1
	if a := strings.TrimSpace(m.Args()); a != "" {
		if n, err := strconv.Atoi(a); err == nil && n > 0 {
			page = n
		}
	}

	all, err := quotesListByChat(m.ChatID())
	if err != nil || len(all) == 0 {
		m.Reply("<b>No quotes saved here yet.</b> Reply to a message with <code>/qsave</code>.")
		return nil
	}

	perPage := 10
	totalPages := (len(all) + perPage - 1) / perPage
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * perPage
	end := start + perPage
	if end > len(all) {
		end = len(all)
	}

	var resp strings.Builder
	resp.WriteString(fmt.Sprintf("<b>Saved Quotes</b> (page %d/%d)\n", page, totalPages))
	resp.WriteString("━━━━━━━━━━━━━━━━\n\n")
	for _, rec := range all[start:end] {
		preview := rec.Text
		if len(preview) > 90 {
			preview = preview[:90] + "..."
		}
		resp.WriteString(fmt.Sprintf("<code>#%d</code> <b>%s</b>\n<i>%s</i>\n\n",
			rec.ID,
			html.EscapeString(rec.UserName),
			html.EscapeString(preview)))
	}
	resp.WriteString(fmt.Sprintf("━━━━━━━━━━━━━━━━\n<b>Total:</b> %d quotes\n", len(all)))
	if totalPages > 1 {
		resp.WriteString(fmt.Sprintf("<i>Use</i> <code>/quotes %d</code> <i>for next page</i>", page+1))
	}
	m.Reply(resp.String())
	return nil
}

func QuoteDeleteHandler(m *tg.NewMessage) error {
	if !m.IsPrivate() {
		if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
			m.Reply("<b>Permission denied.</b> Admins only.")
			return nil
		}
	}
	arg := strings.TrimSpace(m.Args())
	arg = strings.TrimPrefix(arg, "#")
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/delq &lt;id&gt;</code>")
		return nil
	}
	id, err := strconv.ParseUint(arg, 10, 64)
	if err != nil || id == 0 {
		m.Reply("<b>Invalid id.</b>")
		return nil
	}
	if err := quotesEnsureBucket(); err != nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}
	d, derr := db.GetDB()
	if derr != nil || d == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}
	found := false
	_ = d.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(quotesBucket)
		if b == nil {
			return nil
		}
		key := quotesChatKey(m.ChatID(), id)
		if b.Get(key) == nil {
			return nil
		}
		found = true
		return b.Delete(key)
	})
	if !found {
		m.Reply(fmt.Sprintf("<b>Quote not found:</b> <code>#%d</code>", id))
		return nil
	}
	m.Reply(fmt.Sprintf("<b>Quote deleted:</b> <code>#%d</code>", id))
	return nil
}

func QuotesSearchHandler(m *tg.NewMessage) error {
	q := strings.ToLower(strings.TrimSpace(m.Args()))
	if q == "" {
		m.Reply("<b>Usage:</b> <code>/qsearch &lt;keyword&gt;</code>")
		return nil
	}
	all, err := quotesListByChat(m.ChatID())
	if err != nil || len(all) == 0 {
		m.Reply("<b>No quotes to search.</b>")
		return nil
	}
	var matches []quoteRecord
	for _, rec := range all {
		if strings.Contains(strings.ToLower(rec.Text), q) ||
			strings.Contains(strings.ToLower(rec.UserName), q) ||
			strings.Contains(strings.ToLower(rec.UserHandle), q) {
			matches = append(matches, rec)
		}
	}
	if len(matches) == 0 {
		m.Reply(fmt.Sprintf("<b>No quotes match:</b> <code>%s</code>", html.EscapeString(q)))
		return nil
	}
	var resp strings.Builder
	resp.WriteString(fmt.Sprintf("<b>Quote Search:</b> <code>%s</code>\n", html.EscapeString(q)))
	resp.WriteString("━━━━━━━━━━━━━━━━\n\n")
	limit := 15
	for i, rec := range matches {
		if i >= limit {
			resp.WriteString(fmt.Sprintf("\n<i>...and %d more</i>", len(matches)-limit))
			break
		}
		preview := rec.Text
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		resp.WriteString(fmt.Sprintf("<code>#%d</code> <b>%s</b>\n<i>%s</i>\n\n",
			rec.ID, html.EscapeString(rec.UserName), html.EscapeString(preview)))
	}
	resp.WriteString(fmt.Sprintf("━━━━━━━━━━━━━━━━\n<b>Matches:</b> %d", len(matches)))
	m.Reply(resp.String())
	return nil
}

func registerQuotesHandlers() {
	c := Client
	c.On("cmd:q", QuoteImageHandler)
	c.On("cmd:qsave", QuoteSaveHandler)
	c.On("cmd:quotes", QuotesListHandler)
	c.On("cmd:delq", QuoteDeleteHandler)
	c.On("cmd:qsearch", QuotesSearchHandler)
}

func init() {
	QueueHandlerRegistration(registerQuotesHandlers)
}
