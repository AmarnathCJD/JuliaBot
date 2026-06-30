package extras

import (
	"fmt"
	"html"
	"image"
	"image/color"
	modules "main/modules"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

// === from meme.go ===
func memeFontPath(name string) string {
	candidates := []string{
		"./assets/" + name,
		"assets/" + name,
		"../assets/" + name,
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "assets", name),
			filepath.Join(dir, "..", "assets", name),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "assets", name))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func memeLoadFont(dc *gg.Context, size float64) bool {
	primary := memeFontPath("Swiss 721 Black Extended BT.ttf")
	if primary != "" {
		if err := dc.LoadFontFace(primary, size); err == nil {
			return true
		}
	}
	fallback := memeFontPath("Inter_28pt-Bold.ttf")
	if fallback != "" {
		if err := dc.LoadFontFace(fallback, size); err == nil {
			return true
		}
	}
	dc.SetFontFace(basicfont.Face7x13)
	return false
}

func memeParseArgs(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	lower := strings.ToLower(raw)
	topIdx := strings.Index(lower, "top")
	botIdx := strings.Index(lower, "bottom")

	var top, bottom string
	if topIdx >= 0 {
		end := len(raw)
		if botIdx > topIdx {
			end = botIdx
		}
		top = strings.TrimSpace(raw[topIdx+3 : end])
		top = memeStripQuotes(top)
	}
	if botIdx >= 0 {
		end := len(raw)
		if topIdx > botIdx {
			end = topIdx
		}
		bottom = strings.TrimSpace(raw[botIdx+6 : end])
		bottom = memeStripQuotes(bottom)
	}
	if top == "" && bottom == "" && topIdx < 0 && botIdx < 0 {
		parts := strings.SplitN(raw, "|", 2)
		if len(parts) == 2 {
			top = strings.TrimSpace(parts[0])
			bottom = strings.TrimSpace(parts[1])
		} else {
			top = raw
		}
	}
	return strings.ToUpper(top), strings.ToUpper(bottom)
}

func memeStripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		first := s[0]
		last := s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			s = s[1 : len(s)-1]
		}
	}
	return strings.TrimSpace(s)
}

func memeSkyGradientBg(dc *gg.Context, w, h int) {
	top := color.RGBA{R: 0x87, G: 0xce, B: 0xeb}
	mid := color.RGBA{R: 0xb0, G: 0xe0, B: 0xff}
	bottom := color.RGBA{R: 0xe6, G: 0xf4, B: 0xff}
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h-1)
		var r, g, b float64
		if t < 0.5 {
			lt := t / 0.5
			r = float64(top.R)*(1-lt) + float64(mid.R)*lt
			g = float64(top.G)*(1-lt) + float64(mid.G)*lt
			b = float64(top.B)*(1-lt) + float64(mid.B)*lt
		} else {
			lt := (t - 0.5) / 0.5
			r = float64(mid.R)*(1-lt) + float64(bottom.R)*lt
			g = float64(mid.G)*(1-lt) + float64(bottom.G)*lt
			b = float64(mid.B)*(1-lt) + float64(bottom.B)*lt
		}
		dc.SetRGB255(int(r), int(g), int(b))
		dc.DrawRectangle(0, float64(y), float64(w), 1)
		dc.Fill()
	}
	dc.SetRGBA(1, 1, 1, 0.85)
	dc.DrawCircle(180, 140, 36)
	dc.Fill()
	dc.SetRGBA(1, 1, 1, 0.75)
	dc.DrawCircle(230, 130, 30)
	dc.Fill()
	dc.SetRGBA(1, 1, 1, 0.8)
	dc.DrawCircle(260, 155, 28)
	dc.Fill()

	dc.SetRGBA(1, 1, 1, 0.75)
	dc.DrawCircle(820, 200, 42)
	dc.Fill()
	dc.SetRGBA(1, 1, 1, 0.65)
	dc.DrawCircle(880, 195, 36)
	dc.Fill()
	dc.SetRGBA(1, 1, 1, 0.7)
	dc.DrawCircle(770, 220, 30)
	dc.Fill()
}

