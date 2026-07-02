package extras

import (
	"fmt"
	"hash/fnv"
	"html"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	modules "main/modules"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

// === from image_crop.go ===
func circleCropApplyMask(src image.Image) image.Image {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	size := w
	if h < size {
		size = h
	}
	if size <= 0 {
		return src
	}

	offX := bounds.Min.X + (w-size)/2
	offY := bounds.Min.Y + (h-size)/2
	squareRect := image.Rect(offX, offY, offX+size, offY+size)

	square := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(square, square.Bounds(), src, squareRect.Min, draw.Src)

	dc := gg.NewContext(size, size)
	dc.DrawCircle(float64(size)/2, float64(size)/2, float64(size)/2)
	dc.Clip()
	dc.DrawImage(square, 0, 0)
	return dc.Image()
}

func circleCropDecodeFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func circleCropSavePNG(img image.Image, outPath string) error {
	return gg.NewContextForImage(img).SavePNG(outPath)
}

func CircleCropHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("reply to a photo with <code>/circle</code> to crop it into a circle")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("error: " + html.EscapeString(err.Error()))
		return nil
	}

	if r.Photo() == nil {
		m.Reply("the replied message is not a photo")
		return nil
	}

	status, _ := m.Reply("<code>cropping into circle...</code>")

	inPath := filepath.Join(os.TempDir(), fmt.Sprintf("circle_in_%d.jpg", time.Now().UnixNano()))
	_, err = m.Client.DownloadMedia(r.Media(), &tg.DownloadOptions{FileName: inPath})
	if err != nil {
		msg := "error downloading photo: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(inPath)

	src, err := circleCropDecodeFile(inPath)
	if err != nil {
		msg := "error decoding image: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	b := src.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		msg := "invalid image dimensions"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	out := circleCropApplyMask(src)

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("circle_out_%d.png", time.Now().UnixNano()))
	if err := circleCropSavePNG(out, outPath); err != nil {
		msg := "error saving png: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(outPath)

	side := int(math.Min(float64(b.Dx()), float64(b.Dy())))
	caption := fmt.Sprintf("<b>Circle Crop</b>\n<b>Size:</b> <code>%dx%d</code>", side, side)

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		Caption:  caption,
		FileName: "circle.png",
		MimeType: "image/png",
	})
	if merr != nil {
		m.Reply("upload failed: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerImageCropHandlers() {
	c := modules.Client
	c.On("cmd:circle", CircleCropHandler)
}

func initFromSrc_image_crop_0_1() { modules.QueueHandlerRegistration(registerImageCropHandlers) }

// === from colors.go ===
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
	"orange":               "#FFA500",
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
	name := modules.GetRandomFont()
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
	c := modules.Client
	c.On("cmd:color", ColorHandler)
	c.On("cmd:colors", ColorsHandler)
}

func initFromSrc_colors_1_1() {
	modules.QueueHandlerRegistration(registerColorsHandlers)
}

// === from wallpaper.go ===
func WallpaperHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("usage: <code>/wallpaper &lt;query&gt;</code> or <code>/wallpaper random</code>")
		return nil
	}

	var endpoint string
	var label string
	if strings.EqualFold(query, "random") {
		endpoint = fmt.Sprintf("https://picsum.photos/1280/720?rand=%d", time.Now().UnixNano())
		label = "random"
	} else {
		endpoint = "https://picsum.photos/seed/" + url.PathEscape(query) + "/1280/720"
		label = query
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		m.Reply("error: " + html.EscapeString(err.Error()))
		return nil
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.Reply("error fetching wallpaper: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.Reply(fmt.Sprintf("wallpaper api returned status %d", resp.StatusCode))
		return nil
	}

	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("wallpaper_%d.jpg", time.Now().UnixNano()))
	out, err := os.Create(tmpPath)
	if err != nil {
		m.Reply("error creating temp file: " + html.EscapeString(err.Error()))
		return nil
	}

	if _, err := io.Copy(out, io.LimitReader(resp.Body, 15*1024*1024)); err != nil {
		out.Close()
		os.Remove(tmpPath)
		m.Reply("error writing wallpaper: " + html.EscapeString(err.Error()))
		return nil
	}
	out.Close()
	defer os.Remove(tmpPath)

	caption := "<b>Wallpaper</b>\n<code>" + html.EscapeString(label) + "</code>"
	_, err = m.ReplyMedia(tmpPath, &tg.MediaOptions{Caption: caption})
	if err != nil {
		m.Reply("error sending wallpaper: " + html.EscapeString(err.Error()))
		return nil
	}
	return nil
}

