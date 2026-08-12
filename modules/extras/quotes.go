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
			if u.Username != "" {
				f.Photo = &qaPhoto{URL: fmt.Sprintf("https://t.me/i/userpic/320/%s.jpg", u.Username)}
			} else {
				f.Photo = &qaPhoto{URL: fmt.Sprintf("%s/avatar/%d.jpg", avatarSelfURL(), senderID)}
			}
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

type quoteFlags struct {
	IncludeReply bool
	N            int
	ImageMode    bool
	HD           bool
	IsQuote      bool
	Brand        string
	Scale        int
	Bg           string
}

func quoteParseFlags(args string) quoteFlags {
	f := quoteFlags{}
	for _, tok := range strings.Fields(args) {
		lower := strings.ToLower(tok)
		switch {
		case lower == "r":
			f.IncludeReply = true
		case lower == "p":
			f.ImageMode = true
		case lower == "hd":
			f.HD = true
		case lower == "q":
			f.IsQuote = true
		case strings.HasPrefix(lower, "brand="):
			f.Brand = strings.TrimPrefix(lower, "brand=")
		case strings.HasPrefix(lower, "scale="):
			if n, err := strconv.Atoi(strings.TrimPrefix(lower, "scale=")); err == nil && n >= 1 && n <= 20 {
				f.Scale = n
			}
		default:
			if n, err := strconv.Atoi(tok); err == nil && n >= 1 && n <= 10 {
				f.N = n
			} else if f.Bg == "" {
				f.Bg = tok
			}
		}
	}
	return f
}

func quoteCollectMessages(client *tg.Client, m *tg.NewMessage, target *tg.NewMessage, flags quoteFlags) []qaMessage {
	chatID := m.ChatID()
	messages := []qaMessage{quoteBuildMessage(client, target, chatID, flags.IncludeReply)}
	if flags.IsQuote {
		messages[0].IsQuote = true
	}
	if flags.N <= 1 {
		return messages
	}
	history, err := client.GetMessages(chatID, &tg.SearchOption{
		MaxID: target.ID,
		Limit: int32(flags.N - 1),
	})
	if err != nil || len(history) == 0 {
		return messages
	}
	extra := make([]qaMessage, 0, len(history))
	for i := range history {
		older := &history[i]
		if older.ID == target.ID {
			continue
		}
		if strings.TrimSpace(older.RawText()) == "" {
			continue
		}
		bm := quoteBuildMessage(client, older, chatID, false)
		if flags.IsQuote {
			bm.IsQuote = true
		}
		extra = append(extra, bm)
	}
	all := make([]qaMessage, 0, len(extra)+len(messages))
	for i := len(extra) - 1; i >= 0; i-- {
		all = append(all, extra[i])
	}
	all = append(all, messages...)
	return all
}

func quoteRateBucket() []byte { return []byte("quote_rate") }
func quoteVotesBucket() []byte { return []byte("quote_votes") }
func quotePacksBucket() []byte { return []byte("quote_packs_by_chat") }

func quoteRateIsOn(chatID int64) bool {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return false
	}
	on := false
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(quoteRateBucket())
		if b == nil {
			return nil
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(chatID))
		v := b.Get(key)
		if len(v) == 1 && v[0] == 1 {
			on = true
		}
		return nil
	})
	return on
}

func quoteRateSet(chatID int64, on bool) error {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return fmt.Errorf("db unavailable")
	}
	return d.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(quoteRateBucket())
		if e != nil {
			return e
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(chatID))
		val := []byte{0}
		if on {
			val = []byte{1}
		}
		return b.Put(key, val)
	})
}

type quoteVoteRecord struct {
	ChatID     int64           `json:"chat_id"`
	QuoteID    uint64          `json:"quote_id"`
	Up         map[int64]bool  `json:"up"`
	Down       map[int64]bool  `json:"down"`
	QuoterID   int64           `json:"quoter_id"`
	QuoterName string          `json:"quoter_name"`
	QuotedID   int64           `json:"quoted_id"`
	QuotedName string          `json:"quoted_name"`
	Preview    string          `json:"preview"`
	MessageID  int32           `json:"message_id"`
	Timestamp  int64           `json:"ts"`
}

