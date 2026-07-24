package extras

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

// ============================================================
// JuliaBot CTF challenge  (revert after event: delete this file)
// ============================================================
//
// The chain (nothing here is user-visible from source unless they read Go):
//   1. /help  →  spot the "Owl" module (name is out-of-family).
//   2. Owl help text describes a "watch": sha1("owl|" + botusername)[:8].
//      Whisper it → bot sends an owl PNG (no caption).
//   3. Inspect the PNG:
//        - Left eye pixel color  = first 3 bytes of half-A (6 hex chars)
//        - Right eye pixel color = last  3 bytes of half-A (6 hex chars)
//        - PNG tEXt chunk: "hint = numbers 0..7"
//   4. Send integer messages "0" .. "7". Bot reacts to each with one of
//      16 emoji whose index encodes one hex nibble of half-B.
//      Two reactions per byte, so 8 reactions = 4 bytes.
//   5. Assemble key = half-A || half-B (8 bytes / 16 hex chars).
//   6. .owl <16-hex> → bot XOR-decrypts the flag and prints it.

// Flag: flag{th3_0w1_s33s_wh4t_th3_pix3ls_h1de_1n_c010r_&_react_2u}
// Encoded below with a per-position keystream = sha256("owl-flag|" + i).
// Static — not grep-able as flag{...}.

var owlFlagObf = []byte{
	// generated at build design; matches decoder logic below
}

// key = eight bytes:
//   half-A (4 bytes): eyeLeft{R,G,B} || eyeRight{R}
//   half-B (4 bytes): from reactions
// We PICK the key deliberately so the two eye colors look plausible.
//   half-A = 0xDE, 0xAD, 0xBE, 0xEF   →  eyes will be #DEADBE and #EF----
//   half-B = 0xC0, 0xFF, 0xEE, 0x42
var owlKeyHalfA = [4]byte{0xDE, 0xAD, 0xBE, 0xEF}
var owlKeyHalfB = [4]byte{0xC0, 0xFF, 0xEE, 0x42}

// The plaintext flag — encoded into owlFlagObf via init().
const owlFlagPlaintext = "flag{th3_0w1_s33s_wh4t_th3_pix3ls_h1de_1n_c010r_&_react_2u}"

func init() {
	full := append([]byte{}, owlKeyHalfA[:]...)
	full = append(full, owlKeyHalfB[:]...)
	// key-derived keystream
	ks := owlKeystream(hex.EncodeToString(full), len(owlFlagPlaintext))
	owlFlagObf = make([]byte, len(owlFlagPlaintext))
	for i := range owlFlagPlaintext {
		owlFlagObf[i] = owlFlagPlaintext[i] ^ ks[i]
	}
	modules.QueueHandlerRegistration(owlRegisterHandlers)
}

func owlKeystream(seed string, n int) []byte {
	out := make([]byte, 0, n)
	for i := 0; len(out) < n; i++ {
		h := sha256.Sum256([]byte(fmt.Sprintf("owl-flag|%s|%d", seed, i)))
		out = append(out, h[:]...)
	}
	return out[:n]
}

func owlDecryptWithKey(keyHex string) (string, bool) {
	if len(keyHex) != 16 {
		return "", false
	}
	key, err := hex.DecodeString(strings.ToLower(keyHex))
	if err != nil {
		return "", false
	}
	ks := owlKeystream(hex.EncodeToString(key), len(owlFlagObf))
	out := make([]byte, len(owlFlagObf))
	for i := range owlFlagObf {
		out[i] = owlFlagObf[i] ^ ks[i]
	}
	s := string(out)
	if !strings.HasPrefix(s, "flag{") || !strings.HasSuffix(s, "}") {
		return "", false
	}
	return s, true
}

// ---- Owl discovery layer (unchanged) ----

func owlWatch(username string) string {
	h := sha1.Sum([]byte("owl|" + strings.ToLower(strings.TrimSpace(username))))
	return hex.EncodeToString(h[:])[:8]
}