func initFromSrc_wallpaper_2_1() { modules.QueueHandlerRegistration(registerWallpaperHandlers) }

func registerWallpaperHandlers() {
	c := modules.Client
	c.On("cmd:wallpaper", WallpaperHandler)
}

// === from profile_cards.go ===
type pcardPalette struct {
	BgTop   color.RGBA
	BgMid   color.RGBA
	BgBot   color.RGBA
	Accent  color.RGBA
	Accent2 color.RGBA
	Surface color.RGBA
}

type pcardTarget struct {
	UserID    int64
	FirstName string
	LastName  string
	Username  string
	Photo     tg.UserProfilePhoto
}

var pcardPalettes = []pcardPalette{
	{color.RGBA{0x0b, 0x0f, 0x1c, 0xff}, color.RGBA{0x14, 0x18, 0x2c, 0xff}, color.RGBA{0x1d, 0x22, 0x3d, 0xff}, color.RGBA{0x7c, 0x3a, 0xed, 0xff}, color.RGBA{0xc0, 0x84, 0xfc, 0xff}, color.RGBA{0x1a, 0x1f, 0x35, 0xff}},
	{color.RGBA{0x0a, 0x1a, 0x14, 0xff}, color.RGBA{0x0e, 0x26, 0x1f, 0xff}, color.RGBA{0x13, 0x33, 0x2a, 0xff}, color.RGBA{0x10, 0xb9, 0x81, 0xff}, color.RGBA{0x6e, 0xe7, 0xb7, 0xff}, color.RGBA{0x10, 0x28, 0x22, 0xff}},
	{color.RGBA{0x1a, 0x07, 0x0c, 0xff}, color.RGBA{0x2c, 0x0c, 0x18, 0xff}, color.RGBA{0x40, 0x12, 0x24, 0xff}, color.RGBA{0xf4, 0x3f, 0x5e, 0xff}, color.RGBA{0xfd, 0xa4, 0xaf, 0xff}, color.RGBA{0x33, 0x10, 0x1d, 0xff}},
	{color.RGBA{0x08, 0x10, 0x22, 0xff}, color.RGBA{0x0d, 0x1b, 0x36, 0xff}, color.RGBA{0x12, 0x26, 0x4c, 0xff}, color.RGBA{0x38, 0xbd, 0xf8, 0xff}, color.RGBA{0xa5, 0xe5, 0xfd, 0xff}, color.RGBA{0x12, 0x21, 0x40, 0xff}},
	{color.RGBA{0x1b, 0x10, 0x05, 0xff}, color.RGBA{0x2c, 0x1a, 0x09, 0xff}, color.RGBA{0x40, 0x26, 0x0d, 0xff}, color.RGBA{0xf5, 0x9e, 0x0b, 0xff}, color.RGBA{0xfc, 0xd3, 0x4d, 0xff}, color.RGBA{0x33, 0x21, 0x0d, 0xff}},
	{color.RGBA{0x12, 0x09, 0x1f, 0xff}, color.RGBA{0x1f, 0x10, 0x33, 0xff}, color.RGBA{0x32, 0x18, 0x51, 0xff}, color.RGBA{0xd9, 0x46, 0xef, 0xff}, color.RGBA{0xf0, 0xab, 0xfc, 0xff}, color.RGBA{0x29, 0x14, 0x45, 0xff}},
	{color.RGBA{0x05, 0x14, 0x1b, 0xff}, color.RGBA{0x08, 0x22, 0x2d, 0xff}, color.RGBA{0x0c, 0x33, 0x42, 0xff}, color.RGBA{0x06, 0xb6, 0xd4, 0xff}, color.RGBA{0x67, 0xe8, 0xf9, 0xff}, color.RGBA{0x0a, 0x2c, 0x38, 0xff}},
	{color.RGBA{0x18, 0x05, 0x05, 0xff}, color.RGBA{0x29, 0x0a, 0x0a, 0xff}, color.RGBA{0x3d, 0x10, 0x10, 0xff}, color.RGBA{0xef, 0x44, 0x44, 0xff}, color.RGBA{0xfc, 0xa5, 0xa5, 0xff}, color.RGBA{0x33, 0x0d, 0x0d, 0xff}},
	{color.RGBA{0x0a, 0x16, 0x05, 0xff}, color.RGBA{0x14, 0x26, 0x0c, 0xff}, color.RGBA{0x1f, 0x37, 0x12, 0xff}, color.RGBA{0x84, 0xcc, 0x16, 0xff}, color.RGBA{0xbe, 0xf2, 0x64, 0xff}, color.RGBA{0x1a, 0x2e, 0x0e, 0xff}},
	{color.RGBA{0x18, 0x10, 0x05, 0xff}, color.RGBA{0x29, 0x1c, 0x09, 0xff}, color.RGBA{0x3d, 0x2a, 0x0d, 0xff}, color.RGBA{0xea, 0x58, 0x0c, 0xff}, color.RGBA{0xfd, 0xba, 0x74, 0xff}, color.RGBA{0x33, 0x24, 0x0d, 0xff}},
	{color.RGBA{0x0c, 0x0c, 0x16, 0xff}, color.RGBA{0x16, 0x16, 0x26, 0xff}, color.RGBA{0x22, 0x22, 0x3a, 0xff}, color.RGBA{0x64, 0x74, 0xff, 0xff}, color.RGBA{0xa5, 0xb4, 0xfc, 0xff}, color.RGBA{0x1c, 0x1c, 0x33, 0xff}},
	{color.RGBA{0x1a, 0x0b, 0x14, 0xff}, color.RGBA{0x2a, 0x10, 0x21, 0xff}, color.RGBA{0x3d, 0x17, 0x30, 0xff}, color.RGBA{0xec, 0x48, 0x99, 0xff}, color.RGBA{0xf9, 0xa8, 0xd4, 0xff}, color.RGBA{0x33, 0x14, 0x29, 0xff}},
}

