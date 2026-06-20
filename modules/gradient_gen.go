package modules

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

func gradientParseHex(s string) (int, int, int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex %q", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid hex %q", s)
	}
	r := int((v >> 16) & 0xff)
	g := int((v >> 8) & 0xff)
	b := int(v & 0xff)
	return r, g, b, nil
}

func gradientLerp(a, b float64, t float64) float64 {
	return a*(1-t) + b*t
}

func gradientDrawHorizontal(dc *gg.Context, w, h int, stops [][3]int) {
	n := len(stops)
	if n < 2 {
		return
	}
	segs := n - 1
	for x := 0; x < w; x++ {
		t := float64(x) / float64(w-1)
		pos := t * float64(segs)
		idx := int(pos)
		if idx >= segs {
			idx = segs - 1
		}
		local := pos - float64(idx)
		c1 := stops[idx]
		c2 := stops[idx+1]
		r := gradientLerp(float64(c1[0]), float64(c2[0]), local)
		g := gradientLerp(float64(c1[1]), float64(c2[1]), local)
		b := gradientLerp(float64(c1[2]), float64(c2[2]), local)
		dc.SetRGB255(int(r), int(g), int(b))
		dc.DrawRectangle(float64(x), 0, 1, float64(h))
		dc.Fill()
	}
}

func gradientRender(stops [][3]int) (string, error) {
	const w, h = 1280, 256
	dc := gg.NewContext(w, h)
	gradientDrawHorizontal(dc, w, h, stops)
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("gradient_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func gradientFormatCaption(stops [][3]int, raw []string) string {
	parts := make([]string, len(raw))
	for i, h := range raw {
		s := strings.TrimSpace(h)
		s = strings.TrimPrefix(s, "#")
		parts[i] = "#" + strings.ToUpper(s)
	}
	return "<b>Gradient</b>\n<code>" + html.EscapeString(strings.Join(parts, " -> ")) + "</code>"
}

func gradientHandle(m *tg.NewMessage, want int) error {
	args := strings.Fields(strings.TrimSpace(m.Args()))
	if len(args) < want {
		usage := fmt.Sprintf("usage: <code>/gradient%s %s</code>", map[int]string{2: "", 3: "3"}[want], strings.Repeat("&lt;hex&gt; ", want))
		m.Reply(strings.TrimSpace(usage))
		return nil
	}
	args = args[:want]
	stops := make([][3]int, 0, want)
	for _, a := range args {
		r, g, b, err := gradientParseHex(a)
		if err != nil {
			m.Reply("error: " + html.EscapeString(err.Error()))
			return nil
		}
		stops = append(stops, [3]int{r, g, b})
	}

	status, _ := m.Reply("<code>generating gradient...</code>")

	outPath, err := gradientRender(stops)
	if err != nil {
		msg := "error rendering: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	caption := gradientFormatCaption(stops, args)

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		Caption:  caption,
		FileName: "gradient.png",
		MimeType: "image/png",
	})
	os.Remove(outPath)
	if merr != nil {
		m.Reply("error sending: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func GradientHandler(m *tg.NewMessage) error {
	return gradientHandle(m, 2)
}

func Gradient3Handler(m *tg.NewMessage) error {
	return gradientHandle(m, 3)
}

func registerGradientGenHandlers() {
	c := Client
	c.On("cmd:gradient", GradientHandler)
	c.On("cmd:gradient3", Gradient3Handler)
}

func init() { QueueHandlerRegistration(registerGradientGenHandlers) }