var owlHelpText = `<b>Owl</b>
<i>night telemetry — internal</i>

The <b>Owl</b> watches over telemetry when no operator is present.
Its channel is silent by day and speaks only through echoes.

Owl accepts one greeting: <b>whisper the current watch</b>.
It is <code>8 hex</code> characters, derived from the caretaker's <b>handle</b>
via a single well-known one-way function.

<i>Reply to this message with the watch. Only the daughters hear.</i>`

func isOwlWatchShaped(s string) bool {
	if len(s) != 8 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// Bot reacts with one of these 16, index = hex nibble value.
// Ordered to look like natural "owl phase" emoji — but the ORDER is the code.
var owlEmojiAlphabet = []string{
	"🌑", "🌒", "🌓", "🌔", // 0-3
	"🌕", "🌖", "🌗", "🌘", // 4-7
	"🦉", "🌙", "⭐", "🌌", // 8-B
	"🪶", "🕯", "🔥", "🕸", // C-F
}

// Half-B nibble for numeric input N in "0".."7".
// Returns nibble (0..15) determined by owlKeyHalfB and a shuffle so
// hex order isn't the same as message order.
func owlNibbleForNumber(n int) int {
	if n < 0 || n > 7 {
		return -1
	}
	// half-B has 8 nibbles (indexed 0..7). Serve them in a small permutation
	// so players can't blindly assume "message 0 = first nibble".
	perm := []int{3, 0, 5, 2, 7, 4, 1, 6}
	nib := perm[n]
	byteIdx := nib / 2
	if nib%2 == 0 {
		return int(owlKeyHalfB[byteIdx] >> 4)
	}
	return int(owlKeyHalfB[byteIdx] & 0x0F)
}

// Owl replies with a hex-8 challenge check.
func owlReplyWatcher(m *tg.NewMessage) error {
	txt := strings.TrimSpace(strings.ToLower(m.Text()))
	if !isOwlWatchShaped(txt) {
		return nil
	}
	me, _ := m.Client.GetMe()
	if me == nil || me.Username == "" {
		return nil
	}
	if txt != owlWatch(me.Username) {
		return nil
	}
	sendOwlImage(m)
	return nil
}

func sendOwlImage(m *tg.NewMessage) {
	imgBytes, err := renderOwlPNG(owlKeyHalfA)
	if err != nil {
		m.Reply("<i>the owl is silent tonight.</i>")
		return
	}
	// write to temp file so Telegram treats it as a proper photo
	tmp, err := os.CreateTemp("", "owl_*.png")
	if err != nil {
		m.Reply("<i>the owl is silent tonight.</i>")
		return
	}
	if _, err := tmp.Write(imgBytes); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		m.Reply("<i>the owl is silent tonight.</i>")
		return
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	m.ReplyMedia(tmp.Name(), &tg.MediaOptions{
		FileName: "owl.png",
		MimeType: "image/png",
	})
}

// Numeric message handler: bot silently reacts with a phase emoji.
func owlNumberReactor(m *tg.NewMessage) error {
	txt := strings.TrimSpace(m.Text())
	if len(txt) == 0 || len(txt) > 2 {
		return nil
	}
	n, err := strconv.Atoi(txt)
	if err != nil || n < 0 || n > 7 {
		return nil
	}
	nib := owlNibbleForNumber(n)
	if nib < 0 {
		return nil
	}
	_ = m.React(owlEmojiAlphabet[nib])
	return nil
}

// Final decoder: .owl <16-hex>
func owlDecodeHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(strings.ToLower(m.Args()))
	if arg == "" {
		return nil
	}
	flag, ok := owlDecryptWithKey(arg)
	if !ok {
		_ = m.React("🌑")
		return nil
	}
	m.Reply("<b>the owl closes its eyes.</b>\n<code>" + flag + "</code>")
	return nil
}

// ---- PNG rendering with encoded eyes + tEXt hint ----