var pcardTitles = []string{
	"Adventurer", "Mystic", "Visionary", "Wanderer", "Sage", "Trailblazer",
	"Dreamweaver", "Stargazer", "Pathfinder", "Lorekeeper", "Nightowl", "Daybreaker",
	"Stormcaller", "Ironheart", "Lightbringer", "Shadowdancer", "Voidwalker", "Skyweaver",
	"Frostborn", "Emberforged", "Tidecaller", "Worldshaper", "Mythmaker", "Realmrider",
	"Spellbound", "Soulforged", "Runekeeper", "Echoseeker", "Phoenixsworn", "Starborn",
}

var pcardAuras = []string{
	"✨", "\U0001f30c", "\U0001f525", "\U0001f30a", "⚡", "\U0001f33f",
	"\U0001f31a", "\U0001f320", "\U0001f308", "\U0001f4ab", "\U0001f52e", "\U0001f343",
}

var pcardStatNames = []string{"POWER", "AURA", "VIBE", "LUCK", "CHAOS", "GRACE", "MAGIC"}

func pcardHash(userID int64, salt string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(strconv.FormatInt(userID, 10)))
	h.Write([]byte("|"))
	h.Write([]byte(salt))
	return h.Sum64()
}

func pcardPick(userID int64, salt string, n int) int {
	if n <= 0 {
		return 0
	}
	return int(pcardHash(userID, salt) % uint64(n))
}

func pcardPalette4(userID int64) pcardPalette {
	return pcardPalettes[pcardPick(userID, "palette", len(pcardPalettes))]
}

func pcardTitleFor(userID int64) string {
	return pcardTitles[pcardPick(userID, "title", len(pcardTitles))]
}

func pcardAuraFor(userID int64) string {
	return pcardAuras[pcardPick(userID, "aura", len(pcardAuras))]
}

func pcardStatsFor(userID int64) []struct {
	Name  string
	Value int
} {
	used := map[int]bool{}
	out := []struct {
		Name  string
		Value int
	}{}
	for i := 0; i < 3; i++ {
		idx := pcardPick(userID, fmt.Sprintf("statname_%d", i), len(pcardStatNames))
		for used[idx] {
			idx = (idx + 1) % len(pcardStatNames)
		}
		used[idx] = true
		val := 40 + int(pcardHash(userID, fmt.Sprintf("statval_%d", i))%61)
		out = append(out, struct {
			Name  string
			Value int
		}{pcardStatNames[idx], val})
	}
	return out
}

