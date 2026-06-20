package modules

import (
	"fmt"
	"html"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

func paletteParseHex(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	s = strings.ToLower(s)
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid hex length")
	}
	r, err := strconv.ParseUint(s[0:2], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	g, err := strconv.ParseUint(s[2:4], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	b, err := strconv.ParseUint(s[4:6], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), 0xff}, nil
}

func paletteRGBToHSL(c color.RGBA) (float64, float64, float64) {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l := (max + min) / 2.0
	if max == min {
		return 0, 0, l
	}
	d := max - min
	var s float64
	if l > 0.5 {
		s = d / (2.0 - max - min)
	} else {
		s = d / (max + min)
	}
	var h float64
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h /= 6
	return h * 360, s, l
}

func paletteHueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 0.5 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func paletteHSLToRGB(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	hn := h / 360.0
	if s == 0 {
		v := uint8(math.Round(l * 255))
		return color.RGBA{v, v, v, 0xff}
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	r := paletteHueToRGB(p, q, hn+1.0/3.0)
	g := paletteHueToRGB(p, q, hn)
	b := paletteHueToRGB(p, q, hn-1.0/3.0)
	return color.RGBA{
		uint8(math.Round(r * 255)),
		uint8(math.Round(g * 255)),
		uint8(math.Round(b * 255)),
		0xff,
	}
}

func paletteHexFromRGB(c color.RGBA) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func paletteShiftHue(base color.RGBA, deg float64) color.RGBA {
	h, s, l := paletteRGBToHSL(base)
	return paletteHSLToRGB(h+deg, s, l)
}

func paletteRelativeLuminance(c color.RGBA) float64 {
	chan_ := func(v uint8) float64 {
		x := float64(v) / 255.0
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*chan_(c.R) + 0.7152*chan_(c.G) + 0.0722*chan_(c.B)
}

func paletteTextColorFor(bg color.RGBA) color.RGBA {
	if paletteRelativeLuminance(bg) > 0.5 {
		return color.RGBA{0x14, 0x14, 0x18, 0xff}
	}
	return color.RGBA{0xff, 0xff, 0xff, 0xff}
}

type paletteSwatch struct {
	Color color.RGBA
	Label string
}

func paletteBuild(base color.RGBA) []paletteSwatch {
	return []paletteSwatch{
		{paletteShiftHue(base, -30), "ANALOG"},
		{paletteShiftHue(base, 120), "TRIAD"},
		{base, "INPUT"},
		{paletteShiftHue(base, 180), "COMPL"},
		{paletteShiftHue(base, 90), "TETRA"},
		{paletteShiftHue(base, 30), "ANALOG"},
	}
}

func paletteLoadFont(dc *gg.Context, size float64) {
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

func paletteRenderImage(base color.RGBA) (string, error) {
	const W, H = 1024, 256
	dc := gg.NewContext(W, H)

	dc.SetRGB(0.06, 0.06, 0.08)
	dc.Clear()

	swatches := paletteBuild(base)
	n := len(swatches)
	blockW := float64(W) / float64(n)

	for i, sw := range swatches {
		x := float64(i) * blockW
		dc.SetRGBA255(int(sw.Color.R), int(sw.Color.G), int(sw.Color.B), 255)
		dc.DrawRectangle(x, 0, blockW, float64(H))
		dc.Fill()

		isInput := sw.Label == "INPUT"
		if isInput {
			accent := paletteTextColorFor(sw.Color)
			dc.SetRGBA255(int(accent.R), int(accent.G), int(accent.B), 220)
			dc.SetLineWidth(3)
			dc.DrawRectangle(x+6, 6, blockW-12, float64(H)-12)
			dc.Stroke()
		}

		textCol := paletteTextColorFor(sw.Color)
		dc.SetRGBA255(int(textCol.R), int(textCol.G), int(textCol.B), 255)

		paletteLoadFont(dc, 20)
		dc.DrawStringAnchored(sw.Label, x+blockW/2, float64(H)/2-26, 0.5, 0.5)

		paletteLoadFont(dc, 26)
		dc.DrawStringAnchored(paletteHexFromRGB(sw.Color), x+blockW/2, float64(H)/2+8, 0.5, 0.5)

		paletteLoadFont(dc, 14)
		r, g, b := sw.Color.R, sw.Color.G, sw.Color.B
		dc.DrawStringAnchored(fmt.Sprintf("rgb %d %d %d", r, g, b), x+blockW/2, float64(H)/2+40, 0.5, 0.5)
	}

	out := filepath.Join(os.TempDir(), fmt.Sprintf("palette_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func PaletteHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("usage: <code>/palette &lt;hex&gt;</code>\nexample: <code>/palette #ff6a3d</code>")
		return nil
	}
	token := strings.Fields(arg)[0]
	base, err := paletteParseHex(token)
	if err != nil {
		m.Reply("invalid hex color: " + html.EscapeString(token))
		return nil
	}

	status, _ := m.Reply("<code>building palette...</code>")

	out, err := paletteRenderImage(base)
	if err != nil {
		msg := "failed to render: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	swatches := paletteBuild(base)
	var b strings.Builder
	b.WriteString("<b>Color Palette</b>\n")
	b.WriteString(fmt.Sprintf("input  <code>%s</code>\n", paletteHexFromRGB(base)))
	for _, sw := range swatches {
		if sw.Label == "INPUT" {
			continue
		}
		b.WriteString(fmt.Sprintf("%s  <code>%s</code>\n", strings.ToLower(sw.Label), paletteHexFromRGB(sw.Color)))
	}

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  b.String(),
		FileName: "palette.png",
		MimeType: "image/png",
	})
	os.Remove(out)
	if merr != nil {
		m.Reply("upload failed: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerPaletteHandlers() {
	c := Client
	c.On("cmd:palette", PaletteHandler)
}

func init() {
	QueueHandlerRegistration(registerPaletteHandlers)
}
