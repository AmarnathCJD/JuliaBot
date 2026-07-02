package extras

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
	"html"
	modules "main/modules"
	"main/modules/db"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// === from notes.go ===
// Variable replacements supported in notes
var variableReplacements = map[string]func(*tg.NewMessage) string{
	"{mention}": func(m *tg.NewMessage) string {
		if m.Sender == nil {
			return ""
		}
		return fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", m.SenderID(), m.Sender.FirstName)
	},
	"{firstname}": func(m *tg.NewMessage) string {
		if m.Sender == nil {
			return ""
		}
		return m.Sender.FirstName
	},
	"{lastname}": func(m *tg.NewMessage) string {
		if m.Sender == nil {
			return ""
		}
		return m.Sender.LastName
	},
	"{fullname}": func(m *tg.NewMessage) string {
		if m.Sender == nil {
			return ""
		}
		return m.Sender.FirstName + " " + m.Sender.LastName
	},
	"{username}": func(m *tg.NewMessage) string {
		if m.Sender == nil {
			return ""
		}
		if m.Sender.Username != "" {
			return "@" + m.Sender.Username
		}
		return m.Sender.FirstName
	},
	"{chatname}": func(m *tg.NewMessage) string { return m.Chat.Title },
	"{userid}":   func(m *tg.NewMessage) string { return fmt.Sprint(m.SenderID()) },
	"{chatid}":   func(m *tg.NewMessage) string { return fmt.Sprint(m.ChatID()) },
}

// Button represents a single button with text and URL
type Button struct {
	Text string
	URL  string
}

// ParseButtonsFromText extracts buttons in format: [Text1](url1) | [Text2](url2)
func ParseButtonsFromText(text string) ([]Button, string) {
	var buttons []Button
	cleanText := text

	// Match pattern: [text](url)
	buttonPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	matches := buttonPattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			buttons = append(buttons, Button{Text: match[1], URL: match[2]})
		}
	}

	// Remove button syntax from text
	cleanText = buttonPattern.ReplaceAllString(cleanText, "")
	cleanText = strings.ReplaceAll(cleanText, "| |", "|")
	cleanText = strings.Trim(strings.TrimSpace(cleanText), "|")

	return buttons, cleanText
}

// BuildButtonKeyboard creates an inline keyboard from buttons
func BuildButtonKeyboard(buttons []Button) *tg.ReplyInlineMarkup {
	if len(buttons) == 0 {
		return nil
	}

	b := tg.Button
	kb := tg.NewKeyboard()

	// Add buttons to keyboard (max 2 per row for readability)
	for i := 0; i < len(buttons); i += 2 {
		if i+1 < len(buttons) {
			kb.AddRow(b.URL(buttons[i].Text, buttons[i].URL), b.URL(buttons[i+1].Text, buttons[i+1].URL))
		} else {
			kb.AddRow(b.URL(buttons[i].Text, buttons[i].URL))
		}
	}

	return kb.Build()
}

func SaveNoteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Notes work in groups only.</b>")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> You need Change Info permission.")
		return nil
	}

	args := m.Args()
	if args == "" && !m.IsReply() {
		m.Reply("<b>Usage:</b> <code>/save notename content</code> or reply to a message with <code>/save notename</code>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	noteName := strings.ToLower(strings.TrimSpace(parts[0]))

	if noteName == "" {
		m.Reply("<b>Error:</b> Note name required.")
		return nil
	}

	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(noteName) {
		m.Reply("<b>Invalid name.</b> Use only lowercase letters, numbers, and underscores.")
		return nil
	}

	note := &db.Note{
		Name:      noteName,
		CreatedBy: m.SenderID(),
	}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error getting reply message")
			return nil
		}

		if reply.IsMedia() {
			if reply.Photo() != nil {
				note.MediaType = "photo"
				note.FileID = reply.File.FileID
			} else if reply.Document() != nil {
				note.MediaType = "document"
				note.FileID = reply.File.FileID
			} else if reply.Video() != nil {
				note.MediaType = "video"
				note.FileID = reply.File.FileID
			} else if reply.Audio() != nil {
				note.MediaType = "audio"
				note.FileID = reply.File.FileID
			} else if reply.Sticker() != nil {
				note.MediaType = "sticker"
				note.FileID = reply.File.FileID
			} else if reply.Animation() != nil {
				note.MediaType = "animation"
				note.FileID = reply.File.FileID
			}
		}

		note.Content = reply.RawText()
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			contentArg := strings.TrimSpace(parts[1])
			testContent := contentArg
			testContent = strings.ReplaceAll(testContent, "{admin}", "")
			testContent = strings.ReplaceAll(testContent, "{private}", "")
			testContent = strings.TrimSpace(testContent)
			if testContent != "" {
				note.Content = contentArg
			}
		}
	} else {
		if len(parts) < 2 {
			m.Reply("<b>Error:</b> Provide note content or reply to a message.")
			return nil
		}
		note.Content = parts[1]
	}

	// Handle special tags
	privateMode := false
	if strings.Contains(note.Content, "{private}") {
		privateMode = true
		note.Content = strings.ReplaceAll(note.Content, "{private}", "")
		note.Content = strings.TrimSpace(note.Content)
	}

	if strings.Contains(note.Content, "{admin}") {
		note.AdminOnly = true
		note.Content = strings.ReplaceAll(note.Content, "{admin}", "")
		note.Content = strings.TrimSpace(note.Content)
	}

	if note.Content == "" && note.FileID == "" {
		m.Reply("<b>Error:</b> Note must have text or media.")
		return nil
	}

	// Store private mode flag in a custom field if needed (extend Note struct)
	if err := db.SaveNote(m.ChatID(), note); err != nil {
		m.Reply("<b>Failed to save note.</b> Please try again.")
		return nil
	}

	// Build success message
	var tags []string
	if note.AdminOnly {
		tags = append(tags, "Admin Only")
	}
	if privateMode {
		tags = append(tags, "PM Mode")
	}
	if note.FileID != "" {
		tags = append(tags, "Has Media")
	}

	tagsStr := ""
	if len(tags) > 0 {
		tagsStr = " [" + strings.Join(tags, ", ") + "]"
	}

	m.Reply(fmt.Sprintf("<b>Note saved:</b> <code>#%s</code>%s", noteName, tagsStr))
	return nil
}

// GetNoteHandler handles /note and #notename
func GetNoteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	noteName := strings.ToLower(strings.TrimSpace(m.Args()))
	if noteName == "" {
		m.Reply("<b>Usage:</b> <code>/note notename</code> or <code>#notename</code>")
		return nil
	}

	return sendNote(m, noteName)
}

// NoteHashHandler handles #notename triggers
func NoteHashHandler(m *tg.NewMessage) error {
	fmt.Println("Hash note triggered")
	if m.IsPrivate() {
		return nil
	}

	text := m.Text()
	if !strings.HasPrefix(text, "#") {
		return nil
	}

	// Extract note name from #notename
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil
	}

	noteName := strings.ToLower(strings.TrimPrefix(parts[0], "#"))
	if noteName == "" {
		return nil
	}

	return sendNote(m, noteName)
}

func sendNote(m *tg.NewMessage, noteName string) error {
	note, err := db.GetNote(m.ChatID(), noteName)
	if err != nil || note == nil {
		return nil // Silent fail for hash triggers
	}

	// Check admin-only restriction
	if note.AdminOnly && !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		m.Reply("<b>Admin-only note.</b> Only admins can view this.")
		return nil
	}

	// Replace variables in content
	content := note.Content
	for variable, replacer := range variableReplacements {
		if strings.Contains(content, variable) {
			content = strings.ReplaceAll(content, variable, replacer(m))
		}
	}

	// Parse buttons from content if present
	buttons, cleanContent := ParseButtonsFromText(content)

	if note.FileID != "" {
		opts := &tg.MediaOptions{Caption: cleanContent}
		if len(buttons) > 0 {
			opts.ReplyMarkup = BuildButtonKeyboard(buttons)
		}
		media, err := tg.ResolveBotFileID(note.FileID)
		if err != nil {
			m.Reply("<b>Media not found.</b> The media may have been deleted.")
			return nil
		}
		m.ReplyMedia(media, opts)
		return nil
	}

	if cleanContent != "" {
		if len(buttons) > 0 {
			m.Reply(cleanContent, &tg.SendOptions{ReplyMarkup: BuildButtonKeyboard(buttons)})
		} else {
			m.Reply(cleanContent)
		}
	}

	return nil
}

func ListNotesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Notes work in groups only.</b>")
		return nil
	}

	notes, err := db.GetAllNotes(m.ChatID())
	if err != nil || len(notes) == 0 {
		m.Reply("<b>No notes saved yet.</b> Use <code>/save notename content</code> to create one.")
		return nil
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Name < notes[j].Name
	})

	var resp strings.Builder
	resp.WriteString("<b>Saved Notes</b>\n")
	resp.WriteString("━━━━━━━━━━━━━━━━\n\n")

	adminCount := 0
	mediaCount := 0
	tempCount := 0

	for _, note := range notes {
		var badges []string
		if note.AdminOnly {
			badges = append(badges, "[A]")
			adminCount++
		}
		if note.FileID != "" {
			badges = append(badges, "[M]")
			mediaCount++
		}
		if !note.ExpiresAt.IsZero() {
			badges = append(badges, "[T]")
			tempCount++
		}

		badgeStr := ""
		if len(badges) > 0 {
			badgeStr = " " + strings.Join(badges, " ")
		}

		resp.WriteString(fmt.Sprintf(" • <code>#%s</code>%s\n", note.Name, badgeStr))
	}

	resp.WriteString(fmt.Sprintf("\n━━━━━━━━━━━━━━━━\n<b>Total:</b> %d notes", len(notes)))

	var stats []string
	if adminCount > 0 {
		stats = append(stats, fmt.Sprintf("%d admin-only", adminCount))
	}
	if mediaCount > 0 {
		stats = append(stats, fmt.Sprintf("%d with media", mediaCount))
	}
	if tempCount > 0 {
		stats = append(stats, fmt.Sprintf("%d temporary", tempCount))
	}

	if len(stats) > 0 {
		resp.WriteString("\n<i>")
		resp.WriteString(strings.Join(stats, " • "))
		resp.WriteString("</i>")
	}

	resp.WriteString("\n\n<i>Type #notename to get a note</i>")

	m.Reply(resp.String())
	return nil
}

func ClearNoteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Notes work in groups only.</b>")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> You need Change Info permission.")
		return nil
	}

	noteName := strings.ToLower(strings.TrimSpace(m.Args()))
	if noteName == "" {
		m.Reply("<b>Usage:</b> <code>/clear notename</code>")
		return nil
	}

	note, _ := db.GetNote(m.ChatID(), noteName)
	if note == nil {
		m.Reply(fmt.Sprintf("<b>Note not found:</b> <code>#%s</code>", noteName))
		return nil
	}

	if err := db.DeleteNote(m.ChatID(), noteName); err != nil {
		m.Reply("<b>Failed to delete note.</b> Please try again.")
		return nil
	}

	m.Reply(fmt.Sprintf("<b>Note deleted:</b> <code>#%s</code>", noteName))
	return nil
}

func ClearAllNotesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Notes work in groups only.</b>")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> You need Change Info permission.")
		return nil
	}

	count, _ := db.GetNotesCount(m.ChatID())
	if count == 0 {
		m.Reply("<b>No notes to delete.</b>")
		return nil
	}

	b := tg.Button
	m.Reply(
		fmt.Sprintf("<b>Delete all %d notes?</b>\n\n"+
			"<i>This action cannot be undone.</i>", count),
		&tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				b.Data("Yes, Delete All", fmt.Sprintf("clearallnotes_%d", m.SenderID())),
				b.Data("Cancel", fmt.Sprintf("cancelnotes_%d", m.SenderID())),
			).Build(),
		},
	)

	return nil
}