func pcardMemberSince(userID int64) string {
	now := time.Now().Year()
	years := []int{2013, 2014, 2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024}
	if userID < 1_000_000 {
		return "2013"
	}
	if userID < 10_000_000 {
		return "2014"
	}
	if userID < 100_000_000 {
		return "2015"
	}
	if userID < 200_000_000 {
		return "2017"
	}
	if userID < 500_000_000 {
		return "2018"
	}
	if userID < 1_000_000_000 {
		return "2020"
	}
	if userID < 1_500_000_000 {
		return "2021"
	}
	if userID < 2_000_000_000 {
		return "2022"
	}
	if userID < 5_000_000_000 {
		return "2023"
	}
	if now < 2024 {
		now = 2024
	}
	return strconv.Itoa(years[len(years)-1])
}

func pcardFontPath(name string) string {
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

func pcardLoadFont(dc *gg.Context, size float64) bool {
	for _, name := range []string{"Inter_28pt-Bold.ttf", "Swiss 721 Black Extended BT.ttf"} {
		p := pcardFontPath(name)
		if p == "" {
			continue
		}
		if err := dc.LoadFontFace(p, size); err == nil {
			return true
		}
	}
	dc.SetFontFace(basicfont.Face7x13)
	return false
}

func pcardLerpColor(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: 0xff,
	}
}

func pcardDrawBackground(dc *gg.Context, w, h int, pal pcardPalette, userID int64) {
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h-1)
		var c color.RGBA
		if t < 0.5 {
			c = pcardLerpColor(pal.BgTop, pal.BgMid, t/0.5)
		} else {
			c = pcardLerpColor(pal.BgMid, pal.BgBot, (t-0.5)/0.5)
		}
		dc.SetRGB255(int(c.R), int(c.G), int(c.B))
		dc.DrawRectangle(0, float64(y), float64(w), 1)
		dc.Fill()
	}

	blobSeed := pcardHash(userID, "blobs")
	blobs := []struct {
		cx, cy, r float64
		alpha     int
	}{
		{float64(w) * (0.15 + float64(blobSeed%17)*0.01), float64(h) * (0.2 + float64((blobSeed>>4)%13)*0.015), float64(w) * 0.35, 55},
		{float64(w) * (0.82 - float64((blobSeed>>8)%19)*0.008), float64(h) * (0.75 - float64((blobSeed>>12)%11)*0.012), float64(w) * 0.28, 65},
		{float64(w) * 0.55, float64(h) * (0.5 + float64((blobSeed>>16)%7)*0.02), float64(w) * 0.22, 45},
	}
	for _, b := range blobs {
		for r := b.r; r > b.r*0.3; r -= 6 {
			a := int(float64(b.alpha) * (1 - (b.r-r)/b.r))
			if a < 1 {
				a = 1
			}
			dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), a/8)
			dc.DrawCircle(b.cx, b.cy, r)
			dc.Fill()
		}
	}

	dc.SetRGBA255(255, 255, 255, 8)
	dc.SetLineWidth(1)
	gridSize := 64.0
	for x := 0.0; x < float64(w); x += gridSize {
		dc.DrawLine(x, 0, x, float64(h))
		dc.Stroke()
	}
	for y := 0.0; y < float64(h); y += gridSize {
		dc.DrawLine(0, y, float64(w), y)
		dc.Stroke()
	}

	starSeed := pcardHash(userID, "stars")
	for i := 0; i < 80; i++ {
		sx := float64((starSeed>>uint(i%32))%uint64(w)) + float64(i*17%w)
		sy := float64((starSeed>>uint((i*3)%32))%uint64(h)) + float64(i*29%h)
		sx = math.Mod(sx, float64(w))
		sy = math.Mod(sy, float64(h))
		rad := 0.6 + float64(i%5)*0.3
		alpha := 25 + (i*7)%50
		dc.SetRGBA255(255, 255, 255, alpha)
		dc.DrawCircle(sx, sy, rad)
		dc.Fill()
	}
}

