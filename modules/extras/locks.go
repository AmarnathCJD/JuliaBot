package extras

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	modules "main/modules"
	"main/modules/db"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
)

const locksBucket = "locks_v3"

type lockValue struct {
	On  bool    `json:"on"`
	IDs []int64 `json:"ids,omitempty"`
}

var supportedLockTypes = []string{
	"text",
	"photo",
	"video",
	"gif",
	"animation",
	"sticker",
	"voice",
	"audio",
	"file",
	"link",
	"url",
	"forward",
	"mention",
	"hashtag",
	"invite",
	"reply",
	"poll",
	"location",
	"contact",
	"game",
	"all",
	"emoji",
	"command",
	"edit",
	"inline",
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
	"links":        "link",
	"urls":         "url",
	"forwards":     "forward",
	"mentions":     "mention",
	"hashtags":     "hashtag",
	"invites":      "invite",
	"invitelink":   "invite",
	"invitelinks":  "invite",
	"replies":      "reply",
	"polls":        "poll",
	"loc":          "location",
	"locations":    "location",
	"contacts":     "contact",
	"games":        "game",
	"emojis":       "emoji",
	"commands":     "command",
	"cmd":          "command",
	"cmds":         "command",
	"edits":        "edit",
	"editing":      "edit",
	"inlines":      "inline",
	"viabot":       "inline",
	"services":     "service",
	"servicemsg":   "service",
	"previews":     "preview",
	"linkpreview":  "preview",
	"webpage":      "preview",
	"videonote":    "video_note",
	"roundvideo":   "video_note",
	"vn":           "video_note",
	"album":        "mediagroup",
	"albums":       "mediagroup",
	"group":        "mediagroup",
	"anon":         "anonymous",
	"anonusers":    "anonymous",
	"spoilers":     "spoiler",
	"premiumemoji": "premium_emoji",
	"customemoji":  "custom_emoji",
	"stickers":     "sticker",
	"photos":       "photo",
	"videos":       "video",
	"gifs":         "gif",
	"animations":   "animation",
	"voices":       "voice",
	"audios":       "audio",
	"documents":    "file",
	"document":     "file",
	"files":        "file",
	"texts":        "text",
	"messages":     "text",
	"msg":          "text",
}

var (
	locksCache   = make(map[int64]map[string]lockValue)
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
	return slices.Contains(supportedLockTypes, t)
}

func locksKey(chatID int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(chatID))
	return b
}

func loadLocks(chatID int64) map[string]lockValue {
	locksCacheMu.RLock()
	if v, ok := locksCache[chatID]; ok {
		out := make(map[string]lockValue, len(v))
		for k, lv := range v {
			out[k] = lv
		}
		locksCacheMu.RUnlock()
		return out
	}
	locksCacheMu.RUnlock()

	out := make(map[string]lockValue)
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
	cp := make(map[string]lockValue, len(out))
	for k, lv := range out {
		cp[k] = lv
	}
	locksCache[chatID] = cp
	locksCacheMu.Unlock()
	return out
}

