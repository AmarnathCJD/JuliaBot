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
	"Cloudwalker", "Duskbringer", "Ashborn", "Moonrider", "Sunseeker", "Wavebreaker",
	"Thornheart", "Silvertongue", "Gladewatcher", "Suncrafter", "Emberwitch", "Rimebound",
	"Windshaper", "Ravensworn", "Ashenwolf", "Steelmind", "Owlbound", "Riftwalker",
	"Prismborn", "Grimshade", "Ironquill", "Songkeeper", "Firelark", "Coldforged",
	"Latecomer", "Firstlight", "Solargazer", "Duneskipper", "Cinderpath", "Loomweaver",
	"Nightscribe", "Ashenroamer",
}


var pcardAuraColors = []color.RGBA{
	{0xFF, 0xD7, 0x54, 0xFF}, {0x8B, 0x5C, 0xF6, 0xFF},
	{0xF9, 0x5D, 0x3B, 0xFF}, {0x38, 0xBD, 0xF8, 0xFF},
	{0xFA, 0xCC, 0x15, 0xFF}, {0x4A, 0xDE, 0x80, 0xFF},
	{0xFB, 0xBF, 0x24, 0xFF}, {0xEC, 0x4C, 0x8B, 0xFF},
	{0xFF, 0x66, 0x99, 0xFF}, {0xA5, 0x8B, 0xFF, 0xFF},
	{0x60, 0xA5, 0xFA, 0xFF}, {0x86, 0xEF, 0xAC, 0xFF},
	{0xF4, 0x72, 0xB6, 0xFF}, {0x2D, 0xD4, 0xBF, 0xFF},
	{0xE1, 0x1D, 0x48, 0xFF}, {0x22, 0xC5, 0x5E, 0xFF},
	{0xC0, 0x82, 0xFF, 0xFF}, {0xF8, 0x71, 0x71, 0xFF},
	{0x0E, 0xA5, 0xE9, 0xFF}, {0x84, 0xCC, 0x16, 0xFF},
	{0xF9, 0x73, 0x16, 0xFF}, {0xBE, 0x18, 0x5D, 0xFF},
	{0x14, 0xB8, 0xA6, 0xFF}, {0xF5, 0x9E, 0x0B, 0xFF},
}

func pcardAuraColorFor(userID int64) color.RGBA {
	return pcardAuraColors[pcardPick(userID, "auracolor", len(pcardAuraColors))]
}

var pcardStatNames = []string{
	"POWER", "AURA", "VIBE", "LUCK", "CHAOS", "GRACE", "MAGIC",
	"FOCUS", "ECHO", "SPARK", "CHILL", "REACH", "PULSE", "GRIT", "FLOW",
}

type pcardMetric struct {
	Name   string
	Salt   string
	FromFn func(*userPerf) int
}

var pcardRealMetrics = []pcardMetric{
	{"POWER", "power", func(p *userPerf) int { return pcardScaleLog(p.TotalMsgs, 5000) }},
	{"CHAOS", "chaos", func(p *userPerf) int { return pcardScaleRatio(p.NightMsgs, p.TotalMsgs) }},
	{"AURA", "aura", func(p *userPerf) int { return pcardScaleLog(int64(len(p.Chats)), 40) }},
	{"MAGIC", "magic", func(p *userPerf) int { return pcardScaleLog(int64(len(p.Commands)), 30) }},
	{"VIBE", "vibe", func(p *userPerf) int { return pcardScaleRatio(p.ReplyMsgs, p.TotalMsgs) }},
	{"ECHO", "echo", func(p *userPerf) int { return pcardScaleRatio(p.StickerMsgs, p.TotalMsgs) }},
	{"SPARK", "spark", func(p *userPerf) int { return pcardScaleRatio(p.LinkMsgs, p.TotalMsgs) }},
	{"REACH", "reach", func(p *userPerf) int { return pcardScaleRatio(p.MediaMsgs, p.TotalMsgs) }},
	{"FLOW", "flow", func(p *userPerf) int {
		if p.TotalMsgs == 0 {
			return 0
		}
		avg := float64(p.CharSum) / float64(p.TotalMsgs)
		v := int(avg * 1.4)
		if v > 100 {
			v = 100
		}
		return v
	}},
	{"FOCUS", "focus", func(p *userPerf) int { return pcardScaleRatio(p.CmdMsgs, p.TotalMsgs) }},
	{"PULSE", "pulse", func(p *userPerf) int {
		if p.LastSeen == 0 {
			return 0
		}
		diff := time.Now().Unix() - p.LastSeen
		if diff < 3600 {
			return 100
		}
		if diff < 86400 {
			return 80
		}
		if diff < 7*86400 {
			return 50
		}
		if diff < 30*86400 {
			return 25
		}
		return 5
	}},
	{"GRIT", "grit", func(p *userPerf) int {
		if p.FirstSeen == 0 {
			return 0
		}
		days := (time.Now().Unix() - p.FirstSeen) / 86400
		return pcardScaleLog(days, 180)
	}},
}