func pcardDrawGlassPanel(dc *gg.Context, x, y, w, h, radius float64, pal pcardPalette) {
	dc.SetRGBA255(int(pal.Surface.R), int(pal.Surface.G), int(pal.Surface.B), 200)
	dc.DrawRoundedRectangle(x, y, w, h, radius)
	dc.Fill()

	dc.SetRGBA(1, 1, 1, 0.05)
	dc.DrawRoundedRectangle(x, y, w, h*0.5, radius)
	dc.Fill()

	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 120)
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(x, y, w, h, radius)
	dc.Stroke()

	dc.SetRGBA(1, 1, 1, 0.08)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(x+2, y+2, w-4, h-4, radius-2)
	dc.Stroke()
}

func pcardDrawAvatarRing(dc *gg.Context, cx, cy, r float64, pal pcardPalette, userID int64, avatarPath string) {
	for i := 0; i < 6; i++ {
		alpha := 25 - i*3
		if alpha < 4 {
			alpha = 4
		}
		dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), alpha)
		dc.DrawCircle(cx, cy, r+18+float64(i)*4)
		dc.Fill()
	}

	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 255)
	dc.SetLineWidth(6)
	dc.DrawCircle(cx, cy, r+10)
	dc.Stroke()

	dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 220)
	dc.SetLineWidth(2)
	dc.DrawCircle(cx, cy, r+16)
	dc.Stroke()

	if avatarPath != "" {
		f, err := os.Open(avatarPath)
		if err == nil {
			defer f.Close()
			img, _, derr := image.Decode(f)
			if derr == nil {
				b := img.Bounds()
				srcW := float64(b.Dx())
				srcH := float64(b.Dy())
				size := r * 2
				scale := size / srcW
				if srcH < srcW {
					scale = size / srcH
				}
				tmp := gg.NewContext(int(size), int(size))
				tmp.ScaleAbout(scale, scale, srcW/2, srcH/2)
				tmp.DrawImageAnchored(img, int(size/2), int(size/2), 0.5, 0.5)

				dc.Push()
				dc.DrawCircle(cx, cy, r)
				dc.Clip()
				dc.DrawImageAnchored(tmp.Image(), int(cx), int(cy), 0.5, 0.5)
				dc.ResetClip()
				dc.Pop()
				return
			}
		}
	}

	dc.SetRGBA255(int(pal.Accent.R)/2, int(pal.Accent.G)/2, int(pal.Accent.B)/2, 255)
	dc.DrawCircle(cx, cy, r)
	dc.Fill()
}

func pcardInitials(name string) string {
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
		return strings.ToUpper(string(r[0:2]))
	}
	a := []rune(parts[0])
	b := []rune(parts[len(parts)-1])
	if len(a) == 0 || len(b) == 0 {
		return "?"
	}
	return strings.ToUpper(string(a[0]) + string(b[0]))
}

func pcardDrawInitialsAvatar(dc *gg.Context, cx, cy, r float64, initials string, pal pcardPalette) {
	dc.Push()
	defer dc.Pop()
	dc.DrawCircle(cx, cy, r)
	dc.SetRGBA255(int(pal.Accent.R)/2, int(pal.Accent.G)/2, int(pal.Accent.B)/2, 255)
	dc.Fill()
	pcardLoadFont(dc, r*0.85)
	dc.SetRGB(1, 1, 1)
	if initials == "" {
		initials = "?"
	}
	dc.DrawStringAnchored(initials, cx, cy, 0.5, 0.55)
}

func pcardDrawStatBar(dc *gg.Context, x, y, w, h float64, label string, value int, pal pcardPalette) {
	pcardLoadFont(dc, 20)
	dc.SetRGBA(1, 1, 1, 0.55)
	dc.DrawString(label, x, y-6)

	pcardLoadFont(dc, 22)
	dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 255)
	valStr := fmt.Sprintf("%d", value)
	vw, _ := dc.MeasureString(valStr)
	dc.DrawString(valStr, x+w-vw, y-6)

	dc.SetRGBA(1, 1, 1, 0.1)
	dc.DrawRoundedRectangle(x, y, w, h, h/2)
	dc.Fill()

	fillW := w * float64(value) / 100.0
	if fillW < h {
		fillW = h
	}
	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 255)
	dc.DrawRoundedRectangle(x, y, fillW, h, h/2)
	dc.Fill()

	dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 255)
	dc.DrawCircle(x+fillW, y+h/2, h*0.55)
	dc.Fill()
}