func ClearAllNotesCallback(c *tg.CallbackQuery) error {
	data := c.DataString()

	if after, ok := strings.CutPrefix(data, "cancelnotes_"); ok {
		userID := after
		if fmt.Sprint(c.SenderID) != userID {
			c.Answer("This button is not for you.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		c.Edit("<b>Cancelled.</b> No notes were deleted.")
		return nil
	}

	if strings.HasPrefix(data, "clearallnotes_") {
		userID := strings.TrimPrefix(data, "clearallnotes_")
		if fmt.Sprint(c.SenderID) != userID {
			c.Answer("This button is not for you.", &tg.CallbackOptions{Alert: true})
			return nil
		}

		chatID := c.ChatID
		count, _ := db.GetNotesCount(chatID)

		if err := db.DeleteAllNotes(chatID); err != nil {
			c.Edit("<b>Failed to delete notes.</b> Please try again.")
			return nil
		}

		c.Edit(fmt.Sprintf("<b>All notes deleted.</b> Removed %d notes.", count))
	}

	return nil
}

func SaveTempNoteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Notes work in groups only.</b>")
		return nil
	}

	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> You need Change Info permission.")
		return nil
	}

	args := m.Args()
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/tempnote &lt;duration&gt; &lt;name&gt; [content]</code>\n" +
			"<b>Example:</b> <code>/tempnote 1h sale Sale ends soon!</code>\n" +
			"<b>Formats:</b> 30s, 10m, 1h, 24h")
		return nil
	}

	parts := strings.SplitN(args, " ", 3)
	if len(parts) < 2 {
		m.Reply("<b>Invalid format.</b> Use: <code>/tempnote &lt;duration&gt; &lt;name&gt; [content]</code>")
		return nil
	}

	durationStr := parts[0]
	noteName := strings.ToLower(strings.TrimSpace(parts[1]))
	content := ""
	if len(parts) > 2 {
		content = parts[2]
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		m.Reply("<b>Invalid duration.</b> Use: <code>30s</code>, <code>10m</code>, <code>1h</code>, <code>24h</code>")
		return nil
	}

	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(noteName) {
		m.Reply("<b>Invalid name.</b> Use only lowercase letters, numbers, and underscores.")
		return nil
	}

	note := &db.Note{
		Name:      noteName,
		CreatedBy: m.SenderID(),
		ExpiresAt: time.Now().Add(duration),
	}

	if m.IsReply() {
		reply, _ := m.GetReplyMessage()
		if reply.IsMedia() {
			if reply.Photo() != nil {
				note.MediaType = "photo"
				note.FileID = reply.File.FileID
			} else if reply.Document() != nil {
				note.MediaType = "document"
				note.FileID = reply.File.FileID
			} else if reply.Video() != nil {
				note.MediaType = "video"
				note.FileID = reply.File.FileID
			} else if reply.Audio() != nil {
				note.MediaType = "audio"
				note.FileID = reply.File.FileID
			} else if reply.Sticker() != nil {
				note.MediaType = "sticker"
				note.FileID = reply.File.FileID
			} else if reply.Animation() != nil {
				note.MediaType = "animation"
				note.FileID = reply.File.FileID
			}
		}
		if content == "" {
			note.Content = reply.RawText()
		} else {
			note.Content = content
		}
	} else {
		if content == "" {
			m.Reply("<b>Missing content.</b> Provide text or reply to a message.")
			return nil
		}
		note.Content = content
	}

	if err := db.SaveNote(m.ChatID(), note); err != nil {
		m.Reply("<b>Failed to save note.</b> Please try again.")
		return nil
	}

	expiryTime := time.Now().Add(duration).Format("3:04 PM")
	m.Reply(fmt.Sprintf("<b>Temporary note created:</b> <code>#%s</code>\n"+
		"<b>Expires in:</b> %s (at %s)",
		noteName, formatDuration(duration), expiryTime))
	return nil
}


func registerNoteHandlers() {
	c := modules.Client
	c.On("cmd:save", SaveNoteHandler)
	c.On("cmd:note", GetNoteHandler)
	c.On("cmd:notes", ListNotesHandler)
	c.On("cmd:listnotes", ListNotesHandler)
	c.On("cmd:clear", ClearNoteHandler)
	c.On("cmd:clearallnotes", ClearAllNotesHandler)
	c.On("callback:clearallnotes_", ClearAllNotesCallback)
	c.On("callback:cancelnotes_", ClearAllNotesCallback)
	c.On("message:^#", NoteHashHandler)
}

func initFromSrc_notes_0_1() {
	modules.QueueHandlerRegistration(registerNoteHandlers)

	modules.Mods.AddModule("Notes", `<b>Notes Module</b>

<b>Commands:</b>
 • /save <name> [content] - Save a note
 • /note <name> or #notename - Get a note
 • /notes - List all notes
 • /clear <name> - Delete a note
 • /clearallnotes - Delete all notes
 • /tempnote <duration> <name> [content] - Temporary note
 • /noteinfo <name> - View note details
 • /searchnotes <keyword> - Search notes
 • /rename <old> <new> - Rename a note

<b>Special Tags:</b>
 • {admin} - Admin-only note
 • {mention}, {firstname}, {lastname}, {username}, {fullname} - User variables
 • {chatname}, {userid}, {chatid} - Chat variables

<b>Add Buttons:</b>
Use format: <code>[Button Text](https://example.com)</code>
Example: <code>/save welcome Hello! [Visit](url) | [Help](url)</code>

<b>Permission:</b> Admins with Change Info permission can manage notes.`)
}
// === from reminders.go ===
type reminderEntry struct {
	ID         uint64 `json:"id"`
	UserID     int64  `json:"user_id"`
	ChatID     int64  `json:"chat_id"`
	Msg        string `json:"msg"`
	FireAtUnix int64  `json:"fire_at_unix"`
}