func memeWrapLines(dc *gg.Context, text string, maxWidth float64) []string {
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	current := ""
	for _, w := range words {
		trial := current
		if trial == "" {
			trial = w
		} else {
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
	return lines
}

func memeFitText(dc *gg.Context, text string, maxWidth, maxHeight, startSize, minSize float64) (float64, []string) {
	size := startSize
	for size >= minSize {
		memeLoadFont(dc, size)
		lines := memeWrapLines(dc, text, maxWidth)
		if len(lines) == 0 {
			return size, nil
		}
		lineH := size * 1.15
		total := lineH * float64(len(lines))
		if total <= maxHeight && len(lines) <= 4 {
			return size, lines
		}
		size -= 4
	}
	memeLoadFont(dc, minSize)
	return minSize, memeWrapLines(dc, text, maxWidth)
}

func memeDrawOutlinedText(dc *gg.Context, line string, cx, cy float64, outline float64) {
	dc.SetRGB(0, 0, 0)
	for dy := -outline; dy <= outline; dy++ {
		for dx := -outline; dx <= outline; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			dc.DrawStringAnchored(line, cx+dx, cy+dy, 0.5, 0.5)
		}
	}
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(line, cx, cy, 0.5, 0.5)
}

func memeDrawCaption(dc *gg.Context, text string, w int, yAnchor float64, fromTop bool) {
	if text == "" {
		return
	}
	maxWidth := float64(w) - 60
	maxHeight := float64(w) * 0.28
	startSize := float64(w) / 9.5
	if startSize < 36 {
		startSize = 36
	}
	minSize := 22.0
	size, lines := memeFitText(dc, text, maxWidth, maxHeight, startSize, minSize)
	if len(lines) == 0 {
		return
	}
	memeLoadFont(dc, size)
	lineH := size * 1.15
	outline := size * 0.06
	if outline < 2 {
		outline = 2
	}

	if fromTop {
		y := yAnchor + size*0.6
		for _, ln := range lines {
			memeDrawOutlinedText(dc, ln, float64(w)/2, y, outline)
			y += lineH
		}
	} else {
		total := lineH * float64(len(lines))
		y := yAnchor - total + size*0.6
		for _, ln := range lines {
			memeDrawOutlinedText(dc, ln, float64(w)/2, y, outline)
			y += lineH
		}
	}
}

func memeRenderFromImage(img image.Image, top, bottom string) (string, error) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w < 1 || h < 1 {
		return "", fmt.Errorf("invalid image")
	}
	maxDim := 1200
	scale := 1.0
	if w > maxDim || h > maxDim {
		if w >= h {
			scale = float64(maxDim) / float64(w)
		} else {
			scale = float64(maxDim) / float64(h)
		}
	}
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)

	dc := gg.NewContext(nw, nh)
	dc.DrawImageAnchored(img, nw/2, nh/2, 0.5, 0.5)
	if scale != 1.0 {
		dc = gg.NewContext(nw, nh)
		dc.Push()
		dc.Scale(scale, scale)
		dc.DrawImage(img, 0, 0)
		dc.Pop()
	}

	memeDrawCaption(dc, top, nw, 20, true)
	memeDrawCaption(dc, bottom, nw, float64(nh)-20, false)

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("meme_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func memeRenderDefault(top, bottom string) (string, error) {
	const w, h = 960, 720
	dc := gg.NewContext(w, h)
	memeSkyGradientBg(dc, w, h)

	dc.SetRGBA(1, 1, 1, 0.6)
	dc.DrawRoundedRectangle(120, 260, w-240, 200, 30)
	dc.Fill()
	dc.SetRGBA(0, 0, 0, 0.25)
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(120, 260, w-240, 200, 30)
	dc.Stroke()

	memeLoadFont(dc, 80)
	dc.SetRGBA(0, 0, 0, 0.6)
	dc.DrawStringAnchored("?", w/2+3, 363, 0.5, 0.5)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored("?", w/2, 360, 0.5, 0.5)

	memeDrawCaption(dc, top, w, 20, true)
	memeDrawCaption(dc, bottom, w, float64(h)-20, false)

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("meme_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func MemeHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	top, bottom := memeParseArgs(args)

	if top == "" && bottom == "" {
		m.Reply("<b>Meme Generator</b>\n\n<b>Usage:</b> <code>/meme top &lt;text&gt; bottom &lt;text&gt;</code>\n\nExample: <code>/meme top \"WHEN YOU\" bottom \"DEBUG YOUR OWN CODE\"</code>\n\nReply to a photo to overlay text on it, or use without a reply for a default template.")
		return nil
	}

	status, _ := m.Reply("<i>cooking the meme...</i>")

	var outPath string
	var rerr error

	if m.IsReply() {
		reply, gerr := m.GetReplyMessage()
		if gerr == nil && reply != nil && reply.Photo() != nil {
			localPath, derr := reply.Download(&tg.DownloadOptions{})
			if derr == nil && localPath != "" {
				defer os.Remove(localPath)
				img, ierr := gg.LoadImage(localPath)
				if ierr == nil {
					outPath, rerr = memeRenderFromImage(img, top, bottom)
				} else {
					rerr = ierr
				}
			} else {
				rerr = derr
			}
		}
	}

	if outPath == "" {
		outPath, rerr = memeRenderDefault(top, bottom)
	}

	if rerr != nil || outPath == "" {
		errMsg := "render failed"
		if rerr != nil {
			errMsg = html.EscapeString(rerr.Error())
		}
		if status != nil {
			status.Edit("failed: " + errMsg)
		}
		return nil
	}

	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		FileName: "meme.png",
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

func registerMemeHandlers() {
	c := modules.Client
	c.On("cmd:meme", MemeHandler)
}

func initFromSrc_meme_0_1() {
	modules.QueueHandlerRegistration(registerMemeHandlers)
}

// === from meme_generator.go ===
var dramaMemeRng = rand.New(rand.NewSource(time.Now().UnixNano()))

func dramaMemeGradientBg(dc *gg.Context, w, h int) {
	palettes := [][2]color.RGBA{
		{{0x1a, 0x1a, 0x2e, 0xff}, {0xe9, 0x4b, 0x69, 0xff}},
		{{0x0f, 0x0c, 0x29, 0xff}, {0xff, 0x6a, 0x00, 0xff}},
		{{0x42, 0x27, 0x5a, 0xff}, {0x73, 0x4b, 0x6d, 0xff}},
		{{0x23, 0x07, 0x4d, 0xff}, {0xcc, 0x53, 0x33, 0xff}},
		{{0x00, 0x0c, 0x40, 0xff}, {0xf0, 0x47, 0x1b, 0xff}},
		{{0x14, 0x00, 0x21, 0xff}, {0xff, 0x36, 0x55, 0xff}},
	}
	pal := palettes[dramaMemeRng.Intn(len(palettes))]
	top := pal[0]
	bot := pal[1]
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h-1)
		r := float64(top.R)*(1-t) + float64(bot.R)*t
		g := float64(top.G)*(1-t) + float64(bot.G)*t
		b := float64(top.B)*(1-t) + float64(bot.B)*t
		dc.SetRGB255(int(r), int(g), int(b))
		dc.DrawRectangle(0, float64(y), float64(w), 1)
		dc.Fill()
	}
	for i := 0; i < 80; i++ {
		x := dramaMemeRng.Float64() * float64(w)
		yy := dramaMemeRng.Float64() * float64(h)
		rad := dramaMemeRng.Float64()*2.5 + 0.5
		alpha := dramaMemeRng.Float64()*0.4 + 0.2
		dc.SetRGBA(1, 1, 1, alpha)
		dc.DrawCircle(x, yy, rad)
		dc.Fill()
	}
}

