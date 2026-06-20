package modules

import (
	"fmt"
	"html"
	"image/color"
	"math/rand"
	"os"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func randomColorPick() color.RGBA {
	return color.RGBA{
		R: uint8(rand.Intn(256)),
		G: uint8(rand.Intn(256)),
		B: uint8(rand.Intn(256)),
		A: 0xff,
	}
}

func RandColorHandler(m *tg.NewMessage) error {
	c := randomColorPick()

	status, _ := m.Reply("<code>rolling color...</code>")

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

	hex := colorsHexFromRGB(c)
	h, s, l := colorsRGBToHSL(c)
	nearest := colorsNearest(c, 1)

	var b strings.Builder
	b.WriteString("<b>Random Color</b>\n")
	b.WriteString(fmt.Sprintf("hex   <code>%s</code>\n", hex))
	b.WriteString(fmt.Sprintf("rgb   <code>%d, %d, %d</code>\n", c.R, c.G, c.B))
	b.WriteString(fmt.Sprintf("hsl   <code>%.0f, %.0f%%, %.0f%%</code>\n", h, s, l))
	if len(nearest) > 0 {
		b.WriteString(fmt.Sprintf("name  <code>%s</code> <i>(Δ %.0f)</i>", html.EscapeString(nearest[0].Name), nearest[0].Dist))
	}

	if status != nil {
		status.Delete()
	}

	_, merr := m.ReplyMedia(out, &tg.MediaOptions{
		Caption:  b.String(),
		FileName: "randcolor.png",
		MimeType: "image/png",
	})
	os.Remove(out)
	if merr != nil {
		m.Reply("upload failed: " + html.EscapeString(merr.Error()))
	}
	return nil
}

func registerRandomColorHandlers() {
	c := Client
	c.On("cmd:randcolor", RandColorHandler)
}

func init() {
	QueueHandlerRegistration(registerRandomColorHandlers)
}
