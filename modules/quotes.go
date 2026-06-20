package modules

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	_ "image/jpeg"
	"main/modules/db"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/image/font/basicfont"
)

var quotesBucket = []byte("quotes")

var quotesRng = rand.New(rand.NewSource(time.Now().UnixNano()))

type quoteRecord struct {
	ID          uint64 `json:"id"`
	ChatID      int64  `json:"chat_id"`
	UserID      int64  `json:"user_id"`
	UserName    string `json:"user_name"`
	UserHandle  string `json:"user_handle"`
	Text        string `json:"text"`
	SavedBy     int64  `json:"saved_by"`
	SavedByName string `json:"saved_by_name"`
	Timestamp   int64  `json:"ts"`
}

type quoteBlock struct {
	Name   string
	Handle string
	Text   string
	Avatar string
	UserID int64
	Color  color.RGBA
	Date   int64
}

func quotesAccentPalette() []color.RGBA {
	return []color.RGBA{
		{0xff, 0x6b, 0x6b, 0xff},
		{0x4e, 0xcd, 0xc4, 0xff},
		{0xff, 0xd9, 0x3d, 0xff},
		{0x95, 0xe1, 0xd3, 0xff},
		{0xf3, 0x8b, 0xa0, 0xff},
		{0xa8, 0xe6, 0xcf, 0xff},
		{0xff, 0x8b, 0x94, 0xff},
		{0xc7, 0x9e, 0xff, 0xff},
		{0x6c, 0x5c, 0xe7, 0xff},
		{0xfd, 0x79, 0xa8, 0xff},
		{0xfd, 0xcb, 0x6e, 0xff},
		{0x55, 0xef, 0xc4, 0xff},
		{0x74, 0xb9, 0xff, 0xff},
		{0xff, 0x76, 0x75, 0xff},
		{0xe1, 0x7e, 0xb5, 0xff},
		{0xff, 0xa6, 0x2b, 0xff},
		{0x8e, 0xd8, 0x1f, 0xff},
		{0x00, 0xcf, 0xc1, 0xff},
	}
}

func quotesPickAccent(userID int64) color.RGBA {
	pal := quotesAccentPalette()
	if userID == 0 {
		return pal[quotesRng.Intn(len(pal))]
	}
	return pal[int(uint64(userID))%len(pal)]
}

func quotesLoadFont(dc *gg.Context, size float64) bool {
	primary := memeFontPath("Inter_28pt-Bold.ttf")
	if primary != "" {
		if err := dc.LoadFontFace(primary, size); err == nil {
			return true
		}
	}
	fallback := memeFontPath("Swiss 721 Black Extended BT.ttf")
	if fallback != "" {
		if err := dc.LoadFontFace(fallback, size); err == nil {
			return true
		}
	}
	dc.SetFontFace(basicfont.Face7x13)
	return false
}

func quotesDrawCircleAvatar(dc *gg.Context, path string, cx, cy, radius float64, accent color.RGBA) {
	dc.Push()
	defer dc.Pop()

	if path == "" {
		quotesDrawInitialsAvatar(dc, cx, cy, radius, "?", accent)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		quotesDrawInitialsAvatar(dc, cx, cy, radius, "?", accent)
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		quotesDrawInitialsAvatar(dc, cx, cy, radius, "?", accent)
		return
	}
	b := img.Bounds()
	srcW := float64(b.Dx())
	srcH := float64(b.Dy())
	scale := (radius * 2) / srcW
	if srcH > srcW {
		scale = (radius * 2) / srcH
	}
	scaled := gg.NewContext(int(radius*2)+2, int(radius*2)+2)
	scaled.ScaleAbout(scale, scale, srcW/2, srcH/2)
	scaled.DrawImageAnchored(img, int(radius), int(radius), 0.5, 0.5)

	dc.DrawCircle(cx, cy, radius)
	dc.Clip()
	dc.DrawImageAnchored(scaled.Image(), int(cx), int(cy), 0.5, 0.5)
	dc.ResetClip()
}

