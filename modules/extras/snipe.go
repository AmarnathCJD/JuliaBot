package extras

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
	modules "main/modules"
)

const snipeBucket = "snipe_cfg"
const snipeRingSize = 20

type snipedMsg struct {
	MessageID  int32
	ChatID     int64
	SenderID   int64
	SenderName string
	Text       string
	FileID     string
	IsSticker  bool
	OldText    string
	NewText    string
	IsEdit     bool
	Time       int64
}

type snipeCfg struct {
	Enabled bool `json:"enabled"`
}

var (
	snipeStore      = make(map[int64][]*snipedMsg)
	snipeEditStore  = make(map[int64][]*snipedMsg)
	snipeMu         sync.RWMutex
	snipeCache      = make(map[int64]*snipeCfg)
	snipeCacheMu    sync.RWMutex
	snipeMsgCache   = make(map[int64]map[int32]*snipedMsg)
	snipeMsgCacheMu sync.RWMutex
)

func snipeKey(chatID int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(chatID))
	return b
}

func defaultSnipeCfg() *snipeCfg {
	return &snipeCfg{Enabled: true}
}

func loadSnipeCfg(chatID int64) *snipeCfg {
	snipeCacheMu.RLock()
	if v, ok := snipeCache[chatID]; ok {
		cp := *v
		snipeCacheMu.RUnlock()
		return &cp
	}
	snipeCacheMu.RUnlock()

	cfg := defaultSnipeCfg()
	d, err := db.GetDB()
	if err != nil || d == nil {
		return cfg
	}
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(snipeBucket))
		if b == nil {
			return nil
		}
		raw := b.Get(snipeKey(chatID))
		if len(raw) == 0 {
			return nil
		}
		_ = json.Unmarshal(raw, cfg)
		return nil
	})

	snipeCacheMu.Lock()
	cp := *cfg
	snipeCache[chatID] = &cp
	snipeCacheMu.Unlock()
	return cfg
}

func saveSnipeCfg(chatID int64, cfg *snipeCfg) error {
	d, err := db.GetDB()
	if err != nil || d == nil {
		return fmt.Errorf("db unavailable")
	}
	err = d.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(snipeBucket))
		if err != nil {
			return err
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return b.Put(snipeKey(chatID), raw)
	})
	if err != nil {
		return err
	}
	snipeCacheMu.Lock()
	cp := *cfg
	snipeCache[chatID] = &cp
	snipeCacheMu.Unlock()
	return nil
}

func snipeAuthorName(m *tg.NewMessage) string {
	if m.Sender == nil {
		return "Unknown"
	}
	name := strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
	if name == "" {
		name = m.Sender.Username
	}
	if name == "" {
		name = strconv.FormatInt(m.Sender.ID, 10)
	}
	return name
}

func snipeCacheMessage(m *tg.NewMessage) {
	if m == nil || m.Message == nil {
		return
	}
	chatID := m.ChatID()
	if chatID == 0 {
		return
	}
	if !loadSnipeCfg(chatID).Enabled {
		return
	}

	entry := &snipedMsg{
		MessageID:  m.ID,
		ChatID:     chatID,
		SenderID:   m.SenderID(),
		SenderName: snipeAuthorName(m),
		Text:       m.Text(),
		Time:       time.Now().Unix(),
	}

	if m.IsMedia() && m.File != nil {
		entry.FileID = m.File.FileID
		if m.Sticker() != nil {
			entry.IsSticker = true
		}
	}

	snipeMsgCacheMu.Lock()
	if _, ok := snipeMsgCache[chatID]; !ok {
		snipeMsgCache[chatID] = make(map[int32]*snipedMsg)
	}
	snipeMsgCache[chatID][m.ID] = entry
	if len(snipeMsgCache[chatID]) > 500 {
		var oldestID int32
		var oldestTime int64 = 1<<62
		for id, e := range snipeMsgCache[chatID] {
			if e.Time < oldestTime {
				oldestTime = e.Time
				oldestID = id
			}
		}
		delete(snipeMsgCache[chatID], oldestID)
	}
	snipeMsgCacheMu.Unlock()
}