const remindersBucket = "reminders"

var reminderTimers sync.Map

func parseReminderDuration(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	var total time.Duration
	var numBuf strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			numBuf.WriteRune(r)
		case r == 'd':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("missing number before 'd'")
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			total += time.Duration(n) * 24 * time.Hour
		case r == 'h':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("missing number before 'h'")
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			total += time.Duration(n) * time.Hour
		case r == 'm':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("missing number before 'm'")
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			total += time.Duration(n) * time.Minute
		case r == 's':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("missing number before 's'")
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			total += time.Duration(n) * time.Second
		default:
			return 0, fmt.Errorf("invalid character '%c'", r)
		}
	}
	if numBuf.Len() > 0 {
		n, _ := strconv.Atoi(numBuf.String())
		total += time.Duration(n) * time.Second
	}
	if total <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}
	return total, nil
}

func formatReminderDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	var parts []string
	if days := d / (24 * time.Hour); days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
		d -= days * 24 * time.Hour
	}
	if hours := d / time.Hour; hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
		d -= hours * time.Hour
	}
	if mins := d / time.Minute; mins > 0 {
		parts = append(parts, fmt.Sprintf("%dm", mins))
		d -= mins * time.Minute
	}
	if secs := d / time.Second; secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}
	return strings.Join(parts, " ")
}

func reminderKey(id uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, id)
	return b
}

func saveReminder(entry *reminderEntry) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db error")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(remindersBucket))
		if err != nil {
			return err
		}
		if entry.ID == 0 {
			id, _ := bkt.NextSequence()
			entry.ID = id
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return bkt.Put(reminderKey(entry.ID), data)
	})
}

func deleteReminder(id uint64) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db error")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(remindersBucket))
		if bkt == nil {
			return nil
		}
		return bkt.Delete(reminderKey(id))
	})
}

func listReminders(userID int64) ([]*reminderEntry, error) {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil, fmt.Errorf("db error")
	}
	var out []*reminderEntry
	err = database.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(remindersBucket))
		if bkt == nil {
			return nil
		}
		return bkt.ForEach(func(k, v []byte) error {
			var entry reminderEntry
			if err := json.Unmarshal(v, &entry); err != nil {
				return nil
			}
			if userID == 0 || entry.UserID == userID {
				out = append(out, &entry)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].FireAtUnix < out[j].FireAtUnix
	})
	return out, nil
}

func loadAllReminders() ([]*reminderEntry, error) {
	return listReminders(0)
}

func fireReminder(entry *reminderEntry) {
	reminderTimers.Delete(entry.ID)
	_ = deleteReminder(entry.ID)
	if modules.Client == nil {
		return
	}
	mention := fmt.Sprintf("<a href=\"tg://user?id=%d\">user</a>", entry.UserID)
	text := fmt.Sprintf("%s reminder: %s", mention, html.EscapeString(entry.Msg))
	modules.Client.SendMessage(entry.ChatID, text, &tg.SendOptions{})
}

func scheduleReminder(entry *reminderEntry) {
	now := time.Now().Unix()
	delay := time.Duration(entry.FireAtUnix-now) * time.Second
	if delay <= 0 {
		go fireReminder(entry)
		return
	}
	t := time.AfterFunc(delay, func() {
		fireReminder(entry)
	})
	reminderTimers.Store(entry.ID, t)
}

