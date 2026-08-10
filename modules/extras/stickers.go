package extras

import (
	"encoding/binary"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	modules "main/modules"
	"main/modules/db"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	_ "golang.org/x/image/webp"
)

const MaxStickersPerPack = 120

func stickerFriendlyError(m *tg.NewMessage, err error) string {
	msg := err.Error()
	upper := strings.ToUpper(msg)
	switch {
	case strings.Contains(upper, "PEER_ID_INVALID"):
		if me := m.Client.Me(); me != nil && me.Username != "" {
			return fmt.Sprintf("I can't create sticker packs for you because you haven't started me in DM (or you've blocked me). Open <a href=\"https://t.me/%s\">@%s</a> and press <b>Start</b>, then try again.", me.Username, me.Username)
		}
		return "I can't create sticker packs for you because you haven't started me in DM. Open my DM and press <b>Start</b>, then try again."
	case strings.Contains(upper, "STICKERPACK_STICKERS_TOO_MUCH"):
		return "This sticker pack is full. Try again — I'll spill into a new pack."
	case strings.Contains(upper, "SHORTNAME_OCCUPY_FAILED"), strings.Contains(upper, "PACK_SHORT_NAME_OCCUPIED"):
		return "That pack name is already taken. Retry to get a new one."
	case strings.Contains(upper, "STICKER_TGS_NODOC"), strings.Contains(upper, "STICKER_VIDEO_NODOC"):
		return "The uploaded sticker document is empty or missing. Reply to the original sticker/file directly and try again."
	case strings.Contains(upper, "STICKER_TGS_NOTFOUND"), strings.Contains(upper, "STICKER_VIDEO_NOTFOUND"):
		return "Telegram couldn't find the uploaded sticker document. Try again — the upload may have expired."
	case strings.Contains(upper, "STICKER_MIME_INVALID"):
		return "Wrong file type for this pack kind. Video sticker packs accept only .webm/VP9, animated only .tgs, static only .webp/.png."
	case strings.Contains(upper, "STICKER_EMOJI_INVALID"), strings.Contains(upper, "STICKER_EMOJI_EMPTY"):
		return "Emoji is invalid or empty. Provide a real emoji after the command, like <code>/kang 😀</code>."
	case strings.Contains(upper, "STICKER_PNG_DIMENSIONS"), strings.Contains(upper, "STICKER_PNG_NOPNG"):
		return "Static sticker must be 512×512 PNG or WEBP with transparency."
	case strings.Contains(upper, "STICKER_INVALID"), strings.Contains(upper, "STICKER_DOCUMENT_INVALID"):
		return "Sticker file is invalid — must be a 512×512 PNG/WEBP or a valid animated/video sticker."
	case strings.Contains(upper, "FLOOD"):
		return "Telegram is rate-limiting sticker operations. Wait a bit and try again."
	}
	return "Sticker operation failed: " + html.EscapeString(msg)
}

func GifToSticker(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Error:</b> Please reply to a GIF or short MP4 to convert it to a video sticker.")
		return nil
	}
	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Error:</b> Unable to fetch the replied message.")
		return nil
	}
	if !r.IsMedia() {
		m.Reply("<b>Error:</b> The replied message has no media.")
		return nil
	}

	fn := ""
	if r.File != nil {
		fn = strings.ToLower(r.File.Name)
	}
	if fn != "" && !(strings.HasSuffix(fn, ".mp4") || strings.HasSuffix(fn, ".gif") || strings.HasSuffix(fn, ".webm") || strings.HasSuffix(fn, ".mov")) {
		m.Reply("<b>Error:</b> Only .mp4, .gif, .webm, or .mov files are supported.")
		return nil
	}

	emoji := strings.TrimSpace(m.Args())
	if emoji == "" {
		emoji = "😍"
	}

	ts := time.Now().UnixNano()
	srcExt := ".mp4"
	if r.File != nil && r.File.Name != "" {
		if e := strings.ToLower(filepath.Ext(r.File.Name)); e != "" {
			srcExt = e
		}
	}
	inPath := filepath.Join(os.TempDir(), fmt.Sprintf("gif2sticker_%d_in%s", ts, srcExt))
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("gif2sticker_%d.webm", ts))
	fi, err := r.Download(&tg.DownloadOptions{FileName: inPath})
	if err != nil {
		m.Reply("<b>Error:</b> Unable to download the source: " + html.EscapeString(err.Error()))
		return nil
	}
	defer os.Remove(fi)
	defer os.Remove(outPath)

	if err := encodeVideoSticker(fi, outPath); err != nil {
		m.Reply("<b>Error:</b> ffmpeg failed: " + html.EscapeString(err.Error()))
		return nil
	}

	if _, err := m.ReplyMedia(outPath, &tg.MediaOptions{
		MimeType: "video/webm",
		Attributes: []tg.DocumentAttribute{
			&tg.DocumentAttributeSticker{
				Alt:        emoji,
				Stickerset: &tg.InputStickerSetEmpty{},
			},
			&tg.DocumentAttributeFilename{FileName: "sticker.webm"},
		},
	}); err != nil {
		m.Reply("<b>Error:</b> upload failed: " + html.EscapeString(err.Error()))
	}
	return nil
}