func renderOwlPNG(halfA [4]byte) ([]byte, error) {
	const W, H = 512, 512
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// gradient background: deep-night sky, brighter near center
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			dx := float64(x-W/2) / float64(W/2)
			dy := float64(y-H/2) / float64(H/2)
			d := dx*dx + dy*dy
			if d > 1 {
				d = 1
			}
			r := uint8(8 + int(18*(1-d)))
			g := uint8(10 + int(16*(1-d)))
			b := uint8(22 + int(35*(1-d)))
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// faint scattered "stars" — deterministic pseudo-random
	starRnd := uint32(0xB16B00B5)
	for i := 0; i < 140; i++ {
		starRnd = starRnd*1664525 + 1013904223
		sx := int(starRnd % uint32(W))
		starRnd = starRnd*1664525 + 1013904223
		sy := int(starRnd % uint32(H))
		starRnd = starRnd*1664525 + 1013904223
		br := 100 + int(starRnd%156)
		if (sx-W/2)*(sx-W/2)+(sy-H/2)*(sy-H/2) < 180*180 {
			continue // don't put stars over the owl
		}
		img.Set(sx, sy, color.RGBA{R: uint8(br), G: uint8(br), B: uint8(br + 20), A: 255})
	}

	cx, cy := W/2, H/2

	// body (below head) — teardrop-ish
	bodyCol := color.RGBA{R: 60, G: 45, B: 40, A: 255}
	for y := cy; y < H; y++ {
		widthAtY := int(140.0 * (1.0 - float64(y-cy)/300.0*0.4))
		if widthAtY < 20 {
			continue
		}
		for x := cx - widthAtY; x <= cx+widthAtY; x++ {
			// oval edge
			dx := float64(x - cx)
			dy := float64(y-cy) * 1.6
			if dx*dx+dy*dy < float64(widthAtY*widthAtY)*1.2 {
				img.Set(x, y, bodyCol)
			}
		}
	}

	// belly speckles / feathers
	speckleRnd := uint32(0x0FFEE)
	for i := 0; i < 220; i++ {
		speckleRnd = speckleRnd*22695477 + 1
		px := cx + int(speckleRnd%160) - 80
		speckleRnd = speckleRnd*22695477 + 1
		py := cy + 30 + int(speckleRnd%230)
		if px < 0 || px >= W || py < 0 || py >= H {
			continue
		}
		r, g, b, a := img.At(px, py).RGBA()
		if a == 0 {
			continue
		}
		if uint8(r>>8) == bodyCol.R && uint8(g>>8) == bodyCol.G && uint8(b>>8) == bodyCol.B {
			shade := color.RGBA{R: 90, G: 70, B: 60, A: 255}
			drawDisk(img, px, py, 2, shade)
		}
	}

	// head — big round with slight shading
	rHead := 130
	headCol := color.RGBA{R: 78, G: 60, B: 55, A: 255}
	headShadow := color.RGBA{R: 50, G: 38, B: 35, A: 255}
	for y := cy - rHead; y <= cy+rHead; y++ {
		for x := cx - rHead; x <= cx+rHead; x++ {
			dx := x - cx
			dy := y - cy
			d := dx*dx + dy*dy
			if d < rHead*rHead {
				// darker at bottom-right (shadow)
				if dx+dy > rHead/2 {
					img.Set(x, y, headShadow)
				} else {
					img.Set(x, y, headCol)
				}
			}
		}
	}

	// ear tufts — pointy triangles at top
	for y := cy - rHead - 40; y < cy-rHead+30; y++ {
		width := (y - (cy - rHead - 40)) / 3
		for dx := -width; dx <= width; dx++ {
			img.Set(cx-70+dx, y, headCol)
			img.Set(cx+70+dx, y, headCol)
		}
	}

	// facial disc — pale heart shape
	disc := color.RGBA{R: 220, G: 210, B: 190, A: 255}
	for y := cy - 60; y < cy+60; y++ {
		for x := cx - 90; x < cx+90; x++ {
			dx := float64(x - cx)
			dy := float64(y-cy) * 1.1
			if dx*dx+dy*dy < 85*85 {
				img.Set(x, y, disc)
			}
		}
	}

	// small horizontal "beak-line" splits between the eye discs
	for y := cy - 5; y < cy+5; y++ {
		for x := cx - 8; x <= cx+8; x++ {
			img.Set(x, y, color.RGBA{R: 90, G: 60, B: 30, A: 255})
		}
	}

	// EYE SOCKETS — big pale circles carved into the facial disc, ringed
	socket := color.RGBA{R: 240, G: 235, B: 220, A: 255}
	drawDisk(img, cx-45, cy-15, 42, socket)
	drawDisk(img, cx+45, cy-15, 42, socket)

	// ring around sockets (darker)
	ringR := 42
	ringCol := color.RGBA{R: 40, G: 30, B: 30, A: 255}
	drawRing(img, cx-45, cy-15, ringR, 2, ringCol)
	drawRing(img, cx+45, cy-15, ringR, 2, ringCol)

	// EYES — three concentric disks so it looks like an actual eye,
	// but the OUTER iris colors carry the key bytes verbatim.
	leftEye := color.RGBA{R: halfA[0], G: halfA[1], B: halfA[2], A: 255}
	rightEye := color.RGBA{R: halfA[3], G: halfA[1] ^ halfA[0], B: halfA[2] ^ halfA[3], A: 255}
	drawDisk(img, cx-45, cy-15, 24, leftEye)
	drawDisk(img, cx+45, cy-15, 24, rightEye)

	// pupils (black) + tiny catch-light
	pupil := color.RGBA{R: 8, G: 8, B: 12, A: 255}
	drawDisk(img, cx-45, cy-15, 10, pupil)
	drawDisk(img, cx+45, cy-15, 10, pupil)
	catch := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	drawDisk(img, cx-42, cy-18, 3, catch)
	drawDisk(img, cx+48, cy-18, 3, catch)

	// beak — orange downward triangle, shaded
	beakTop := cy + 15
	beakBottom := cy + 55
	for y := beakTop; y < beakBottom; y++ {
		w := (beakBottom - y) * 3 / 4
		for x := cx - w; x <= cx+w; x++ {
			shadeR := uint8(215)
			shadeG := uint8(130)
			shadeB := uint8(50)
			if x > cx {
				shadeR = 190
				shadeG = 110
				shadeB = 40
			}
			img.Set(x, y, color.RGBA{R: shadeR, G: shadeG, B: shadeB, A: 255})
		}
	}

	// signature: faint text row at bottom
	// (not strictly needed for the chal but adds polish)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return injectTextChunk(buf.Bytes(), "hint", "numbers 0..7 speak to me"), nil
}