func quoteVoteKey(chatID int64, quoteID uint64) []byte {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[0:8], uint64(chatID))
	binary.BigEndian.PutUint64(b[8:16], quoteID)
	return b
}

func quoteVoteGet(chatID int64, quoteID uint64) *quoteVoteRecord {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return nil
	}
	var rec quoteVoteRecord
	found := false
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(quoteVotesBucket())
		if b == nil {
			return nil
		}
		raw := b.Get(quoteVoteKey(chatID, quoteID))
		if raw == nil {
			return nil
		}
		if jerr := json.Unmarshal(raw, &rec); jerr == nil {
			found = true
			if rec.Up == nil {
				rec.Up = map[int64]bool{}
			}
			if rec.Down == nil {
				rec.Down = map[int64]bool{}
			}
		}
		return nil
	})
	if !found {
		return nil
	}
	return &rec
}

func quoteVoteSave(rec *quoteVoteRecord) error {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return fmt.Errorf("db unavailable")
	}
	return d.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(quoteVotesBucket())
		if e != nil {
			return e
		}
		raw, jerr := json.Marshal(rec)
		if jerr != nil {
			return jerr
		}
		return b.Put(quoteVoteKey(rec.ChatID, rec.QuoteID), raw)
	})
}

func quoteVoteKeyboard(chatID int64, quoteID uint64, up, down int) *tg.ReplyInlineMarkup {
	upBtn := tg.Button.Data(fmt.Sprintf("👍 %d", up), fmt.Sprintf("qvote:%d:up", quoteID))
	downBtn := tg.Button.Data(fmt.Sprintf("👎 %d", down), fmt.Sprintf("qvote:%d:down", quoteID))
	return tg.NewKeyboard().AddRow(upBtn, downBtn).Build()
}