func encodeVideoSticker(src, dst string) error {
	dur := 0.0
	probe := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", src)
	if out, err := probe.Output(); err == nil {
		dur, _ = strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	}

	vf := "scale='if(gt(iw,ih),512,-2)':'if(gt(iw,ih),-2,512)':flags=lanczos,format=yuva420p,fps=30"
	args := []string{"-y", "-loglevel", "error", "-i", src, "-vf", vf,
		"-c:v", "libvpx-vp9",
		"-pix_fmt", "yuva420p",
		"-b:v", "0",
		"-crf", "34",
		"-deadline", "good",
		"-cpu-used", "4",
		"-row-mt", "1",
		"-threads", "4",
		"-auto-alt-ref", "0",
		"-an", "-sn",
		dst,
	}
	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}

	if dur > 3.0 {
		if err := spoofWebmDuration(dst, 2999.0); err != nil {
			os.Remove(dst)
			return fmt.Errorf("spoof duration: %w", err)
		}
	}
	return nil
}

func spoofWebmDuration(path string, targetMs float64) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Element IDs as they appear in the file (VINT-encoded, with marker bits):
	// Segment=0x18538067 (4 bytes), Info=0x1549A966 (4 bytes), Duration=0x4489 (2 bytes).
	segStart, segLen, ok := ebmlFindChild(data, 0, len(data), []byte{0x18, 0x53, 0x80, 0x67})
	if !ok {
		return fmt.Errorf("Segment element not found")
	}
	infoStart, infoLen, ok := ebmlFindChild(data, segStart, segStart+segLen, []byte{0x15, 0x49, 0xA9, 0x66})
	if !ok {
		return fmt.Errorf("Info element not found")
	}
	durStart, durLen, ok := ebmlFindChild(data, infoStart, infoStart+infoLen, []byte{0x44, 0x89})
	if !ok {
		return fmt.Errorf("Duration element not found")
	}
	switch durLen {
	case 4:
		bits := math.Float32bits(float32(targetMs))
		binary.BigEndian.PutUint32(data[durStart:durStart+4], bits)
	case 8:
		bits := math.Float64bits(targetMs)
		binary.BigEndian.PutUint64(data[durStart:durStart+8], bits)
	default:
		return fmt.Errorf("unexpected Duration payload length %d", durLen)
	}
	return os.WriteFile(path, data, 0o644)
}

func ebmlFindChild(data []byte, start, end int, wantID []byte) (payloadStart, payloadLen int, ok bool) {
	s := start
	for s < end {
		if s+len(wantID) > end {
			return 0, 0, false
		}
		idLen := ebmlVIntLen(data[s])
		if idLen == 0 || s+idLen > end {
			return 0, 0, false
		}
		id := data[s : s+idLen]
		s += idLen
		sizeLen := ebmlVIntLen(data[s])
		if sizeLen == 0 || s+sizeLen > end {
			return 0, 0, false
		}
		size := ebmlVIntValue(data[s : s+sizeLen])
		s += sizeLen
		if len(id) == len(wantID) && bytesEqual(id, wantID) {
			return s, int(size), true
		}
		s += int(size)
	}
	return 0, 0, false
}

func ebmlVIntLen(first byte) int {
	if first == 0 {
		return 0
	}
	for i := 0; i < 8; i++ {
		if first&(0x80>>i) != 0 {
			return i + 1
		}
	}
	return 0
}

func ebmlVIntValue(b []byte) uint64 {
	if len(b) == 0 {
		return 0
	}
	mask := byte(0x80 >> (len(b) - 1))
	v := uint64(b[0] &^ mask)
	for i := 1; i < len(b); i++ {
		v = (v << 8) | uint64(b[i])
	}
	return v
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type kangSource struct {
	Doc            *tg.DocumentObj
	Kind           string
	IsAlreadyValid bool
	Emoji          string
}

func kangDetectSource(reply *tg.NewMessage) (*kangSource, error) {
	if reply == nil || !reply.IsMedia() || reply.Media() == nil {
		return nil, fmt.Errorf("reply is not media")
	}
	md, ok := reply.Media().(*tg.MessageMediaDocument)
	if !ok {
		return nil, fmt.Errorf("reply is not a document")
	}
	doc, ok := md.Document.(*tg.DocumentObj)
	if !ok {
		return nil, fmt.Errorf("document is not a DocumentObj")
	}
	src := &kangSource{Doc: doc}
	mime := strings.ToLower(doc.MimeType)
	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeSticker:
			if src.Emoji == "" && a.Alt != "" {
				src.Emoji = a.Alt
			}
			src.IsAlreadyValid = true
		case *tg.DocumentAttributeVideo:
			if src.Kind == "" {
				src.Kind = "webm"
			}
		case *tg.DocumentAttributeFilename:
			if strings.HasSuffix(strings.ToLower(a.FileName), ".tgs") {
				src.Kind = "tgs"
			}
		}
	}
	switch {
	case strings.Contains(mime, "application/x-tgsticker"):
		src.Kind = "tgs"
	case strings.HasPrefix(mime, "video/"):
		src.Kind = "webm"
	case strings.HasPrefix(mime, "image/"):
		if src.Kind == "" {
			src.Kind = "normal"
		}
	}
	if src.Kind == "" {
		src.Kind = "normal"
	}
	return src, nil
}