func dramaRandomVividColor() color.RGBA {
	hues := []color.RGBA{
		{0xff, 0x6b, 0x6b, 0xff},
		{0x4e, 0xcd, 0xc4, 0xff},
		{0xff, 0xe6, 0x6d, 0xff},
		{0x95, 0xe1, 0xd3, 0xff},
		{0xf3, 0x8b, 0xa0, 0xff},
		{0xa8, 0xe6, 0xcf, 0xff},
		{0xff, 0x8b, 0x94, 0xff},
		{0xc7, 0x9e, 0xff, 0xff},
		{0x6c, 0x5c, 0xe7, 0xff},
		{0xfd, 0x79, 0xa8, 0xff},
		{0xfd, 0xcb, 0x6e, 0xff},
		{0x55, 0xef, 0xc4, 0xff},
		{0x74, 0xb9, 0xff, 0xff},
		{0xff, 0x76, 0x75, 0xff},
		{0xe1, 0x7e, 0xb5, 0xff},
	}
	return hues[dramaMemeRng.Intn(len(hues))]
}

func dramaDrawPhotoPanel(dc *gg.Context, x, y, w, h float64) {
	base := dramaRandomVividColor()
	accent := dramaRandomVividColor()
	for i := 0; i < int(h); i++ {
		t := float64(i) / h
		r := float64(base.R)*(1-t) + float64(accent.R)*t
		g := float64(base.G)*(1-t) + float64(accent.G)*t
		b := float64(base.B)*(1-t) + float64(accent.B)*t
		dc.SetRGB255(int(r), int(g), int(b))
		dc.DrawRectangle(x, y+float64(i), w, 1)
		dc.Fill()
	}
	dc.SetRGBA(0, 0, 0, 0.35)
	for i := 0; i < 6; i++ {
		cx := x + dramaMemeRng.Float64()*w
		cy := y + dramaMemeRng.Float64()*h
		rad := dramaMemeRng.Float64()*w*0.18 + 20
		dc.SetRGBA(1, 1, 1, 0.08+dramaMemeRng.Float64()*0.12)
		dc.DrawCircle(cx, cy, rad)
		dc.Fill()
	}
	dc.SetRGB(1, 1, 1)
	dc.SetLineWidth(8)
	dc.DrawRectangle(x, y, w, h)
	dc.Stroke()
	dc.SetRGBA(0, 0, 0, 0.55)
	dc.SetLineWidth(2)
	dc.DrawRectangle(x+4, y+4, w-8, h-8)
	dc.Stroke()
}