func quotesDrawInitialsAvatar(dc *gg.Context, cx, cy, radius float64, initials string, accent color.RGBA) {
	dc.Push()
	defer dc.Pop()
	dc.DrawCircle(cx, cy, radius)
	dc.SetRGBA255(int(accent.R)/2, int(accent.G)/2, int(accent.B)/2, 255)
	dc.Fill()
	quotesLoadFont(dc, radius*0.9)
	dc.SetRGB(1, 1, 1)
	if initials == "" {
		initials = "?"
	}
	dc.DrawStringAnchored(strings.ToUpper(initials), cx, cy, 0.5, 0.5)
}

var quoteHTMLTags = regexp.MustCompile(`<[^>]+>`)

func quoteSanitizeText(s string) string {
	if s == "" {
		return ""
	}
	stripped := quoteHTMLTags.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(stripped))
}

func quotesWrapLines(dc *gg.Context, text string, maxWidth float64) []string {
	if text == "" {
		return nil
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
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
	}
	return lines
}

func quotesInitials(name string) string {
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
			return string(r[0])
		}
		return string(r[0]) + string(r[1])
	}
	a := []rune(parts[0])
	b := []rune(parts[len(parts)-1])
	if len(a) == 0 || len(b) == 0 {
		return "?"
	}
	return string(a[0]) + string(b[0])
}

func quotesGetAccessHash(c *tg.Client, userID int64) int64 {
	peer, err := c.ResolvePeer(userID)
	if err != nil {
		return 0
	}
	if pu, ok := peer.(*tg.InputPeerUser); ok {
		return pu.AccessHash
	}
	return 0
}

