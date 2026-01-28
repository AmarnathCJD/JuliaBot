package modules

import (
	"fmt"
	"main/modules/db"
	"sort"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func AddBlacklistHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Blacklist can only be used in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to modify blacklist")
		return nil
	}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply.IsMedia() && reply.File != nil {
			fileID := reply.File.FileID
			if fileID == "" {
				m.Reply("Unable to get file ID from this media")
				return nil
			}

			if db.IsBlacklisted(m.ChatID(), fileID) {
				m.Reply("This media is already blacklisted")
				return nil
			}

			entry := &db.BlacklistEntry{
				Word:    fileID,
				FileID:  fileID,
				AddedBy: m.SenderID(),
			}

			if err := db.AddBlacklist(m.ChatID(), entry); err != nil {
				m.Reply("Failed to add media to blacklist")
				return nil
			}

			m.Reply("Media added to blacklist. Any matching media will be deleted.")
			return nil
		}
	}

	word := strings.TrimSpace(strings.ToLower(m.Args()))
	if word == "" {
		m.Reply("Usage: /addbl <word/phrase> or reply to media with /addbl")
		return nil
	}

	if len(word) < 2 {
		m.Reply("Blacklisted word must be at least 2 characters")
		return nil
	}

	if db.IsBlacklisted(m.ChatID(), word) {
		m.Reply(fmt.Sprintf("<code>%s</code> is already in the blacklist", word))
		return nil
	}

	entry := &db.BlacklistEntry{
		Word:    word,
		AddedBy: m.SenderID(),
	}

	if err := db.AddBlacklist(m.ChatID(), entry); err != nil {
		m.Reply("Failed to add to blacklist")
		return nil
	}

	m.Reply(fmt.Sprintf("Added <code>%s</code> to the blacklist", word))
	return nil
}

func RemoveBlacklistHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Blacklist can only be used in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to modify blacklist")
		return nil
	}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply.IsMedia() && reply.File != nil {
			fileID := reply.File.FileID
			if fileID == "" {
				m.Reply("Unable to get file ID from this media")
				return nil
			}

			if !db.IsBlacklisted(m.ChatID(), fileID) {
				m.Reply("This media is not in the blacklist")
				return nil
			}

			if err := db.RemoveBlacklist(m.ChatID(), fileID); err != nil {
				m.Reply("Failed to remove media from blacklist")
				return nil
			}

			m.Reply("Media removed from blacklist")
			return nil
		}
	}

	word := strings.TrimSpace(strings.ToLower(m.Args()))
	if word == "" {
		m.Reply("Usage: /rmbl <word> or reply to media")
		return nil
	}

	if !db.IsBlacklisted(m.ChatID(), word) {
		m.Reply(fmt.Sprintf("<code>%s</code> is not in the blacklist", word))
		return nil
	}

	if err := db.RemoveBlacklist(m.ChatID(), word); err != nil {
		m.Reply("Failed to remove from blacklist")
		return nil
	}

	m.Reply(fmt.Sprintf("Removed <code>%s</code> from the blacklist", word))
	return nil
}

func ListBlacklistHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Blacklist can only be used in groups")
		return nil
	}

	entries, err := db.GetBlacklist(m.ChatID())
	if err != nil || len(entries) == 0 {
		m.Reply("No blacklisted words in this chat")
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Word < entries[j].Word
	})

	settings, _ := db.GetBlacklistSettings(m.ChatID())
	actionStr := string(settings.Action)
	if settings.Duration != "" {
		actionStr += " (" + settings.Duration + ")"
	}

	var resp strings.Builder
	resp.WriteString("<b>Blacklisted items:</b>\n\n")

	wordCount := 0
	mediaCount := 0
	for i, entry := range entries {
		if entry.FileID != "" {
			resp.WriteString(fmt.Sprintf("%d. [Media File]\n", i+1))
			mediaCount++
		} else {
			resp.WriteString(fmt.Sprintf("%d. <code>%s</code>\n", i+1, entry.Word))
			wordCount++
		}
	}

	resp.WriteString(fmt.Sprintf("\nTotal: <b>%d</b> items", len(entries)))
	if wordCount > 0 && mediaCount > 0 {
		resp.WriteString(fmt.Sprintf(" (%d words, %d media)", wordCount, mediaCount))
	}
	resp.WriteString(fmt.Sprintf("\nAction: <b>%s</b>", actionStr))

	m.Reply(resp.String())
	return nil
}

func SetBlacklistActionHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Blacklist can only be used in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to modify blacklist settings")
		return nil
	}

	args := strings.Fields(strings.ToLower(m.Args()))
	if len(args) == 0 {
		current, _ := db.GetBlacklistSettings(m.ChatID())
		currentAction := string(current.Action)
		if current.Duration != "" {
			currentAction += " (" + current.Duration + ")"
		}

		m.Reply(fmt.Sprintf(`<b>Blacklist Action Settings</b>

Current action: <b>%s</b>

Usage: /setblaction <action> [duration]

<b>Available actions:</b>
 - <code>delete</code> - Delete the message (default)
 - <code>ban</code> - Ban the user
 - <code>mute</code> - Mute the user permanently
 - <code>tban</code> - Temporary ban (requires duration)
 - <code>tmute</code> - Temporary mute (requires duration)

<b>Duration examples:</b> 1h, 2d, 1w, 30m`, currentAction))
		return nil
	}

	action := args[0]
	duration := ""

	if len(args) > 1 {
		duration = args[1]
	}

	var blAction db.BlacklistAction
	switch action {
	case "delete", "del":
		blAction = db.ActionDelete
	case "ban":
		blAction = db.ActionBan
	case "mute":
		blAction = db.ActionMute
	case "tban":
		if duration == "" {
			m.Reply("tban requires a duration. Example: /setblaction tban 1h")
			return nil
		}
		blAction = db.ActionTBan
	case "tmute":
		if duration == "" {
			m.Reply("tmute requires a duration. Example: /setblaction tmute 1d")
			return nil
		}
		blAction = db.ActionTMute
	default:
		m.Reply("Unknown action. Use: delete, ban, mute, tban, tmute")
		return nil
	}

	settings := &db.BlacklistSettings{
		Action:   blAction,
		Duration: duration,
	}

	if err := db.SetBlacklistSettings(m.ChatID(), settings); err != nil {
		m.Reply("Failed to update settings")
		return nil
	}

	actionMsg := string(blAction)
	if duration != "" {
		actionMsg += " (" + duration + ")"
	}

	m.Reply(fmt.Sprintf("Blacklist action set to: <b>%s</b>", actionMsg))
	return nil
}

func BlacklistWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	entries, err := db.GetBlacklist(m.ChatID())
	if err != nil || len(entries) == 0 {
		return nil
	}

	var matchedWord string

	// Check media files first
	if m.IsMedia() && m.File != nil && m.File.FileID != "" {
		for _, entry := range entries {
			if entry.FileID != "" && entry.FileID == m.File.FileID {
				matchedWord = "media"
				break
			}
		}
	}

	// Check text if no media match
	if matchedWord == "" && m.Text() != "" {
		msgLower := strings.ToLower(m.Text())
		for _, entry := range entries {
			if entry.FileID == "" && strings.Contains(msgLower, entry.Word) {
				matchedWord = entry.Word
				break
			}
		}
	}

	if matchedWord == "" {
		return nil
	}

	m.Delete()

	settings, _ := db.GetBlacklistSettings(m.ChatID())

	user, err := m.Client.ResolvePeer(m.SenderID())
	if err != nil {
		return nil
	}

	switch settings.Action {
	case db.ActionBan:
		m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Ban: true})
		m.Respond(fmt.Sprintf("<b>%s</b> was banned for using blacklisted word", m.Sender.FirstName))

	case db.ActionMute:
		m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Mute: true})
		m.Respond(fmt.Sprintf("<b>%s</b> was muted for using blacklisted word", m.Sender.FirstName))

	case db.ActionTBan:
		duration, err := parseAdminDuration(settings.Duration)
		if err != nil || duration == 0 {
			duration = time.Hour
		}
		untilDate := int32(time.Now().Add(duration).Unix())
		m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Ban: true, TillDate: untilDate})
		m.Respond(fmt.Sprintf("<b>%s</b> was banned for %s for using blacklisted word", m.Sender.FirstName, duration.String()))

	case db.ActionTMute:
		duration, err := parseAdminDuration(settings.Duration)
		if err != nil || duration == 0 {
			duration = time.Hour
		}
		untilDate := int32(time.Now().Add(duration).Unix())
		m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Mute: true, TillDate: untilDate})
		m.Respond(fmt.Sprintf("<b>%s</b> was muted for %s for using blacklisted word", m.Sender.FirstName, duration.String()))

	case db.ActionDelete:
	}

	return nil
}

func ClearBlacklistHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Blacklist can only be used in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to clear blacklist")
		return nil
	}

	count, _ := db.GetBlacklistCount(m.ChatID())
	if count == 0 {
		m.Reply("Blacklist is already empty")
		return nil
	}

	b := tg.Button
	m.Reply(
		fmt.Sprintf("<b>Are you sure you want to clear all %d blacklisted words?</b>", count),
		&tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				b.Data("Yes, clear all", fmt.Sprintf("clearbl_%d", m.SenderID())),
				b.Data("Cancel", fmt.Sprintf("cancelbl_%d", m.SenderID())),
			).Build(),
		},
	)

	return nil
}

func ClearBlacklistCallback(c *tg.CallbackQuery) error {
	data := c.DataString()

	if strings.HasPrefix(data, "cancelbl_") {
		userID := strings.TrimPrefix(data, "cancelbl_")
		if fmt.Sprint(c.SenderID) != userID {
			c.Answer("This is not for you", &tg.CallbackOptions{Alert: true})
			return nil
		}
		c.Edit("Operation cancelled")
		return nil
	}

	if strings.HasPrefix(data, "clearbl_") {
		userID := strings.TrimPrefix(data, "clearbl_")
		if fmt.Sprint(c.SenderID) != userID {
			c.Answer("This is not for you", &tg.CallbackOptions{Alert: true})
			return nil
		}

		chatID := c.ChatID
		count, _ := db.GetBlacklistCount(chatID)

		if err := db.ClearBlacklist(chatID); err != nil {
			c.Edit("Failed to clear blacklist")
			return nil
		}

		c.Edit(fmt.Sprintf("Cleared <b>%d</b> blacklisted words", count))
	}

	return nil
}

func init() {
	Mods.AddModule("Blacklist", `<b>Blacklist Module</b>

Block specific words/phrases or media in your group.

<b>Commands:</b>
 - /addbl <word> - Add word to blacklist
 - /addbl [reply to media] - Add media to blacklist
 - /addblacklist <word> - Same as above
 - /rmbl <word> - Remove word from blacklist
 - /rmblacklist <word> - Same as above
 - /listbl - List all blacklisted items
 - /blacklist - Same as above
 - /setblaction <action> - Set action for violations
 - /clearbl - Clear all blacklisted items

<b>Actions:</b>
 - delete - Delete message (default)
 - ban - Ban the user
 - mute - Mute forever
 - tban <duration> - Temporary ban
 - tmute <duration> - Temporary mute

<b>Note:</b> Admins are exempt from blacklist checks.`)
}