func snipeCacheGet(chatID int64, msgID int32) *snipedMsg {
	snipeMsgCacheMu.RLock()
	defer snipeMsgCacheMu.RUnlock()
	if m, ok := snipeMsgCache[chatID]; ok {
		if e, ok := m[msgID]; ok {
			return e
		}
	}
	return nil
}

func snipeCacheRemove(chatID int64, msgID int32) {
	snipeMsgCacheMu.Lock()
	defer snipeMsgCacheMu.Unlock()
	if m, ok := snipeMsgCache[chatID]; ok {
		delete(m, msgID)
	}
}

func snipePush(chatID int64, entry *snipedMsg) {
	snipeMu.Lock()
	defer snipeMu.Unlock()
	if entry.IsEdit {
		snipeEditStore[chatID] = append([]*snipedMsg{entry}, snipeEditStore[chatID]...)
		if len(snipeEditStore[chatID]) > snipeRingSize {
			snipeEditStore[chatID] = snipeEditStore[chatID][:snipeRingSize]
		}
	} else {
		snipeStore[chatID] = append([]*snipedMsg{entry}, snipeStore[chatID]...)
		if len(snipeStore[chatID]) > snipeRingSize {
			snipeStore[chatID] = snipeStore[chatID][:snipeRingSize]
		}
	}
}

func snipeGetDeleted(chatID int64) []*snipedMsg {
	snipeMu.RLock()
	defer snipeMu.RUnlock()
	res := make([]*snipedMsg, len(snipeStore[chatID]))
	copy(res, snipeStore[chatID])
	return res
}

func snipeGetLastEdit(chatID int64) *snipedMsg {
	snipeMu.RLock()
	defer snipeMu.RUnlock()
	if len(snipeEditStore[chatID]) > 0 {
		return snipeEditStore[chatID][0]
	}
	return nil
}

func SnipeCacheHandler(m *tg.NewMessage) error {
	if m == nil || m.IsPrivate() {
		return nil
	}
	if m.SenderID() == 0 {
		return nil
	}
	snipeCacheMessage(m)
	return nil
}

func SnipeDeleteHandler(d *tg.DeleteMessage) error {
	if d == nil {
		return nil
	}

	var chatID int64
	if d.ChannelID != 0 {
		chatID = -1_000_000_000_000 - d.ChannelID
	}

	for _, msgID := range d.Messages {
		var entry *snipedMsg
		if chatID != 0 {
			entry = snipeCacheGet(chatID, msgID)
		}
		if entry == nil {
			snipeMsgCacheMu.RLock()
			for cid, m := range snipeMsgCache {
				if e, ok := m[msgID]; ok {
					entry = e
					chatID = cid
					break
				}
			}
			snipeMsgCacheMu.RUnlock()
		}
		if entry == nil {
			continue
		}
		if !loadSnipeCfg(entry.ChatID).Enabled {
			snipeCacheRemove(entry.ChatID, msgID)
			continue
		}
		snipePush(entry.ChatID, entry)
		snipeCacheRemove(entry.ChatID, msgID)
	}
	return nil
}

func SnipeEditHandler(m *tg.NewMessage) error {
	if m == nil || m.IsPrivate() {
		return nil
	}
	if m.SenderID() == 0 {
		return nil
	}
	chatID := m.ChatID()
	if !loadSnipeCfg(chatID).Enabled {
		snipeCacheMessage(m)
		return nil
	}

	old := snipeCacheGet(chatID, m.ID)
	newText := m.Text()
	if old != nil && old.Text != newText {
		entry := &snipedMsg{
			MessageID:  m.ID,
			ChatID:     chatID,
			SenderID:   m.SenderID(),
			SenderName: snipeAuthorName(m),
			OldText:    old.Text,
			NewText:    newText,
			IsEdit:     true,
			Time:       time.Now().Unix(),
		}
		snipePush(chatID, entry)
	}

	snipeCacheMessage(m)
	return nil
}

