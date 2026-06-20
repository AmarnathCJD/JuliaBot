package modules

import (
	"encoding/binary"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
	"main/modules/db"
)

const locksBucket = "locks_v2"

var supportedLockTypes = []string{
	"text",
	"photo",
	"video",
	"gif",
	"animation",
	"sticker",
	"voice",
	"audio",
	"document",
	"file",
	"link",
	"url",
	"forward",
	"mention",
	"hashtag",
	"bot",
	"invite",
	"reply",
	"poll",
	"location",
	"contact",
	"dice",
	"game",
	"all",
	"button",
	"emoji",
	"command",
	"edit",
	"service",
	"preview",
	"video_note",
	"mediagroup",
	"anonymous",
	"spoiler",
	"premium_emoji",
	"custom_emoji",
}

var lockAliases = map[string]string{
	"links":         "link",
	"urls":          "url",
	"forwards":      "forward",
	"mentions":      "mention",
	"hashtags":      "hashtag",
	"bots":          "bot",
	"invites":       "invite",
	"invitelink":    "invite",
	"invitelinks":   "invite",
	"replies":       "reply",
	"polls":         "poll",
	"loc":           "location",
	"locations":    "location",
	"contacts":      "contact",
	"dices":         "dice",
	"games":         "game",
	"buttons":       "button",
	"emojis":        "emoji",
	"commands":      "command",
	"cmd":           "command",
	"cmds":          "command",
	"edits":         "edit",
	"editing":       "edit",
	"services":      "service",
	"servicemsg":    "service",
	"previews":      "preview",
	"linkpreview":   "preview",
	"webpage":       "preview",
	"videonote":     "video_note",
	"roundvideo":    "video_note",
	"vn":            "video_note",
	"album":         "mediagroup",
	"albums":        "mediagroup",
	"group":         "mediagroup",
	"anon":          "anonymous",
	"anonusers":     "anonymous",
	"spoilers":      "spoiler",
	"premiumemoji":  "premium_emoji",
	"customemoji":   "custom_emoji",
	"stickers":      "sticker",
	"photos":        "photo",
	"videos":        "video",
	"gifs":          "gif",
	"animations":    "animation",
	"voices":        "voice",
	"audios":        "audio",
	"documents":     "document",
	"files":         "file",
	"texts":         "text",
	"messages":      "text",
	"msg":           "text",
}

var (
	locksCache   = make(map[int64]map[string]bool)
	locksCacheMu sync.RWMutex
)

func normalizeLockType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if v, ok := lockAliases[t]; ok {
		return v
	}
	return t
}

func isValidLockType(t string) bool {
	for _, s := range supportedLockTypes {
		if s == t {
			return true
		}
	}
	return false
}

func locksKey(chatID int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(chatID))
	return b
}

func loadLocks(chatID int64) map[string]bool {
	locksCacheMu.RLock()
	if v, ok := locksCache[chatID]; ok {
		out := make(map[string]bool, len(v))
		for k, b := range v {
			out[k] = b
		}
		locksCacheMu.RUnlock()
		return out
	}
	locksCacheMu.RUnlock()

	out := make(map[string]bool)
	d, err := db.GetDB()
	if err != nil {
		return out
	}
	_ = d.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(locksBucket))
		if b == nil {
			return nil
		}
		raw := b.Get(locksKey(chatID))
		if len(raw) == 0 {
			return nil
		}
		_ = json.Unmarshal(raw, &out)
		return nil
	})

	locksCacheMu.Lock()
	cp := make(map[string]bool, len(out))
	for k, b := range out {
		cp[k] = b
	}
	locksCache[chatID] = cp
	locksCacheMu.Unlock()
	return out
}

func saveLocks(chatID int64, m map[string]bool) error {
	d, err := db.GetDB()
	if err != nil {
		return err
	}
	clean := make(map[string]bool, len(m))
	for k, v := range m {
		if v {
			clean[k] = true
		}
	}
	err = d.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(locksBucket))
		if err != nil {
			return err
		}
		if len(clean) == 0 {
			return b.Delete(locksKey(chatID))
		}
		raw, err := json.Marshal(clean)
		if err != nil {
			return err
		}
		return b.Put(locksKey(chatID), raw)
	})
	if err != nil {
		return err
	}
	locksCacheMu.Lock()
	if len(clean) == 0 {
		delete(locksCache, chatID)
	} else {
		locksCache[chatID] = clean
	}
	locksCacheMu.Unlock()
	return nil
}

func anyLockedExceptAll(locks map[string]bool) bool {
	for k, v := range locks {
		if v && k != "all" {
			return true
		}
	}
	return false
}

