package modules

import (
	"fmt"
	"main/modules/db"
	"regexp"
	"sort"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// Variable replacements supported in notes
var variableReplacements = map[string]func(*tg.NewMessage) string{
	"{mention}": func(m *tg.NewMessage) string {
		return fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", m.SenderID(), m.Sender.FirstName)
	},
	"{firstname}": func(m *tg.NewMessage) string { return m.Sender.FirstName },
	"{lastname}":  func(m *tg.NewMessage) string { return m.Sender.LastName },
	"{fullname}":  func(m *tg.NewMessage) string { return m.Sender.FirstName + " " + m.Sender.LastName },
	"{username}": func(m *tg.NewMessage) string {
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

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
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
	if note.AdminOnly && !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
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
		resp.WriteString("\n<i>" + strings.Join(stats, " • ") + "</i>")
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

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
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

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
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

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
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

func formatDurationv2(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1f hours", d.Hours())
	}
	return fmt.Sprintf("%.1f days", d.Hours()/24)
}

// NoteInfoHandler shows detailed info about a note
func NoteInfoHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Notes work in groups only.</b>")
		return nil
	}

	noteName := strings.ToLower(strings.TrimSpace(m.Args()))
	if noteName == "" {
		m.Reply("<b>Usage:</b> <code>/noteinfo notename</code>")
		return nil
	}

	note, err := db.GetNote(m.ChatID(), noteName)
	if err != nil || note == nil {
		m.Reply(fmt.Sprintf("<b>Note not found:</b> <code>#%s</code>", noteName))
		return nil
	}

	var info strings.Builder
	info.WriteString("<b>Note Information</b>\n")
	info.WriteString("━━━━━━━━━━━━━━━━\n\n")
	info.WriteString(fmt.Sprintf("<b>Name:</b> <code>#%s</code>\n", note.Name))

	creator, _ := m.Client.GetUser(note.CreatedBy)
	if creator != nil {
		info.WriteString(fmt.Sprintf("<b>Created by:</b> %s\n", creator.FirstName))
	}

	if note.FileID != "" {
		info.WriteString(fmt.Sprintf("<b>Media:</b> %s\n", note.MediaType))
	}

	contentLen := len(note.Content)
	if contentLen > 0 {
		info.WriteString(fmt.Sprintf("<b>Length:</b> %d characters\n", contentLen))
	}

	if note.AdminOnly {
		info.WriteString("<b>Access:</b> Admin only\n")
	} else {
		info.WriteString("<b>Access:</b> Everyone\n")
	}

	if !note.ExpiresAt.IsZero() {
		remaining := time.Until(note.ExpiresAt)
		if remaining > 0 {
			info.WriteString(fmt.Sprintf("<b>Expires in:</b> %s\n", formatDuration(remaining)))
		} else {
			info.WriteString("<b>Status:</b> Expired\n")
		}
	}

	info.WriteString("\n<i>Use <code>#" + noteName + "</code> to retrieve</i>")

	m.Reply(info.String())
	return nil
}

// SearchNotesHandler searches notes by name or content
func SearchNotesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Notes work in groups only.</b>")
		return nil
	}

	query := strings.ToLower(strings.TrimSpace(m.Args()))
	if query == "" {
		m.Reply("<b>Usage:</b> <code>/searchnotes keyword</code>")
		return nil
	}

	allNotes, err := db.GetAllNotes(m.ChatID())
	if err != nil || len(allNotes) == 0 {
		m.Reply("<b>No notes to search.</b>")
		return nil
	}

	var matches []*db.Note
	for _, note := range allNotes {
		if strings.Contains(note.Name, query) || strings.Contains(strings.ToLower(note.Content), query) {
			matches = append(matches, note)
		}
	}

	if len(matches) == 0 {
		m.Reply(fmt.Sprintf("<b>No results for:</b> <code>%s</code>", query))
		return nil
	}

	var resp strings.Builder
	resp.WriteString(fmt.Sprintf("<b>Search Results:</b> <code>%s</code>\n", query))
	resp.WriteString("━━━━━━━━━━━━━━━━\n\n")

	for i, note := range matches {
		if i >= 20 {
			resp.WriteString(fmt.Sprintf("\n<i>...and %d more</i>", len(matches)-20))
			break
		}

		var badges []string
		if note.AdminOnly {
			badges = append(badges, "[A]")
		}
		if note.FileID != "" {
			badges = append(badges, "[M]")
		}

		badgeStr := ""
		if len(badges) > 0 {
			badgeStr = " " + strings.Join(badges, " ")
		}

		resp.WriteString(fmt.Sprintf(" • <code>#%s</code>%s\n", note.Name, badgeStr))
	}

	resp.WriteString(fmt.Sprintf("\n━━━━━━━━━━━━━━━━\n<b>Found:</b> %d matches", len(matches)))
	m.Reply(resp.String())
	return nil
}

// RenameNoteHandler renames an existing note
func RenameNoteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Notes work in groups only.</b>")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> You need Change Info permission.")
		return nil
	}

	args := m.Args()
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		m.Reply("<b>Usage:</b> <code>/rename oldname newname</code>")
		return nil
	}

	oldName := strings.ToLower(strings.TrimSpace(parts[0]))
	newName := strings.ToLower(strings.TrimSpace(parts[1]))

	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(newName) {
		m.Reply("<b>Invalid name.</b> Use only lowercase letters, numbers, and underscores.")
		return nil
	}

	oldNote, _ := db.GetNote(m.ChatID(), oldName)
	if oldNote == nil {
		m.Reply(fmt.Sprintf("<b>Note not found:</b> <code>#%s</code>", oldName))
		return nil
	}

	existingNote, _ := db.GetNote(m.ChatID(), newName)
	if existingNote != nil {
		m.Reply(fmt.Sprintf("<b>Name already exists:</b> <code>#%s</code>", newName))
		return nil
	}

	// Create new note with new name
	oldNote.Name = newName
	if err := db.SaveNote(m.ChatID(), oldNote); err != nil {
		m.Reply("<b>Failed to rename note.</b>")
		return nil
	}

	// Delete old note
	db.DeleteNote(m.ChatID(), oldName)

	m.Reply(fmt.Sprintf("<b>Note renamed:</b> <code>#%s</code> → <code>#%s</code>", oldName, newName))
	return nil
}

func init() {
	Mods.AddModule("Notes", `<b>Notes Module</b>

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
