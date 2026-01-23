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

func SaveNoteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Notes can only be saved in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to save notes")
		return nil
	}

	args := m.Args()
	if args == "" && !m.IsReply() {
		m.Reply("Usage: /save <notename> [content] or reply to a message with /save <notename>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	noteName := strings.ToLower(strings.TrimSpace(parts[0]))

	if noteName == "" {
		m.Reply("Please provide a note name")
		return nil
	}

	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(noteName) {
		m.Reply("Note name can only contain lowercase letters, numbers, and underscores")
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
		if len(parts) > 1 {
			note.Content = parts[1]
		}
	} else {
		if len(parts) < 2 {
			m.Reply("Please provide content for the note or reply to a message")
			return nil
		}
		note.Content = parts[1]
	}

	if strings.Contains(note.Content, "{admin}") {
		note.AdminOnly = true
		note.Content = strings.ReplaceAll(note.Content, "{admin}", "")
		note.Content = strings.TrimSpace(note.Content)
	}

	if note.Content == "" && note.FileID == "" {
		m.Reply("Note cannot be empty")
		return nil
	}

	if err := db.SaveNote(m.ChatID(), note); err != nil {
		m.Reply("Failed to save note")
		return nil
	}

	adminTag := ""
	if note.AdminOnly {
		adminTag = " [Admin Only]"
	}

	m.Reply(fmt.Sprintf("Note <code>%s</code> saved successfully%s", noteName, adminTag))
	return nil
}

// GetNoteHandler handles /note and #notename
func GetNoteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	noteName := strings.ToLower(strings.TrimSpace(m.Args()))
	if noteName == "" {
		m.Reply("Usage: /note <notename> or #notename")
		return nil
	}

	return sendNote(m, noteName)
}

// NoteHashHandler handles #notename triggers
func NoteHashHandler(m *tg.NewMessage) error {
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
	if note.AdminOnly && !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "") {
		m.Reply("This note is restricted to admins only")
		return nil
	}

	if note.FileID != "" {
		opts := &tg.MediaOptions{Caption: note.Content}
		media, err := tg.ResolveBotFileID(note.FileID)
		if err != nil {
			m.Reply("Failed to resolve media")
			return nil
		}
		m.ReplyMedia(media, opts)
		return nil
	}

	if note.Content != "" {
		m.Reply(note.Content)
	}

	return nil
}

func ListNotesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Notes can only be listed in groups")
		return nil
	}

	notes, err := db.GetAllNotes(m.ChatID())
	if err != nil || len(notes) == 0 {
		m.Reply("No notes saved in this chat")
		return nil
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Name < notes[j].Name
	})

	var resp strings.Builder
	resp.WriteString("<b>Notes in this chat:</b>\n\n")

	for _, note := range notes {
		marker := ""
		if note.AdminOnly {
			marker = " [A]"
		}
		if note.FileID != "" {
			marker += " [M]"
		}
		resp.WriteString(fmt.Sprintf(" - <code>#%s</code>%s\n", note.Name, marker))
	}

	resp.WriteString(fmt.Sprintf("\nTotal: <b>%d</b> notes", len(notes)))
	resp.WriteString("\n\n<i>[A] = Admin only, [M] = Has media</i>")

	m.Reply(resp.String())
	return nil
}

func ClearNoteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Notes can only be cleared in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to clear notes")
		return nil
	}

	noteName := strings.ToLower(strings.TrimSpace(m.Args()))
	if noteName == "" {
		m.Reply("Usage: /clear <notename>")
		return nil
	}

	note, _ := db.GetNote(m.ChatID(), noteName)
	if note == nil {
		m.Reply(fmt.Sprintf("Note <code>%s</code> does not exist", noteName))
		return nil
	}

	if err := db.DeleteNote(m.ChatID(), noteName); err != nil {
		m.Reply("Failed to delete note")
		return nil
	}

	m.Reply(fmt.Sprintf("Note <code>%s</code> has been deleted", noteName))
	return nil
}

func ClearAllNotesHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Notes can only be cleared in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to clear all notes")
		return nil
	}

	count, _ := db.GetNotesCount(m.ChatID())
	if count == 0 {
		m.Reply("No notes to delete")
		return nil
	}

	b := tg.Button
	m.Reply(
		fmt.Sprintf("<b>Are you sure you want to delete all %d notes?</b>\n\nThis action cannot be undone.", count),
		&tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				b.Data("Yes, delete all", fmt.Sprintf("clearallnotes_%d", m.SenderID())),
				b.Data("Cancel", fmt.Sprintf("cancelnotes_%d", m.SenderID())),
			).Build(),
		},
	)

	return nil
}

func ClearAllNotesCallback(c *tg.CallbackQuery) error {
	data := c.DataString()

	if strings.HasPrefix(data, "cancelnotes_") {
		userID := strings.TrimPrefix(data, "cancelnotes_")
		if fmt.Sprint(c.SenderID) != userID {
			c.Answer("This is not for you", &tg.CallbackOptions{Alert: true})
			return nil
		}
		c.Edit("Operation cancelled")
		return nil
	}

	if strings.HasPrefix(data, "clearallnotes_") {
		userID := strings.TrimPrefix(data, "clearallnotes_")
		if fmt.Sprint(c.SenderID) != userID {
			c.Answer("This is not for you", &tg.CallbackOptions{Alert: true})
			return nil
		}

		chatID := c.ChatID
		count, _ := db.GetNotesCount(chatID)

		if err := db.DeleteAllNotes(chatID); err != nil {
			c.Edit("Failed to delete notes")
			return nil
		}

		c.Edit(fmt.Sprintf("Successfully deleted <b>%d</b> notes", count))
	}

	return nil
}

func SaveTempNoteHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Notes can only be saved in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to save notes")
		return nil
	}

	args := m.Args()
	if args == "" {
		m.Reply("Usage: /tempnote <duration> <name> [content]")
		return nil
	}

	parts := strings.SplitN(args, " ", 3)
	if len(parts) < 2 {
		m.Reply("Usage: /tempnote <duration> <name> [content]")
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
		m.Reply("Invalid duration. Examples: 10m, 1h, 30s")
		return nil
	}

	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(noteName) {
		m.Reply("Note name can only contain lowercase letters, numbers, and underscores")
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
			m.Reply("Please provide content or reply to a message")
			return nil
		}
		note.Content = content
	}

	if err := db.SaveNote(m.ChatID(), note); err != nil {
		m.Reply("Failed to save note")
		return nil
	}

	m.Reply(fmt.Sprintf("Temporary note <code>#%s</code> saved (Expires in %s)", noteName, duration))
	return nil
}

func init() {
	Mods.AddModule("Notes", `<b>Notes Module</b>

Save and retrieve notes in your group.

<b>Commands:</b>
 - /save <name> [content] - Save a note (reply to msg or provide content)
 - /tempnote <duration> <name> [content] - Save a self-destructing note
 - /note <name> - Get a note
 - #notename - Quick way to get a note
 - /notes or /listnotes - List all notes
 - /clear <name> - Delete a specific note
 - /clearallnotes - Delete all notes (with confirmation)

<b>Special Tags:</b>
 - Add {admin} in content to make note admin-only

<b>Note:</b> Only admins with Change Info permission can save/delete notes.`)
}