func messageHasURL(m *tg.NewMessage) bool {
	if m.Message != nil {
		for _, e := range m.Message.Entities {
			switch e.(type) {
			case *tg.MessageEntityURL, *tg.MessageEntityTextURL:
				return true
			}
		}
	}
	t := strings.ToLower(m.Text())
	if t == "" {
		return false
	}
	return strings.Contains(t, "http://") || strings.Contains(t, "https://") || strings.Contains(t, "t.me/") || strings.Contains(t, "telegram.me/") || strings.Contains(t, "www.")
}

func messageHasInviteLink(m *tg.NewMessage) bool {
	t := strings.ToLower(m.Text())
	if strings.Contains(t, "t.me/joinchat/") || strings.Contains(t, "t.me/+") || strings.Contains(t, "telegram.me/joinchat/") || strings.Contains(t, "telegram.me/+") {
		return true
	}
	if m.Message != nil {
		for _, e := range m.Message.Entities {
			if u, ok := e.(*tg.MessageEntityTextURL); ok {
				lu := strings.ToLower(u.URL)
				if strings.Contains(lu, "t.me/joinchat/") || strings.Contains(lu, "t.me/+") || strings.Contains(lu, "telegram.me/joinchat/") || strings.Contains(lu, "telegram.me/+") {
					return true
				}
			}
		}
	}
	return false
}

func messageHasMention(m *tg.NewMessage) bool {
	if m.Message == nil {
		return false
	}
	for _, e := range m.Message.Entities {
		switch e.(type) {
		case *tg.MessageEntityMention, *tg.MessageEntityMentionName:
			return true
		}
	}
	return false
}

func messageHasHashtag(m *tg.NewMessage) bool {
	if m.Message == nil {
		return false
	}
	for _, e := range m.Message.Entities {
		if _, ok := e.(*tg.MessageEntityHashtag); ok {
			return true
		}
	}
	return false
}

func messageHasCustomEmoji(m *tg.NewMessage) bool {
	if m.Message == nil {
		return false
	}
	for _, e := range m.Message.Entities {
		if _, ok := e.(*tg.MessageEntityCustomEmoji); ok {
			return true
		}
	}
	return false
}

func messageHasSpoiler(m *tg.NewMessage) bool {
	if m.Message != nil {
		for _, e := range m.Message.Entities {
			if _, ok := e.(*tg.MessageEntitySpoiler); ok {
				return true
			}
		}
	}
	if doc, ok := m.Media().(*tg.MessageMediaDocument); ok && doc != nil {
		if doc.Spoiler {
			return true
		}
	}
	if ph, ok := m.Media().(*tg.MessageMediaPhoto); ok && ph != nil {
		if ph.Spoiler {
			return true
		}
	}
	return false
}

func messageHasButtons(m *tg.NewMessage) bool {
	if m.Message == nil {
		return false
	}
	switch m.Message.ReplyMarkup.(type) {
	case *tg.ReplyInlineMarkup:
		return true
	}
	return false
}

func messageIsVideoNote(m *tg.NewMessage) bool {
	doc := m.Document()
	if doc == nil {
		return false
	}
	for _, attr := range doc.Attributes {
		if v, ok := attr.(*tg.DocumentAttributeVideo); ok {
			if v.RoundMessage {
				return true
			}
		}
	}
	return false
}

func messageIsMediaGroup(m *tg.NewMessage) bool {
	return m.Message != nil && m.Message.GroupedID != 0
}

func messageHasPreview(m *tg.NewMessage) bool {
	if _, ok := m.Media().(*tg.MessageMediaWebPage); ok {
		return true
	}
	return false
}

func senderIsBot(m *tg.NewMessage) bool {
	if m.Sender != nil && m.Sender.Bot {
		return true
	}
	if m.Message != nil {
		if m.Message.ViaBotID != 0 {
			return true
		}
	}
	return false
}

func messageIsCommand(m *tg.NewMessage) bool {
	if m.IsCommand() {
		return true
	}
	t := strings.TrimSpace(m.Text())
	if len(t) > 1 && (t[0] == '/' || t[0] == '!' || t[0] == '.' || t[0] == '-' || t[0] == '?') {
		return true
	}
	return false
}

func messageIsText(m *tg.NewMessage) bool {
	if m.Media() != nil {
		return false
	}
	if m.IsService() {
		return false
	}
	return strings.TrimSpace(m.Text()) != ""
}