func pcardScaleLog(n, cap int64) int {
	if n <= 0 {
		return 0
	}
	if n >= cap {
		return 100
	}
	f := float64(n) / float64(cap)
	v := int(100 * (0.5 + 0.5*f*f))
	if v > 100 {
		v = 100
	}
	if v < 5 {
		v = 5
	}
	return v
}

func pcardScaleRatio(part, total int64) int {
	if total <= 0 {
		return 0
	}
	ratio := float64(part) / float64(total)
	v := int(ratio * 250)
	if v > 100 {
		v = 100
	}
	return v
}

func pcardHasRealData(p *userPerf) bool {
	return p != nil && p.TotalMsgs >= 25
}

type pcardStat struct {
	Name  string
	Value int
}

func pcardStatsFor(userID int64) []pcardStat {
	perf := UserPerfGet(userID)
	if pcardHasRealData(perf) {
		buckets := make([]pcardMetric, len(pcardRealMetrics))
		copy(buckets, pcardRealMetrics)
		type scored struct {
			m pcardMetric
			v int
			s uint64
		}
		out := make([]scored, 0, len(buckets))
		for _, m := range buckets {
			v := m.FromFn(perf)
			if v <= 0 {
				continue
			}
			out = append(out, scored{m, v, pcardHash(userID, m.Salt)})
		}
		if len(out) >= 3 {
			for i := range out {
				j := i + int(out[i].s%uint64(len(out)-i))
				out[i], out[j] = out[j], out[i]
			}
			out = out[:3]
			result := make([]pcardStat, 0, 3)
			for _, s := range out {
				result = append(result, pcardStat{s.m.Name, s.v})
			}
			return result
		}
	}
	used := map[int]bool{}
	out := make([]pcardStat, 0, 3)
	for i := 0; i < 3; i++ {
		idx := pcardPick(userID, fmt.Sprintf("statname_%d", i), len(pcardStatNames))
		for used[idx] {
			idx = (idx + 1) % len(pcardStatNames)
		}
		used[idx] = true
		val := 40 + int(pcardHash(userID, fmt.Sprintf("statval_%d", i))%61)
		out = append(out, pcardStat{pcardStatNames[idx], val})
	}
	return out
}

type pcardRank struct {
	Name  string
	Color color.RGBA
}

var pcardRanks = []pcardRank{
	{"BRONZE", color.RGBA{0xCD, 0x7F, 0x32, 0xFF}},
	{"SILVER", color.RGBA{0xC0, 0xC0, 0xC0, 0xFF}},
	{"GOLD", color.RGBA{0xFF, 0xD7, 0x00, 0xFF}},
	{"PLATINUM", color.RGBA{0xE5, 0xE4, 0xE2, 0xFF}},
	{"DIAMOND", color.RGBA{0x67, 0xE8, 0xF9, 0xFF}},
	{"MYTHIC", color.RGBA{0xC0, 0x82, 0xFF, 0xFF}},
}

func pcardRankFor(perf *userPerf) pcardRank {
	if perf == nil {
		return pcardRanks[0]
	}
	n := perf.TotalMsgs
	switch {
	case n >= 20000:
		return pcardRanks[5]
	case n >= 8000:
		return pcardRanks[4]
	case n >= 3000:
		return pcardRanks[3]
	case n >= 1000:
		return pcardRanks[2]
	case n >= 250:
		return pcardRanks[1]
	default:
		return pcardRanks[0]
	}
}