func kangPrepareInputDoc(m *tg.NewMessage, src *kangSource) (tg.InputDocument, func(), error) {
	cleanup := func() {}

	if src.Kind == "normal" && src.IsAlreadyValid {
		return &tg.InputDocumentObj{
			ID:            src.Doc.ID,
			AccessHash:    src.Doc.AccessHash,
			FileReference: src.Doc.FileReference,
		}, cleanup, nil
	}

	dl, err := m.Client.DownloadMedia(&tg.MessageMediaDocument{Document: src.Doc})
	if err != nil {
		return nil, cleanup, fmt.Errorf("download: %w", err)
	}

	srcW, srcH := 0, 0
	for _, attr := range src.Doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeImageSize:
			srcW, srcH = int(a.W), int(a.H)
		case *tg.DocumentAttributeVideo:
			if srcW == 0 {
				srcW, srcH = int(a.W), int(a.H)
			}
		}
	}
	alreadySized := (srcW == 512 && srcH <= 512) || (srcH == 512 && srcW <= 512)

	outPath := dl
	ts := time.Now().UnixNano()
	switch src.Kind {
	case "tgs":
	case "webm":
		if !(src.IsAlreadyValid && alreadySized) {
			outPath = filepath.Join(os.TempDir(), fmt.Sprintf("kang_%d.webm", ts))
			if err := encodeVideoSticker(dl, outPath); err != nil {
				os.Remove(dl)
				return nil, cleanup, err
			}
		}
	default:
		if !(src.IsAlreadyValid && alreadySized) {
			outPath = filepath.Join(os.TempDir(), fmt.Sprintf("kang_%d.webp", ts))
			cmd := exec.Command("ffmpeg",
				"-y", "-loglevel", "error",
				"-i", dl,
				"-vf", "scale='if(gt(iw,ih),512,-2)':'if(gt(iw,ih),-2,512)':flags=lanczos",
				"-c:v", "libwebp", "-pix_fmt", "yuva420p",
				"-lossless", "0", "-compression_level", "4", "-q:v", "80",
				"-preset", "picture", "-an", "-sn", "-threads", "4",
				outPath,
			)
			if out, err := cmd.CombinedOutput(); err != nil {
				os.Remove(dl)
				return nil, cleanup, fmt.Errorf("ffmpeg webp: %v: %s", err, strings.TrimSpace(string(out)))
			}
		}
	}

	cleanup = func() {
		os.Remove(dl)
		if outPath != dl {
			os.Remove(outPath)
		}
	}

	media, err := m.Client.GetSendableMedia(outPath, &tg.MediaMetadata{Inline: true})
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("prepare media: %w", err)
	}
	inMedia, ok := media.(*tg.InputMediaDocument)
	if !ok {
		cleanup()
		return nil, func() {}, fmt.Errorf("unexpected sendable media type")
	}
	return inMedia.ID, cleanup, nil
}

