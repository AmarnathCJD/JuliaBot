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
		m.Reply("Filters can only be used in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to add filters")
		return nil
	}

	args := m.Args()
	if args == "" && !m.IsReply() {
		m.Reply("Usage: /filter <keyword> [response] or reply to a message with /filter <keyword>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	keyword := strings.ToLower(strings.TrimSpace(parts[0]))

	if keyword == "" {
		m.Reply("Please provide a keyword")
		return nil
	}

	if len(keyword) < 2 {
		m.Reply("Keyword must be at least 2 characters")
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

		if len(parts) > 1 {
			filter.Content = parts[1]
		}
	} else {
		if len(parts) < 2 {
			m.Reply("Please provide a response or reply to a message")
			return nil
		}
		filter.Content = parts[1]
	}

	if filter.Content == "" && filter.FileID == "" {
		m.Reply("Filter response cannot be empty")
		return nil
	}

	if err := db.SaveFilter(m.ChatID(), filter); err != nil {
		m.Reply("Failed to save filter")
		return nil
	}

	m.Reply(fmt.Sprintf("Filter for <code>%s</code> saved successfully", keyword))
	return nil
}

func StopFilterHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Filters can only be used in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to remove filters")
		return nil
	}

	keyword := strings.ToLower(strings.TrimSpace(m.Args()))
	if keyword == "" {
		m.Reply("Usage: /stop <keyword>")
		return nil
	}

	filter, _ := db.GetFilter(m.ChatID(), keyword)
	if filter == nil {
		m.Reply(fmt.Sprintf("No filter found for <code>%s</code>", keyword))
		return nil
	}

	if err := db.DeleteFilter(m.ChatID(), keyword); err != nil {
		m.Reply("Failed to delete filter")
		return nil
	}

	m.Reply(fmt.Sprintf("Filter for <code>%s</code> has been removed", keyword))
	return nil
}

func ListFiltersHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Filters can only be used in groups")
		return nil
	}

	filters, err := db.GetAllFilters(m.ChatID())
	if err != nil || len(filters) == 0 {
		m.Reply("No filters in this chat")
		return nil
	}

	sort.Slice(filters, func(i, j int) bool {
		return filters[i].Keyword < filters[j].Keyword
	})

	var resp strings.Builder
	resp.WriteString("<b>Filters in this chat:</b>\n\n")

	for _, filter := range filters {
		marker := ""
		if filter.FileID != "" {
			marker = " [M]"
		}
		resp.WriteString(fmt.Sprintf(" - <code>%s</code>%s\n", filter.Keyword, marker))
	}

	resp.WriteString(fmt.Sprintf("\nTotal: <b>%d</b> filters", len(filters)))
	resp.WriteString("\n\n<i>[M] = Has media</i>")

	m.Reply(resp.String())
	return nil
}

func StopAllFiltersHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Filters can only be used in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to remove filters")
		return nil
	}

	count, _ := db.GetFiltersCount(m.ChatID())
	if count == 0 {
		m.Reply("No filters to delete")
		return nil
	}

	b := tg.Button
	m.Reply(
		fmt.Sprintf("<b>Are you sure you want to delete all %d filters?</b>\n\nThis action cannot be undone.", count),
		&tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				b.Data("Yes, delete all", fmt.Sprintf("stopall_%d", m.SenderID())),
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
			c.Answer("This is not for you", &tg.CallbackOptions{Alert: true})
			return nil
		}
		c.Edit("Operation cancelled")
		return nil
	}

	if after, ok := strings.CutPrefix(data, "stopall_"); ok {
		userID := after
		if fmt.Sprint(c.SenderID) != userID {
			c.Answer("This is not for you", &tg.CallbackOptions{Alert: true})
			return nil
		}

		chatID := c.ChatID
		count, _ := db.GetFiltersCount(chatID)

		if err := db.DeleteAllFilters(chatID); err != nil {
			c.Edit("Failed to delete filters")
			return nil
		}

		c.Edit(fmt.Sprintf("Successfully deleted <b>%d</b> filters", count))
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
			if filter.FileID != "" {
				opts := &tg.MediaOptions{Caption: filter.Content}
				file, _ := tg.ResolveBotFileID(filter.FileID)
				m.ReplyMedia(file, opts)
			} else if filter.Content != "" {
				m.Reply(filter.Content)
			}
			break
		}
	}

	return nil
}

func init() {
	Mods.AddModule("Filters", `<b>Filters Module</b>

Set automatic responses to specific keywords.

<b>Commands:</b>
 - /filter <keyword> [response] - Add a filter (or reply to message)
 - /stop <keyword> - Remove a filter
 - /filters - List all filters
 - /stopall - Remove all filters (with confirmation)

<b>How it works:</b>
Filters trigger when the keyword appears as a whole word in messages.
Example: Filter "hello" triggers on "hello there" but not on "helloworld"

<b>Note:</b> Only admins with Change Info permission can modify filters.`)
}
