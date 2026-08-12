package extras

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	modules "main/modules"
	"main/modules/db"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
)

var quotesBucket = []byte("quotes")

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

func quoteAPIBase() string {
	if v := strings.TrimSuffix(strings.TrimSpace(os.Getenv("QUOTE_API")), "/"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

const (
	avatarCacheTTL   = 6 * time.Hour
	avatarCacheDir   = "tmp/avatars"
	avatarNoneMarker = ".none"
)

var (
	avatarClient   *tg.Client
	avatarClientMu sync.RWMutex
)

func AvatarServerInit(c *tg.Client) {
	avatarClientMu.Lock()
	avatarClient = c
	avatarClientMu.Unlock()
	_ = os.MkdirAll(avatarCacheDir, 0o755)
	http.HandleFunc("/avatar/", avatarServeHTTP)
}

func avatarSelfURL() string {
	if v := strings.TrimSuffix(strings.TrimSpace(os.Getenv("BOT_PUBLIC_URL")), "/"); v != "" {
		return v
	}
	return "http://localhost:6060"
}

func avatarServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/avatar/")
	name = strings.TrimSuffix(name, ".jpg")
	uid, err := strconv.ParseInt(name, 10, 64)
	if err != nil || uid == 0 {
		http.NotFound(w, r)
		return
	}

	cached := filepath.Join(avatarCacheDir, fmt.Sprintf("%d.jpg", uid))
	miss := filepath.Join(avatarCacheDir, fmt.Sprintf("%d%s", uid, avatarNoneMarker))

	if st, err := os.Stat(cached); err == nil && time.Since(st.ModTime()) < avatarCacheTTL {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeFile(w, r, cached)
		return
	}
	if st, err := os.Stat(miss); err == nil && time.Since(st.ModTime()) < avatarCacheTTL {
		http.NotFound(w, r)
		return
	}

	avatarClientMu.RLock()
	c := avatarClient
	avatarClientMu.RUnlock()
	if c == nil {
		http.Error(w, "bot not ready", http.StatusServiceUnavailable)
		return
	}

	u, err := c.GetUser(uid)
	if err != nil || u == nil || u.Photo == nil {
		_ = os.WriteFile(miss, nil, 0o644)
		http.NotFound(w, r)
		return
	}
	full, err := c.UsersGetFullUser(&tg.InputUserObj{UserID: uid, AccessHash: u.AccessHash})
	if err != nil || full == nil {
		_ = os.WriteFile(miss, nil, 0o644)
		http.NotFound(w, r)
		return
	}
	var photo tg.Photo
	if full.FullUser.ProfilePhoto != nil {
		photo = full.FullUser.ProfilePhoto
	} else if full.FullUser.PersonalPhoto != nil {
		photo = full.FullUser.PersonalPhoto
	} else if full.FullUser.FallbackPhoto != nil {
		photo = full.FullUser.FallbackPhoto
	}
	p, ok := photo.(*tg.PhotoObj)
	if !ok || p == nil {
		_ = os.WriteFile(miss, nil, 0o644)
		http.NotFound(w, r)
		return
	}
	if _, err := c.DownloadMedia(p, &tg.DownloadOptions{FileName: cached}); err != nil {
		os.Remove(cached)
		http.Error(w, "download failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, cached)
}

type qaEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url,omitempty"`
	UserID int64  `json:"user_id,omitempty"`
}

type qaPhoto struct {
	URL string `json:"url,omitempty"`
}

type qaFrom struct {
	ID           int64    `json:"id"`
	FirstName    string   `json:"first_name,omitempty"`
	LastName     string   `json:"last_name,omitempty"`
	Name         string   `json:"name,omitempty"`
	Username     string   `json:"username,omitempty"`
	Title        string   `json:"title,omitempty"`
	EmojiStatus  string   `json:"emoji_status,omitempty"`
	IsPremium    bool     `json:"is_premium,omitempty"`
	IsVerified   bool     `json:"is_verified,omitempty"`
	IsBot        bool     `json:"is_bot,omitempty"`
	Photo        *qaPhoto `json:"photo,omitempty"`
}

type qaReplyMessage struct {
	Name     string     `json:"name,omitempty"`
	Text     string     `json:"text,omitempty"`
	Entities []qaEntity `json:"entities,omitempty"`
	ChatID   int64      `json:"chatId,omitempty"`
	From     *qaFrom    `json:"from,omitempty"`
}

type qaForward struct {
	Label string `json:"label,omitempty"`
}

type qaMessage struct {
	From         qaFrom          `json:"from"`
	Text         string          `json:"text"`
	Entities     []qaEntity      `json:"entities,omitempty"`
	Avatar       bool            `json:"avatar"`
	ReplyMessage *qaReplyMessage `json:"replyMessage,omitempty"`
	Forward      *qaForward      `json:"forward,omitempty"`
	ViaBot       string          `json:"viaBot,omitempty"`
	SenderTag    string          `json:"senderTag,omitempty"`
	IsQuote      bool            `json:"isQuote,omitempty"`
}

type qaRequest struct {
	Type            string      `json:"type,omitempty"`
	Format          string      `json:"format,omitempty"`
	Ext             string      `json:"ext,omitempty"`
	BackgroundColor string      `json:"backgroundColor,omitempty"`
	Width           int         `json:"width,omitempty"`
	Height          int         `json:"height,omitempty"`
	Scale           int         `json:"scale,omitempty"`
	EmojiBrand      string      `json:"emojiBrand,omitempty"`
	Messages        []qaMessage `json:"messages"`
}

func quoteEntityType(e tg.MessageEntity) (string, string, int64) {
	switch e.(type) {
	case *tg.MessageEntityBold:
		return "bold", "", 0
	case *tg.MessageEntityItalic:
		return "italic", "", 0
	case *tg.MessageEntityUnderline:
		return "underline", "", 0
	case *tg.MessageEntityStrike:
		return "strikethrough", "", 0
	case *tg.MessageEntitySpoiler:
		return "spoiler", "", 0
	case *tg.MessageEntityCode:
		return "code", "", 0
	case *tg.MessageEntityPre:
		return "pre", "", 0
	case *tg.MessageEntityBlockquote:
		return "blockquote", "", 0
	case *tg.MessageEntityBotCommand:
		return "bot_command", "", 0
	case *tg.MessageEntityURL:
		return "url", "", 0
	case *tg.MessageEntityMention:
		return "mention", "", 0
	case *tg.MessageEntityHashtag:
		return "hashtag", "", 0
	case *tg.MessageEntityCashtag:
		return "cashtag", "", 0
	case *tg.MessageEntityEmail:
		return "email", "", 0
	case *tg.MessageEntityPhone:
		return "phone_number", "", 0
	case *tg.MessageEntityTextURL:
		if v, ok := e.(*tg.MessageEntityTextURL); ok {
			return "text_link", v.URL, 0
		}
	case *tg.MessageEntityMentionName:
		if v, ok := e.(*tg.MessageEntityMentionName); ok {
			return "text_mention", "", v.UserID
		}
	case *tg.MessageEntityCustomEmoji:
		return "", "", 0
	}
	return "", "", 0
}

func quoteBuildEntities(src []tg.MessageEntity) []qaEntity {
	if len(src) == 0 {
		return nil
	}
	out := make([]qaEntity, 0, len(src))
	for _, e := range src {
		var (
			offset, length int32
			t              string
			url            string
			uid            int64
		)
		switch v := e.(type) {
		case *tg.MessageEntityBold:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityItalic:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityUnderline:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityStrike:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntitySpoiler:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityCode:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityPre:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityBlockquote:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityBotCommand:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityURL:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityMention:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityHashtag:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityCashtag:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityEmail:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityPhone:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityTextURL:
			offset, length = v.Offset, v.Length
		case *tg.MessageEntityMentionName:
			offset, length = v.Offset, v.Length
		}
		t, url, uid = quoteEntityType(e)
		if t == "" {
			continue
		}
		out = append(out, qaEntity{
			Type:   t,
			Offset: int(offset),
			Length: int(length),
			URL:    url,
			UserID: uid,
		})
	}
	return out
}

func quoteResolveFrom(c *tg.Client, senderID, chatID int64) qaFrom {
	f := qaFrom{ID: senderID}
	if senderID == 0 {
		f.Name = "User"
		return f
	}
	u, err := c.GetUser(senderID)
	if err != nil || u == nil {
		f.Name = fmt.Sprintf("User %d", senderID)
		return f
	}
	f.FirstName = u.FirstName
	f.LastName = u.LastName
	f.Username = u.Username
	f.IsPremium = u.Premium
	f.IsVerified = u.Verified
	f.IsBot = u.Bot
	if u.Photo != nil {
		if p, ok := u.Photo.(*tg.UserProfilePhotoObj); ok && p != nil && p.PhotoID != 0 {
			f.Photo = &qaPhoto{URL: fmt.Sprintf("%s/avatar/%d.jpg", avatarSelfURL(), senderID)}
		}
	}
	if es, ok := u.EmojiStatus.(*tg.EmojiStatusObj); ok && es.DocumentID != 0 {
		f.EmojiStatus = strconv.FormatInt(es.DocumentID, 10)
	} else if esc, ok := u.EmojiStatus.(*tg.EmojiStatusCollectible); ok && esc.DocumentID != 0 {
		f.EmojiStatus = strconv.FormatInt(esc.DocumentID, 10)
	}
	if strings.TrimSpace(f.FirstName+f.LastName) == "" {
		if u.Username != "" {
			f.Name = "@" + u.Username
		} else {
			f.Name = fmt.Sprintf("User %d", senderID)
		}
	}
	if chatID != 0 && !u.Bot {
		if part, perr := c.GetChatMember(chatID, senderID); perr == nil && part != nil {
			if part.Rank != "" {
				f.Title = part.Rank
			} else if part.Status == tg.Creator {
				f.Title = "Owner"
			} else if part.Status == tg.Admin {
				f.Title = "Admin"
			}
		}
	}
	return f
}

func quoteBuildMessage(client *tg.Client, msg *tg.NewMessage, chatID int64, includeReply bool) qaMessage {
	from := quoteResolveFrom(client, msg.SenderID(), chatID)
	out := qaMessage{
		From:     from,
		Text:     msg.RawText(),
		Entities: quoteBuildEntities(msg.Message.Entities),
		Avatar:   true,
	}
	if from.Title != "" {
		out.SenderTag = from.Title
	}
	if msg.Message != nil {
		if msg.Message.ViaBotID != 0 {
			if u, err := client.GetUser(msg.Message.ViaBotID); err == nil && u != nil && u.Username != "" {
				out.ViaBot = "@" + u.Username
			}
		}
		if fh := msg.Message.FwdFrom; fh != nil {
			label := ""
			switch fp := fh.FromID.(type) {
			case *tg.PeerUser:
				if u, err := client.GetUser(fp.UserID); err == nil && u != nil {
					label = strings.TrimSpace(u.FirstName + " " + u.LastName)
					if label == "" && u.Username != "" {
						label = "@" + u.Username
					}
				}
			case *tg.PeerChannel:
				if ch, err := client.GetChannel(fp.ChannelID); err == nil && ch != nil {
					label = ch.Title
				}
			case *tg.PeerChat:
				if ch, err := client.GetChat(fp.ChatID); err == nil && ch != nil {
					label = ch.Title
				}
			}
			if label == "" && fh.FromName != "" {
				label = fh.FromName
			}
			if label != "" {
				out.Forward = &qaForward{Label: label}
			}
		}
	}
	if !includeReply || !msg.IsReply() {
		return out
	}
	prev, err := msg.GetReplyMessage()
	if err != nil || prev == nil {
		return out
	}
	prevFrom := quoteResolveFrom(client, prev.SenderID(), chatID)
	prevName := strings.TrimSpace(prevFrom.FirstName + " " + prevFrom.LastName)
	if prevName == "" {
		prevName = prevFrom.Name
	}
	if prevName == "" {
		prevName = "User"
	}
	rm := &qaReplyMessage{
		Name:     prevName,
		Text:     prev.RawText(),
		Entities: quoteBuildEntities(prev.Message.Entities),
		ChatID:   chatID,
		From:     &prevFrom,
	}
	if strings.TrimSpace(rm.Text) == "" && len(rm.Entities) == 0 {
		return out
	}
	out.ReplyMessage = rm
	return out
}

func quoteRequestImage(req qaRequest, wantPNG bool) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	endpoint := "/generate.webp"
	if wantPNG {
		endpoint = "/generate.png"
	}
	url := quoteAPIBase() + endpoint
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, rerr := io.ReadAll(resp.Body)
	if rerr != nil {
		return nil, rerr
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("quote-api %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("quote-api error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("quote-api returned unexpected json: %s", strings.TrimSpace(string(data)))
	}
	return data, nil
}

func QuoteImageHandler(m *tg.NewMessage) error {
	return quoteImageHandlerImpl(m, false)
}

func QuoteHDImageHandler(m *tg.NewMessage) error {
	return quoteImageHandlerImpl(m, true)
}

func quoteImageHandlerImpl(m *tg.NewMessage, hd bool) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a message with <code>/q</code> to generate a quote.")
		return nil
	}
	target, err := m.GetReplyMessage()
	if err != nil || target == nil {
		m.Reply("<b>Could not read the replied message.</b>")
		return nil
	}
	if strings.TrimSpace(target.RawText()) == "" {
		m.Reply("<b>Nothing to quote.</b> Reply to a text message.")
		return nil
	}

	status, _ := m.Reply("<i>painting your quote...</i>")

	scale := 2
	if hd {
		scale = 5
	}
	bgColor := "#1b1429"
	if fields := strings.Fields(m.Args()); len(fields) > 0 {
		bgColor = fields[0]
	}

	req := qaRequest{
		BackgroundColor: bgColor,
		Width:           512,
		Height:          768,
		Scale:           scale,
		EmojiBrand:      "apple",
		Messages:        []qaMessage{quoteBuildMessage(m.Client, target, m.ChatID(), true)},
	}

	data, rerr := quoteRequestImage(req, hd)
	if rerr != nil {
		if status != nil {
			status.Edit("failed: " + html.EscapeString(rerr.Error()))
		}
		return nil
	}

	ext := ".webp"
	mime := "image/webp"
	if hd {
		ext = ".png"
		mime = "image/png"
	}
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("quote_%d%s", time.Now().UnixNano(), ext))
	if werr := os.WriteFile(outPath, data, 0o644); werr != nil {
		if status != nil {
			status.Edit("failed to write image: " + html.EscapeString(werr.Error()))
		}
		return nil
	}
	defer os.Remove(outPath)

	opts := &tg.MediaOptions{
		FileName: "quote" + ext,
		MimeType: mime,
	}
	if !hd {
		opts.Attributes = []tg.DocumentAttribute{
			&tg.DocumentAttributeSticker{
				Alt:        "💬",
				Stickerset: &tg.InputStickerSetEmpty{},
			},
			&tg.DocumentAttributeFilename{FileName: "quote.webp"},
		}
	}
	if _, merr := m.ReplyMedia(outPath, opts); merr != nil {
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
	for k, _ := c.Seek(prefix); len(k) >= 16; k, _ = c.Next() {
		if !bytes.HasPrefix(k, prefix) {
			break
		}
		id := binary.BigEndian.Uint64(k[8:16])
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1
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
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
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
	end := min(start+perPage, len(all))

	var resp strings.Builder
	fmt.Fprintf(&resp, "<b>Saved Quotes</b> (page %d/%d)\n", page, totalPages)
	resp.WriteString("━━━━━━━━━━━━━━━━\n\n")
	for _, rec := range all[start:end] {
		preview := rec.Text
		if len(preview) > 90 {
			preview = preview[:90] + "..."
		}
		fmt.Fprintf(&resp, "<code>#%d</code> <b>%s</b>\n<i>%s</i>\n\n",
			rec.ID,
			html.EscapeString(rec.UserName),
			html.EscapeString(preview))
	}
	fmt.Fprintf(&resp, "━━━━━━━━━━━━━━━━\n<b>Total:</b> %d quotes\n", len(all))
	if totalPages > 1 {
		fmt.Fprintf(&resp, "<i>Use</i> <code>/quotes %d</code> <i>for next page</i>", page+1)
	}
	m.Reply(resp.String())
	return nil
}

func QuoteDeleteHandler(m *tg.NewMessage) error {
	if !m.IsPrivate() {
		if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
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
	fmt.Fprintf(&resp, "<b>Quote Search:</b> <code>%s</code>\n", html.EscapeString(q))
	resp.WriteString("━━━━━━━━━━━━━━━━\n\n")
	limit := 15
	for i, rec := range matches {
		if i >= limit {
			fmt.Fprintf(&resp, "\n<i>...and %d more</i>", len(matches)-limit)
			break
		}
		preview := rec.Text
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		fmt.Fprintf(&resp, "<code>#%d</code> <b>%s</b>\n<i>%s</i>\n\n",
			rec.ID, html.EscapeString(rec.UserName), html.EscapeString(preview))
	}
	fmt.Fprintf(&resp, "━━━━━━━━━━━━━━━━\n<b>Matches:</b> %d", len(matches))
	m.Reply(resp.String())
	return nil
}

func registerQuotesHandlers() {
	c := modules.Client
	c.On("cmd:q", QuoteImageHandler)
	c.On("cmd:qhd", QuoteHDImageHandler)
	c.On("cmd:qsave", QuoteSaveHandler)
	c.On("cmd:quotes", QuotesListHandler)
	c.On("cmd:delq", QuoteDeleteHandler)
	c.On("cmd:qsearch", QuotesSearchHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerQuotesHandlers)
}

func memeFontPath(name string) string {
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