func quotesDownloadAvatar(c *tg.Client, userID int64) string {
	if userID == 0 {
		return ""
	}
	full, err := c.UsersGetFullUser(&tg.InputUserObj{
		UserID:     userID,
		AccessHash: quotesGetAccessHash(c, userID),
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
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("qavatar_%d_%d.jpg", userID, time.Now().UnixNano()))
	_, err = c.DownloadMedia(p, &tg.DownloadOptions{FileName: tmp})
	if err != nil {
		os.Remove(tmp)
		return ""
	}
	return tmp
}

func quotesCollectChain(m *tg.NewMessage, maxDepth int) []quoteBlock {
	var blocks []quoteBlock
	current, err := m.GetReplyMessage()
	if err != nil || current == nil {
		return blocks
	}
	for depth := 0; depth < maxDepth && current != nil; depth++ {
		text := quoteSanitizeText(current.RawText())
		if text == "" {
			text = "(no text)"
		}
		if len(text) > 600 {
			text = text[:600] + "..."
		}
		name := "User"
		handle := ""
		var userID int64
		if current.SenderID() != 0 {
			userID = current.SenderID()
			u, uerr := m.Client.GetUser(userID)
			if uerr == nil && u != nil {
				name = strings.TrimSpace(u.FirstName + " " + u.LastName)
				if name == "" {
					name = "User"
				}
				handle = u.Username
			}
		}
		avatar := quotesDownloadAvatar(m.Client, userID)
		blocks = append(blocks, quoteBlock{
			Name:   name,
			Handle: handle,
			Text:   text,
			Avatar: avatar,
			UserID: userID,
			Color:  quotesPickAccent(userID),
			Date:   int64(current.Date()),
		})
		if !current.IsReply() {
			break
		}
		next, nerr := current.GetReplyMessage()
		if nerr != nil || next == nil {
			break
		}
		current = next
	}
	return blocks
}

func quotesMeasureBlock(dc *gg.Context, block quoteBlock, contentW float64) (float64, float64, []string) {
	avatarSize := 96.0
	nameSize := 38.0
	handleSize := 22.0
	textSize := 40.0

	quotesLoadFont(dc, nameSize)
	_, nameH := dc.MeasureString(block.Name)
	if nameH == 0 {
		nameH = nameSize
	}

	headerH := nameH
	if block.Handle != "" {
		headerH = nameH + 6 + handleSize*1.2
	}
	if headerH < avatarSize {
		headerH = avatarSize
	}

	quotesLoadFont(dc, textSize)
	lines := quotesWrapLines(dc, block.Text, contentW)
	lineH := textSize * 1.35
	textH := lineH * float64(len(lines))
	if textH < lineH && len(lines) > 0 {
		textH = lineH
	}

	bodyGap := 24.0
	total := headerH + bodyGap + textH
	return total, headerH, lines
}

func quotesDrawBlock(dc *gg.Context, block quoteBlock, x, y, contentW float64) float64 {
	avatarSize := 96.0
	nameSize := 38.0
	handleSize := 22.0
	textSize := 40.0
	gapAvatarText := 22.0

	total, headerH, lines := quotesMeasureBlock(dc, block, contentW)

	avX := x + avatarSize/2
	avY := y + avatarSize/2
	if block.Avatar != "" {
		quotesDrawCircleAvatar(dc, block.Avatar, avX, avY, avatarSize/2, block.Color)
	} else {
		quotesDrawInitialsAvatar(dc, avX, avY, avatarSize/2, quotesInitials(block.Name), block.Color)
	}

	textX := x + avatarSize + gapAvatarText
	quotesLoadFont(dc, nameSize)
	_, nameH := dc.MeasureString(block.Name)
	if nameH == 0 {
		nameH = nameSize
	}
	nameY := y + nameH
	dc.SetRGBA255(int(block.Color.R), int(block.Color.G), int(block.Color.B), 255)
	dc.DrawString(block.Name, textX, nameY)

	if block.Handle != "" {
		quotesLoadFont(dc, handleSize)
		handleY := nameY + 6 + handleSize
		dc.SetRGBA(0.78, 0.78, 0.86, 0.95)
		dc.DrawString("@"+block.Handle, textX, handleY)
	}

	quotesLoadFont(dc, textSize)
	lineH := textSize * 1.35
	bodyStartY := y + headerH + 24 + textSize*0.85
	dc.SetRGBA(0.95, 0.95, 0.96, 1.0)
	for i, ln := range lines {
		dc.DrawString(ln, x, bodyStartY+float64(i)*lineH)
	}

	return total
}

func quotesAutoCropBottom(img *image.RGBA, pad int) *image.RGBA {
	b := img.Bounds()
	lastY := b.Min.Y
	for y := b.Max.Y - 1; y >= b.Min.Y; y-- {
		rowHasContent := false
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				rowHasContent = true
				break
			}
		}
		if rowHasContent {
			lastY = y
			break
		}
	}
	newH := lastY - b.Min.Y + 1 + pad
	if newH > b.Dy() {
		newH = b.Dy()
	}
	if newH < 1 {
		newH = 1
	}
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < b.Dx(); x++ {
			out.Set(x, y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

func quotesRenderImage(blocks []quoteBlock, botName string) (string, error) {
	if len(blocks) == 0 {
		return "", fmt.Errorf("no blocks")
	}
	const W = 1024
	const maxH = 4096
	padding := 32.0
	contentW := float64(W) - padding*2
	blockGap := 20.0
	footerH := 36.0

	measure := gg.NewContext(W, 100)
	totalH := padding
	for i, b := range blocks {
		h, _, _ := quotesMeasureBlock(measure, b, contentW)
		totalH += h
		if i < len(blocks)-1 {
			totalH += blockGap
		}
	}
	totalH += footerH + padding

	H := int(totalH)
	if H < 256 {
		H = 256
	}
	if H > maxH {
		H = maxH
	}

	rgba := image.NewRGBA(image.Rect(0, 0, W, H))
	dc := gg.NewContextForRGBA(rgba)

	y := padding
	for i, b := range blocks {
		blockH := quotesDrawBlock(dc, b, padding, y, contentW)
		y += blockH
		if i < len(blocks)-1 {
			y += blockGap
		}
	}

	quotesLoadFont(dc, 16)
	dc.SetRGBA(0.78, 0.78, 0.86, 0.45)
	mark := "— via @" + botName
	if botName == "" {
		mark = "— via @JuliaBot"
	}
	footerY := y + footerH - 8
	dc.DrawStringAnchored(mark, float64(W)-padding, footerY, 1.0, 0.5)

	cropped := quotesAutoCropBottom(rgba, int(padding))

	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("quote_%d.png", time.Now().UnixNano()))
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := png.Encode(f, cropped); err != nil {
		return "", err
	}
	return outPath, nil
}

func quotesAutoStackPreceding(m *tg.NewMessage, replied *tg.NewMessage) []quoteBlock {
	var extras []quoteBlock
	if replied == nil {
		return extras
	}
	repliedDate := int64(replied.Date())
	repliedSender := replied.SenderID()
	if repliedSender == 0 {
		return extras
	}
	startID := replied.ID - 1
	for i := 0; i < 2 && startID > 0; i, startID = i+1, startID-1 {
		msgs, err := m.Client.GetMessages(m.ChatID(), &tg.SearchOption{IDs: startID})
		if err != nil || len(msgs) == 0 {
			break
		}
		prev := msgs[0]
		if prev.SenderID() != repliedSender {
			break
		}
		prevDate := int64(prev.Date())
		if repliedDate-prevDate > 30 || repliedDate-prevDate < 0 {
			break
		}
		text := quoteSanitizeText(prev.RawText())
		if text == "" {
			break
		}
		if len(text) > 600 {
			text = text[:600] + "..."
		}
		name := "User"
		handle := ""
		u, uerr := m.Client.GetUser(repliedSender)
		if uerr == nil && u != nil {
			name = strings.TrimSpace(u.FirstName + " " + u.LastName)
			if name == "" {
				name = "User"
			}
			handle = u.Username
		}
		extras = append(extras, quoteBlock{
			Name:   name,
			Handle: handle,
			Text:   text,
			UserID: repliedSender,
			Color:  quotesPickAccent(repliedSender),
			Date:   prevDate,
		})
	}
	return extras
}

func QuoteImageHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a message with <code>/q</code> to generate a quote image.")
		return nil
	}

	status, _ := m.Reply("<i>painting your quote...</i>")

	blocks := quotesCollectChain(m, 3)
	if len(blocks) == 0 {
		if status != nil {
			status.Edit("could not read reply")
		}
		return nil
	}

	botName := "JuliaBot"
	if me, err := m.Client.GetMe(); err == nil && me != nil {
		if me.Username != "" {
			botName = me.Username
		} else if me.FirstName != "" {
			botName = me.FirstName
		}
	}

	defer func() {
		for _, b := range blocks {
			if b.Avatar != "" {
				os.Remove(b.Avatar)
			}
		}
	}()

	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	outPath, err := quotesRenderImage(blocks, botName)
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
	defer os.Remove(outPath)

	_, merr := m.ReplyMedia(outPath, &tg.MediaOptions{
		FileName: "quote.png",
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

func quotesEnsureBucket() error {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return fmt.Errorf("db unavailable")
	}
	return d.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(quotesBucket)
		return e
	})
}