func pcardBadgesFor(perf *userPerf) []string {
	if perf == nil || perf.TotalMsgs < 50 {
		return nil
	}
	var out []string
	if perf.TotalMsgs >= 5000 {
		out = append(out, "Chatterbox")
	}
	if perf.TotalMsgs > 0 && float64(perf.NightMsgs)/float64(perf.TotalMsgs) >= 0.35 {
		out = append(out, "Night Owl")
	}
	if perf.TotalMsgs > 0 && float64(perf.StickerMsgs)/float64(perf.TotalMsgs) >= 0.25 {
		out = append(out, "Sticker King")
	}
	if perf.TotalMsgs > 0 && float64(perf.MediaMsgs)/float64(perf.TotalMsgs) >= 0.4 {
		out = append(out, "Media Hoarder")
	}
	if perf.TotalMsgs > 0 && float64(perf.ReplyMsgs)/float64(perf.TotalMsgs) >= 0.5 {
		out = append(out, "Reply Guy")
	}
	if perf.TotalMsgs > 0 && float64(perf.LinkMsgs)/float64(perf.TotalMsgs) >= 0.2 {
		out = append(out, "Linkposter")
	}
	if len(perf.Chats) >= 10 {
		out = append(out, "Nomad")
	}
	if len(perf.Commands) >= 20 {
		out = append(out, "Power User")
	}
	if perf.TotalMsgs > 0 && float64(perf.CmdMsgs)/float64(perf.TotalMsgs) >= 0.5 {
		out = append(out, "Bot Whisperer")
	}
	if perf.TotalMsgs > 0 {
		avg := float64(perf.CharSum) / float64(perf.TotalMsgs)
		if avg >= 120 {
			out = append(out, "Wordsmith")
		}
		if avg <= 8 && perf.TotalMsgs >= 200 {
			out = append(out, "Terse")
		}
	}
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

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


func pcardMemberSinceLegacy(userID int64) string {
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

func pcardMemberSince(userID int64) string {
	if ts := modules.NewUserDateEstimator().Estimate(userID); ts > 0 {
		return time.Unix(ts, 0).Format("Jan 2006")
	}
	perf := UserPerfGet(userID)
	if perf != nil && perf.FirstSeen > 0 {
		return time.Unix(perf.FirstSeen, 0).Format("Jan 2006")
	}
	return pcardMemberSinceLegacy(userID)
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
	for _, name := range []string{"GoNotoCurrent-Bold.ttf", "GoNotoCurrent-Regular.ttf", "NotoSans-Bold.ttf", "NotoSans-Regular.ttf", "Inter_28pt-Bold.ttf"} {
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

func pcardStripEmoji(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if pcardIsEmojiRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func pcardIsEmojiRune(r rune) bool {
	switch {
	case r == 0x200D, r == 0xFE0F, r == 0x20E3:
		return true
	case r >= 0x1F000 && r <= 0x1FFFF:
		return true
	case r >= 0x2600 && r <= 0x27BF:
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF:
		return true
	case r >= 0x2700 && r <= 0x27BF:
		return true
	case r >= 0x1F900 && r <= 0x1F9FF:
		return true
	case r >= 0x1FA70 && r <= 0x1FAFF:
		return true
	case r >= 0x2300 && r <= 0x23FF:
		return true
	case r >= 0x25A0 && r <= 0x25FF:
		return true
	case r == 0x2934 || r == 0x2935, r >= 0x2B00 && r <= 0x2BFF:
		return true
	case r >= 0x1F3FB && r <= 0x1F3FF:
		return true
	}
	return false
}


func pcardDrawAvatarRing(dc *gg.Context, cx, cy, r float64, pal pcardPalette, avatarPath string) {
	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 30)
	dc.DrawCircle(cx, cy, r+14)
	dc.Fill()

	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 220)
	dc.SetLineWidth(3)
	dc.DrawCircle(cx, cy, r+6)
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
				side := srcW
				if srcH < side {
					side = srcH
				}
				sx := (srcW - side) / 2
				sy := (srcH - side) / 2
				diameter := int(math.Ceil(r * 2))
				scale := float64(diameter) / side
				tmp := gg.NewContext(diameter, diameter)
				tmp.Scale(scale, scale)
				tmp.DrawImage(img, int(-sx), int(-sy))

				dc.Push()
				dc.DrawCircle(cx, cy, r)
				dc.Clip()
				dc.DrawImage(tmp.Image(), int(cx-r), int(cy-r))
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
	pcardLoadFont(dc, 18)
	dc.SetRGBA(1, 1, 1, 0.6)
	dc.DrawString(label, x, y-8)

	pcardLoadFont(dc, 18)
	dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 240)
	valStr := fmt.Sprintf("%d", value)
	vw, _ := dc.MeasureString(valStr)
	dc.DrawString(valStr, x+w-vw, y-8)

	dc.SetRGBA(1, 1, 1, 0.09)
	dc.DrawRoundedRectangle(x, y, w, h, h/2)
	dc.Fill()

	fillPct := float64(value) / 100.0
	if fillPct < 0 {
		fillPct = 0
	}
	if fillPct > 1 {
		fillPct = 1
	}
	fillW := w * fillPct
	if fillW < h {
		fillW = h
	}
	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 235)
	dc.DrawRoundedRectangle(x, y, fillW, h, h/2)
	dc.Fill()
}

func pcardDrawLabel(dc *gg.Context, text string, x, y float64) {
	pcardLoadFont(dc, 12)
	dc.SetRGBA(1, 1, 1, 0.38)
	dc.DrawString(text, x, y)
}

func pcardDrawCornerBrackets(dc *gg.Context, x, y, w, h float64, pal pcardPalette) {
	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 220)
	dc.SetLineWidth(2)
	const armLen = 22.0
	const inset = 16.0
	dc.DrawLine(x+inset, y+inset+armLen, x+inset, y+inset)
	dc.DrawLine(x+inset, y+inset, x+inset+armLen, y+inset)
	dc.Stroke()
	dc.DrawLine(x+w-inset-armLen, y+inset, x+w-inset, y+inset)
	dc.DrawLine(x+w-inset, y+inset, x+w-inset, y+inset+armLen)
	dc.Stroke()
	dc.DrawLine(x+inset, y+h-inset-armLen, x+inset, y+h-inset)
	dc.DrawLine(x+inset, y+h-inset, x+inset+armLen, y+h-inset)
	dc.Stroke()
	dc.DrawLine(x+w-inset-armLen, y+h-inset, x+w-inset, y+h-inset)
	dc.DrawLine(x+w-inset, y+h-inset, x+w-inset, y+h-inset-armLen)
	dc.Stroke()
}

