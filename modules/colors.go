package modules

import (
	"fmt"
	"html"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

var namedColors = map[string]string{
	"aliceblue":            "#F0F8FF",
	"antiquewhite":         "#FAEBD7",
	"aqua":                 "#00FFFF",
	"aquamarine":           "#7FFFD4",
	"azure":                "#F0FFFF",
	"beige":                "#F5F5DC",
	"bisque":               "#FFE4C4",
	"black":                "#000000",
	"blanchedalmond":       "#FFEBCD",
	"blue":                 "#0000FF",
	"blueviolet":           "#8A2BE2",
	"brown":                "#A52A2A",
	"burlywood":            "#DEB887",
	"cadetblue":            "#5F9EA0",
	"chartreuse":           "#7FFF00",
	"chocolate":            "#D2691E",
	"coral":                "#FF7F50",
	"cornflowerblue":       "#6495ED",
	"cornsilk":             "#FFF8DC",
	"crimson":              "#DC143C",
	"cyan":                 "#00FFFF",
	"darkblue":             "#00008B",
	"darkcyan":             "#008B8B",
	"darkgoldenrod":        "#B8860B",
	"darkgray":             "#A9A9A9",
	"darkgreen":            "#006400",
	"darkkhaki":            "#BDB76B",
	"darkmagenta":          "#8B008B",
	"darkolivegreen":       "#556B2F",
	"darkorange":           "#FF8C00",
	"darkorchid":           "#9932CC",
	"darkred":              "#8B0000",
	"darksalmon":           "#E9967A",
	"darkseagreen":         "#8FBC8F",
	"darkslateblue":        "#483D8B",
	"darkslategray":        "#2F4F4F",
	"darkturquoise":        "#00CED1",
	"darkviolet":           "#9400D3",
	"deeppink":             "#FF1493",
	"deepskyblue":          "#00BFFF",
	"dimgray":              "#696969",
	"dodgerblue":           "#1E90FF",
	"firebrick":            "#B22222",
	"floralwhite":          "#FFFAF0",
	"forestgreen":          "#228B22",
	"fuchsia":              "#FF00FF",
	"gainsboro":            "#DCDCDC",
	"ghostwhite":           "#F8F8FF",
	"gold":                 "#FFD700",
	"goldenrod":            "#DAA520",
	"gray":                 "#808080",
	"green":                "#008000",
	"greenyellow":          "#ADFF2F",
	"honeydew":             "#F0FFF0",
	"hotpink":              "#FF69B4",
	"indianred":            "#CD5C5C",
	"indigo":               "#4B0082",
	"ivory":                "#FFFFF0",
	"khaki":                "#F0E68C",
	"lavender":             "#E6E6FA",
	"lavenderblush":        "#FFF0F5",
	"lawngreen":            "#7CFC00",
	"lemonchiffon":         "#FFFACD",
	"lightblue":            "#ADD8E6",
	"lightcoral":           "#F08080",
	"lightcyan":            "#E0FFFF",
	"lightgoldenrodyellow": "#FAFAD2",
	"lightgray":            "#D3D3D3",
	"lightgreen":           "#90EE90",
	"lightpink":            "#FFB6C1",
	"lightsalmon":          "#FFA07A",
	"lightseagreen":        "#20B2AA",
	"lightskyblue":         "#87CEFA",
	"lightslategray":       "#778899",
	"lightsteelblue":       "#B0C4DE",
	"lightyellow":          "#FFFFE0",
	"lime":                 "#00FF00",
	"limegreen":            "#32CD32",
	"linen":                "#FAF0E6",
	"magenta":              "#FF00FF",
	"maroon":               "#800000",
	"mediumaquamarine":     "#66CDAA",
	"mediumblue":           "#0000CD",
	"mediumorchid":         "#BA55D3",
	"mediumpurple":         "#9370DB",
	"mediumseagreen":       "#3CB371",
	"mediumslateblue":      "#7B68EE",
	"mediumspringgreen":    "#00FA9A",
	"mediumturquoise":      "#48D1CC",
	"mediumvioletred":      "#C71585",
	"midnightblue":         "#191970",
	"mintcream":            "#F5FFFA",
	"mistyrose":            "#FFE4E1",
	"moccasin":             "#FFE4B5",
	"navajowhite":          "#FFDEAD",
	"navy":                 "#000080",
	"oldlace":              "#FDF5E6",
	"olive":                "#808000",
	"olivedrab":            "#6B8E23",
	"orange":                "#FFA500",
	"orangered":            "#FF4500",
	"orchid":               "#DA70D6",
	"palegoldenrod":        "#EEE8AA",
	"palegreen":            "#98FB98",
	"paleturquoise":        "#AFEEEE",
	"palevioletred":        "#DB7093",
	"papayawhip":           "#FFEFD5",
	"peachpuff":            "#FFDAB9",
	"peru":                 "#CD853F",
	"pink":                 "#FFC0CB",
	"plum":                 "#DDA0DD",
	"powderblue":           "#B0E0E6",
	"purple":               "#800080",
	"rebeccapurple":        "#663399",
	"red":                  "#FF0000",
	"rosybrown":            "#BC8F8F",
	"royalblue":            "#4169E1",
	"saddlebrown":          "#8B4513",
	"salmon":               "#FA8072",
	"sandybrown":           "#F4A460",
	"seagreen":             "#2E8B57",
	"seashell":             "#FFF5EE",
	"sienna":               "#A0522D",
	"silver":               "#C0C0C0",
	"skyblue":              "#87CEEB",
	"slateblue":            "#6A5ACD",
	"slategray":            "#708090",
	"snow":                 "#FFFAFA",
	"springgreen":          "#00FF7F",
	"steelblue":            "#4682B4",
	"tan":                  "#D2B48C",
	"teal":                 "#008080",
	"thistle":              "#D8BFD8",
	"tomato":               "#FF6347",
	"turquoise":            "#40E0D0",
	"violet":               "#EE82EE",
	"wheat":                "#F5DEB3",
	"white":                "#FFFFFF",
	"whitesmoke":           "#F5F5F5",
	"yellow":               "#FFFF00",
	"yellowgreen":          "#9ACD32",
}

func colorsParseHex(s string) (color.RGBA, error) {
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

func colorsResolve(token string) (color.RGBA, string, error) {
	t := strings.TrimSpace(token)
	if t == "" {
		return color.RGBA{}, "", fmt.Errorf("empty input")
	}
	lower := strings.ToLower(t)
	lower = strings.ReplaceAll(lower, " ", "")
	if hex, ok := namedColors[lower]; ok {
		c, err := colorsParseHex(hex)
		if err != nil {
			return color.RGBA{}, "", err
		}
		return c, lower, nil
	}
	c, err := colorsParseHex(t)
	if err != nil {
		return color.RGBA{}, "", fmt.Errorf("unknown color: %s", t)
	}
	return c, "", nil
}

func colorsRGBToHSL(c color.RGBA) (float64, float64, float64) {
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
	return h * 360, s * 100, l * 100
}

func colorsHexFromRGB(c color.RGBA) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func colorsRelativeLuminance(c color.RGBA) float64 {
	chx := func(v uint8) float64 {
		x := float64(v) / 255.0
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*chx(c.R) + 0.7152*chx(c.G) + 0.0722*chx(c.B)
}

func colorsTextColorFor(bg color.RGBA) color.RGBA {
	if colorsRelativeLuminance(bg) > 0.5 {
		return color.RGBA{0x14, 0x14, 0x18, 0xff}
	}
	return color.RGBA{0xff, 0xff, 0xff, 0xff}
}

func colorsLoadFont(dc *gg.Context, size float64) {
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

func colorsRenderSwatch(c color.RGBA) (string, error) {
	const W, H = 200, 200
	dc := gg.NewContext(W, H)
	dc.SetRGBA255(int(c.R), int(c.G), int(c.B), 255)
	dc.Clear()

	hex := colorsHexFromRGB(c)
	tc := colorsTextColorFor(c)

	dc.SetRGBA255(int(tc.R), int(tc.G), int(tc.B), 230)
	dc.SetLineWidth(2)
	dc.DrawRectangle(8, 8, W-16, H-16)
	dc.Stroke()

	dc.SetRGBA255(int(tc.R), int(tc.G), int(tc.B), 255)
	colorsLoadFont(dc, 28)
	dc.DrawStringAnchored(hex, W/2, H/2-8, 0.5, 0.5)

	colorsLoadFont(dc, 14)
	dc.DrawStringAnchored(fmt.Sprintf("rgb %d %d %d", c.R, c.G, c.B), W/2, H/2+22, 0.5, 0.5)

	out := filepath.Join(os.TempDir(), fmt.Sprintf("color_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func colorsDistance(a, b color.RGBA) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

type colorMatch struct {
	Name string
	Hex  string
	Dist float64
}

func colorsNearest(target color.RGBA, n int) []colorMatch {
	matches := make([]colorMatch, 0, len(namedColors))
	for name, hex := range namedColors {
		c, err := colorsParseHex(hex)
		if err != nil {
			continue
		}
		matches = append(matches, colorMatch{
			Name: name,
			Hex:  hex,
			Dist: colorsDistance(target, c),
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Dist < matches[j].Dist
	})
	if len(matches) > n {
		matches = matches[:n]
	}
	return matches
}

func ColorHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("usage: <code>/color &lt;name_or_hex&gt;</code>\nexamples: <code>/color red</code> or <code>/color #FF0000</code>")
		return nil
	}
	token := strings.Fields(arg)[0]
	c, name, err := colorsResolve(token)
	if err != nil {
		m.Reply("invalid color: " + html.EscapeString(token))
		return nil
	}

	status, _ := m.Reply("<code>rendering swatch...</code>")

	out, err := colorsRenderSwatch(c)
	if err != nil {
		msg := "failed to render: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	h, s, l := colorsRGBToHSL(c)
	hex := colorsHexFromRGB(c)

	var b strings.Builder
	b.WriteString("<b>Color</b>\n")
	if name != "" {
		b.WriteString(fmt.Sprintf("name  <code>%s</code>\n", html.EscapeString(name)))
	}
	b.WriteString(fmt.Sprintf("hex   <code>%s</code>\n", hex))
	b.WriteString(fmt.Sprintf("rgb   <code>%d, %d, %d</code>\n", c.R, c.G, c.B))
	b.WriteString(fmt.Sprintf("hsl   <code>%.0f, %.0f%%, %.0f%%</code>", h, s, l))

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  b.String(),
		FileName: "color.png",
		MimeType: "image/png",
	})
	os.Remove(out)
	if merr != nil {
		m.Reply("upload failed: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func ColorsHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("usage: <code>/colors &lt;hex&gt;</code>\nexample: <code>/colors #ff6a3d</code>")
		return nil
	}
	token := strings.Fields(arg)[0]
	c, err := colorsParseHex(token)
	if err != nil {
		m.Reply("invalid hex color: " + html.EscapeString(token))
		return nil
	}

	matches := colorsNearest(c, 8)
	if len(matches) == 0 {
		m.Reply("no matches found")
		return nil
	}

	var b strings.Builder
	b.WriteString("<b>Nearest Named Colors</b>\n")
	b.WriteString(fmt.Sprintf("input  <code>%s</code>\n\n", colorsHexFromRGB(c)))
	for i, mt := range matches {
		b.WriteString(fmt.Sprintf("%d. <b>%s</b>  <code>%s</code>  Δ %.0f\n",
			i+1, html.EscapeString(mt.Name), mt.Hex, mt.Dist))
	}
	m.Reply(b.String())
	return nil
}

func registerColorsHandlers() {
	c := Client
	c.On("cmd:color", ColorHandler)
	c.On("cmd:colors", ColorsHandler)
}

func init() {
	QueueHandlerRegistration(registerColorsHandlers)
}