func KangSticker(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a sticker or image to kang it.\n<b>Usage:</b> <code>/kang [emoji]</code>")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Failed to get replied message: " + html.EscapeString(err.Error()))
		return nil
	}
	src, err := kangDetectSource(reply)
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}

	emoji := strings.TrimSpace(m.Args())
	if emoji == "" {
		if src.Emoji != "" {
			emoji = src.Emoji
		} else {
			emoji = "👍"
		}
	}

	userID := m.SenderID()
	username := m.Sender.Username
	if username == "" {
		username = fmt.Sprintf("user%d", userID)
	}
	me := m.Client.Me()
	if me == nil || me.Username == "" {
		m.Reply("<b>Error:</b> bot has no username; can't create sticker sets.")
		return nil
	}

	pack, _ := db.GetActivePack(userID, src.Kind)
	needNewPack := pack == nil || pack.StickerCount >= MaxStickersPerPack

	doc, cleanup, err := kangPrepareInputDoc(m, src)
	defer cleanup()
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}

	if needNewPack {
		packs, _ := db.GetUserPacks(userID)
		packNumber := len(packs[src.Kind]) + 1
		shortName := fmt.Sprintf("x%s_%s_%d_by_%s", username, src.Kind, packNumber, me.Username)
		title := fmt.Sprintf("%s's %s Stickers #%d", username, asciiTitle(src.Kind), packNumber)

		newPack := &db.PackInfo{
			ShortName:    shortName,
			Title:        title,
			Type:         src.Kind,
			StickerCount: 1,
			PackNumber:   packNumber,
		}
		_, createErr := m.Client.StickersCreateStickerSet(&tg.StickersCreateStickerSetParams{
			UserID:    &tg.InputUserObj{UserID: userID, AccessHash: m.Sender.AccessHash},
			Title:     title,
			ShortName: shortName,
			Stickers: []*tg.InputStickerSetItem{
				{Document: doc, Emoji: emoji},
			},
		})
		if createErr != nil {
			m.Reply(stickerFriendlyError(m, createErr))
			return nil
		}
		db.SavePack(userID, newPack)
		m.Reply(fmt.Sprintf(
			"<b>Created new %s sticker pack.</b>\n"+
				"Pack: <a href=\"https://t.me/addstickers/%s\">%s</a>\n"+
				"Stickers: 1/%d",
			src.Kind, shortName, html.EscapeString(title), MaxStickersPerPack,
		))
		return nil
	}

	_, addErr := m.Client.StickersAddStickerToSet(
		&tg.InputStickerSetShortName{ShortName: pack.ShortName},
		&tg.InputStickerSetItem{Document: doc, Emoji: emoji},
	)
	if addErr != nil {
		m.Reply(stickerFriendlyError(m, addErr))
		return nil
	}
	db.IncrementPackCount(userID, pack)

	msg := fmt.Sprintf(
		"<b>Added to pack.</b>\n"+
			"Pack: <a href=\"https://t.me/addstickers/%s\">%s</a>\n"+
			"Stickers: %d/%d",
		pack.ShortName, html.EscapeString(pack.Title), pack.StickerCount, MaxStickersPerPack,
	)
	if pack.StickerCount >= MaxStickersPerPack {
		msg += "\n\n<b>Pack is full.</b> Next kang will create a new pack."
	}
	m.Reply(msg)
	return nil
}

func RemoveKangedSticker(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a sticker in your pack to remove it.\n<b>Usage:</b> <code>/rmkang</code>")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Failed to get replied message: " + html.EscapeString(err.Error()))
		return nil
	}
	if !reply.IsMedia() {
		m.Reply("Please reply to a sticker.")
		return nil
	}

	var (
		stickerFile tg.InputDocument
		setInput    tg.InputStickerSet
	)
	if md, ok := reply.Media().(*tg.MessageMediaDocument); ok {
		if document, ok := md.Document.(*tg.DocumentObj); ok {
			stickerFile = &tg.InputDocumentObj{
				ID:            document.ID,
				AccessHash:    document.AccessHash,
				FileReference: document.FileReference,
			}
			for _, attr := range document.Attributes {
				if s, ok := attr.(*tg.DocumentAttributeSticker); ok {
					setInput = s.Stickerset
					break
				}
			}
		}
	}
	if stickerFile == nil {
		m.Reply("Unable to extract sticker file.")
		return nil
	}

	if _, err := m.Client.StickersRemoveStickerFromSet(stickerFile); err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()) + " (you can only remove stickers from packs you created)")
		return nil
	}

	if setInput != nil {
		if setRes, err := m.Client.MessagesGetStickerSet(setInput, 0); err == nil {
			if resp, ok := setRes.(*tg.MessagesStickerSetObj); ok && resp.Set != nil {
				if pack, err := db.GetPackByShortName(m.Sender.ID, resp.Set.ShortName); err == nil && pack != nil {
					db.DecrementPackCount(m.Sender.ID, pack)
				}
			}
		}
	}

	m.Reply("Removed sticker from your pack.")
	return nil
}

