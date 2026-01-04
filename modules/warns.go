package modules

import (
	"fmt"
	"main/modules/db"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func WarnUserHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warns can only be used in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to warn users")
		return nil
	}

	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need the Ban Users right to enforce warns")
		return nil
	}

	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Usage: /warn <user> [reason] or reply to a message with /warn [reason]")
		return nil
	}

	userID := m.Client.GetPeerID(user)

	if IsUserAdmin(m.Client, int(userID), int(m.ChatID()), "") {
		m.Reply("Cannot warn admins")
		return nil
	}

	if reason == "" {
		reason = "No reason specified"
	}

	warn := &db.Warn{
		Reason:    reason,
		AdminID:   m.SenderID(),
		Timestamp: time.Now(),
	}

	count, err := db.AddWarn(m.ChatID(), userID, warn)
	if err != nil {
		m.Reply("Failed to add warn")
		return nil
	}

	settings, _ := db.GetWarnSettings(m.ChatID())

	userInfo, _ := m.Client.GetUser(userID)
	userName := "User"
	if userInfo != nil {
		userName = userInfo.FirstName
	}

	if count >= settings.MaxWarns {
		switch settings.Action {
		case db.WarnActionBan:
			m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Ban: true})
			db.ResetWarns(m.ChatID(), userID)
			m.Reply(fmt.Sprintf("<b>%s</b> has been banned after reaching %d/%d warns\n<b>Reason:</b> %s",
				userName, count, settings.MaxWarns, reason))
		case db.WarnActionMute:
			m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Mute: true})
			db.ResetWarns(m.ChatID(), userID)
			m.Reply(fmt.Sprintf("<b>%s</b> has been muted after reaching %d/%d warns\n<b>Reason:</b> %s",
				userName, count, settings.MaxWarns, reason))
		case db.WarnActionKick:
			m.Client.KickParticipant(m.ChatID(), user)
			db.ResetWarns(m.ChatID(), userID)
			m.Reply(fmt.Sprintf("<b>%s</b> has been kicked after reaching %d/%d warns\n<b>Reason:</b> %s",
				userName, count, settings.MaxWarns, reason))
		}
		return nil
	}

	b := tg.Button
	m.Reply(
		fmt.Sprintf("<b>%s</b> has been warned (%d/%d)\n<b>Reason:</b> %s",
			userName, count, settings.MaxWarns, reason),
		&tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				b.Data("Remove Warn", fmt.Sprintf("rmwarn_%d_%d", userID, m.SenderID())),
			).Build(),
		},
	)
	return nil
}

func RemoveWarnCallback(c *tg.CallbackQuery) error {
	data := c.DataString()

	if !strings.HasPrefix(data, "rmwarn_") {
		return nil
	}

	parts := strings.Split(strings.TrimPrefix(data, "rmwarn_"), "_")
	if len(parts) != 2 {
		return nil
	}

	userID, _ := strconv.ParseInt(parts[0], 10, 64)
	adminID, _ := strconv.ParseInt(parts[1], 10, 64)

	if c.SenderID != adminID && !IsUserAdmin(c.Client, int(c.SenderID), int(c.ChatID), "ban") {
		c.Answer("Only the warning admin or other admins can remove this warn", &tg.CallbackOptions{Alert: true})
		return nil
	}

	warns, _ := db.GetWarns(c.ChatID, userID)
	if len(warns) == 0 {
		c.Edit("No warns to remove")
		return nil
	}

	db.ResetWarns(c.ChatID, userID)

	settings, _ := db.GetWarnSettings(c.ChatID)
	newCount := max(len(warns)-1, 0)

	for i := 0; i < newCount; i++ {
		db.AddWarn(c.ChatID, userID, warns[i])
	}

	c.Edit(fmt.Sprintf("Warn removed. User now has %d/%d warns", newCount, settings.MaxWarns))
	return nil
}

func ListWarnsHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warns can only be checked in groups")
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		user, _ = m.Client.ResolvePeer(m.SenderID())
	}

	userID := m.Client.GetPeerID(user)
	warns, err := db.GetWarns(m.ChatID(), userID)
	if err != nil || len(warns) == 0 {
		m.Reply("This user has no warns")
		return nil
	}

	settings, _ := db.GetWarnSettings(m.ChatID())

	userInfo, _ := m.Client.GetUser(userID)
	userName := "User"
	if userInfo != nil {
		userName = userInfo.FirstName
	}

	var resp strings.Builder
	resp.WriteString(fmt.Sprintf("<b>Warns for %s:</b> %d/%d\n\n", userName, len(warns), settings.MaxWarns))

	for i, warn := range warns {
		adminInfo, _ := m.Client.GetUser(warn.AdminID)
		adminName := "Unknown"
		if adminInfo != nil {
			adminName = adminInfo.FirstName
		}
		resp.WriteString(fmt.Sprintf("%d. %s\n   <i>By %s - %s</i>\n",
			i+1, warn.Reason, adminName, warn.Timestamp.Format("02 Jan 2006")))
	}

	m.Reply(resp.String())
	return nil
}

func ResetWarnsHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warns can only be reset in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to reset warns")
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Usage: /resetwarns <user> or reply to a user")
		return nil
	}

	userID := m.Client.GetPeerID(user)

	warns, _ := db.GetWarns(m.ChatID(), userID)
	if len(warns) == 0 {
		m.Reply("This user has no warns to reset")
		return nil
	}

	if err := db.ResetWarns(m.ChatID(), userID); err != nil {
		m.Reply("Failed to reset warns")
		return nil
	}

	userInfo, _ := m.Client.GetUser(userID)
	userName := "User"
	if userInfo != nil {
		userName = userInfo.FirstName
	}

	m.Reply(fmt.Sprintf("Reset %d warn(s) for <b>%s</b>", len(warns), userName))
	return nil
}

func RemoveWarnHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warns can only be removed in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "ban") {
		m.Reply("You need Ban Users permission to remove warns")
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Usage: /rmwarn <user> or reply to a user")
		return nil
	}

	userID := m.Client.GetPeerID(user)

	warns, _ := db.GetWarns(m.ChatID(), userID)
	if len(warns) == 0 {
		m.Reply("This user has no warns to remove")
		return nil
	}

	db.ResetWarns(m.ChatID(), userID)

	for i := 0; i < len(warns)-1; i++ {
		db.AddWarn(m.ChatID(), userID, warns[i])
	}

	settings, _ := db.GetWarnSettings(m.ChatID())
	newCount := len(warns) - 1

	userInfo, _ := m.Client.GetUser(userID)
	userName := "User"
	if userInfo != nil {
		userName = userInfo.FirstName
	}

	m.Reply(fmt.Sprintf("Removed last warn for <b>%s</b>. Now at %d/%d warns", userName, newCount, settings.MaxWarns))
	return nil
}

func SetWarnLimitHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warn settings can only be changed in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to modify warn settings")
		return nil
	}

	args := strings.TrimSpace(m.Args())
	if args == "" {
		settings, _ := db.GetWarnSettings(m.ChatID())
		m.Reply(fmt.Sprintf("<b>Current warn limit:</b> %d\n\nUsage: /setwarnlimit <number>", settings.MaxWarns))
		return nil
	}

	limit, err := strconv.Atoi(args)
	if err != nil || limit < 1 || limit > 20 {
		m.Reply("Warn limit must be a number between 1 and 20")
		return nil
	}

	settings, _ := db.GetWarnSettings(m.ChatID())
	settings.MaxWarns = limit

	if err := db.SetWarnSettings(m.ChatID(), settings); err != nil {
		m.Reply("Failed to update warn limit")
		return nil
	}

	m.Reply(fmt.Sprintf("Warn limit set to <b>%d</b>", limit))
	return nil
}

func SetWarnActionHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warn settings can only be changed in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "change_info") {
		m.Reply("You need Change Info permission to modify warn settings")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))
	settings, _ := db.GetWarnSettings(m.ChatID())

	if args == "" {
		m.Reply(fmt.Sprintf(`<b>Warn Action Settings</b>

Current action: <b>%s</b>
Current limit: <b>%d</b>

Usage: /setwarnaction <action>

<b>Available actions:</b>
 - <code>ban</code> - Ban user when limit is reached
 - <code>mute</code> - Mute user when limit is reached
 - <code>kick</code> - Kick user when limit is reached`, settings.Action, settings.MaxWarns))
		return nil
	}

	var action db.WarnAction
	switch args {
	case "ban":
		action = db.WarnActionBan
	case "mute":
		action = db.WarnActionMute
	case "kick":
		action = db.WarnActionKick
	default:
		m.Reply("Unknown action. Use: ban, mute, kick")
		return nil
	}

	settings.Action = action

	if err := db.SetWarnSettings(m.ChatID(), settings); err != nil {
		m.Reply("Failed to update warn action")
		return nil
	}

	m.Reply(fmt.Sprintf("Warn action set to <b>%s</b>", action))
	return nil
}

func WarnSettingsHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warn settings can only be viewed in groups")
		return nil
	}

	settings, _ := db.GetWarnSettings(m.ChatID())

	m.Reply(fmt.Sprintf(`<b>Warn Settings</b>

<b>Limit:</b> %d warns
<b>Action:</b> %s

Use /setwarnlimit to change the limit
Use /setwarnaction to change the action`, settings.MaxWarns, settings.Action))
	return nil
}

func init() {
	Mods.AddModule("Warns", `<b>Warns Module</b>

Warn users for rule violations.

<b>Commands:</b>
 - /warn <user> [reason] - Warn a user
 - /warns [user] - Check warns for a user
 - /rmwarn <user> - Remove last warn from a user
 - /resetwarns <user> - Reset all warns for a user
 - /setwarnlimit <num> - Set max warns before action (1-20)
 - /setwarnaction <action> - Set action when limit reached
 - /warnsettings - View current warn settings

<b>Actions:</b>
 - ban - Ban user when limit is reached
 - mute - Mute user when limit is reached
 - kick - Kick user when limit is reached

<b>Note:</b> Default is 3 warns with ban action.`)
}