func pcardResolveTarget(m *tg.NewMessage) (pcardTarget, error) {
	var info pcardTarget
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply != nil && reply.SenderID() != 0 {
			u, uerr := m.Client.GetUser(reply.SenderID())
			if uerr == nil && u != nil {
				info.UserID = u.ID
				info.FirstName = u.FirstName
				info.LastName = u.LastName
				info.Username = u.Username
				info.Photo = u.Photo
				return info, nil
			}
			info.UserID = reply.SenderID()
			info.FirstName = "User"
			return info, nil
		}
	}
	args := strings.TrimSpace(m.Args())
	if args != "" {
		token := strings.Fields(args)[0]
		token = strings.TrimPrefix(token, "@")
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
			return info, fmt.Errorf("could not resolve user %d", n)
		}
		peer, err := m.Client.ResolvePeer(token)
		if err != nil {
			return info, err
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
	if m.Sender != nil {
		info.UserID = m.Sender.ID
		info.FirstName = m.Sender.FirstName
		info.LastName = m.Sender.LastName
		info.Username = m.Sender.Username
		info.Photo = m.Sender.Photo
		return info, nil
	}
	info.UserID = m.SenderID()
	info.FirstName = "User"
	return info, nil
}

func pcardGetAccessHash(c *tg.Client, userID int64) int64 {
	peer, err := c.ResolvePeer(userID)
	if err != nil {
		return 0
	}
	if pu, ok := peer.(*tg.InputPeerUser); ok {
		return pu.AccessHash
	}
	return 0
}

