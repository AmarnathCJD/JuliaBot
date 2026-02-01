package modules

import (
	"fmt"
	"main/modules/db"
	"regexp"
	"sort"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func FilterHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Filters work in groups only.</b>")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> You need Change Info permission to add filters.")
		return nil
	}

	args := m.Args()
	if args == "" && !m.IsReply() {
		m.Reply("<b>Usage:</b> <code>/filter keyword response</code> or reply with <code>/filter keyword</code>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	keyword := strings.ToLower(strings.TrimSpace(parts[0]))

	if keyword == "" {
		m.Reply("<b>Error:</b> Keyword required.")
		return nil
	}

	if len(keyword) < 2 {
		m.Reply("<b>Error:</b> Keyword must be at least 2 characters.")
		return nil
	}

	filter := &db.Filter{
		Keyword: keyword,
		AddedBy: m.SenderID(),
	}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error getting reply message")
			return nil
		}

		if reply.IsMedia() {
			if reply.Photo() != nil {
				filter.MediaType = "photo"
				filter.FileID = reply.File.FileID
			} else if reply.Document() != nil {
				filter.MediaType = "document"
				filter.FileID = reply.File.FileID
			} else if reply.Video() != nil {
				filter.MediaType = "video"
				filter.FileID = reply.File.FileID
			} else if reply.Audio() != nil {
				filter.MediaType = "audio"
				filter.FileID = reply.File.FileID
			} else if reply.Sticker() != nil {
				filter.MediaType = "sticker"
				filter.FileID = reply.File.FileID
			} else if reply.Animation() != nil {
				filter.MediaType = "animation"
				filter.FileID = reply.File.FileID
			}
		}

		filter.Content = reply.Text()
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			filter.Content = parts[1]
		}
	} else {
		if len(parts) < 2 {
			m.Reply("<b>Error:</b> Provide response or reply to a message.")
			return nil
		}
		filter.Content = parts[1]
	}

	if filter.Content == "" && filter.FileID == "" {
		m.Reply("<b>Error:</b> Filter response required.")
		return nil
	}

	if err := db.SaveFilter(m.ChatID(), filter); err != nil {
		m.Reply("<b>Failed to save filter.</b> Please try again.")
		return nil
	}

	m.Reply(fmt.Sprintf("<b>Filter saved:</b> <code>%s</code>", keyword))
	return nil
}

func StopFilterHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Filters work in groups only.</b>")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> You need Change Info permission to remove filters.")
		return nil
	}

	keyword := strings.ToLower(strings.TrimSpace(m.Args()))
	if keyword == "" {
		m.Reply("<b>Usage:</b> <code>/stop keyword</code>")
		return nil
	}

	filter, _ := db.GetFilter(m.ChatID(), keyword)
	if filter == nil {
		m.Reply(fmt.Sprintf("<b>Not found:</b> No filter for <code>%s</code>", keyword))
		return nil
	}

	if err := db.DeleteFilter(m.ChatID(), keyword); err != nil {
		m.Reply("<b>Failed to delete filter.</b> Please try again.")
		return nil
	}

	m.Reply(fmt.Sprintf("<b>Filter removed:</b> <code>%s</code>", keyword))
	return nil
}

func ListFiltersHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Filters work in groups only.</b>")
		return nil
	}

	filters, err := db.GetAllFilters(m.ChatID())
	if err != nil || len(filters) == 0 {
		m.Reply("<b>No filters yet.</b> Create one with <code>/filter keyword response</code>")
		return nil
	}

	sort.Slice(filters, func(i, j int) bool {
		return filters[i].Keyword < filters[j].Keyword
	})

	var resp strings.Builder
	resp.WriteString("<b>Saved Filters</b>\n")
	resp.WriteString("━━━━━━━━━━━━━━━━\n\n")

	mediaCount := 0
	for _, filter := range filters {
		marker := ""
		if filter.FileID != "" {
			marker = " [M]"
			mediaCount++
		}
		resp.WriteString(fmt.Sprintf(" • <code>%s</code>%s\n", filter.Keyword, marker))
	}

	resp.WriteString(fmt.Sprintf("\n━━━━━━━━━━━━━━━━\n<b>Total:</b> %d filters", len(filters)))
	if mediaCount > 0 {
		resp.WriteString(fmt.Sprintf(" (%d with media)", mediaCount))
	}

	m.Reply(resp.String())
	return nil
}

func StopAllFiltersHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Filters work in groups only.</b>")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("<b>Permission denied.</b> You need Change Info permission to remove filters.")
		return nil
	}

	count, _ := db.GetFiltersCount(m.ChatID())
	if count == 0 {
		m.Reply("<b>No filters to delete.</b>")
		return nil
	}

	b := tg.Button
	m.Reply(
		fmt.Sprintf("<b>Delete all %d filters?</b>\n\nThis cannot be undone.", count),
		&tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				b.Data("Delete", fmt.Sprintf("stopall_%d", m.SenderID())),
				b.Data("Cancel", fmt.Sprintf("cancelfilters_%d", m.SenderID())),
			).Build(),
		},
	)

	return nil
}

func StopAllFiltersCallback(c *tg.CallbackQuery) error {
	data := c.DataString()

	if after, ok := strings.CutPrefix(data, "cancelfilters_"); ok {
		userID := after
		if fmt.Sprint(c.SenderID) != userID {
			c.Answer("Not for you", &tg.CallbackOptions{Alert: true})
			return nil
		}
		c.Edit("<b>Cancelled.</b>")
		return nil
	}

	if after, ok := strings.CutPrefix(data, "stopall_"); ok {
		userID := after
		if fmt.Sprint(c.SenderID) != userID {
			c.Answer("Not for you", &tg.CallbackOptions{Alert: true})
			return nil
		}

		chatID := c.ChatID
		count, _ := db.GetFiltersCount(chatID)

		if err := db.DeleteAllFilters(chatID); err != nil {
			c.Edit("<b>Failed to delete filters.</b>")
			return nil
		}

		c.Edit(fmt.Sprintf("<b>Deleted:</b> %d filters", count))
	}

	return nil
}

func FilterWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() || m.Text() == "" || m.IsCommand() {
		return nil
	}

	filters, err := db.GetAllFilters(m.ChatID())
	if err != nil || len(filters) == 0 {
		return nil
	}

	msgLower := strings.ToLower(m.Text())

	for _, filter := range filters {
		pattern := `(?i)\b` + regexp.QuoteMeta(filter.Keyword) + `\b`
		matched, _ := regexp.MatchString(pattern, msgLower)

		if matched {
			// Parse buttons from filter content
			buttons, cleanContent := ParseButtonsFromText(filter.Content)

			if filter.FileID != "" {
				opts := &tg.MediaOptions{Caption: cleanContent}
				if len(buttons) > 0 {
					opts.ReplyMarkup = BuildButtonKeyboard(buttons)
				}
				file, _ := tg.ResolveBotFileID(filter.FileID)
				m.ReplyMedia(file, opts)
			} else if cleanContent != "" {
				if len(buttons) > 0 {
					m.Reply(cleanContent, &tg.SendOptions{ReplyMarkup: BuildButtonKeyboard(buttons)})
				} else {
					m.Reply(cleanContent)
				}
			}
			break
		}
	}

	return nil
}

func init() {
	Mods.AddModule("Filters", `<b>Filters Module</b>

Set automatic responses to keywords.

<b>Commands:</b>
 - /filter keyword response - Add filter (or reply)
 - /stop keyword - Remove filter
 - /filters - List filters
 - /stopall - Delete all filters

<b>Behavior:</b>
Triggers when keyword appears as a complete word. Example: \"hello\" triggers on \"hello there\" but not \"helloworld\".

<b>Add Buttons:</b>
Use format: <code>[Button Text](https://example.com)</code>
Example: <code>/filter spam [Report](url) | [Info](url)</code>

<b>Permission:</b> Admins with Change Info permission only.`)
}