func dramaLoadFont(dc *gg.Context, size float64) {
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

func dramaWrapLines(dc *gg.Context, text string, maxWidth float64) []string {
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
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
	return lines
}

func dramaFitCaption(dc *gg.Context, text string, maxWidth, maxHeight, startSize, minSize float64) (float64, []string) {
	size := startSize
	for size >= minSize {
		dramaLoadFont(dc, size)
		lines := dramaWrapLines(dc, text, maxWidth)
		if len(lines) == 0 {
			return size, nil
		}
		lineH := size * 1.2
		total := lineH * float64(len(lines))
		if total <= maxHeight && len(lines) <= 5 {
			return size, lines
		}
		size -= 3
	}
	dramaLoadFont(dc, minSize)
	return minSize, dramaWrapLines(dc, text, maxWidth)
}

func dramaDrawCaptionText(dc *gg.Context, text string, areaX, areaY, areaW, areaH float64) {
	if text == "" {
		return
	}
	maxWidth := areaW - 60
	startSize := areaH / 3.2
	if startSize > areaW/8 {
		startSize = areaW / 8
	}
	if startSize < 28 {
		startSize = 28
	}
	size, lines := dramaFitCaption(dc, text, maxWidth, areaH-30, startSize, 22)
	if len(lines) == 0 {
		return
	}
	dramaLoadFont(dc, size)
	lineH := size * 1.2
	total := lineH * float64(len(lines))
	y := areaY + (areaH-total)/2 + size*0.75
	outline := size * 0.07
	if outline < 2 {
		outline = 2
	}
	cx := areaX + areaW/2
	for _, ln := range lines {
		dc.SetRGB(0, 0, 0)
		for dy := -outline; dy <= outline; dy++ {
			for dx := -outline; dx <= outline; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				dc.DrawStringAnchored(ln, cx+dx, y+dy, 0.5, 0.5)
			}
		}
		dc.SetRGB(1, 1, 1)
		dc.DrawStringAnchored(ln, cx, y, 0.5, 0.5)
		y += lineH
	}
}