func pcardSerialFor(userID int64) string {
	h := pcardHash(userID, "serial")
	block := func(shift uint) string {
		v := (h >> shift) & 0xFFFF
		return fmt.Sprintf("%04X", v)
	}
	return fmt.Sprintf("N° %s · %s · %s", block(0), block(16), block(32))
}

func pcardPeakHourLabel(perf *userPerf) (string, int) {
	if perf == nil {
		return "—", -1
	}
	var total int64
	for _, v := range perf.HourBuckets {
		total += v
	}
	if total < 5 {
		return "—", -1
	}
	best := 0
	bestVal := perf.HourBuckets[0]
	for i, v := range perf.HourBuckets {
		if v > bestVal {
			bestVal = v
			best = i
		}
	}
	start := best - 1
	end := best + 2
	if start < 0 {
		start += 24
	}
	if end > 23 {
		end -= 24
	}
	return fmt.Sprintf("%02d:00–%02d:00", start, end), best
}

func pcardDrawSparkline(dc *gg.Context, x, y, w, h float64, perf *userPerf, pal pcardPalette) {
	dc.SetRGBA255(255, 255, 255, 8)
	dc.DrawRoundedRectangle(x, y, w, h, 10)
	dc.Fill()
	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 55)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(x, y, w, h, 10)
	dc.Stroke()

	pcardLoadFont(dc, 11)
	dc.SetRGBA(1, 1, 1, 0.38)
	dc.DrawString("LAST 14 DAYS", x+12, y+16)

	const days = 14
	buckets := make([]int64, days)
	if perf != nil && perf.DailyMsgs != nil {
		now := time.Now()
		for i := 0; i < days; i++ {
			k := now.AddDate(0, 0, -(days - 1 - i)).Format("2006-01-02")
			buckets[i] = perf.DailyMsgs[k]
		}
	}
	maxV := int64(1)
	for _, v := range buckets {
		if v > maxV {
			maxV = v
		}
	}

	plotX := x + 12
	plotY := y + 22
	plotW := w - 24
	plotH := h - 30
	barW := plotW / float64(days)
	for i, v := range buckets {
		bh := plotH * float64(v) / float64(maxV)
		if v > 0 && bh < 3 {
			bh = 3
		}
		bx := plotX + float64(i)*barW + 1
		by := plotY + plotH - bh
		bar := barW - 2
		alpha := uint8(120)
		if i == days-1 {
			alpha = 235
		} else if i >= days-3 {
			alpha = 180
		}
		dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), int(alpha))
		dc.DrawRoundedRectangle(bx, by, bar, bh, 2)
		dc.Fill()
	}

	total := int64(0)
	for _, v := range buckets {
		total += v
	}
	summary := fmt.Sprintf("%d msgs", total)
	pcardLoadFont(dc, 11)
	dc.SetRGBA(1, 1, 1, 0.55)
	sw, _ := dc.MeasureString(summary)
	dc.DrawString(summary, x+w-12-sw, y+16)
}