func RemindHandler(m *tg.NewMessage) error {
	args := m.Args()
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/remind &lt;duration&gt; &lt;message&gt;</code>\n<b>Example:</b> <code>/remind 10m take a break</code>\n<i>Duration units: s, m, h, d</i>")
		return nil
	}
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		m.Reply("<b>Usage:</b> <code>/remind &lt;duration&gt; &lt;message&gt;</code>")
		return nil
	}
	duration, err := parseReminderDuration(parts[0])
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	msg := strings.TrimSpace(parts[1])
	if msg == "" {
		m.Reply("<b>Error:</b> reminder message cannot be empty")
		return nil
	}
	entry := &reminderEntry{
		UserID:     m.SenderID(),
		ChatID:     m.ChatID(),
		Msg:        msg,
		FireAtUnix: time.Now().Add(duration).Unix(),
	}
	if err := saveReminder(entry); err != nil {
		m.Reply("<b>Error saving reminder:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	scheduleReminder(entry)
	m.Reply(fmt.Sprintf("Reminder #<b>%d</b> set for <b>%s</b>", entry.ID, formatReminderDuration(duration)))
	return nil
}

func RemindersHandler(m *tg.NewMessage) error {
	entries, err := listReminders(m.SenderID())
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	if len(entries) == 0 {
		m.Reply("You have no pending reminders.")
		return nil
	}
	var sb strings.Builder
	sb.WriteString("<b>Your pending reminders:</b>\n\n")
	now := time.Now().Unix()
	for _, e := range entries {
		remaining := time.Duration(e.FireAtUnix-now) * time.Second
		sb.WriteString(fmt.Sprintf("• <b>#%d</b> in <b>%s</b> — %s\n", e.ID, formatReminderDuration(remaining), html.EscapeString(e.Msg)))
	}
	sb.WriteString("\n<i>Use</i> <code>/delremind &lt;id&gt;</code> <i>to cancel.</i>")
	m.Reply(sb.String())
	return nil
}

func DelRemindHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/delremind &lt;id&gt;</code>")
		return nil
	}
	id, err := strconv.ParseUint(args, 10, 64)
	if err != nil {
		m.Reply("<b>Error:</b> invalid id")
		return nil
	}
	entries, err := listReminders(m.SenderID())
	if err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	var owned *reminderEntry
	for _, e := range entries {
		if e.ID == id {
			owned = e
			break
		}
	}
	if owned == nil {
		m.Reply("<b>Error:</b> reminder not found or not yours")
		return nil
	}
	if v, ok := reminderTimers.LoadAndDelete(id); ok {
		if t, ok := v.(*time.Timer); ok {
			t.Stop()
		}
	}
	if err := deleteReminder(id); err != nil {
		m.Reply("<b>Error:</b> " + html.EscapeString(err.Error()))
		return nil
	}
	m.Reply(fmt.Sprintf("Reminder #<b>%d</b> cancelled.", id))
	return nil
}

func bootstrapReminders() {
	entries, err := loadAllReminders()
	if err != nil {
		return
	}
	for _, e := range entries {
		scheduleReminder(e)
	}
}

func registerRemindersHandlers() {
	c := modules.Client
	c.On("cmd:remind", RemindHandler)
	c.On("cmd:reminders", RemindersHandler)
	c.On("cmd:delremind", DelRemindHandler)
	go bootstrapReminders()
}

func initFromSrc_reminders_1_1() {
	modules.QueueHandlerRegistration(registerRemindersHandlers)
}
// === from timer.go ===
type timerData struct {
	chatID   int64
	userID   int64
	message  string
	media    tg.MessageMedia
	client   *tg.Client
	duration time.Duration
}

var (
	activeTimers   = make(map[string]*timerData)
	activeTimersMu sync.RWMutex
)

func SetTimerHandler(m *tg.NewMessage) error {
	args := m.Args()
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/timer &lt;duration&gt; &lt;message&gt;</code>\n<b>Example:</b> <code>/timer 1h30m Take a break!</code>\n\n<i>Reply to media to include it in the reminder</i>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	message := ""
	if len(parts) >= 2 {
		message = parts[1]
	}

	duration, err := parseDuration(parts[0])
	if err != nil {
		m.Reply("<b>Error:</b> " + err.Error())
		return nil
	}

	timer := &timerData{
		chatID:   m.ChatID(),
		userID:   m.SenderID(),
		message:  message,
		client:   m.Client,
		duration: duration,
	}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply.IsMedia() {
			timer.media = reply.Media()
		}
	}

	timerID := fmt.Sprintf("%d_%d_%d", m.ChatID(), m.SenderID(), time.Now().UnixNano())

	activeTimersMu.Lock()
	activeTimers[timerID] = timer
	activeTimersMu.Unlock()

	time.AfterFunc(duration, func() {
		sendTimerNotification(timerID)
	})

	m.Reply(fmt.Sprintf("Timer set for <b>%s</b>", formatDuration(duration)))
	return nil
}