func drawRing(img *image.RGBA, cx, cy, r, thickness int, c color.RGBA) {
	rSq := r * r
	rInner := (r - thickness)
	rInnerSq := rInner * rInner
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			d := dx*dx + dy*dy
			if d <= rSq && d >= rInnerSq {
				img.Set(x, y, c)
			}
		}
	}
}

func drawDisk(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, c)
			}
		}
	}
}

// injectTextChunk inserts a PNG tEXt chunk right after the IHDR chunk.
// Format of a tEXt chunk (RFC): 4-byte length + "tEXt" + keyword + 0x00 + text + 4-byte CRC.
func injectTextChunk(pngBytes []byte, keyword, text string) []byte {
	// PNG signature is 8 bytes; IHDR chunk is next: 4(len)+4(type)+13(data)+4(crc) = 25 bytes.
	const sig = 8
	const ihdrEnd = sig + 25
	if len(pngBytes) < ihdrEnd {
		return pngBytes
	}
	payload := append([]byte(keyword), 0x00)
	payload = append(payload, []byte(text)...)
	chunkType := []byte("tEXt")
	crcInput := append([]byte{}, chunkType...)
	crcInput = append(crcInput, payload...)
	crc := crc32.ChecksumIEEE(crcInput)

	chunk := make([]byte, 0, 12+len(payload))
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(payload)))
	chunk = append(chunk, lenBuf...)
	chunk = append(chunk, chunkType...)
	chunk = append(chunk, payload...)
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, crc)
	chunk = append(chunk, crcBuf...)

	out := make([]byte, 0, len(pngBytes)+len(chunk))
	out = append(out, pngBytes[:ihdrEnd]...)
	out = append(out, chunk...)
	out = append(out, pngBytes[ihdrEnd:]...)
	return out
}

// ---- registration ----

func owlRegisterHandlers() {
	c := modules.Client
	c.On("cmd:owl", owlDecodeHandler)
	c.On("message:.*", owlReplyWatcher)
	c.On("message:.*", owlNumberReactor)
	modules.Mods.AddModule("Owl", owlHelpText)
}