func pcardNextRankThreshold(current pcardRank) (string, int64) {
	tiers := []struct {
		Name string
		Need int64
	}{
		{"BRONZE", 0},
		{"SILVER", 250},
		{"GOLD", 1000},
		{"PLATINUM", 3000},
		{"DIAMOND", 8000},
		{"MYTHIC", 20000},
	}
	for i, t := range tiers {
		if t.Name == current.Name && i+1 < len(tiers) {
			return tiers[i+1].Name, tiers[i+1].Need
		}
	}
	return "", 0
}

func pcardCurrentRankFloor(rank pcardRank) int64 {
	switch rank.Name {
	case "BRONZE":
		return 0
	case "SILVER":
		return 250
	case "GOLD":
		return 1000
	case "PLATINUM":
		return 3000
	case "DIAMOND":
		return 8000
	case "MYTHIC":
		return 20000
	}
	return 0
}

func pcardDrawRankProgress(dc *gg.Context, x, y, w float64, perf *userPerf, rank pcardRank, pal pcardPalette) {
	nextName, nextNeed := pcardNextRankThreshold(rank)
	if nextName == "" {
		pcardLoadFont(dc, 12)
		dc.SetRGBA(1, 1, 1, 0.5)
		dc.DrawString(rank.Name+" · MAX RANK", x, y+10)
		return
	}
	floor := pcardCurrentRankFloor(rank)
	span := float64(nextNeed - floor)
	progress := float64(perf.TotalMsgs - floor)
	if progress < 0 {
		progress = 0
	}
	if progress > span {
		progress = span
	}
	pct := 0.0
	if span > 0 {
		pct = progress / span
	}

	pcardLoadFont(dc, 12)
	dc.SetRGBA(1, 1, 1, 0.45)
	label := fmt.Sprintf("%s → %s   %d / %d", rank.Name, nextName, perf.TotalMsgs, nextNeed)
	dc.DrawString(label, x, y+10)

	barY := y + 18
	barH := 6.0
	dc.SetRGBA(1, 1, 1, 0.09)
	dc.DrawRoundedRectangle(x, barY, w, barH, barH/2)
	dc.Fill()
	fillW := w * pct
	if fillW < barH {
		fillW = barH
	}
	dc.SetRGBA255(int(rank.Color.R), int(rank.Color.G), int(rank.Color.B), 220)
	dc.DrawRoundedRectangle(x, barY, fillW, barH, barH/2)
	dc.Fill()
	_ = pal
}

func pcardDrawTopCommands(dc *gg.Context, x, y, w float64, perf *userPerf, pal pcardPalette) {
	pcardLoadFont(dc, 12)
	dc.SetRGBA(1, 1, 1, 0.38)
	dc.DrawString("TOP COMMANDS", x, y+10)

	if perf == nil || len(perf.Commands) == 0 {
		pcardLoadFont(dc, 13)
		dc.SetRGBA(1, 1, 1, 0.32)
		dc.DrawString("no commands used yet", x+130, y+10)
		return
	}
	type kv struct {
		K string
		V int64
	}
	all := make([]kv, 0, len(perf.Commands))
	for k, v := range perf.Commands {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].V > all[j].V })
	n := 3
	if len(all) < n {
		n = len(all)
	}
	cx := x + 130
	pcardLoadFont(dc, 13)
	for i := 0; i < n; i++ {
		entry := fmt.Sprintf("/%s ×%d", all[i].K, all[i].V)
		if i > 0 {
			dc.SetRGBA(1, 1, 1, 0.2)
			dc.DrawString("·", cx, y+10)
			cx += 12
		}
		dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 220)
		ew, _ := dc.MeasureString(entry)
		if cx+ew > x+w-4 {
			break
		}
		dc.DrawString(entry, cx, y+10)
		cx += ew + 8
	}
}