func PackInfoHandle(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a sticker to get pack info.")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Failed to get replied message.")
		return nil
	}
	if !reply.IsMedia() {
		m.Reply("Please reply to a sticker.")
		return nil
	}

	var stickerAttr *tg.DocumentAttributeSticker
	if md, ok := reply.Media().(*tg.MessageMediaDocument); ok {
		if document, ok := md.Document.(*tg.DocumentObj); ok {
			for _, attr := range document.Attributes {
				if s, ok := attr.(*tg.DocumentAttributeSticker); ok {
					stickerAttr = s
					break
				}
			}
		}
	}
	if stickerAttr == nil || stickerAttr.Stickerset == nil {
		m.Reply("This is not a valid sticker or doesn't belong to a pack.")
		return nil
	}
	if _, empty := stickerAttr.Stickerset.(*tg.InputStickerSetEmpty); empty {
		m.Reply("Sticker has no associated pack.")
		return nil
	}

	result, err := m.Client.MessagesGetStickerSet(stickerAttr.Stickerset, 0)
	if err != nil {
		m.Reply("Failed to get sticker pack info: " + html.EscapeString(err.Error()))
		return nil
	}
	resp, ok := result.(*tg.MessagesStickerSetObj)
	if !ok || resp.Set == nil {
		m.Reply("Unexpected response from Telegram.")
		return nil
	}
	set := resp.Set

	kind := "Static"
	for _, d := range resp.Documents {
		obj, ok := d.(*tg.DocumentObj)
		if !ok {
			continue
		}
		mime := strings.ToLower(obj.MimeType)
		switch {
		case strings.Contains(mime, "application/x-tgsticker"):
			kind = "Animated (.tgs)"
		case strings.HasPrefix(mime, "video/"):
			kind = "Video (.webm)"
		case strings.HasPrefix(mime, "image/"):
			kind = "Static (.webp)"
		}
		break
	}
	if set.Emojis {
		kind += " · Emoji pack"
	}
	if set.Masks {
		kind += " · Mask pack"
	}

	text := fmt.Sprintf(
		"<b>Sticker Pack Info</b>\n\n"+
			"<b>Title:</b> %s\n"+
			"<b>Short name:</b> <code>%s</code>\n"+
			"<b>Stickers:</b> %d\n"+
			"<b>Type:</b> %s\n"+
			"<b>Set ID:</b> <code>%d</code>\n"+
			"<b>Link:</b> <a href=\"https://t.me/addstickers/%s\">Add pack</a>",
		html.EscapeString(set.Title),
		html.EscapeString(set.ShortName),
		set.Count,
		html.EscapeString(kind),
		set.ID,
		html.EscapeString(set.ShortName),
	)
	m.Reply(text)
	return nil
}

func MyPacksHandler(m *tg.NewMessage) error {
	packs, err := db.GetUserPacks(m.SenderID())
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	total := 0
	var b strings.Builder
	b.WriteString("<b>Your Sticker Packs</b>\n")
	for _, kind := range []string{"normal", "webm", "tgs"} {
		list := packs[kind]
		if len(list) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n<b>%s</b>\n", asciiTitle(kind))
		for _, p := range list {
			total += p.StickerCount
			fmt.Fprintf(&b, "• <a href=\"https://t.me/addstickers/%s\">%s</a> (%d/%d)\n",
				p.ShortName, html.EscapeString(p.Title), p.StickerCount, MaxStickersPerPack)
		}
	}
	if total == 0 {
		m.Reply("You haven't kanged any stickers yet. Reply to a sticker with <code>/kang</code>.")
		return nil
	}
	m.Reply(b.String())
	return nil
}

func RenamePackHandler(m *tg.NewMessage) error {
	fields := strings.SplitN(strings.TrimSpace(m.Args()), " ", 2)
	if len(fields) < 2 {
		m.Reply("<b>Usage:</b> <code>/renamepack &lt;short_name&gt; &lt;new title&gt;</code>")
		return nil
	}
	shortName, newTitle := strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1])
	if shortName == "" || newTitle == "" {
		m.Reply("<b>Usage:</b> <code>/renamepack &lt;short_name&gt; &lt;new title&gt;</code>")
		return nil
	}
	pack, err := db.GetPackByShortName(m.SenderID(), shortName)
	if err != nil || pack == nil {
		m.Reply("Pack not found in your collection.")
		return nil
	}
	if _, err := m.Client.StickersRenameStickerSet(&tg.InputStickerSetShortName{ShortName: shortName}, newTitle); err != nil {
		m.Reply(stickerFriendlyError(m, err))
		return nil
	}
	pack.Title = newTitle
	db.SavePack(m.SenderID(), pack)
	m.Reply(fmt.Sprintf("Renamed pack to <b>%s</b>.", html.EscapeString(newTitle)))
	return nil
}

func DeletePackHandler(m *tg.NewMessage) error {
	shortName := strings.TrimSpace(m.Args())
	if shortName == "" {
		m.Reply("<b>Usage:</b> <code>/deletepack &lt;short_name&gt;</code>")
		return nil
	}
	pack, err := db.GetPackByShortName(m.SenderID(), shortName)
	if err != nil || pack == nil {
		m.Reply("Pack not found in your collection.")
		return nil
	}
	if _, err := m.Client.StickersDeleteStickerSet(&tg.InputStickerSetShortName{ShortName: shortName}); err != nil {
		m.Reply(stickerFriendlyError(m, err))
		return nil
	}
	if _, err := db.DeletePack(m.SenderID(), shortName); err != nil {
		m.Reply(fmt.Sprintf("Deleted on Telegram, but local record cleanup failed: %s", html.EscapeString(err.Error())))
		return nil
	}
	m.Reply(fmt.Sprintf("Deleted pack <b>%s</b>.", html.EscapeString(pack.Title)))
	return nil
}