func quoteRenderCore(m *tg.NewMessage, target *tg.NewMessage, flags quoteFlags) error {
	if strings.TrimSpace(target.RawText()) == "" {
		m.Reply("<b>Nothing to quote.</b> Reply to a text message.")
		return nil
	}

	status, _ := m.Reply("<i>painting your quote...</i>")

	scale := 2
	if flags.HD {
		scale = 5
	}
	if flags.Scale > 0 {
		scale = flags.Scale
	}
	bg := flags.Bg
	if bg == "" {
		bg = "#1b1429"
	}
	brand := flags.Brand
	if brand == "" {
		brand = "apple"
	}

	messages := quoteCollectMessages(m.Client, m, target, flags)
	req := qaRequest{
		BackgroundColor: bg,
		Width:           512,
		Height:          768,
		Scale:           scale,
		EmojiBrand:      brand,
		Messages:        messages,
	}
	if flags.ImageMode {
		req.Type = "image"
	}

	wantPNG := flags.ImageMode || flags.HD
	data, rerr := quoteRequestImage(req, wantPNG)
	if rerr != nil {
		if status != nil {
			status.Edit("failed: " + html.EscapeString(rerr.Error()))
		}
		return nil
	}

	ext := ".webp"
	mime := "image/webp"
	if wantPNG {
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

	quotedID, quotedName := quoteExtractQuoted(m.Client, target)
	quoteID, _ := quoteAutoSave(m, target, quotedID, quotedName)

	opts := &tg.MediaOptions{
		FileName: "quote" + ext,
		MimeType: mime,
	}
	if !wantPNG {
		opts.Attributes = []tg.DocumentAttribute{
			&tg.DocumentAttributeSticker{
				Alt:        "💬",
				Stickerset: &tg.InputStickerSetEmpty{},
			},
			&tg.DocumentAttributeFilename{FileName: "quote.webp"},
		}
	}
	if quoteRateIsOn(m.ChatID()) && quoteID != 0 {
		opts.ReplyMarkup = quoteVoteKeyboard(m.ChatID(), quoteID, 0, 0)
	}

	sent, merr := m.ReplyMedia(outPath, opts)
	if merr != nil {
		if status != nil {
			status.Edit("upload failed: " + html.EscapeString(merr.Error()))
		}
		return nil
	}
	if sent != nil && quoteID != 0 {
		rec := &quoteVoteRecord{
			ChatID:     m.ChatID(),
			QuoteID:    quoteID,
			Up:         map[int64]bool{},
			Down:       map[int64]bool{},
			QuoterID:   m.SenderID(),
			QuoterName: quoteSenderName(m),
			QuotedID:   quotedID,
			QuotedName: quotedName,
			Preview:    quotePreview(target.RawText()),
			MessageID:  sent.ID,
			Timestamp:  time.Now().Unix(),
		}
		_ = quoteVoteSave(rec)
	}
	if status != nil {
		status.Delete()
	}
	return nil
}

func quoteExtractQuoted(client *tg.Client, target *tg.NewMessage) (int64, string) {
	if target == nil {
		return 0, ""
	}
	uid := target.SenderID()
	if uid == 0 {
		return 0, "User"
	}
	u, err := client.GetUser(uid)
	if err != nil || u == nil {
		return uid, fmt.Sprintf("User %d", uid)
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		if u.Username != "" {
			name = "@" + u.Username
		} else {
			name = fmt.Sprintf("User %d", uid)
		}
	}
	return uid, name
}

func quoteSenderName(m *tg.NewMessage) string {
	if m.Sender == nil {
		return "User"
	}
	name := strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
	if name == "" {
		if m.Sender.Username != "" {
			name = "@" + m.Sender.Username
		} else {
			name = "User"
		}
	}
	return name
}

func quotePreview(text string) string {
	text = strings.TrimSpace(text)
	if len(text) > 120 {
		return text[:120] + "..."
	}
	return text
}

func quoteAutoSave(m *tg.NewMessage, target *tg.NewMessage, quotedID int64, quotedName string) (uint64, error) {
	text := strings.TrimSpace(target.RawText())
	if text == "" {
		return 0, nil
	}
	if len(text) > 4000 {
		text = text[:4000]
	}
	if err := quotesEnsureBucket(); err != nil {
		return 0, err
	}
	d, err := db.GetDB()
	if err != nil || d == nil {
		return 0, fmt.Errorf("db unavailable")
	}
	handle := ""
	if u, uerr := m.Client.GetUser(quotedID); uerr == nil && u != nil {
		handle = u.Username
	}
	savedByName := quoteSenderName(m)
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
			UserID:      quotedID,
			UserName:    quotedName,
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
	return newID, werr
}

func QuoteImageHandler(m *tg.NewMessage) error {
	flags := quoteParseFlags(m.Args())
	target, err := quoteResolveTarget(m, flags)
	if err != nil || target == nil {
		m.Reply("<b>Usage:</b> reply to a message with <code>/q</code>.\nFlags: <code>r</code> include reply, <code>N</code> stack N messages, <code>p</code> image mode, <code>hd</code>, <code>q</code> blockquote style, <code>brand=apple|google|twitter</code>, <code>scale=1-20</code>, color name or hex.")
		return nil
	}
	return quoteRenderCore(m, target, flags)
}

func QuoteHDImageHandler(m *tg.NewMessage) error {
	flags := quoteParseFlags(m.Args())
	flags.HD = true
	target, err := quoteResolveTarget(m, flags)
	if err != nil || target == nil {
		m.Reply("<b>Usage:</b> reply to a message with <code>/qhd</code>.")
		return nil
	}
	return quoteRenderCore(m, target, flags)
}

func quoteResolveTarget(m *tg.NewMessage, _ quoteFlags) (*tg.NewMessage, error) {
	if !m.IsReply() {
		return nil, fmt.Errorf("no reply")
	}
	target, err := m.GetReplyMessage()
	if err != nil || target == nil {
		return nil, fmt.Errorf("could not read reply")
	}
	return target, nil
}

func QuoteVoteCallback(m *tg.CallbackQuery) error {
	data := string(m.Data)
	parts := strings.Split(data, ":")
	if len(parts) != 3 {
		m.Answer("bad vote", &tg.CallbackOptions{Alert: false})
		return nil
	}
	quoteID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		m.Answer("bad vote", &tg.CallbackOptions{Alert: false})
		return nil
	}
	direction := parts[2]
	if direction != "up" && direction != "down" {
		m.Answer("bad vote", &tg.CallbackOptions{Alert: false})
		return nil
	}
	chatID := m.ChatID
	if !quoteRateIsOn(chatID) {
		m.Answer("voting is disabled here", &tg.CallbackOptions{Alert: false})
		return nil
	}
	rec := quoteVoteGet(chatID, quoteID)
	if rec == nil {
		m.Answer("quote not found", &tg.CallbackOptions{Alert: false})
		return nil
	}
	uid := m.SenderID
	if rec.Up == nil {
		rec.Up = map[int64]bool{}
	}
	if rec.Down == nil {
		rec.Down = map[int64]bool{}
	}
	toast := ""
	if direction == "up" {
		if rec.Up[uid] {
			delete(rec.Up, uid)
			toast = "removed 👍"
		} else {
			rec.Up[uid] = true
			delete(rec.Down, uid)
			toast = "👍"
		}
	} else {
		if rec.Down[uid] {
			delete(rec.Down, uid)
			toast = "removed 👎"
		} else {
			rec.Down[uid] = true
			delete(rec.Up, uid)
			toast = "👎"
		}
	}
	_ = quoteVoteSave(rec)
	kb := quoteVoteKeyboard(chatID, quoteID, len(rec.Up), len(rec.Down))
	_, _ = m.Client.EditMessage(chatID, rec.MessageID, "", &tg.SendOptions{ReplyMarkup: kb})
	m.Answer(toast, &tg.CallbackOptions{Alert: false})
	return nil
}

func QuoteRateHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>/qrate</b> works in groups only.")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		m.Reply("<b>Admins only.</b>")
		return nil
	}
	arg := strings.ToLower(strings.TrimSpace(m.Args()))
	switch arg {
	case "on", "enable":
		if err := quoteRateSet(m.ChatID(), true); err != nil {
			m.Reply("<b>DB error.</b>")
			return nil
		}
		m.Reply("<b>Quote voting enabled.</b> New <code>/q</code> messages will get 👍/👎 buttons.")
	case "off", "disable":
		if err := quoteRateSet(m.ChatID(), false); err != nil {
			m.Reply("<b>DB error.</b>")
			return nil
		}
		m.Reply("<b>Quote voting disabled.</b>")
	default:
		state := "off"
		if quoteRateIsOn(m.ChatID()) {
			state = "on"
		}
		m.Reply(fmt.Sprintf("<b>Quote voting:</b> <code>%s</code>\n<i>Usage:</i> <code>/qrate on|off</code>", state))
	}
	return nil
}

func QuoteRandHandler(m *tg.NewMessage) error {
	all, err := quotesListByChat(m.ChatID())
	if err != nil || len(all) == 0 {
		m.Reply("<b>No quotes saved here yet.</b> Run <code>/q</code> on a message first.")
		return nil
	}
	idx := int(time.Now().UnixNano()) % len(all)
	if idx < 0 {
		idx = -idx
	}
	rec := all[idx]
	preview := rec.Text
	if len(preview) > 300 {
		preview = preview[:300] + "..."
	}
	m.Reply(fmt.Sprintf("<b>%s</b> once said <code>#%d</code>:\n\n<i>%s</i>",
		html.EscapeString(rec.UserName), rec.ID, html.EscapeString(preview)))
	return nil
}