func messageMatchesLock(m *tg.NewMessage, lockType string) bool {
	switch lockType {
	case "text":
		return messageIsText(m)
	case "photo":
		return m.Photo() != nil
	case "video":
		return m.Video() != nil
	case "gif", "animation":
		return m.Animation() != nil
	case "sticker":
		return m.Sticker() != nil
	case "voice":
		if doc, ok := m.Media().(*tg.MessageMediaDocument); ok && doc != nil {
			if doc.Voice {
				return true
			}
		}
		if doc := m.Document(); doc != nil {
			for _, attr := range doc.Attributes {
				if a, ok := attr.(*tg.DocumentAttributeAudio); ok && a.Voice {
					return true
				}
			}
		}
		return false
	case "audio":
		if doc := m.Document(); doc != nil {
			for _, attr := range doc.Attributes {
				if a, ok := attr.(*tg.DocumentAttributeAudio); ok && !a.Voice {
					return true
				}
			}
		}
		return false
	case "document", "file":
		doc := m.Document()
		if doc == nil {
			return false
		}
		for _, attr := range doc.Attributes {
			switch attr.(type) {
			case *tg.DocumentAttributeSticker, *tg.DocumentAttributeAnimated:
				return false
			}
			if v, ok := attr.(*tg.DocumentAttributeVideo); ok && v.RoundMessage {
				return false
			}
		}
		return true
	case "link", "url":
		return messageHasURL(m)
	case "forward":
		return m.IsForward()
	case "mention":
		return messageHasMention(m)
	case "hashtag":
		return messageHasHashtag(m)
	case "bot":
		return senderIsBot(m)
	case "invite":
		return messageHasInviteLink(m)
	case "reply":
		return m.IsReply()
	case "poll":
		return m.Poll() != nil
	case "location":
		return m.Geo() != nil || m.Venue() != nil
	case "contact":
		return m.Contact() != nil
	case "dice":
		if _, ok := m.Media().(*tg.MessageMediaDice); ok {
			return true
		}
		return false
	case "game":
		return m.Game() != nil
	case "button":
		return messageHasButtons(m)
	case "emoji", "custom_emoji", "premium_emoji":
		return messageHasCustomEmoji(m)
	case "command":
		return messageIsCommand(m)
	case "service":
		return m.IsService()
	case "preview":
		return messageHasPreview(m)
	case "video_note":
		return messageIsVideoNote(m)
	case "mediagroup":
		return messageIsMediaGroup(m)
	case "anonymous":
		return m.IsAnonymous()
	case "spoiler":
		return messageHasSpoiler(m)
	case "all":
		return true
	}
	return false
}

func formatLockTypesList() string {
	cp := make([]string, len(supportedLockTypes))
	copy(cp, supportedLockTypes)
	sort.Strings(cp)
	var sb strings.Builder
	sb.WriteString("<b>Supported Lock Types</b>\n\n")
	for _, t := range cp {
		sb.WriteString(" - <code>")
		sb.WriteString(t)
		sb.WriteString("</code>\n")
	}
	sb.WriteString("\n<i>Usage:</i> <code>/lock photo video sticker</code>")
	return sb.String()
}

func formatCurrentLocks(chatID int64) string {
	locks := loadLocks(chatID)
	active := make([]string, 0, len(locks))
	for k, v := range locks {
		if v {
			active = append(active, k)
		}
	}
	if len(active) == 0 {
		return "<b>Current Locks</b>\n\n<i>No types are locked in this chat.</i>"
	}
	sort.Strings(active)
	var sb strings.Builder
	sb.WriteString("<b>Current Locks</b>\n\n")
	for _, t := range active {
		sb.WriteString(" - <code>")
		sb.WriteString(t)
		sb.WriteString("</code>\n")
	}
	return sb.String()
}

func LockHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Locks only work in groups.")
		return nil
	}
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		m.Reply("You need Delete Messages permission to manage locks.")
		return nil
	}

	args := m.ArgsList()
	if len(args) == 0 {
		m.Reply("<i>Usage:</i> <code>/lock &lt;type&gt; [type2 ...]</code>\nSee <code>/locktypes</code> for the full list.")
		return nil
	}

	locks := loadLocks(m.ChatID())
	if locks == nil {
		locks = make(map[string]bool)
	}

	var locked, unknown []string
	for _, a := range args {
		t := normalizeLockType(a)
		if !isValidLockType(t) {
			unknown = append(unknown, a)
			continue
		}
		locks[t] = true
		locked = append(locked, t)
	}

	if err := saveLocks(m.ChatID(), locks); err != nil {
		m.Reply("Failed to save locks.")
		return nil
	}

	var sb strings.Builder
	if len(locked) > 0 {
		sb.WriteString("<b>Locked:</b> <code>")
		sb.WriteString(strings.Join(locked, ", "))
		sb.WriteString("</code>")
	}
	if len(unknown) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("<b>Unknown:</b> <code>")
		sb.WriteString(strings.Join(unknown, ", "))
		sb.WriteString("</code>\nSee <code>/locktypes</code>.")
	}
	if sb.Len() == 0 {
		sb.WriteString("Nothing changed.")
	}
	m.Reply(sb.String())

	if len(locked) > 0 {
		actor := ""
		if m.Sender != nil {
			actor = strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
		}
		LogModerationAction(m.ChatID(), "lock", actor, "", strings.Join(locked, ", "))
	}
	return nil
}

func UnlockHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Locks only work in groups.")
		return nil
	}
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		m.Reply("You need Delete Messages permission to manage locks.")
		return nil
	}

	args := m.ArgsList()
	if len(args) == 0 {
		m.Reply("<i>Usage:</i> <code>/unlock &lt;type&gt; [type2 ...]</code> or <code>/unlock all</code>.")
		return nil
	}

	locks := loadLocks(m.ChatID())
	if locks == nil {
		locks = make(map[string]bool)
	}

	if len(args) == 1 && normalizeLockType(args[0]) == "all" {
		_ = saveLocks(m.ChatID(), map[string]bool{})
		m.Reply("<b>All locks cleared.</b>")
		actor := ""
		if m.Sender != nil {
			actor = strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
		}
		LogModerationAction(m.ChatID(), "unlock", actor, "", "all")
		return nil
	}

	var unlocked, unknown []string
	for _, a := range args {
		t := normalizeLockType(a)
		if !isValidLockType(t) {
			unknown = append(unknown, a)
			continue
		}
		delete(locks, t)
		unlocked = append(unlocked, t)
	}

	if err := saveLocks(m.ChatID(), locks); err != nil {
		m.Reply("Failed to save locks.")
		return nil
	}

	var sb strings.Builder
	if len(unlocked) > 0 {
		sb.WriteString("<b>Unlocked:</b> <code>")
		sb.WriteString(strings.Join(unlocked, ", "))
		sb.WriteString("</code>")
	}
	if len(unknown) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("<b>Unknown:</b> <code>")
		sb.WriteString(strings.Join(unknown, ", "))
		sb.WriteString("</code>\nSee <code>/locktypes</code>.")
	}
	if sb.Len() == 0 {
		sb.WriteString("Nothing changed.")
	}
	m.Reply(sb.String())

	if len(unlocked) > 0 {
		actor := ""
		if m.Sender != nil {
			actor = strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
		}
		LogModerationAction(m.ChatID(), "unlock", actor, "", strings.Join(unlocked, ", "))
	}
	return nil
}

func LocksStatusHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Locks only work in groups.")
		return nil
	}
	m.Reply(formatCurrentLocks(m.ChatID()))
	return nil
}

func LockTypesHandler(m *tg.NewMessage) error {
	m.Reply(formatLockTypesList())
	return nil
}

func LocksWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	locks := loadLocks(m.ChatID())
	if len(locks) == 0 {
		return nil
	}
	if IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		return nil
	}

	if locks["all"] {
		m.Delete()
		return nil
	}

	matched := ""
	for t, on := range locks {
		if !on || t == "all" || t == "edit" {
			continue
		}
		if messageMatchesLock(m, t) {
			matched = t
			break
		}
	}

	if matched != "" {
		m.Delete()
	}
	return nil
}

func LocksEditWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	locks := loadLocks(m.ChatID())
	if !locks["edit"] && !locks["all"] {
		return nil
	}
	if IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		return nil
	}
	m.Delete()
	return nil
}

func registerLocksHandlers() {
	c := Client
	c.On("cmd:lock", LockHandler)
	c.On("cmd:unlock", UnlockHandler)
	c.On("cmd:locks", LocksStatusHandler)
	c.On("cmd:lockstatus", LocksStatusHandler)
	c.On("cmd:locktypes", LockTypesHandler)
	c.On(tg.OnNewMessage, LocksWatcher)
	c.On(tg.OnEditMessage, LocksEditWatcher)
}

func init() {
	QueueHandlerRegistration(registerLocksHandlers)

	Mods.AddModule("Locks", `<b>Locks</b>

Per-chat content locks. When a type is locked, new messages of that type are deleted (admins exempt).

<b>Commands:</b>
 - /lock &lt;type&gt; [type2 ...] - Lock one or more types
 - /unlock &lt;type&gt; [type2 ...] - Unlock specific types
 - /unlock all - Clear every lock
 - /locks - Show current locked types
 - /lockstatus - Same as /locks
 - /locktypes - List every supported lock type

<b>Permission:</b> Admins with Delete Messages can manage locks.`)
}