func pcardDownloadAvatar(c *tg.Client, info pcardTarget) string {
	if info.UserID == 0 || info.Photo == nil {
		return ""
	}
	full, err := c.UsersGetFullUser(&tg.InputUserObj{
		UserID:     info.UserID,
		AccessHash: pcardGetAccessHash(c, info.UserID),
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
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("pcard_av_%d_%d.jpg", info.UserID, time.Now().UnixNano()))
	_, err = c.DownloadMedia(p, &tg.DownloadOptions{FileName: tmp})
	if err != nil {
		os.Remove(tmp)
		return ""
	}
	return tmp
}

func pcardRender(info pcardTarget, avatarPath string) (string, error) {
	const W, H = 1280, 720

	dc := gg.NewContext(W, H)
	pal := pcardPalette4(info.UserID)

	bg := gg.NewLinearGradient(0, 0, 0, H)
	bg.AddColorStop(0, pal.BgTop)
	bg.AddColorStop(1, pal.BgBot)

	dc.SetFillStyle(bg)
	dc.DrawRectangle(0, 0, W, H)
	dc.Fill()

	for i := 0; i < 3; i++ {
		r := 300.0 + float64(i*80)

		x := []float64{200, 1000, 700}[i]
		y := []float64{180, 540, 300}[i]

		g := gg.NewRadialGradient(
			x, y, 0,
			x, y, r,
		)

		g.AddColorStop(0, color.RGBA{
			pal.Accent.R,
			pal.Accent.G,
			pal.Accent.B,
			40,
		})

		g.AddColorStop(1, color.RGBA{
			pal.Accent.R,
			pal.Accent.G,
			pal.Accent.B,
			0,
		})

		dc.SetFillStyle(g)
		dc.DrawCircle(x, y, r)
		dc.Fill()
	}

	cardX := 50.0
	cardY := 50.0
	cardW := 1180.0
	cardH := 620.0

	dc.SetRGBA255(20, 24, 35, 230)
	dc.DrawRoundedRectangle(cardX, cardY, cardW, cardH, 35)
	dc.Fill()

	dc.SetRGBA255(
		int(pal.Accent.R),
		int(pal.Accent.G),
		int(pal.Accent.B),
		80,
	)

	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(cardX, cardY, cardW, cardH, 35)
	dc.Stroke()

	sidebarW := 320.0

	dc.SetRGBA255(255, 255, 255, 8)
	dc.DrawRoundedRectangle(
		cardX,
		cardY,
		sidebarW,
		cardH,
		35,
	)
	dc.Fill()

	dc.SetRGBA255(255, 255, 255, 20)
	dc.DrawLine(
		cardX+sidebarW,
		cardY,
		cardX+sidebarW,
		cardY+cardH,
	)
	dc.Stroke()

	avCX := cardX + sidebarW/2
	avCY := cardY + 180
	avR := 105.0

	dc.SetRGBA255(
		int(pal.Accent.R),
		int(pal.Accent.G),
		int(pal.Accent.B),
		40,
	)

	dc.DrawCircle(avCX, avCY, avR+18)
	dc.Fill()

	dc.SetRGBA255(
		int(pal.Accent2.R),
		int(pal.Accent2.G),
		int(pal.Accent2.B),
		255,
	)

	dc.SetLineWidth(4)
	dc.DrawCircle(avCX, avCY, avR+8)
	dc.Stroke()

	if avatarPath != "" {
		pcardDrawAvatarRing(
			dc,
			avCX,
			avCY,
			avR,
			pal,
			info.UserID,
			avatarPath,
		)
	} else {
		name := strings.TrimSpace(
			info.FirstName + " " + info.LastName,
		)

		pcardDrawInitialsAvatar(
			dc,
			avCX,
			avCY,
			avR,
			pcardInitials(name),
			pal,
		)
	}

	leftX := cardX + 45
	infoY := cardY + 390

	pcardLoadFont(dc, 20)
	dc.SetRGBA(1, 1, 1, 0.45)
	dc.DrawString("USER ID", leftX, infoY)

	pcardLoadFont(dc, 28)
	dc.SetRGB(1, 1, 1)
	dc.DrawString(
		strconv.FormatInt(info.UserID, 10),
		leftX,
		infoY+35,
	)

	infoY += 110

	pcardLoadFont(dc, 20)
	dc.SetRGBA(1, 1, 1, 0.45)
	dc.DrawString("MEMBER SINCE", leftX, infoY)

	pcardLoadFont(dc, 28)
	dc.SetRGB(1, 1, 1)
	dc.DrawString(
		pcardMemberSince(info.UserID),
		leftX,
		infoY+35,
	)

	contentX := cardX + sidebarW + 55

	displayName := strings.TrimSpace(
		info.FirstName + " " + info.LastName,
	)

	if displayName == "" {
		displayName = "User"
	}

	if len(displayName) > 24 {
		displayName = displayName[:24] + "..."
	}

	titleY := cardY + 170

	pcardLoadFont(dc, 60)
	dc.SetRGB(1, 1, 1)
	dc.DrawString(
		displayName,
		contentX,
		titleY,
	)

	if info.Username != "" {
		pcardLoadFont(dc, 30)
		dc.SetRGBA(1, 1, 1, 0.6)

		dc.DrawString(
			"@"+info.Username,
			contentX,
			titleY+50,
		)
	}

	badgeY := titleY + 95

	dc.SetRGBA255(
		int(pal.Accent.R),
		int(pal.Accent.G),
		int(pal.Accent.B),
		60,
	)

	dc.DrawRoundedRectangle(
		contentX,
		badgeY,
		240,
		55,
		20,
	)
	dc.Fill()

	title := pcardTitleFor(info.UserID)

	pcardLoadFont(dc, 24)
	dc.SetRGBA255(
		int(pal.Accent2.R),
		int(pal.Accent2.G),
		int(pal.Accent2.B),
		255,
	)

	dc.DrawString(
		title,
		contentX+20,
		badgeY+36,
	)

	aura := pcardAuraFor(info.UserID)

	pcardLoadFont(dc, 30)
	dc.SetRGB(1, 1, 1)

	dc.DrawString(
		aura,
		contentX+280,
		badgeY+36,
	)

	statsX := contentX
	statsY := cardY + 350
	statsW := 700.0
	statsH := 220.0

	dc.SetRGBA255(255, 255, 255, 8)
	dc.DrawRoundedRectangle(
		statsX,
		statsY,
		statsW,
		statsH,
		22,
	)
	dc.Fill()

	pcardLoadFont(dc, 24)
	dc.SetRGBA(1, 1, 1, 0.7)
	dc.DrawString(
		"STATS",
		statsX+25,
		statsY+35,
	)

	stats := pcardStatsFor(info.UserID)

	for i, s := range stats {
		y := statsY + 80 + float64(i)*50

		pcardDrawStatBar(
			dc,
			statsX+25,
			y,
			statsW-50,
			16,
			s.Name,
			s.Value,
			pal,
		)
	}

	pcardLoadFont(dc, 24)
	dc.SetRGBA(1, 1, 1, 0.3)

	dc.DrawString(
		"JULIABOT",
		cardX+cardW-180,
		cardY+45,
	)

	outPath := filepath.Join(
		os.TempDir(),
		fmt.Sprintf(
			"pcard_%d_%d.png",
			info.UserID,
			time.Now().UnixNano(),
		),
	)

	if err := dc.SavePNG(outPath); err != nil {
		return "", err
	}

	return outPath, nil
}

func ProfileCardHandler(m *tg.NewMessage) error {
	status, _ := m.Reply("<i>forging your card...</i>")

	info, err := pcardResolveTarget(m)
	if err != nil {
		if status != nil {
			status.Edit("failed: " + html.EscapeString(err.Error()))
		}
		return nil
	}
	if info.UserID == 0 {
		if status != nil {
			status.Edit("could not resolve user")
		}
		return nil
	}

	avatarPath := pcardDownloadAvatar(m.Client, info)
	defer func() {
		if avatarPath != "" {
			os.Remove(avatarPath)
		}
	}()

	outPath, rerr := pcardRender(info, avatarPath)
	if rerr != nil || outPath == "" {
		msg := "render failed"
		if rerr != nil {
			msg = html.EscapeString(rerr.Error())
		}
		if status != nil {
			status.Edit("failed: " + msg)
		}
		return nil
	}
	defer os.Remove(outPath)

	displayName := strings.TrimSpace(info.FirstName + " " + info.LastName)
	if displayName == "" {
		displayName = "User"
	}
	caption := fmt.Sprintf("<b>%s</b>", html.EscapeString(displayName))
	if info.Username != "" {
		caption += fmt.Sprintf(" · @%s", html.EscapeString(info.Username))
	}
	caption += fmt.Sprintf("\n<i>%s</i> %s", html.EscapeString(pcardTitleFor(info.UserID)), pcardAuraFor(info.UserID))

	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		Caption:  caption,
		FileName: fmt.Sprintf("card_%d.png", info.UserID),
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

func registerProfileCardsHandlers() {
	c := modules.Client
	c.On("cmd:card", ProfileCardHandler)
}

func initFromSrc_profile_cards_3_1() {
	modules.QueueHandlerRegistration(registerProfileCardsHandlers)
}

// === from screenshot.go ===
func fetchScreenshot(target string) (string, error) {
	endpoint := "https://image.thum.io/get/width/1280/" + target
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("thum.io returned status %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return "", fmt.Errorf("unexpected content-type: %s", ct)
	}
	f, err := os.CreateTemp("", "ss-*.png")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 15*1024*1024)); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	st, err := os.Stat(f.Name())
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	if st.Size() == 0 {
		os.Remove(f.Name())
		return "", fmt.Errorf("empty screenshot response")
	}
	return f.Name(), nil
}

