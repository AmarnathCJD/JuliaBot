package modules

import (
	"fmt"
	"html"
	"image/color"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

var drakeRng = rand.New(rand.NewSource(time.Now().UnixNano()))

func drakeLoadFont(dc *gg.Context, size float64) {
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

func drakeWrapLines(dc *gg.Context, text string, maxWidth float64) []string {
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

func drakeFitText(dc *gg.Context, text string, maxWidth, maxHeight, startSize, minSize float64) (float64, []string) {
	size := startSize
	for size >= minSize {
		drakeLoadFont(dc, size)
		lines := drakeWrapLines(dc, text, maxWidth)
		if len(lines) == 0 {
			return size, nil
		}
		lineH := size * 1.25
		total := lineH * float64(len(lines))
		if total <= maxHeight && len(lines) <= 6 {
			return size, lines
		}
		size -= 2
	}
	drakeLoadFont(dc, minSize)
	return minSize, drakeWrapLines(dc, text, maxWidth)
}

func drakeDrawText(dc *gg.Context, text string, areaX, areaY, areaW, areaH float64) {
	if text == "" {
		return
	}
	pad := 30.0
	maxWidth := areaW - pad*2
	startSize := areaH / 4.5
	if startSize > 64 {
		startSize = 64
	}
	if startSize < 28 {
		startSize = 28
	}
	size, lines := drakeFitText(dc, text, maxWidth, areaH-pad*2, startSize, 20)
	if len(lines) == 0 {
		return
	}
	drakeLoadFont(dc, size)
	lineH := size * 1.25
	total := lineH * float64(len(lines))
	y := areaY + (areaH-total)/2 + size*0.75
	cx := areaX + areaW/2
	dc.SetRGB(0.05, 0.05, 0.05)
	for _, ln := range lines {
		dc.DrawStringAnchored(ln, cx, y, 0.5, 0.5)
		y += lineH
	}
}

func drakeDrawHead(dc *gg.Context, cx, cy, scale float64, reject bool) {
	skin := color.RGBA{0x6e, 0x47, 0x32, 0xff}
	skinShadow := color.RGBA{0x4a, 0x2e, 0x1f, 0xff}
	beard := color.RGBA{0x18, 0x12, 0x0c, 0xff}

	headW := 180.0 * scale
	headH := 220.0 * scale

	dc.SetColor(skinShadow)
	dc.DrawEllipse(cx+8*scale, cy+10*scale, headW/2, headH/2)
	dc.Fill()

	dc.SetColor(skin)
	dc.DrawEllipse(cx, cy, headW/2, headH/2)
	dc.Fill()

	dc.SetColor(beard)
	dc.DrawArc(cx, cy+headH*0.12, headW*0.42, math.Pi*0.1, math.Pi*0.9)
	dc.SetLineWidth(headW * 0.10)
	dc.Stroke()

	dc.SetColor(beard)
	dc.DrawEllipse(cx, cy-headH*0.40, headW*0.55, headH*0.16)
	dc.Fill()
	dc.DrawRectangle(cx-headW*0.55, cy-headH*0.40, headW*1.10, headH*0.18)
	dc.Fill()

	eyeY := cy - headH*0.05
	eyeOffX := headW * 0.18
	eyeW := headW * 0.07

	if reject {
		dc.SetRGB(0.05, 0.05, 0.05)
		dc.SetLineWidth(6 * scale)
		dc.DrawLine(cx-eyeOffX-eyeW, eyeY-headH*0.04, cx-eyeOffX+eyeW, eyeY+headH*0.04)
		dc.Stroke()
		dc.DrawLine(cx+eyeOffX-eyeW, eyeY+headH*0.04, cx+eyeOffX+eyeW, eyeY-headH*0.04)
		dc.Stroke()

		dc.SetRGB(0.4, 0.1, 0.1)
		dc.SetLineWidth(8 * scale)
		mouthY := cy + headH*0.15
		dc.DrawArc(cx, mouthY+headH*0.06, headW*0.18, math.Pi*1.15, math.Pi*1.85)
		dc.Stroke()
	} else {
		dc.SetRGB(0.05, 0.05, 0.05)
		dc.DrawEllipse(cx-eyeOffX, eyeY, eyeW*0.55, eyeW*0.7)
		dc.Fill()
		dc.DrawEllipse(cx+eyeOffX, eyeY, eyeW*0.55, eyeW*0.7)
		dc.Fill()
		dc.SetRGB(1, 1, 1)
		dc.DrawEllipse(cx-eyeOffX+eyeW*0.15, eyeY-eyeW*0.15, eyeW*0.15, eyeW*0.18)
		dc.Fill()
		dc.DrawEllipse(cx+eyeOffX+eyeW*0.15, eyeY-eyeW*0.15, eyeW*0.15, eyeW*0.18)
		dc.Fill()

		dc.SetRGB(0.6, 0.18, 0.18)
		dc.SetLineWidth(7 * scale)
		mouthY := cy + headH*0.18
		dc.DrawArc(cx, mouthY-headH*0.04, headW*0.20, math.Pi*0.15, math.Pi*0.85)
		dc.Stroke()
		dc.SetRGB(1, 1, 1)
		dc.SetLineWidth(2 * scale)
		dc.DrawArc(cx, mouthY-headH*0.04, headW*0.16, math.Pi*0.25, math.Pi*0.75)
		dc.Stroke()
	}

	dc.SetColor(skinShadow)
	dc.DrawEllipse(cx, cy+headH*0.02, headW*0.06, headH*0.04)
	dc.Fill()
}

func drakeDrawBody(dc *gg.Context, cx, cy, scale float64, reject bool) {
	jacket := color.RGBA{0xd9, 0xb3, 0x6b, 0xff}
	jacketShadow := color.RGBA{0xa8, 0x82, 0x42, 0xff}
	shirt := color.RGBA{0xf2, 0xe7, 0xd0, 0xff}
	pants := color.RGBA{0x2b, 0x1f, 0x12, 0xff}

	bodyW := 260.0 * scale
	bodyH := 320.0 * scale

	dc.SetColor(pants)
	dc.DrawRectangle(cx-bodyW*0.35, cy+bodyH*0.30, bodyW*0.70, bodyH*0.30)
	dc.Fill()

	dc.SetColor(jacket)
	dc.DrawRoundedRectangle(cx-bodyW/2, cy-bodyH*0.10, bodyW, bodyH*0.55, 30*scale)
	dc.Fill()

	dc.SetColor(shirt)
	dc.DrawRectangle(cx-bodyW*0.08, cy-bodyH*0.10, bodyW*0.16, bodyH*0.30)
	dc.Fill()

	dc.SetColor(jacketShadow)
	dc.SetLineWidth(4 * scale)
	dc.DrawLine(cx-bodyW*0.08, cy-bodyH*0.10, cx-bodyW*0.08, cy+bodyH*0.20)
	dc.Stroke()
	dc.DrawLine(cx+bodyW*0.08, cy-bodyH*0.10, cx+bodyW*0.08, cy+bodyH*0.20)
	dc.Stroke()

	skin := color.RGBA{0x6e, 0x47, 0x32, 0xff}
	if reject {
		dc.SetColor(jacket)
		dc.DrawRoundedRectangle(cx-bodyW*0.78, cy-bodyH*0.18, bodyW*0.30, bodyH*0.50, 25*scale)
		dc.Fill()
		dc.SetColor(skin)
		dc.DrawCircle(cx-bodyW*0.63, cy-bodyH*0.30, 28*scale)
		dc.Fill()
		dc.SetRGB(0.05, 0.05, 0.05)
		dc.SetLineWidth(7 * scale)
		dc.DrawArc(cx-bodyW*0.63, cy-bodyH*0.30, 18*scale, math.Pi*1.1, math.Pi*1.9)
		dc.Stroke()

		dc.SetColor(jacket)
		dc.DrawRoundedRectangle(cx+bodyW*0.48, cy-bodyH*0.18, bodyW*0.30, bodyH*0.50, 25*scale)
		dc.Fill()
		dc.SetColor(skin)
		dc.DrawCircle(cx+bodyW*0.63, cy-bodyH*0.30, 28*scale)
		dc.Fill()
		dc.SetRGB(0.05, 0.05, 0.05)
		dc.SetLineWidth(7 * scale)
		dc.DrawArc(cx+bodyW*0.63, cy-bodyH*0.30, 18*scale, math.Pi*1.1, math.Pi*1.9)
		dc.Stroke()
	} else {
		dc.SetColor(jacket)
		dc.DrawRoundedRectangle(cx-bodyW*0.85, cy-bodyH*0.05, bodyW*0.32, bodyH*0.50, 25*scale)
		dc.Fill()
		dc.SetColor(skin)
		dc.DrawCircle(cx-bodyW*0.72, cy+bodyH*0.30, 30*scale)
		dc.Fill()
		dc.SetLineWidth(3 * scale)
		dc.SetRGB(0.30, 0.18, 0.10)
		for i := 0; i < 4; i++ {
			ang := float64(i)*0.25 + 0.1
			x1 := cx - bodyW*0.72 + math.Cos(ang)*18*scale
			y1 := cy + bodyH*0.30 + math.Sin(ang)*18*scale
			x2 := cx - bodyW*0.72 + math.Cos(ang)*30*scale
			y2 := cy + bodyH*0.30 + math.Sin(ang)*30*scale
			dc.DrawLine(x1, y1, x2, y2)
			dc.Stroke()
		}

		dc.SetColor(jacket)
		dc.DrawRoundedRectangle(cx+bodyW*0.53, cy-bodyH*0.05, bodyW*0.32, bodyH*0.50, 25*scale)
		dc.Fill()
		dc.SetColor(skin)
		dc.DrawCircle(cx+bodyW*0.72, cy+bodyH*0.30, 30*scale)
		dc.Fill()
		for i := 0; i < 4; i++ {
			ang := math.Pi - float64(i)*0.25 - 0.1
			x1 := cx + bodyW*0.72 + math.Cos(ang)*18*scale
			y1 := cy + bodyH*0.30 + math.Sin(ang)*18*scale
			x2 := cx + bodyW*0.72 + math.Cos(ang)*30*scale
			y2 := cy + bodyH*0.30 + math.Sin(ang)*30*scale
			dc.DrawLine(x1, y1, x2, y2)
			dc.Stroke()
		}
	}
}

func drakeDrawPanel(dc *gg.Context, x, y, w, h float64, reject bool) {
	bg := color.RGBA{0xee, 0xea, 0xdc, 0xff}
	dc.SetColor(bg)
	dc.DrawRectangle(x, y, w, h)
	dc.Fill()

	for i := 0; i < 40; i++ {
		px := x + drakeRng.Float64()*w
		py := y + drakeRng.Float64()*h
		dc.SetRGBA(0, 0, 0, 0.04+drakeRng.Float64()*0.05)
		dc.DrawCircle(px, py, 1.5+drakeRng.Float64()*2)
		dc.Fill()
	}

	cx := x + w*0.40
	cy := y + h*0.55
	scale := h / 580.0
	if scale > 0.85 {
		scale = 0.85
	}
	if scale < 0.45 {
		scale = 0.45
	}

	drakeDrawBody(dc, cx, cy+60*scale, scale, reject)
	drakeDrawHead(dc, cx, cy-90*scale, scale, reject)

	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(4)
	dc.DrawRectangle(x, y, w, h)
	dc.Stroke()
}

func drakeRenderMeme(rejectText, approveText string) (string, error) {
	const W, H = 1000, 1100
	dc := gg.NewContext(W, H)

	dc.SetRGB(1, 1, 1)
	dc.Clear()

	panelW := float64(W) / 2
	panelH := float64(H) / 2

	drakeDrawPanel(dc, 0, 0, panelW, panelH, true)
	drakeDrawPanel(dc, 0, panelH, panelW, panelH, false)

	textBgTop := color.RGBA{0xfa, 0xf6, 0xe8, 0xff}
	textBgBot := color.RGBA{0xfa, 0xf6, 0xe8, 0xff}
	dc.SetColor(textBgTop)
	dc.DrawRectangle(panelW, 0, panelW, panelH)
	dc.Fill()
	dc.SetColor(textBgBot)
	dc.DrawRectangle(panelW, panelH, panelW, panelH)
	dc.Fill()

	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(4)
	dc.DrawRectangle(panelW, 0, panelW, panelH)
	dc.Stroke()
	dc.DrawRectangle(panelW, panelH, panelW, panelH)
	dc.Stroke()

	if rejectText == "" {
		rejectText = "REJECT TEXT"
	}
	if approveText == "" {
		approveText = "APPROVE TEXT"
	}

	drakeDrawText(dc, rejectText, panelW, 0, panelW, panelH)
	drakeDrawText(dc, approveText, panelW, panelH, panelW, panelH)

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("drake_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func DrakeHandler(m *tg.NewMessage) error {
	raw := strings.TrimSpace(m.Args())
	var rejectText, approveText string

	if raw != "" {
		parts := strings.SplitN(raw, "|", 2)
		rejectText = strings.TrimSpace(parts[0])
		if len(parts) == 2 {
			approveText = strings.TrimSpace(parts[1])
		}
		if len(rejectText) > 180 {
			rejectText = rejectText[:180]
		}
		if len(approveText) > 180 {
			approveText = approveText[:180]
		}
	}

	if raw != "" && (rejectText == "" || approveText == "") {
		m.Reply("<b>Drake Meme</b>\n\n<b>Usage:</b>\n<code>/drake</code> - blank template\n<code>/drake reject_text | approve_text</code>\n\nExample: <code>/drake writing comments | no comments</code>")
		return nil
	}

	status, _ := m.Reply("<i>generating drake meme...</i>")

	outPath, err := drakeRenderMeme(strings.ToUpper(rejectText), strings.ToUpper(approveText))
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
		FileName: "drake.png",
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

func registerFaceSwapMemeHandlers() {
	c := Client
	c.On("cmd:drake", DrakeHandler)
}

func init() {
	QueueHandlerRegistration(registerFaceSwapMemeHandlers)
}