func formatSnipeTime(ts int64) string {
	d := time.Since(time.Unix(ts, 0))
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func sendSnipedEntry(m *tg.NewMessage, entry *snipedMsg) {
	header := fmt.Sprintf("<b>Sniped from %s</b> <i>(%s)</i>", html.EscapeString(entry.SenderName), formatSnipeTime(entry.Time))

	if entry.FileID != "" {
		caption := header
		if entry.Text != "" {
			caption += "\n\n" + html.EscapeString(entry.Text)
		}
		media, err := tg.ResolveBotFileID(entry.FileID)
		if err == nil {
			if entry.IsSticker {
				m.ReplyMedia(media)
				m.Respond(caption)
			} else {
				m.ReplyMedia(media, &tg.MediaOptions{Caption: caption})
			}
			return
		}
	}

	body := header
	if entry.Text != "" {
		body += "\n\n" + html.EscapeString(entry.Text)
	} else {
		body += "\n\n<i>[no text]</i>"
	}
	m.Reply(body)
}

func SnipeHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Snipe works in groups only.</b>")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		m.Reply("Admins only.")
		return nil
	}

	arg := strings.TrimSpace(m.Args())
	argLower := strings.ToLower(arg)
	switch argLower {
	case "on", "off", "enable", "disable", "yes", "no":
		return SnipeToggleHandler(m)
	}

	if !loadSnipeCfg(m.ChatID()).Enabled {
		m.Reply("<b>Snipe is disabled in this chat.</b>")
		return nil
	}

	n := 1
	if arg != "" {
		v, err := strconv.Atoi(arg)
		if err != nil || v < 1 {
			m.Reply("<b>Usage:</b> <code>/snipe [N]</code> or <code>/snipe on|off</code>")
			return nil
		}
		n = v
	}

	deleted := snipeGetDeleted(m.ChatID())
	if len(deleted) == 0 {
		m.Reply("<b>Nothing to snipe.</b>")
		return nil
	}
	if n > len(deleted) {
		m.Reply(fmt.Sprintf("<b>Only %d deleted message(s) available.</b>", len(deleted)))
		return nil
	}

	sendSnipedEntry(m, deleted[n-1])
	return nil
}

func ESnipeHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Snipe works in groups only.</b>")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		m.Reply("Admins only.")
		return nil
	}
	if !loadSnipeCfg(m.ChatID()).Enabled {
		m.Reply("<b>Snipe is disabled in this chat.</b>")
		return nil
	}

	entry := snipeGetLastEdit(m.ChatID())
	if entry == nil {
		m.Reply("<b>No edits sniped.</b>")
		return nil
	}

	body := fmt.Sprintf("<b>Edit by %s</b> <i>(%s)</i>\n\n<s>%s</s> → <b>%s</b>",
		html.EscapeString(entry.SenderName),
		formatSnipeTime(entry.Time),
		html.EscapeString(entry.OldText),
		html.EscapeString(entry.NewText),
	)
	m.Reply(body)
	return nil
}

func renderSnipesPage(entries []*snipedMsg, page int) (string, int, int) {
	const perPage = 10
	total := len(entries)
	totalPages := (total + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * perPage
	end := start + perPage
	if end > total {
		end = total
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Sniped Messages</b> <i>(page %d/%d)</i>\n", page, totalPages))
	sb.WriteString("━━━━━━━━━━━━━━━━\n\n")

	if total == 0 {
		sb.WriteString("<i>Nothing to snipe.</i>")
		return sb.String(), page, totalPages
	}

	for i := start; i < end; i++ {
		entry := entries[i]
		preview := entry.Text
		if preview == "" {
			if entry.IsSticker {
				preview = "[sticker]"
			} else if entry.FileID != "" {
				preview = "[media]"
			} else {
				preview = "[no text]"
			}
		}
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		sb.WriteString(fmt.Sprintf("<b>%d.</b> <b>%s</b> <i>(%s)</i>\n  %s\n\n",
			i+1,
			html.EscapeString(entry.SenderName),
			formatSnipeTime(entry.Time),
			html.EscapeString(preview),
		))
	}

	return sb.String(), page, totalPages
}

func snipesKeyboard(page, totalPages int, userID int64) *tg.ReplyInlineMarkup {
	if totalPages <= 1 {
		return nil
	}
	b := tg.Button
	kb := tg.NewKeyboard()
	var row []tg.KeyboardButton
	if page > 1 {
		row = append(row, b.Data("« Prev", fmt.Sprintf("snipes_%d_%d", page-1, userID)))
	}
	row = append(row, b.Data(fmt.Sprintf("%d/%d", page, totalPages), "snipes_noop"))
	if page < totalPages {
		row = append(row, b.Data("Next »", fmt.Sprintf("snipes_%d_%d", page+1, userID)))
	}
	kb.AddRow(row...)
	return kb.Build()
}

func SnipesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Snipe works in groups only.</b>")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		m.Reply("Admins only.")
		return nil
	}
	if !loadSnipeCfg(m.ChatID()).Enabled {
		m.Reply("<b>Snipe is disabled in this chat.</b>")
		return nil
	}

	page := 1
	arg := strings.TrimSpace(m.Args())
	if arg != "" {
		v, err := strconv.Atoi(arg)
		if err == nil && v >= 1 {
			page = v
		}
	}

	entries := snipeGetDeleted(m.ChatID())
	body, page, totalPages := renderSnipesPage(entries, page)

	opts := &tg.SendOptions{}
	if kb := snipesKeyboard(page, totalPages, m.SenderID()); kb != nil {
		opts.ReplyMarkup = kb
	}
	m.Reply(body, opts)
	return nil
}