func ScreenshotHandler(m *tg.NewMessage) error {
	target := strings.TrimSpace(m.Args())
	if target == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			target = strings.TrimSpace(r.Text())
		}
	}
	if target == "" {
		m.Reply("usage: <code>/ss &lt;url&gt;</code>")
		return nil
	}

	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	u, err := url.Parse(target)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		m.Reply("invalid url, must be http(s)")
		return nil
	}

	status, _ := m.Reply("<code>capturing screenshot...</code>")

	path, err := fetchScreenshot(target)
	if err != nil {
		msg := "error fetching screenshot: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(path)

	if status != nil {
		status.Delete()
	}

	caption := "<b>Screenshot</b>\n<a href=\"" + html.EscapeString(target) + "\">" + html.EscapeString(target) + "</a>"
	if _, merr := m.ReplyMedia(path, &tg.MediaOptions{
		Caption:  caption,
		FileName: "screenshot.png",
		MimeType: "image/png",
	}); merr != nil {
		m.Reply("error sending: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func initFromSrc_screenshot_4_1() { modules.QueueHandlerRegistration(registerScreenshotHandlers) }

func registerScreenshotHandlers() {
	c := modules.Client
	c.On("cmd:ss", ScreenshotHandler)
}

func init() {
	initFromSrc_image_crop_0_1()
	initFromSrc_colors_1_1()
	initFromSrc_wallpaper_2_1()
	initFromSrc_profile_cards_3_1()
	initFromSrc_screenshot_4_1()
}