func QuoteTopHandler(m *tg.NewMessage) error {
	n := 10
	if a := strings.TrimSpace(m.Args()); a != "" {
		if v, err := strconv.Atoi(a); err == nil && v > 0 && v <= 50 {
			n = v
		}
	}
	d, err := db.GetDB()
	if err != nil || d == nil {
		m.Reply("<b>DB error.</b>")
		return nil
	}
	type ranked struct {
		Rec   *quoteVoteRecord
		Score int
	}
	var out []ranked
	prefix := make([]byte, 8)
	binary.BigEndian.PutUint64(prefix, uint64(m.ChatID()))
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(quoteVotesBucket())
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var rec quoteVoteRecord
			if jerr := json.Unmarshal(v, &rec); jerr != nil {
				continue
			}
			score := len(rec.Up) - len(rec.Down)
			if score == 0 && len(rec.Up)+len(rec.Down) == 0 {
				continue
			}
			out = append(out, ranked{&rec, score})
		}
		return nil
	})
	if len(out) == 0 {
		m.Reply("<b>No voted quotes yet.</b> Enable voting with <code>/qrate on</code>.")
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Rec.Timestamp > out[j].Rec.Timestamp
	})
	if len(out) > n {
		out = out[:n]
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>Top Quotes</b> (top %d)\n━━━━━━━━━━━━━━━━\n\n", len(out))
	for i, r := range out {
		fmt.Fprintf(&sb, "<b>%d.</b> <code>#%d</code> <b>%s</b> — 👍 %d 👎 %d (score %+d)\n<i>%s</i>\n\n",
			i+1, r.Rec.QuoteID, html.EscapeString(r.Rec.QuotedName), len(r.Rec.Up), len(r.Rec.Down), r.Score, html.EscapeString(r.Rec.Preview))
	}
	m.Reply(sb.String())
	return nil
}

type quoteChatPack struct {
	OwnerID      int64  `json:"owner_id"`
	OwnerName    string `json:"owner_name"`
	ShortName    string `json:"short_name"`
	Title        string `json:"title"`
	PackNumber   int    `json:"pack_number"`
	StickerCount int    `json:"sticker_count"`
}

func quoteChatPackGet(chatID int64) *quoteChatPack {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return nil
	}
	var pack quoteChatPack
	found := false
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(quotePacksBucket())
		if b == nil {
			return nil
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(chatID))
		raw := b.Get(key)
		if raw == nil {
			return nil
		}
		if jerr := json.Unmarshal(raw, &pack); jerr == nil {
			found = true
		}
		return nil
	})
	if !found {
		return nil
	}
	return &pack
}

func quoteChatPackSave(chatID int64, pack *quoteChatPack) error {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return fmt.Errorf("db unavailable")
	}
	return d.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(quotePacksBucket())
		if e != nil {
			return e
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(chatID))
		raw, jerr := json.Marshal(pack)
		if jerr != nil {
			return jerr
		}
		return b.Put(key, raw)
	})
}

func QuoteStickerHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>/qs</b> works in groups only.")
		return nil
	}
	flags := quoteParseFlags(m.Args())
	target, err := quoteResolveTarget(m, flags)
	if err != nil || target == nil {
		m.Reply("<b>Usage:</b> reply to a message with <code>/qs</code> to add it to this group's quote sticker pack.")
		return nil
	}
	if strings.TrimSpace(target.RawText()) == "" {
		m.Reply("<b>Nothing to quote.</b>")
		return nil
	}

	status, _ := m.Reply("<i>rendering + adding to pack...</i>")

	scale := 2
	if flags.HD {
		scale = 5
	}
	if flags.Scale > 0 {
		scale = flags.Scale
	}
	bg := flags.Bg
	if bg == "" {
		bg = "#1b1429"
	}
	brand := flags.Brand
	if brand == "" {
		brand = "apple"
	}

	messages := quoteCollectMessages(m.Client, m, target, flags)
	req := qaRequest{
		BackgroundColor: bg,
		Width:           512,
		Height:          768,
		Scale:           scale,
		EmojiBrand:      brand,
		Messages:        messages,
	}
	data, rerr := quoteRequestImage(req, false)
	if rerr != nil {
		if status != nil {
			status.Edit("failed: " + html.EscapeString(rerr.Error()))
		}
		return nil
	}
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("qs_%d.webp", time.Now().UnixNano()))
	if werr := os.WriteFile(outPath, data, 0o644); werr != nil {
		if status != nil {
			status.Edit("failed to write image: " + html.EscapeString(werr.Error()))
		}
		return nil
	}
	defer os.Remove(outPath)

	me := m.Client.Me()
	if me == nil || me.Username == "" {
		if status != nil {
			status.Edit("bot has no username; cannot own sticker sets")
		}
		return nil
	}

	pack := quoteChatPackGet(m.ChatID())
	needNew := pack == nil || pack.StickerCount >= MaxStickersPerPack

	media, gerr := m.Client.GetSendableMedia(outPath, &tg.MediaMetadata{Inline: true})
	if gerr != nil {
		if status != nil {
			status.Edit("prepare media failed: " + html.EscapeString(gerr.Error()))
		}
		return nil
	}
	inMedia, ok := media.(*tg.InputMediaDocument)
	if !ok {
		if status != nil {
			status.Edit("unexpected media type from Telegram")
		}
		return nil
	}
	doc := inMedia.ID
	emoji := "💬"

	if needNew {
		chatIDAbs := m.ChatID()
		if chatIDAbs < 0 {
			chatIDAbs = -chatIDAbs
		}
		packNumber := 1
		if pack != nil {
			packNumber = pack.PackNumber + 1
		}
		title := ""
		if chat, cerr := m.Client.GetChat(m.ChatID()); cerr == nil && chat != nil {
			title = chat.Title
		}
		if title == "" {
			title = fmt.Sprintf("Chat %d", m.ChatID())
		}
		shortName := fmt.Sprintf("xquotes_%d_%d_by_%s", chatIDAbs, packNumber, me.Username)
		fullTitle := fmt.Sprintf("%s — Quotes #%d", title, packNumber)

		ownerID := m.SenderID()
		ownerName := quoteSenderName(m)
		if pack != nil {
			ownerID = pack.OwnerID
			ownerName = pack.OwnerName
		}

		_, createErr := m.Client.StickersCreateStickerSet(&tg.StickersCreateStickerSetParams{
			UserID:    &tg.InputUserObj{UserID: ownerID, AccessHash: quoteResolveAccessHash(m, ownerID)},
			Title:     fullTitle,
			ShortName: shortName,
			Stickers: []*tg.InputStickerSetItem{
				{Document: doc, Emoji: emoji},
			},
		})
		if createErr != nil {
			if status != nil {
				status.Edit(html.EscapeString(stickerFriendlyError(m, createErr)))
			}
			return nil
		}
		newPack := &quoteChatPack{
			OwnerID:      ownerID,
			OwnerName:    ownerName,
			ShortName:    shortName,
			Title:        fullTitle,
			PackNumber:   packNumber,
			StickerCount: 1,
		}
		if err := quoteChatPackSave(m.ChatID(), newPack); err != nil {
			if status != nil {
				status.Edit("saved on Telegram but DB save failed: " + html.EscapeString(err.Error()))
			}
			return nil
		}
		if status != nil {
			status.Edit(fmt.Sprintf(
				"<b>New quote pack created for this chat.</b>\nPack: <a href=\"https://t.me/addstickers/%s\">%s</a>\nStickers: 1/%d",
				shortName, html.EscapeString(fullTitle), MaxStickersPerPack,
			))
		}
		return nil
	}

	_, addErr := m.Client.StickersAddStickerToSet(
		&tg.InputStickerSetShortName{ShortName: pack.ShortName},
		&tg.InputStickerSetItem{Document: doc, Emoji: emoji},
	)
	if addErr != nil {
		if status != nil {
			status.Edit(html.EscapeString(stickerFriendlyError(m, addErr)))
		}
		return nil
	}
	pack.StickerCount++
	_ = quoteChatPackSave(m.ChatID(), pack)
	if status != nil {
		status.Edit(fmt.Sprintf(
			"<b>Added to this chat's quote pack.</b>\nPack: <a href=\"https://t.me/addstickers/%s\">%s</a>\nStickers: %d/%d",
			pack.ShortName, html.EscapeString(pack.Title), pack.StickerCount, MaxStickersPerPack,
		))
	}
	return nil
}

func quoteResolveAccessHash(m *tg.NewMessage, userID int64) int64 {
	if m.Sender != nil && m.Sender.ID == userID {
		return m.Sender.AccessHash
	}
	if u, err := m.Client.GetUser(userID); err == nil && u != nil {
		return u.AccessHash
	}
	return 0
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
	c.On("cmd:qrate", QuoteRateHandler)
	c.On("cmd:qrand", QuoteRandHandler)
	c.On("cmd:qtop", QuoteTopHandler)
	c.On("cmd:qs", QuoteStickerHandler)
	c.On("callback:qvote:", QuoteVoteCallback)
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
