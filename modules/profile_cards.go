package modules

import (
	"fmt"
	"hash/fnv"
	"html"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

type pcardPalette struct {
	BgTop    color.RGBA
	BgMid    color.RGBA
	BgBot    color.RGBA
	Accent   color.RGBA
	Accent2  color.RGBA
	Surface  color.RGBA
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

	orbitSeed := pcardHash(userID, "orbit")
	orbitR := r + 32
	dots := 18
	for i := 0; i < dots; i++ {
		theta := float64(i)/float64(dots)*2*math.Pi + float64(orbitSeed%360)*math.Pi/180
		ox := cx + orbitR*math.Cos(theta)
		oy := cy + orbitR*math.Sin(theta)
		dotR := 2.5
		if i%3 == 0 {
			dotR = 4
		}
		alpha := 180
		if i%2 == 0 {
			alpha = 220
		}
		c := pal.Accent
		if i%2 == 0 {
			c = pal.Accent2
		}
		dc.SetRGBA255(int(c.R), int(c.G), int(c.B), alpha)
		dc.DrawCircle(ox, oy, dotR)
		dc.Fill()
	}

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

	pcardDrawBackground(dc, W, H, pal, info.UserID)

	panelX, panelY := 60.0, 60.0
	panelW, panelH := float64(W)-120, float64(H)-120
	pcardDrawGlassPanel(dc, panelX, panelY, panelW, panelH, 32, pal)

	avCX := panelX + 180
	avCY := panelY + 220
	avR := 100.0

	if avatarPath != "" {
		pcardDrawAvatarRing(dc, avCX, avCY, avR, pal, info.UserID, avatarPath)
	} else {
		for i := 0; i < 6; i++ {
			alpha := 25 - i*3
			if alpha < 4 {
				alpha = 4
			}
			dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), alpha)
			dc.DrawCircle(avCX, avCY, avR+18+float64(i)*4)
			dc.Fill()
		}
		name := strings.TrimSpace(info.FirstName + " " + info.LastName)
		pcardDrawInitialsAvatar(dc, avCX, avCY, avR, pcardInitials(name), pal)
		dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 255)
		dc.SetLineWidth(6)
		dc.DrawCircle(avCX, avCY, avR+10)
		dc.Stroke()
		dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 220)
		dc.SetLineWidth(2)
		dc.DrawCircle(avCX, avCY, avR+16)
		dc.Stroke()
		orbitSeed := pcardHash(info.UserID, "orbit")
		orbitR := avR + 32
		dots := 18
		for i := 0; i < dots; i++ {
			theta := float64(i)/float64(dots)*2*math.Pi + float64(orbitSeed%360)*math.Pi/180
			ox := avCX + orbitR*math.Cos(theta)
			oy := avCY + orbitR*math.Sin(theta)
			dotR := 2.5
			if i%3 == 0 {
				dotR = 4
			}
			c := pal.Accent
			if i%2 == 0 {
				c = pal.Accent2
			}
			dc.SetRGBA255(int(c.R), int(c.G), int(c.B), 220)
			dc.DrawCircle(ox, oy, dotR)
			dc.Fill()
		}
	}

	displayName := strings.TrimSpace(info.FirstName + " " + info.LastName)
	if displayName == "" {
		displayName = "User"
	}
	if len(displayName) > 26 {
		displayName = displayName[:26] + "..."
	}

	textX := avCX + avR + 60
	textY := panelY + 130

	pcardLoadFont(dc, 68)
	dc.SetRGB(1, 1, 1)
	dc.DrawString(displayName, textX, textY)

	textY += 50
	if info.Username != "" {
		pcardLoadFont(dc, 32)
		dc.SetRGBA(1, 1, 1, 0.55)
		dc.DrawString("@"+info.Username, textX, textY)
		textY += 36
	}

	pcardLoadFont(dc, 22)
	dc.SetRGBA(1, 1, 1, 0.45)
	dc.DrawString(fmt.Sprintf("ID  %d", info.UserID), textX, textY)
	textY += 30

	pcardLoadFont(dc, 22)
	dc.SetRGBA(1, 1, 1, 0.45)
	dc.DrawString("Member since  "+pcardMemberSince(info.UserID), textX, textY)
	textY += 50

	title := pcardTitleFor(info.UserID)
	aura := pcardAuraFor(info.UserID)

	pcardLoadFont(dc, 38)
	dc.SetRGBA255(int(pal.Accent2.R), int(pal.Accent2.G), int(pal.Accent2.B), 255)
	dc.DrawString(title, textX, textY)
	tw, _ := dc.MeasureString(title)

	pcardLoadFont(dc, 32)
	dc.SetRGB(1, 1, 1)
	dc.DrawString("  "+aura, textX+tw, textY)

	stats := pcardStatsFor(info.UserID)
	stripY := panelY + panelH - 200
	stripX := panelX + 50
	stripW := panelW - 100

	dc.SetRGBA(1, 1, 1, 0.04)
	dc.DrawRoundedRectangle(stripX, stripY, stripW, 160, 18)
	dc.Fill()
	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 80)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(stripX, stripY, stripW, 160, 18)
	dc.Stroke()

	pcardLoadFont(dc, 18)
	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 220)
	dc.DrawString("STATS", stripX+24, stripY+28)

	barX := stripX + 30
	barW := stripW - 60
	barY := stripY + 60
	for i, s := range stats {
		y := barY + float64(i)*32
		pcardDrawStatBar(dc, barX, y, barW, 14, s.Name, s.Value, pal)
	}

	pcardLoadFont(dc, 24)
	dc.SetRGB(1, 1, 1)
	wm := "JULIABOT"
	wmW, _ := dc.MeasureString(wm)
	wmX := panelX + panelW - wmW - 30
	wmY := panelY + 50
	dc.DrawString(wm, wmX, wmY)

	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 255)
	dc.SetLineWidth(3)
	dc.DrawLine(wmX, wmY+8, wmX+wmW, wmY+8)
	dc.Stroke()

	dc.SetRGBA255(int(pal.Accent.R), int(pal.Accent.G), int(pal.Accent.B), 255)
	dc.DrawCircle(wmX-12, wmY-6, 4)
	dc.Fill()

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("pcard_%d_%d.png", info.UserID, time.Now().UnixNano()))
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
	c := Client
	c.On("cmd:card", ProfileCardHandler)
}

func init() {
	QueueHandlerRegistration(registerProfileCardsHandlers)
}