func quotesChatKey(chatID int64, id uint64) []byte {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[0:8], uint64(chatID))
	binary.BigEndian.PutUint64(b[8:16], id)
	return b
}

func quotesNextID(tx *bolt.Tx, chatID int64) uint64 {
	b := tx.Bucket(quotesBucket)
	if b == nil {
		return 1
	}
	prefix := make([]byte, 8)
	binary.BigEndian.PutUint64(prefix, uint64(chatID))
	c := b.Cursor()
	var maxID uint64
	for k, _ := c.Seek(prefix); k != nil && len(k) >= 16; k, _ = c.Next() {
		if !bytesHasPrefix(k, prefix) {
			break
		}
		id := binary.BigEndian.Uint64(k[8:16])
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1
}

func bytesHasPrefix(a, b []byte) bool {
	if len(a) < len(b) {
		return false
	}
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func quotesListByChat(chatID int64) ([]quoteRecord, error) {
	if err := quotesEnsureBucket(); err != nil {
		return nil, err
	}
	d, err := db.GetDB()
	if err != nil || d == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var out []quoteRecord
	prefix := make([]byte, 8)
	binary.BigEndian.PutUint64(prefix, uint64(chatID))
	err = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(quotesBucket)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytesHasPrefix(k, prefix); k, v = c.Next() {
			var rec quoteRecord
			if jerr := json.Unmarshal(v, &rec); jerr == nil {
				out = append(out, rec)
			}
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, err
}

func QuoteSaveHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a message with <code>/qsave</code> to save it.")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil || reply == nil {
		m.Reply("<b>Could not fetch the replied message.</b>")
		return nil
	}
	text := strings.TrimSpace(reply.RawText())
	if text == "" {
		m.Reply("<b>Nothing to save.</b> The message has no text.")
		return nil
	}
	if len(text) > 4000 {
		text = text[:4000]
	}

	var userID int64
	name := "User"
	handle := ""
	if reply.SenderID() != 0 {
		userID = reply.SenderID()
		u, uerr := m.Client.GetUser(userID)
		if uerr == nil && u != nil {
			name = strings.TrimSpace(u.FirstName + " " + u.LastName)
			if name == "" {
				name = "User"
			}
			handle = u.Username
		}
	}

	savedByName := "User"
	if m.Sender != nil {
		savedByName = strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
		if savedByName == "" {
			savedByName = "User"
		}
	}

	if err := quotesEnsureBucket(); err != nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}
	d, derr := db.GetDB()
	if derr != nil || d == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}

	var newID uint64
	werr := d.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(quotesBucket)
		if e != nil {
			return e
		}
		newID = quotesNextID(tx, m.ChatID())
		rec := quoteRecord{
			ID:          newID,
			ChatID:      m.ChatID(),
			UserID:      userID,
			UserName:    name,
			UserHandle:  handle,
			Text:        text,
			SavedBy:     m.SenderID(),
			SavedByName: savedByName,
			Timestamp:   time.Now().Unix(),
		}
		raw, jerr := json.Marshal(&rec)
		if jerr != nil {
			return jerr
		}
		return b.Put(quotesChatKey(m.ChatID(), newID), raw)
	})
	if werr != nil {
		m.Reply("<b>Failed to save quote.</b>")
		return nil
	}

	preview := text
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}
	m.Reply(fmt.Sprintf("<b>Quote saved.</b> <code>#%d</code>\n\n<b>%s</b>: <i>%s</i>",
		newID, html.EscapeString(name), html.EscapeString(preview)))
	return nil
}