func registerStickersHandlers() {
	c := modules.Client
	c.On("cmd:gif", GifToSticker)
	c.On("cmd:kang", KangSticker)
	c.On("cmd:rmkang", RemoveKangedSticker)
	c.On("cmd:pack", PackInfoHandle)
	c.On("cmd:mypacks", MyPacksHandler)
	c.On("cmd:renamepack", RenamePackHandler)
	c.On("cmd:deletepack", DeletePackHandler)
	c.On("command:doge", modules.DogeSticker)
	c.On("inline:doge", modules.DogeStickerInline)
}

func initFromSrc_stickers_0_1() {
	modules.QueueHandlerRegistration(registerStickersHandlers)
}

func StickerInfoHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a sticker with <code>/stickerinfo</code>")
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Error:</b> unable to fetch replied message.")
		return nil
	}
	if !reply.IsMedia() {
		m.Reply("<b>Error:</b> please reply to a sticker.")
		return nil
	}
	var stickerAttr *tg.DocumentAttributeSticker
	var mime string
	if doc, ok := reply.Media().(*tg.MessageMediaDocument); ok {
		if document, ok := doc.Document.(*tg.DocumentObj); ok {
			mime = document.MimeType
			for _, attr := range document.Attributes {
				if s, ok := attr.(*tg.DocumentAttributeSticker); ok {
					stickerAttr = s
					break
				}
			}
		}
	}
	if stickerAttr == nil || stickerAttr.Stickerset == nil {
		m.Reply("<b>Error:</b> not a valid sticker or has no pack.")
		return nil
	}
	if _, ok := stickerAttr.Stickerset.(*tg.InputStickerSetEmpty); ok {
		m.Reply("<b>Error:</b> sticker has no associated pack.")
		return nil
	}
	result, err := m.Client.MessagesGetStickerSet(stickerAttr.Stickerset, 0)
	if err != nil {
		m.Reply(fmt.Sprintf("<b>Error:</b> failed to fetch pack info: %s", html.EscapeString(err.Error())))
		return nil
	}
	resp, ok := result.(*tg.MessagesStickerSetObj)
	if !ok || resp.Set == nil {
		m.Reply("<b>Error:</b> unexpected response from Telegram.")
		return nil
	}
	set := resp.Set
	kind := "Static"
	switch {
	case strings.Contains(mime, "x-tgsticker"):
		kind = "Animated (.tgs)"
	case strings.Contains(mime, "video/webm"):
		kind = "Video (.webm)"
	case strings.Contains(mime, "image/webp"):
		kind = "Static (.webp)"
	}
	if set.Emojis {
		kind += " · Emoji pack"
	}
	if set.Masks {
		kind += " · Mask pack"
	}
	addLink := fmt.Sprintf("https://t.me/addstickers/%s", set.ShortName)
	text := fmt.Sprintf(
		"<b>Sticker Pack Info</b>\n\n"+
			"<b>Title:</b> %s\n"+
			"<b>Short Name:</b> <code>%s</code>\n"+
			"<b>Stickers:</b> %d\n"+
			"<b>Type:</b> %s\n"+
			"<b>Link:</b> <a href=\"%s\">Add Pack</a>",
		html.EscapeString(set.Title),
		html.EscapeString(set.ShortName),
		set.Count,
		html.EscapeString(kind),
		addLink,
	)
	m.Reply(text)
	return nil
}

func MeActionHandler(m *tg.NewMessage) error {
	action := strings.TrimSpace(m.Args())
	if action == "" {
		m.Reply("<b>Usage:</b> <code>/me &lt;action&gt;</code>\nExample: <code>/me waves hello</code>")
		return nil
	}
	name := ""
	if m.Sender != nil {
		name = strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
		if name == "" && m.Sender.Username != "" {
			name = "@" + m.Sender.Username
		}
	}
	if name == "" {
		name = fmt.Sprintf("user%d", m.SenderID())
	}
	text := fmt.Sprintf("<i>* %s %s</i>", html.EscapeString(name), html.EscapeString(action))
	m.Reply(text)
	return nil
}

func registerStickerPackInfoHandlers() {
	c := modules.Client
	c.On("cmd:stickerinfo", StickerInfoHandler)
	c.On("cmd:me", MeActionHandler)
	c.On("cmd:myself", MeActionHandler)
}

func initFromSrc_sticker_pack_info_1_1() {
	modules.QueueHandlerRegistration(registerStickerPackInfoHandlers)
}