func sendTimerNotification(timerID string) {
	activeTimersMu.RLock()
	timer, exists := activeTimers[timerID]
	activeTimersMu.RUnlock()

	if !exists {
		return
	}

	text := "<b>Timer Alert!</b>"
	if timer.message != "" {
		text += "\n" + timer.message
	}

	snoozeBtn := tg.Button.Data("Snooze 5m", "snooze_"+timerID).Primary()
	dismissBtn := tg.Button.Data("Dismiss", "dismiss_"+timerID).Danger()
	keyboard := tg.NewKeyboard().AddRow(snoozeBtn).AddRow(dismissBtn).Build()

	if timer.media != nil {
		timer.client.SendMedia(timer.chatID, timer.media, &tg.MediaOptions{
			Caption:     text,
			ReplyMarkup: keyboard,
		})
	} else {
		timer.client.SendMessage(timer.chatID, text, &tg.SendOptions{
			ReplyMarkup: keyboard,
		})
	}
}

func TimerCallbackHandler(cb *tg.CallbackQuery) error {
	data := cb.DataString()

	var action, timerID string
	if strings.HasPrefix(data, "snooze_") {
		action = "snooze"
		timerID = strings.TrimPrefix(data, "snooze_")
	} else if strings.HasPrefix(data, "dismiss_") {
		action = "dismiss"
		timerID = strings.TrimPrefix(data, "dismiss_")
	} else {
		return nil
	}

	activeTimersMu.RLock()
	timer, exists := activeTimers[timerID]
	activeTimersMu.RUnlock()

	if !exists {
		cb.Answer("Timer expired", &tg.CallbackOptions{Alert: true})
		return nil
	}

	if cb.Sender.ID != timer.userID {
		cb.Answer("Only the timer setter can do this!", &tg.CallbackOptions{Alert: true})
		return nil
	}

	switch action {
	case "snooze":
		snoozeDuration := 5 * time.Minute
		time.AfterFunc(snoozeDuration, func() {
			sendTimerNotification(timerID)
		})
		cb.Edit("<b>Snoozed for 5 minutes</b>")
		cb.Answer("Snoozed!")

	case "dismiss":
		activeTimersMu.Lock()
		delete(activeTimers, timerID)
		activeTimersMu.Unlock()
		cb.Edit("<b>Timer dismissed</b>")
		cb.Answer("Dismissed!")
	}

	return nil
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	var total time.Duration
	var numBuf strings.Builder

	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			numBuf.WriteRune(r)
		case r == 'd' || r == 'w':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("invalid duration: missing number before '%c'", r)
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			if r == 'd' {
				total += time.Duration(n) * 24 * time.Hour
			} else {
				total += time.Duration(n) * 7 * 24 * time.Hour
			}
		case r == 'h' || r == 'm' || r == 's':
			if numBuf.Len() == 0 {
				return 0, fmt.Errorf("invalid duration: missing number before '%c'", r)
			}
			n, _ := strconv.Atoi(numBuf.String())
			numBuf.Reset()
			switch r {
			case 'h':
				total += time.Duration(n) * time.Hour
			case 'm':
				total += time.Duration(n) * time.Minute
			case 's':
				total += time.Duration(n) * time.Second
			}
		default:
			return 0, fmt.Errorf("invalid character '%c' in duration", r)
		}
	}

	if numBuf.Len() > 0 {
		n, _ := strconv.Atoi(numBuf.String())
		total += time.Duration(n) * time.Second
	}

	if total <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}

	return total, nil
}

func formatDuration(d time.Duration) string {
	var parts []string

	if days := d / (24 * time.Hour); days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
		d -= days * 24 * time.Hour
	}
	if hours := d / time.Hour; hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
		d -= hours * time.Hour
	}
	if mins := d / time.Minute; mins > 0 {
		parts = append(parts, fmt.Sprintf("%dm", mins))
		d -= mins * time.Minute
	}
	if secs := d / time.Second; secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}

	return strings.Join(parts, " ")
}

func registerTimerHandlers() {
	c := modules.Client
	c.On("command:timer", SetTimerHandler)
	c.On("callback:snooze_", TimerCallbackHandler)
	c.On("callback:dismiss_", TimerCallbackHandler)
}

func initFromSrc_timer_2_1() {
	modules.QueueHandlerRegistration(registerTimerHandlers)
}

func init() {
	initFromSrc_notes_0_1()
	initFromSrc_reminders_1_1()
	initFromSrc_timer_2_1()
}
