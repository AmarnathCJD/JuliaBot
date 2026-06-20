package modules

import (
	"fmt"
	"html"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

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
	top := color.RGBA{0x87, 0xce, 0xeb, 0xff}
	mid := color.RGBA{0xb0, 0xe0, 0xff, 0xff}
	bottom := color.RGBA{0xe6, 0xf4, 0xff, 0xff}
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
	c := Client
	c.On("cmd:meme", MemeHandler)
}

func init() {
	QueueHandlerRegistration(registerMemeHandlers)
}
