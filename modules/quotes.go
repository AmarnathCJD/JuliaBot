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
	Name   string
	Handle string
	Text   string
	Avatar string
	UserID int64
	Date   int64
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

func quoteDrawShadow(dc *gg.Context, x, y, w, h float64, r quoteRadii, tailSize, s float64) {
	dc.Push()
	defer dc.Pop()
	dc.Translate(x, y+1*s)
	dc.SetRGBA(0, 0, 0, 0.10)
	quoteBubblePath(dc, w, h, r, tailSize)
	dc.Fill()

	dc.Translate(0, 1*s)
	dc.SetRGBA(0, 0, 0, 0.08)
	quoteBubblePath(dc, w, h, r, tailSize)
	dc.Fill()
}

func quoteLoadFont(dc *gg.Context, size float64, bold bool) {
	primary := memeFontPath("Inter_28pt-Bold.ttf")
	if !bold {
	}
	if primary != "" {
		if err := dc.LoadFontFace(primary, size); err == nil {
			return
		}
	}
	fallback := memeFontPath("Swiss 721 Black Extended BT.ttf")
	if fallback != "" {
		if err := dc.LoadFontFace(fallback, size); err == nil {
			return
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

func quoteInitials(name string) string {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		r := []rune(parts[0])
		if len(r) == 0 {
			return "?"
		}
		if len(r) == 1 {
			return strings.ToUpper(string(r[0]))
		}
		return strings.ToUpper(string(r[0]) + string(r[1]))
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

func quoteDrawAvatarCircle(dc *gg.Context, path string, cx, cy, radius float64, userID int64, name string) {
	if path != "" {
		if f, err := os.Open(path); err == nil {
			defer f.Close()
			if img, _, derr := image.Decode(f); derr == nil {
				dc.Push()
				dc.DrawCircle(cx, cy, radius)
				dc.Clip()
				b := img.Bounds()
				side := math.Min(float64(b.Dx()), float64(b.Dy()))
				scale := (radius * 2) / side
				scaled := gg.NewContext(int(radius*2)+2, int(radius*2)+2)
				scaled.ScaleAbout(scale, scale, float64(b.Dx())/2, float64(b.Dy())/2)
				scaled.DrawImageAnchored(img, int(radius), int(radius), 0.5, 0.5)
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

	initials := quoteInitials(name)
	letterCount := len([]rune(initials))
	fontSize := radius * 2 * 0.48
	if letterCount > 1 {
		fontSize = radius * 2 * 0.38
	}
	quoteLoadFont(dc, fontSize, true)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(initials, cx, cy, 0.5, 0.5)
}

type quoteMeasured struct {
	headerH    float64
	textLines  []string
	textH      float64
	totalH     float64
	contentW   float64
	bubbleW    float64
	nameSize   float64
	handleSize float64
	textSize   float64
	avatarSize float64
	avatarGap  float64
}

func quoteMeasureBlock(dc *gg.Context, b quoteBlock, s float64) quoteMeasured {
	m := quoteMeasured{
		avatarSize: quoteAvatarSize * s,
		avatarGap:  quoteAvatarGap * s,
		nameSize:   22 * s,
		handleSize: 14 * s,
		textSize:   24 * s,
	}
	m.bubbleW = quoteWidthBase * s
	m.contentW = m.bubbleW - 2*quotePadX*s

	quoteLoadFont(dc, m.nameSize, true)
	_, nameH := dc.MeasureString(b.Name)
	if nameH == 0 {
		nameH = m.nameSize
	}
	m.headerH = nameH
	if b.Handle != "" {
		m.headerH = nameH + 4*s + m.handleSize*1.2
	}

	quoteLoadFont(dc, m.textSize, false)
	m.textLines = quoteWrapLines(dc, b.Text, m.contentW)
	lineH := m.textSize * 1.35
	m.textH = lineH * float64(len(m.textLines))
	if len(m.textLines) == 0 {
		m.textH = 0
	}

	m.totalH = quotePadY*s + m.headerH + quoteGap*s + m.textH + quotePadY*s
	if m.totalH < quoteMinWidth*s/2 {
		m.totalH = quoteMinWidth * s / 2
	}
	return m
}

func quoteDrawSingleBubble(dc *gg.Context, b quoteBlock, originX, originY float64, m quoteMeasured, bgOne, bgTwo color.RGBA, isLast bool) float64 {
	s := quoteScale

	bubbleX := originX + m.avatarSize + m.avatarGap
	bubbleY := originY
	bubbleW := m.bubbleW
	bubbleH := m.totalH

	radii := quoteRadii{
		tl: quoteRadius * s,
		tr: quoteRadius * s,
		br: quoteRadius * s,
		bl: quoteRadius * s,
	}
	tail := 0.0
	if isLast {
		tail = quoteTailSize * s
	}

	quoteDrawShadow(dc, bubbleX, bubbleY, bubbleW, bubbleH, radii, tail, s)
	quoteDrawGradientBubble(dc, bubbleX, bubbleY, bubbleW, bubbleH, bgOne, bgTwo, radii, tail)

	if isLast {
		avX := originX + m.avatarSize/2
		avY := bubbleY + bubbleH - m.avatarSize/2 - 2*s
		quoteDrawAvatarCircle(dc, b.Avatar, avX, avY, m.avatarSize/2, b.UserID, b.Name)
	}

	nameColor := quoteNameColor(b.UserID, bgOne, bgTwo)
	textX := bubbleX + quotePadX*s
	nameY := bubbleY + quotePadY*s + m.nameSize*0.85
	quoteLoadFont(dc, m.nameSize, true)
	dc.SetRGBA255(int(nameColor.R), int(nameColor.G), int(nameColor.B), 255)
	dc.DrawString(b.Name, textX, nameY)

	if b.Handle != "" {
		quoteLoadFont(dc, m.handleSize, false)
		handleY := nameY + 4*s + m.handleSize
		dc.SetRGBA(0.78, 0.78, 0.86, 0.85)
		dc.DrawString("@"+b.Handle, textX, handleY)
	}

	quoteLoadFont(dc, m.textSize, false)
	if quoteIsLight(bgOne) {
		dc.SetRGB(0, 0, 0)
	} else {
		dc.SetRGB(0.96, 0.96, 0.97)
	}
	bodyY := bubbleY + quotePadY*s + m.headerH + quoteGap*s + m.textSize*0.85
	lineH := m.textSize * 1.35
	for i, ln := range m.textLines {
		dc.DrawString(ln, textX, bodyY+float64(i)*lineH)
	}

	return bubbleH
}

func quoteCollectChain(m *tg.NewMessage, maxDepth int) []quoteBlock {
	var blocks []quoteBlock
	current, err := m.GetReplyMessage()
	if err != nil || current == nil {
		return blocks
	}
	for depth := 0; depth < maxDepth && current != nil; depth++ {
		text := quoteSanitizeText(current.RawText())
		if text == "" {
			text = "(no text)"
		}
		if len(text) > 600 {
			text = text[:600] + "..."
		}
		name := "User"
		handle := ""
		var userID int64
		if current.SenderID() != 0 {
			userID = current.SenderID()
			u, uerr := m.Client.GetUser(userID)
			if uerr == nil && u != nil {
				name = strings.TrimSpace(u.FirstName + " " + u.LastName)
				if name == "" {
					name = "User"
				}
				handle = u.Username
			}
		}
		avatar := quoteDownloadAvatar(m.Client, userID)
		blocks = append(blocks, quoteBlock{
			Name:   name,
			Handle: handle,
			Text:   text,
			Avatar: avatar,
			UserID: userID,
			Date:   int64(current.Date()),
		})
		if !current.IsReply() {
			break
		}
		next, nerr := current.GetReplyMessage()
		if nerr != nil || next == nil {
			break
		}
		current = next
	}
	return blocks
}

func quoteRenderImage(blocks []quoteBlock, botName string) (string, error) {
	if len(blocks) == 0 {
		return "", fmt.Errorf("no blocks")
	}
	s := quoteScale

	bgOne := quoteColorLuminance(quoteDefaultBg, 0.35)
	bgTwo := quoteColorLuminance(quoteDefaultBg, -0.15)

	measureCtx := gg.NewContext(8, 8)
	measured := make([]quoteMeasured, len(blocks))
	maxBubbleW := 0.0
	totalH := 0.0
	gap := quoteGap * s * 2
	for i, b := range blocks {
		measured[i] = quoteMeasureBlock(measureCtx, b, s)
		if measured[i].bubbleW > maxBubbleW {
			maxBubbleW = measured[i].bubbleW
		}
		totalH += measured[i].totalH
		if i < len(blocks)-1 {
			totalH += gap
		}
	}

	originX := quoteShadowPad * s
	originY := quoteShadowPad * s
	footerH := 24 * s
	canvasW := int(originX + measured[0].avatarSize + measured[0].avatarGap + maxBubbleW + quoteShadowPad*s)
	canvasH := int(originY + totalH + footerH + quoteShadowPad*s)

	rgba := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	dc := gg.NewContextForRGBA(rgba)

	dc.SetRGBA(0, 0, 0, 0)
	dc.Clear()

	y := originY
	for i, b := range blocks {
		isLast := i == len(blocks)-1
		h := quoteDrawSingleBubble(dc, b, originX, y, measured[i], bgOne, bgTwo, isLast)
		y += h
		if !isLast {
			y += gap
		}
	}

	quoteLoadFont(dc, 12*s, false)
	dc.SetRGBA(0.78, 0.78, 0.86, 0.45)
	mark := "— via @" + botName
	if botName == "" {
		mark = "— via @JuliaBot"
	}
	dc.DrawStringAnchored(mark, float64(canvasW)-quoteShadowPad*s, y+footerH-6*s, 1.0, 0.5)

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

func QuoteImageHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a message with <code>/q</code> to generate a quote image.")
		return nil
	}

	status, _ := m.Reply("<i>painting your quote...</i>")

	blocks := quoteCollectChain(m, 3)
	if len(blocks) == 0 {
		if status != nil {
			status.Edit("could not read reply")
		}
		return nil
	}

	botName := "JuliaBot"
	if me, err := m.Client.GetMe(); err == nil && me != nil {
		if me.Username != "" {
			botName = me.Username
		} else if me.FirstName != "" {
			botName = me.FirstName
		}
	}

	defer func() {
		for _, b := range blocks {
			if b.Avatar != "" {
				os.Remove(b.Avatar)
			}
		}
	}()

	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	outPath, err := quoteRenderImage(blocks, botName)
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
	defer os.Remove(outPath)

	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		FileName: "quote.png",
		MimeType: "image/png",
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