func QuotesListHandler(m *tg.NewMessage) error {
	page := 1
	if a := strings.TrimSpace(m.Args()); a != "" {
		if n, err := strconv.Atoi(a); err == nil && n > 0 {
			page = n
		}
	}

	all, err := quotesListByChat(m.ChatID())
	if err != nil || len(all) == 0 {
		m.Reply("<b>No quotes saved here yet.</b> Reply to a message with <code>/qsave</code>.")
		return nil
	}

	perPage := 10
	totalPages := (len(all) + perPage - 1) / perPage
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * perPage
	end := start + perPage
	if end > len(all) {
		end = len(all)
	}

	var resp strings.Builder
	resp.WriteString(fmt.Sprintf("<b>Saved Quotes</b> (page %d/%d)\n", page, totalPages))
	resp.WriteString("━━━━━━━━━━━━━━━━\n\n")
	for _, rec := range all[start:end] {
		preview := rec.Text
		if len(preview) > 90 {
			preview = preview[:90] + "..."
		}
		resp.WriteString(fmt.Sprintf("<code>#%d</code> <b>%s</b>\n<i>%s</i>\n\n",
			rec.ID,
			html.EscapeString(rec.UserName),
			html.EscapeString(preview)))
	}
	resp.WriteString(fmt.Sprintf("━━━━━━━━━━━━━━━━\n<b>Total:</b> %d quotes\n", len(all)))
	if totalPages > 1 {
		resp.WriteString(fmt.Sprintf("<i>Use</i> <code>/quotes %d</code> <i>for next page</i>", page+1))
	}
	m.Reply(resp.String())
	return nil
}

