package modules

import (
	"fmt"
	"hash/fnv"
	"html"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

func placeholderParseDims(arg string) (int, int, error) {
	arg = strings.ToLower(strings.TrimSpace(arg))
	if arg == "" {
		return 0, 0, fmt.Errorf("missing dimensions")
	}
	var sep string
	switch {
	case strings.Contains(arg, "x"):
		sep = "x"
	case strings.Contains(arg, "*"):
		sep = "*"
	case strings.Contains(arg, "X"):
		sep = "X"
	default:
		return 0, 0, fmt.Errorf("invalid format, use WxH")
	}
	parts := strings.SplitN(arg, sep, 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format, use WxH")
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width")
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height")
	}
	if w < 100 || w > 2000 || h < 100 || h > 2000 {
		return 0, 0, fmt.Errorf("dimensions must be between 100 and 2000")
	}
	return w, h, nil
}

func placeholderGradientColors(w, h int) (color.RGBA, color.RGBA) {
	h64 := fnv.New64a()
	h64.Write([]byte(fmt.Sprintf("%dx%d|%d", w, h, time.Now().UnixNano())))
	seed := h64.Sum64()
	palettes := [][2]color.RGBA{
		{{0x66, 0x7e, 0xea, 0xff}, {0x76, 0x4b, 0xa2, 0xff}},
		{{0xf0, 0x93, 0x00, 0xff}, {0xf2, 0x37, 0x67, 0xff}},
		{{0x43, 0xe9, 0x7d, 0xff}, {0x38, 0xf9, 0xd7, 0xff}},
		{{0xfa, 0x70, 0x9a, 0xff}, {0xfe, 0xe1, 0x40, 0xff}},
		{{0x30, 0xcf, 0xd0, 0xff}, {0x33, 0x08, 0x67, 0xff}},
		{{0xff, 0x6e, 0x7f, 0xff}, {0xbf, 0xe9, 0xff, 0xff}},
		{{0x21, 0xd4, 0xfd, 0xff}, {0xb7, 0x21, 0xff, 0xff}},
		{{0x08, 0xae, 0xa4, 0xff}, {0xdf, 0x9f, 0x1c, 0xff}},
		{{0xee, 0x09, 0x79, 0xff}, {0xff, 0x66, 0x00, 0xff}},
		{{0x00, 0xc6, 0xff, 0xff}, {0x00, 0x72, 0xff, 0xff}},
		{{0xfc, 0x46, 0x6b, 0xff}, {0x3f, 0x5e, 0xfb, 0xff}},
		{{0x4f, 0xac, 0xfe, 0xff}, {0x00, 0xf2, 0xfe, 0xff}},
	}
	return palettes[int(seed%uint64(len(palettes)))][0], palettes[int((seed>>16)%uint64(len(palettes)))][1]
}

func placeholderDrawGradient(dc *gg.Context, w, h int, a, b color.RGBA) {
	diag := float64(w + h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(x+y) / diag
			t = t*t*(3-2*t)
			r := float64(a.R)*(1-t) + float64(b.R)*t
			g := float64(a.G)*(1-t) + float64(b.G)*t
			bb := float64(a.B)*(1-t) + float64(b.B)*t
			dc.SetRGB255(int(r), int(g), int(bb))
			dc.SetPixel(x, y)
		}
	}
}

func placeholderFontPath(name string) string {
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

func placeholderLoadFont(dc *gg.Context, size float64) {
	p := placeholderFontPath(getRandomFont())
	if p != "" {
		if err := dc.LoadFontFace(p, size); err == nil {
			return
		}
	}
	p = placeholderFontPath("Inter_28pt-Bold.ttf")
	if p != "" {
		if err := dc.LoadFontFace(p, size); err == nil {
			return
		}
	}
	dc.SetFontFace(basicfont.Face7x13)
}

func placeholderRender(w, h int) (string, error) {
	dc := gg.NewContext(w, h)
	a, b := placeholderGradientColors(w, h)
	placeholderDrawGradient(dc, w, h, a, b)

	minDim := w
	if h < minDim {
		minDim = h
	}

	pad := float64(minDim) * 0.04
	dc.SetRGBA(1, 1, 1, 0.18)
	dc.SetLineWidth(float64(minDim) * 0.006)
	dc.DrawRectangle(pad, pad, float64(w)-2*pad, float64(h)-2*pad)
	dc.Stroke()

	label := fmt.Sprintf("%dx%d", w, h)

	fontSize := float64(minDim) * 0.18
	if fontSize < 18 {
		fontSize = 18
	}
	placeholderLoadFont(dc, fontSize)

	cx := float64(w) / 2
	cy := float64(h) / 2

	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawStringAnchored(label, cx+3, cy+3, 0.5, 0.5)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(label, cx, cy, 0.5, 0.5)

	out := filepath.Join(os.TempDir(), fmt.Sprintf("placeholder_%dx%d_%d.png", w, h, time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func PlaceholderHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("usage: <code>/placeholder &lt;WxH&gt;</code>\nexample: <code>/placeholder 800x600</code>\ndims 100-2000")
		return nil
	}
	w, h, err := placeholderParseDims(arg)
	if err != nil {
		m.Reply("error: " + html.EscapeString(err.Error()))
		return nil
	}

	status, _ := m.Reply(fmt.Sprintf("<code>generating %dx%d placeholder...</code>", w, h))

	outPath, err := placeholderRender(w, h)
	if err != nil {
		msg := "render failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	caption := fmt.Sprintf("<b>Placeholder</b>  <code>%dx%d</code>", w, h)
	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		Caption:  caption,
		FileName: fmt.Sprintf("placeholder_%dx%d.png", w, h),
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

func registerPlaceholderHandlers() {
	c := Client
	c.On("cmd:placeholder", PlaceholderHandler)
}

func init() { QueueHandlerRegistration(registerPlaceholderHandlers) }