func stickerExtractDoc(reply *tg.NewMessage) (*tg.DocumentObj, string) {
	if reply.Media() == nil {
		return nil, ""
	}
	md, ok := reply.Media().(*tg.MessageMediaDocument)
	if !ok {
		return nil, ""
	}
	doc, ok := md.Document.(*tg.DocumentObj)
	if !ok {
		return nil, ""
	}
	kind := "static"
	for _, attr := range doc.Attributes {
		if _, ok := attr.(*tg.DocumentAttributeVideo); ok {
			kind = "video"
		}
		if fn, ok := attr.(*tg.DocumentAttributeFilename); ok {
			if strings.HasSuffix(strings.ToLower(fn.FileName), ".tgs") {
				kind = "tgs"
			}
		}
	}
	if strings.Contains(doc.MimeType, "application/x-tgsticker") {
		kind = "tgs"
	} else if strings.HasPrefix(doc.MimeType, "video/") {
		kind = "video"
	}
	return doc, kind
}

func StickerToImageHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a static sticker with <code>/towebp</code>")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Error:</b> unable to fetch reply: " + html.EscapeString(err.Error()))
		return nil
	}

	if !reply.IsMedia() {
		m.Reply("<b>Error:</b> reply is not a sticker")
		return nil
	}

	_, kind := stickerExtractDoc(reply)
	if kind == "" {
		m.Reply("<b>Error:</b> reply is not a sticker document")
		return nil
	}
	if kind == "tgs" {
		m.Reply("<b>Error:</b> animated (.tgs) stickers are not supported")
		return nil
	}
	if kind == "video" {
		m.Reply("<b>Error:</b> video (.webm) stickers are not supported")
		return nil
	}

	status, _ := m.Reply("<code>converting sticker...</code>")

	ts := time.Now().UnixNano()
	srcPath := filepath.Join(os.TempDir(), fmt.Sprintf("sticker_%d.webp", ts))
	pngPath := filepath.Join(os.TempDir(), fmt.Sprintf("sticker_%d.png", ts))
	jpgPath := filepath.Join(os.TempDir(), fmt.Sprintf("sticker_%d.jpg", ts))

	fi, err := reply.Download(&tg.DownloadOptions{FileName: srcPath})
	if err != nil {
		msg := "<b>Error:</b> download failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(fi)

	f, err := os.Open(fi)
	if err != nil {
		msg := "<b>Error:</b> open failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	img, fmtName, err := image.Decode(f)
	f.Close()
	if err != nil {
		msg := "<b>Error:</b> decode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	pngOut, err := os.Create(pngPath)
	if err != nil {
		msg := "<b>Error:</b> create png failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if err := png.Encode(pngOut, img); err != nil {
		pngOut.Close()
		os.Remove(pngPath)
		msg := "<b>Error:</b> png encode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	pngOut.Close()
	defer os.Remove(pngPath)

	bounds := img.Bounds()
	flat := image.NewRGBA(bounds)
	draw.Draw(flat, bounds, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(flat, bounds, img, bounds.Min, draw.Over)

	jpgOut, err := os.Create(jpgPath)
	if err != nil {
		msg := "<b>Error:</b> create jpg failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if err := jpeg.Encode(jpgOut, flat, &jpeg.Options{Quality: 92}); err != nil {
		jpgOut.Close()
		os.Remove(jpgPath)
		msg := "<b>Error:</b> jpg encode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	jpgOut.Close()
	defer os.Remove(jpgPath)

	caption := fmt.Sprintf("<b>Sticker -&gt; Image</b>\n<b>Source:</b> <code>%s</code>", html.EscapeString(fmtName))

	if _, err := m.ReplyMedia(pngPath, &tg.MediaOptions{
		Caption:       caption + "\n<b>Format:</b> PNG",
		FileName:      "sticker.png",
		MimeType:      "image/png",
		ForceDocument: false,
	}); err != nil {
		msg := "<b>Error:</b> png upload failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if _, err := m.ReplyMedia(jpgPath, &tg.MediaOptions{
		Caption:       caption + "\n<b>Format:</b> JPG",
		FileName:      "sticker.jpg",
		MimeType:      "image/jpeg",
		ForceDocument: false,
	}); err != nil {
		msg := "<b>Error:</b> jpg upload failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if status != nil {
		status.Delete()
	}
	return nil
}

func registerStickerToImageHandlers() {
	c := modules.Client
	c.On("cmd:towebp", StickerToImageHandler)
}

func initFromSrc_sticker_to_image_2_1() {
	modules.QueueHandlerRegistration(registerStickerToImageHandlers)
}

func webpToJpgExtractDoc(reply *tg.NewMessage) (*tg.DocumentObj, string, string) {
	if reply.Media() == nil {
		return nil, "", ""
	}
	md, ok := reply.Media().(*tg.MessageMediaDocument)
	if !ok {
		return nil, "", ""
	}
	doc, ok := md.Document.(*tg.DocumentObj)
	if !ok {
		return nil, "", ""
	}
	fileName := ""
	kind := "image"
	for _, attr := range doc.Attributes {
		if _, ok := attr.(*tg.DocumentAttributeVideo); ok {
			kind = "video"
		}
		if _, ok := attr.(*tg.DocumentAttributeAnimated); ok {
			kind = "animated"
		}
		if fn, ok := attr.(*tg.DocumentAttributeFilename); ok {
			fileName = fn.FileName
			if strings.HasSuffix(strings.ToLower(fn.FileName), ".tgs") {
				kind = "tgs"
			}
		}
		if _, ok := attr.(*tg.DocumentAttributeSticker); ok {
			if kind == "image" {
				kind = "sticker"
			}
		}
	}
	mime := strings.ToLower(doc.MimeType)
	if strings.Contains(mime, "application/x-tgsticker") {
		kind = "tgs"
	} else if strings.HasPrefix(mime, "video/") {
		kind = "video"
	}
	return doc, kind, fileName
}

func WebpToJpgHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a webp image or static sticker with <code>/tojpg</code>")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Error:</b> unable to fetch reply: " + html.EscapeString(err.Error()))
		return nil
	}

	if !reply.IsMedia() {
		m.Reply("<b>Error:</b> reply has no media")
		return nil
	}

	doc, kind, fileName := webpToJpgExtractDoc(reply)
	if doc == nil {
		m.Reply("<b>Error:</b> reply is not a document")
		return nil
	}
	if kind == "tgs" {
		m.Reply("<b>Error:</b> animated (.tgs) stickers are not supported")
		return nil
	}
	if kind == "video" {
		m.Reply("<b>Error:</b> video stickers are not supported")
		return nil
	}
	if kind == "animated" {
		m.Reply("<b>Error:</b> animated media is not supported")
		return nil
	}

	mime := strings.ToLower(doc.MimeType)
	lowerName := strings.ToLower(fileName)
	isWebp := strings.Contains(mime, "webp") || strings.HasSuffix(lowerName, ".webp")
	isImage := strings.HasPrefix(mime, "image/")
	if !isWebp && !isImage && kind != "sticker" {
		m.Reply("<b>Error:</b> reply is not a webp or image")
		return nil
	}

	status, _ := m.Reply("<code>converting to jpg...</code>")

	ts := time.Now().UnixNano()
	ext := ".webp"
	if strings.HasSuffix(lowerName, ".png") {
		ext = ".png"
	} else if strings.HasSuffix(lowerName, ".jpg") || strings.HasSuffix(lowerName, ".jpeg") {
		ext = ".jpg"
	}
	srcPath := filepath.Join(os.TempDir(), fmt.Sprintf("tojpg_%d%s", ts, ext))
	jpgPath := filepath.Join(os.TempDir(), fmt.Sprintf("tojpg_%d.jpg", ts))

	fi, err := reply.Download(&tg.DownloadOptions{FileName: srcPath})
	if err != nil {
		msg := "<b>Error:</b> download failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(fi)

	f, err := os.Open(fi)
	if err != nil {
		msg := "<b>Error:</b> open failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	img, fmtName, err := image.Decode(f)
	f.Close()
	if err != nil {
		msg := "<b>Error:</b> decode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	bounds := img.Bounds()
	flat := image.NewRGBA(bounds)
	draw.Draw(flat, bounds, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(flat, bounds, img, bounds.Min, draw.Over)

	jpgOut, err := os.Create(jpgPath)
	if err != nil {
		msg := "<b>Error:</b> create jpg failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if err := jpeg.Encode(jpgOut, flat, &jpeg.Options{Quality: 92}); err != nil {
		jpgOut.Close()
		os.Remove(jpgPath)
		msg := "<b>Error:</b> jpg encode failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	jpgOut.Close()
	defer os.Remove(jpgPath)

	caption := fmt.Sprintf("<b>WebP -&gt; JPG</b>\n<b>Source:</b> <code>%s</code>\n<b>Size:</b> <code>%dx%d</code>", html.EscapeString(fmtName), bounds.Dx(), bounds.Dy())

	if _, err := m.ReplyMedia(jpgPath, &tg.MediaOptions{
		Caption:       caption,
		FileName:      "converted.jpg",
		MimeType:      "image/jpeg",
		ForceDocument: false,
	}); err != nil {
		msg := "<b>Error:</b> upload failed: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	if status != nil {
		status.Delete()
	}
	return nil
}

func registerWebpToJpgHandlers() {
	c := modules.Client
	c.On("cmd:tojpg", WebpToJpgHandler)
}

func initFromSrc_webp_to_jpg_3_1() {
	modules.QueueHandlerRegistration(registerWebpToJpgHandlers)
}

func init() {
	initFromSrc_stickers_0_1()
	initFromSrc_sticker_pack_info_1_1()
	initFromSrc_sticker_to_image_2_1()
	initFromSrc_webp_to_jpg_3_1()
}

func asciiTitle(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	for i := 1; i < len(b); i++ {
		if b[i-1] == ' ' && b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 32
		}
	}
	return string(b)
}