func saveLocks(chatID int64, m map[string]lockValue) error {
	d, err := db.GetDB()
	if err != nil {
		return err
	}
	clean := make(map[string]lockValue, len(m))
	for k, v := range m {
		if v.On {
			clean[k] = v
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
	case "file":
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
	case "game":
		return m.Game() != nil
	case "inline":
		return m.Message != nil && m.Message.ViaBotID != 0
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
		if v.On {
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
		sb.WriteString("</code>")
		if ids := locks[t].IDs; len(ids) > 0 {
			sb.WriteString(" = <code>")
			parts := make([]string, len(ids))
			for i, id := range ids {
				parts[i] = strconv.FormatInt(id, 10)
			}
			sb.WriteString(strings.Join(parts, ", "))
			sb.WriteString("</code>")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func resolveInlineTarget(c *tg.Client, s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty target")
	}
	if id, err := strconv.ParseInt(s, 10, 64); err == nil {
		return id, nil
	}
	result, err := c.ResolveUsername(strings.TrimPrefix(s, "@"))
	if err != nil {
		return 0, err
	}
	if u, ok := result.(*tg.UserObj); ok {
		return u.ID, nil
	}
	return 0, fmt.Errorf("%s is not a user/bot", s)
}

func parseLockArg(a string) (key string, targets []string) {
	if idx := strings.Index(a, "="); idx >= 0 {
		key = a[:idx]
		vs := a[idx+1:]
		for _, t := range strings.Split(vs, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				targets = append(targets, t)
			}
		}
		return key, targets
	}
	return a, nil
}

func LockHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Locks only work in groups.")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to manage locks.")
		return nil
	}

	args := m.ArgsList()
	if len(args) == 0 {
		m.Reply("<i>Usage:</i> <code>/lock &lt;type&gt; [type2 ...]</code>\nSee <code>/locktypes</code> for the full list.")
		return nil
	}

	locks := loadLocks(m.ChatID())
	if locks == nil {
		locks = make(map[string]lockValue)
	}

	var locked, unknown, badTargets []string
	for _, a := range args {
		rawKey, targets := parseLockArg(a)
		t := normalizeLockType(rawKey)
		if !isValidLockType(t) {
			unknown = append(unknown, rawKey)
			continue
		}
		lv := locks[t]
		lv.On = true
		if len(targets) > 0 {
			if t != "inline" {
				badTargets = append(badTargets, rawKey)
				continue
			}
			ids := make([]int64, 0, len(targets))
			for _, target := range targets {
				id, err := resolveInlineTarget(m.Client, target)
				if err != nil {
					badTargets = append(badTargets, target)
					continue
				}
				ids = append(ids, id)
			}
			lv.IDs = ids
		}
		locks[t] = lv
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
	if len(badTargets) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("<b>Unresolved targets:</b> <code>")
		sb.WriteString(strings.Join(badTargets, ", "))
		sb.WriteString("</code>")
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
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to manage locks.")
		return nil
	}

	args := m.ArgsList()
	if len(args) == 0 {
		m.Reply("<i>Usage:</i> <code>/unlock &lt;type&gt; [type2 ...]</code> or <code>/unlock all</code>.")
		return nil
	}

	locks := loadLocks(m.ChatID())
	if locks == nil {
		locks = make(map[string]lockValue)
	}

	if len(args) == 1 {
		if k, tgts := parseLockArg(args[0]); normalizeLockType(k) == "all" && len(tgts) == 0 {
			_ = saveLocks(m.ChatID(), map[string]lockValue{})
			m.Reply("<b>All locks cleared.</b>")
			actor := ""
			if m.Sender != nil {
				actor = strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
			}
			LogModerationAction(m.ChatID(), "unlock", actor, "", "all")
			return nil
		}
	}

	var unlocked, unknown, badTargets []string
	for _, a := range args {
		rawKey, targets := parseLockArg(a)
		t := normalizeLockType(rawKey)
		if !isValidLockType(t) {
			unknown = append(unknown, rawKey)
			continue
		}
		if len(targets) > 0 && t == "inline" {
			lv := locks[t]
			if !lv.On || len(lv.IDs) == 0 {
				continue
			}
			removeIDs := make(map[int64]bool, len(targets))
			for _, target := range targets {
				id, err := resolveInlineTarget(m.Client, target)
				if err != nil {
					badTargets = append(badTargets, target)
					continue
				}
				removeIDs[id] = true
			}
			kept := make([]int64, 0, len(lv.IDs))
			for _, id := range lv.IDs {
				if !removeIDs[id] {
					kept = append(kept, id)
				}
			}
			if len(kept) == 0 {
				delete(locks, t)
			} else {
				lv.IDs = kept
				locks[t] = lv
			}
			unlocked = append(unlocked, rawKey)
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
	if len(badTargets) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("<b>Unresolved targets:</b> <code>")
		sb.WriteString(strings.Join(badTargets, ", "))
		sb.WriteString("</code>")
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

	if inl := locks["inline"]; inl.On && m.Message != nil && m.Message.ViaBotID != 0 {
		if len(inl.IDs) == 0 {
			m.Delete()
			return nil
		}
		for _, id := range inl.IDs {
			if id == m.Message.ViaBotID {
				m.Delete()
				return nil
			}
		}
	}

	if modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		return nil
	}

	if locks["all"].On {
		m.Delete()
		return nil
	}

	matched := ""
	for t, lv := range locks {
		if !lv.On || t == "all" || t == "inline" || t == "edit" {
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
	if !locks["edit"].On && !locks["all"].On {
		return nil
	}
	if modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		return nil
	}
	m.Delete()
	return nil
}

func registerLocksHandlers() {
	c := modules.Client
	c.On("cmd:lock", LockHandler)
	c.On("cmd:unlock", UnlockHandler)
	c.On("cmd:locks", LocksStatusHandler)
	c.On("cmd:lockstatus", LocksStatusHandler)
	c.On("cmd:locktypes", LockTypesHandler)
	c.On(tg.OnNewMessage, LocksWatcher)
	c.On(tg.OnEditMessage, LocksEditWatcher)
}

func init() {
	modules.QueueHandlerRegistration(registerLocksHandlers)

	modules.Mods.AddModule("Locks", `<b>Locks</b>

Per-chat content locks. When a type is locked, new messages of that type are deleted (admins exempt, except <code>inline</code>).

<b>Commands:</b>
 - /lock &lt;type&gt; [type2 ...] - Lock one or more types
 - /unlock &lt;type&gt; [type2 ...] - Unlock specific types
 - /unlock all - Clear every lock
 - /locks - Show current locked types
 - /lockstatus - Same as /locks
 - /locktypes - List every supported lock type

<b>Inline (blocks messages sent via inline bots, admins included):</b>
 - /lock inline - Block all inline usage
 - /lock inline=@bot1,@bot2 - Block only these specific bots (accepts @username or numeric ID)
 - /unlock inline=@bot - Remove one bot from the target list
 - /unlock inline - Clear the inline lock entirely

<b>Permission:</b> Admins with Change Info can manage locks.`)
}