func SnipesCallback(c *tg.CallbackQuery) error {
	data := c.DataString()
	if data == "snipes_noop" {
		c.Answer("")
		return nil
	}
	if !strings.HasPrefix(data, "snipes_") {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(data, "snipes_"), "_")
	if len(parts) != 2 {
		return nil
	}
	page, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}
	ownerID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil
	}
	if c.SenderID != ownerID {
		c.Answer("This button is not for you.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	entries := snipeGetDeleted(c.ChatID)
	body, page, totalPages := renderSnipesPage(entries, page)

	opts := &tg.SendOptions{}
	if kb := snipesKeyboard(page, totalPages, ownerID); kb != nil {
		opts.ReplyMarkup = kb
	}
	c.Edit(body, opts)
	c.Answer("")
	return nil
}

func SnipeToggleHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Snipe works in groups only.</b>")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> Admins only.")
		return nil
	}

	arg := strings.ToLower(strings.TrimSpace(m.Args()))
	cfg := loadSnipeCfg(m.ChatID())

	switch arg {
	case "on", "enable", "yes":
		cfg.Enabled = true
		if err := saveSnipeCfg(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save.</b>")
			return nil
		}
		m.Reply("<b>Snipe enabled.</b>")
	case "off", "disable", "no":
		cfg.Enabled = false
		if err := saveSnipeCfg(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save.</b>")
			return nil
		}
		snipeMu.Lock()
		delete(snipeStore, m.ChatID())
		delete(snipeEditStore, m.ChatID())
		snipeMu.Unlock()
		snipeMsgCacheMu.Lock()
		delete(snipeMsgCache, m.ChatID())
		snipeMsgCacheMu.Unlock()
		m.Reply("<b>Snipe disabled.</b> Buffer cleared.")
	default:
		state := "enabled"
		if !cfg.Enabled {
			state = "disabled"
		}
		m.Reply(fmt.Sprintf("<b>Snipe is %s.</b>\n<b>Usage:</b> <code>/snipe on|off</code>", state))
	}
	return nil
}

func registerSnipeHandlers() {
	c := modules.Client
	c.On("cmd:snipe", SnipeHandler)
	c.On("cmd:esnipe", ESnipeHandler)
	c.On("cmd:snipes", SnipesHandler)
	c.On("callback:snipes_", SnipesCallback)
	c.On(tg.OnNewMessage, SnipeCacheHandler)
	c.On(tg.OnEditMessage, SnipeEditHandler)
	c.On(tg.OnDeleteMessage, SnipeDeleteHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerSnipeHandlers)

	modules.Mods.AddModule("Snipe", `<b>Snipe Module</b>

Recover deleted and edited messages in your chat.

<b>Commands:</b>
 • /snipe [N] - Recover Nth most recent deleted message (default 1)
 • /esnipe - Show the last edited message (old → new)
 • /snipes [page] - Paginated list of last 10 deleted messages
 • /snipe on|off - Enable/disable snipe in this chat (admins only)

<b>Buffer:</b> Last 20 deleted/edited messages per chat (in-memory)
<b>Storage:</b> Per-chat enable flag persisted via bbolt
<b>Permission:</b> Admins with Change Info can toggle.`)
}