func QuoteDeleteHandler(m *tg.NewMessage) error {
	if !m.IsPrivate() {
		if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
			m.Reply("<b>Permission denied.</b> Admins only.")
			return nil
		}
	}
	arg := strings.TrimSpace(m.Args())
	arg = strings.TrimPrefix(arg, "#")
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/delq &lt;id&gt;</code>")
		return nil
	}
	id, err := strconv.ParseUint(arg, 10, 64)
	if err != nil || id == 0 {
		m.Reply("<b>Invalid id.</b>")
		return nil
	}
	if err := quotesEnsureBucket(); err != nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}
	d, derr := db.GetDB()
	if derr != nil || d == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}
	found := false
	_ = d.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(quotesBucket)
		if b == nil {
			return nil
		}
		key := quotesChatKey(m.ChatID(), id)
		if b.Get(key) == nil {
			return nil
		}
		found = true
		return b.Delete(key)
	})
	if !found {
		m.Reply(fmt.Sprintf("<b>Quote not found:</b> <code>#%d</code>", id))
		return nil
	}
	m.Reply(fmt.Sprintf("<b>Quote deleted:</b> <code>#%d</code>", id))
	return nil
}

func QuotesSearchHandler(m *tg.NewMessage) error {
	q := strings.ToLower(strings.TrimSpace(m.Args()))
	if q == "" {
		m.Reply("<b>Usage:</b> <code>/qsearch &lt;keyword&gt;</code>")
		return nil
	}
	all, err := quotesListByChat(m.ChatID())
	if err != nil || len(all) == 0 {
		m.Reply("<b>No quotes to search.</b>")
		return nil
	}
	var matches []quoteRecord
	for _, rec := range all {
		if strings.Contains(strings.ToLower(rec.Text), q) ||
			strings.Contains(strings.ToLower(rec.UserName), q) ||
			strings.Contains(strings.ToLower(rec.UserHandle), q) {
			matches = append(matches, rec)
		}
	}
	if len(matches) == 0 {
		m.Reply(fmt.Sprintf("<b>No quotes match:</b> <code>%s</code>", html.EscapeString(q)))
		return nil
	}
	var resp strings.Builder
	resp.WriteString(fmt.Sprintf("<b>Quote Search:</b> <code>%s</code>\n", html.EscapeString(q)))
	resp.WriteString("━━━━━━━━━━━━━━━━\n\n")
	limit := 15
	for i, rec := range matches {
		if i >= limit {
			resp.WriteString(fmt.Sprintf("\n<i>...and %d more</i>", len(matches)-limit))
			break
		}
		preview := rec.Text
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		resp.WriteString(fmt.Sprintf("<code>#%d</code> <b>%s</b>\n<i>%s</i>\n\n",
			rec.ID, html.EscapeString(rec.UserName), html.EscapeString(preview)))
	}
	resp.WriteString(fmt.Sprintf("━━━━━━━━━━━━━━━━\n<b>Matches:</b> %d", len(matches)))
	m.Reply(resp.String())
	return nil
}

func registerQuotesHandlers() {
	c := Client
	c.On("cmd:q", QuoteImageHandler)
	c.On("cmd:qsave", QuoteSaveHandler)
	c.On("cmd:quotes", QuotesListHandler)
	c.On("cmd:delq", QuoteDeleteHandler)
	c.On("cmd:qsearch", QuotesSearchHandler)
}

func init() {
	QueueHandlerRegistration(registerQuotesHandlers)
}