func pcardDrawGrain(dc *gg.Context, w, h int, userID int64) {
	seed := pcardHash(userID, "grain")
	dots := 320
	for i := 0; i < dots; i++ {
		sx := math.Mod(float64((seed>>uint(i%32))%uint64(w))+float64(i*11%w), float64(w))
		sy := math.Mod(float64((seed>>uint((i*5)%32))%uint64(h))+float64(i*13%h), float64(h))
		alpha := 6 + (i*3)%14
		dc.SetRGBA255(255, 255, 255, alpha)
		dc.DrawPoint(sx, sy, 0.5)
		dc.Fill()
	}
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
	perf := UserPerfGet(info.UserID)
	rank := pcardRankFor(perf)
	auraColor := pcardAuraColorFor(info.UserID)

	bg := gg.NewLinearGradient(0, 0, float64(W), float64(H))
	bg.AddColorStop(0, pal.BgTop)
	bg.AddColorStop(0.55, pal.BgMid)
	bg.AddColorStop(1, pal.BgBot)
	dc.SetFillStyle(bg)
	dc.DrawRectangle(0, 0, W, H)
	dc.Fill()

	starSeed := pcardHash(info.UserID, "stars")
	for i := 0; i < 22; i++ {
		sx := math.Mod(float64((starSeed>>uint(i%32))%uint64(W))+float64(i*17%W), float64(W))
		sy := math.Mod(float64((starSeed>>uint((i*3)%32))%uint64(H))+float64(i*29%H), float64(H))
		rad := 0.6 + float64(i%4)*0.25
		alpha := 18 + (i*7)%25
		dc.SetRGBA255(255, 255, 255, alpha)
		dc.DrawCircle(sx, sy, rad)
		dc.Fill()
	}

	const (
		cardPadX = 44.0
		cardPadY = 44.0
	)
	cardX := cardPadX
	cardY := cardPadY
	cardW := float64(W) - cardPadX*2
	cardH := float64(H) - cardPadY*2
	cardR := 24.0

	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 90)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(cardX-10, cardY-10, cardW+20, cardH+20, cardR+4)
	dc.Stroke()

	dc.SetRGBA255(16, 20, 30, 245)
	dc.DrawRoundedRectangle(cardX, cardY, cardW, cardH, cardR)
	dc.Fill()

	holo := gg.NewLinearGradient(cardX, cardY, cardX+cardW, cardY+cardH)
	holo.AddColorStop(0, color.RGBA{pal.Accent.R, pal.Accent.G, pal.Accent.B, 160})
	holo.AddColorStop(0.5, color.RGBA{pal.Accent2.R, pal.Accent2.G, pal.Accent2.B, 90})
	holo.AddColorStop(1, color.RGBA{auraColor.R, auraColor.G, auraColor.B, 160})
	dc.SetFillStyle(holo)
	dc.SetLineWidth(2.5)
	dc.DrawRoundedRectangle(cardX, cardY, cardW, cardH, cardR)
	dc.SetStrokeStyle(holo)
	dc.Stroke()

	dc.SetRGBA(1, 1, 1, 0.05)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(cardX+7, cardY+7, cardW-14, cardH-14, cardR-6)
	dc.Stroke()

	pcardDrawCornerBrackets(dc, cardX, cardY, cardW, cardH, pal)

	sidebarW := 330.0
	sidebarX := cardX
	dc.SetRGBA255(255, 255, 255, 4)
	dc.DrawRoundedRectangle(sidebarX+14, cardY+14, sidebarW-14, cardH-28, cardR-4)
	dc.Fill()

	dc.SetRGBA255(255, 255, 255, 18)
	dc.SetLineWidth(1)
	dc.DrawLine(sidebarX+sidebarW, cardY+40, sidebarX+sidebarW, cardY+cardH-40)
	dc.Stroke()

	avCX := sidebarX + sidebarW/2
	avCY := cardY + 178
	avR := 96.0

	if avatarPath != "" {
		pcardDrawAvatarRing(dc, avCX, avCY, avR, pal, avatarPath)
	} else {
		dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 30)
		dc.DrawCircle(avCX, avCY, avR+14)
		dc.Fill()
		dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 220)
		dc.SetLineWidth(3)
		dc.DrawCircle(avCX, avCY, avR+6)
		dc.Stroke()
		name := strings.TrimSpace(info.FirstName + " " + info.LastName)
		pcardDrawInitialsAvatar(dc, avCX, avCY, avR, pcardInitials(name), pal)
	}

	serialY := avCY + avR + 32
	pcardLoadFont(dc, 12)
	dc.SetRGBA(1, 1, 1, 0.35)
	serial := pcardSerialFor(info.UserID)
	sw, _ := dc.MeasureString(serial)
	dc.DrawString(serial, avCX-sw/2, serialY)

	sidebarInnerX := sidebarX + 34
	infoY := serialY + 38

	pcardDrawLabel(dc, "USER ID", sidebarInnerX, infoY)
	pcardLoadFont(dc, 22)
	dc.SetRGB(1, 1, 1)
	dc.DrawString(strconv.FormatInt(info.UserID, 10), sidebarInnerX, infoY+30)

	infoY += 72
	pcardDrawLabel(dc, "MEMBER SINCE", sidebarInnerX, infoY)
	pcardLoadFont(dc, 22)
	dc.SetRGB(1, 1, 1)
	dc.DrawString(pcardMemberSince(info.UserID), sidebarInnerX, infoY+30)

	infoY += 72
	pcardDrawLabel(dc, "ACTIVE IN", sidebarInnerX, infoY)
	pcardLoadFont(dc, 22)
	dc.SetRGB(1, 1, 1)
	dc.DrawString(fmt.Sprintf("%d chats", len(perf.Chats)), sidebarInnerX, infoY+30)

	infoY += 72
	pcardDrawLabel(dc, "PEAK HOUR", sidebarInnerX, infoY)
	pcardLoadFont(dc, 22)
	dc.SetRGB(1, 1, 1)
	peakLabel, peakHour := pcardPeakHourLabel(perf)
	dc.DrawString(peakLabel, sidebarInnerX, infoY+30)

	contentX := sidebarX + sidebarW + 34
	contentRight := cardX + cardW - 34
	contentW := contentRight - contentX

	pcardLoadFont(dc, 11)
	dc.SetRGBA(1, 1, 1, 0.35)
	wm := "· JULIABOT ID CARD ·"
	wmW, _ := dc.MeasureString(wm)
	dc.DrawString(wm, contentRight-wmW, cardY+34)

	displayName := pcardStripEmoji(strings.TrimSpace(info.FirstName + " " + info.LastName))
	if displayName == "" {
		displayName = "User"
	}
	titleY := cardY + 130
	nameSize := 62.0
	pcardLoadFont(dc, nameSize)
	for {
		w, _ := dc.MeasureString(displayName)
		if w <= contentW-20 || nameSize <= 30 {
			break
		}
		nameSize -= 4
		pcardLoadFont(dc, nameSize)
	}
	dc.SetRGB(1, 1, 1)
	dc.DrawString(displayName, contentX, titleY)

	nameUnderY := titleY + 10
	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 220)
	dc.SetLineWidth(2)
	dc.DrawLine(contentX, nameUnderY, contentX+42, nameUnderY)
	dc.Stroke()

	if uname := pcardStripEmoji(info.Username); uname != "" {
		pcardLoadFont(dc, 22)
		dc.SetRGBA(1, 1, 1, 0.55)
		dc.DrawString("@"+uname, contentX, titleY+36)
	}

	pillH := 40.0
	pillR := pillH / 2
	badgeY := titleY + 62

	title := pcardTitleFor(info.UserID)
	pcardLoadFont(dc, 20)
	titleTextW, _ := dc.MeasureString(title)
	titlePillW := titleTextW + pillH

	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 55)
	dc.DrawRoundedRectangle(contentX, badgeY, titlePillW, pillH, pillR)
	dc.Fill()
	dc.SetRGBA255(int(auraColor.R), int(auraColor.G), int(auraColor.B), 255)
	dc.DrawCircle(contentX+pillR-4, badgeY+pillH/2, 5)
	dc.Fill()
	dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 255)
	dc.DrawString(title, contentX+pillR+6, badgeY+pillH/2+7)

	rankGap := 10.0
	rankX := contentX + titlePillW + rankGap
	pcardLoadFont(dc, 18)
	rankTextW, _ := dc.MeasureString(rank.Name)
	rankPillW := rankTextW + pillH
	dc.SetRGBA255(int(rank.Color.R), int(rank.Color.G), int(rank.Color.B), 45)
	dc.DrawRoundedRectangle(rankX, badgeY, rankPillW, pillH, pillR)
	dc.Fill()
	dc.SetRGBA255(int(rank.Color.R), int(rank.Color.G), int(rank.Color.B), 180)
	dc.SetLineWidth(1.5)
	dc.DrawRoundedRectangle(rankX, badgeY, rankPillW, pillH, pillR)
	dc.Stroke()
	dc.SetRGBA255(int(rank.Color.R), int(rank.Color.G), int(rank.Color.B), 240)
	dc.DrawString(rank.Name, rankX+pillR, badgeY+pillH/2+6)

	sparkY := badgeY + pillH + 22
	sparkH := 46.0
	pcardDrawSparkline(dc, contentX, sparkY, contentW, sparkH, perf, pal)

	statsX := contentX
	statsW := contentW
	statsY := sparkY + sparkH + 20
	statsH := 176.0

	dc.SetRGBA255(255, 255, 255, 8)
	dc.DrawRoundedRectangle(statsX, statsY, statsW, statsH, 16)
	dc.Fill()
	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 60)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(statsX, statsY, statsW, statsH, 16)
	dc.Stroke()

	pcardDrawLabel(dc, "STATS", statsX+22, statsY+26)
	provisional := !pcardHasRealData(perf)
	if provisional {
		pcardLoadFont(dc, 11)
		dc.SetRGBA(1, 1, 1, 0.35)
		tag := "PROVISIONAL · " + strconv.FormatInt(perf.TotalMsgs, 10) + "/25"
		tagW, _ := dc.MeasureString(tag)
		dc.DrawString(tag, statsX+statsW-tagW-22, statsY+26)
	}

	stats := pcardStatsFor(info.UserID)
	statLeft := statsX + 22
	statRight := statsX + statsW - 22
	barW := statRight - statLeft
	rowH := 38.0
	firstRowY := statsY + 56
	for i, s := range stats {
		y := firstRowY + float64(i)*rowH
		pcardDrawStatBar(dc, statLeft, y, barW, 12, s.Name, s.Value, pal)
	}

	progY := statsY + statsH + 18
	pcardDrawRankProgress(dc, statsX, progY, statsW, perf, rank, pal)

	topCmdsY := progY + 42
	pcardDrawTopCommands(dc, statsX, topCmdsY, statsW, perf, pal)

	badges := pcardBadgesFor(perf)
	if len(badges) > 0 {
		badgeRowY := cardY + cardH - 46
		bx := statsX
		pcardLoadFont(dc, 13)
		for _, b := range badges {
			bw, _ := dc.MeasureString(b)
			padW := bw + 22
			if bx+padW > cardX+cardW-32 {
				break
			}
			dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 50)
			dc.DrawRoundedRectangle(bx, badgeRowY, padW, 24, 12)
			dc.Fill()
			dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 150)
			dc.SetLineWidth(1)
			dc.DrawRoundedRectangle(bx, badgeRowY, padW, 24, 12)
			dc.Stroke()
			dc.SetRGB(1, 1, 1)
			dc.DrawString(b, bx+11, badgeRowY+17)
			bx += padW + 8
		}
		_ = peakHour
	}

	pcardDrawGrain(dc, W, H, info.UserID)

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
	caption += fmt.Sprintf("\n<i>%s</i>", html.EscapeString(pcardTitleFor(info.UserID)))

	if status != nil {
		_, _ = status.Delete()
	}

	if _, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		Caption:  caption,
		FileName: fmt.Sprintf("card_%d.png", info.UserID),
		MimeType: "image/png",
	}); merr != nil {
		m.Reply("upload failed: " + html.EscapeString(merr.Error()))
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