func dramaRenderMeme(text string) (string, error) {
	const W, H = 1000, 1200
	dc := gg.NewContext(W, H)
	dramaMemeGradientBg(dc, W, H)

	margin := 60.0
	panelW := (float64(W) - margin*3) / 2
	panelH := 420.0
	panelY := 90.0

	dramaDrawPhotoPanel(dc, margin, panelY, panelW, panelH)
	dramaDrawPhotoPanel(dc, margin*2+panelW, panelY, panelW, panelH)

	dramaLoadFont(dc, 38)
	labels := []string{"BEFORE", "AFTER", "ME", "MY CODE", "EXPECTATION", "REALITY", "MONDAY", "FRIDAY"}
	li := dramaMemeRng.Intn(len(labels) / 2)
	leftLabel := labels[li*2]
	rightLabel := labels[li*2+1]

	dc.SetRGBA(0, 0, 0, 0.55)
	dc.DrawRoundedRectangle(margin+10, panelY+panelH-70, 220, 50, 10)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(leftLabel, margin+10+110, panelY+panelH-45, 0.5, 0.5)

	dc.SetRGBA(0, 0, 0, 0.55)
	dc.DrawRoundedRectangle(margin*2+panelW+10, panelY+panelH-70, 220, 50, 10)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(rightLabel, margin*2+panelW+10+110, panelY+panelH-45, 0.5, 0.5)

	dramaLoadFont(dc, 56)
	dc.SetRGBA(0, 0, 0, 0.7)
	dc.DrawRoundedRectangle(margin, 20, float64(W)-margin*2, 60, 14)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored("DRAMA", float64(W)/2, 52, 0.5, 0.5)

	captionY := panelY + panelH + 50
	captionH := float64(H) - captionY - 50
	dc.SetRGBA(0, 0, 0, 0.45)
	dc.DrawRoundedRectangle(margin, captionY, float64(W)-margin*2, captionH, 24)
	dc.Fill()
	dc.SetRGBA(1, 1, 1, 0.25)
	dc.SetLineWidth(3)
	dc.DrawRoundedRectangle(margin, captionY, float64(W)-margin*2, captionH, 24)
	dc.Stroke()

	dramaDrawCaptionText(dc, strings.ToUpper(text), margin, captionY, float64(W)-margin*2, captionH)

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("dramameme_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func DramaMemeHandler(m *tg.NewMessage) error {
	text := strings.TrimSpace(m.Args())
	if text == "" {
		m.Reply("<b>Drama Meme Generator</b>\n\n<b>Usage:</b> <code>/dramameme &lt;text&gt;</code>\n\nExample: <code>/dramameme when the build passes on the first try</code>")
		return nil
	}
	if len(text) > 220 {
		text = text[:220]
	}

	status, _ := m.Reply("<i>generating drama...</i>")

	outPath, err := dramaRenderMeme(text)
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

	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		FileName: "dramameme.png",
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

func registerMemeGeneratorHandlers() {
	c := modules.Client
	c.On("cmd:dramameme", DramaMemeHandler)
}

func initFromSrc_meme_generator_1_1() {
	modules.QueueHandlerRegistration(registerMemeGeneratorHandlers)
}

func init() {
	initFromSrc_meme_0_1()
	initFromSrc_meme_generator_1_1()
}
