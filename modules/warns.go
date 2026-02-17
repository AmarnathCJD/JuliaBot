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
		m.Reply("Warning system is only available in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		m.Reply("You need ban Users permission to issue warnings")
		return nil
	}

	if !CanBot(m.Client, m.Channel, "ban") {
		m.Reply("I need ban Users permission to enforce warnings")
		return nil
	}

	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Usage: /warn <user> [reason] or reply to a message with /warn [reason]")
		return nil
	}

	userID := m.Client.GetPeerID(user)

	if IsUserAdmin(m.Client, userID, m.ChatID(), "") {
		m.Reply("Administrators cannot be warned")
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
		m.Reply("Failed to add warning")
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
			m.Reply(fmt.Sprintf("%s has been banned for reaching %d warning(s)\nReason: %s",
				userName, settings.MaxWarns, reason))
		case db.WarnActionMute:
			m.Client.EditBanned(m.ChatID(), user, &tg.BannedOptions{Mute: true})
			db.ResetWarns(m.ChatID(), userID)
			m.Reply(fmt.Sprintf("%s has been muted for reaching %d warning(s)\nReason: %s",
				userName, settings.MaxWarns, reason))
		case db.WarnActionKick:
			m.Client.KickParticipant(m.ChatID(), user)
			db.ResetWarns(m.ChatID(), userID)
			m.Reply(fmt.Sprintf("%s has been removed for reaching %d warning(s)\nReason: %s",
				userName, settings.MaxWarns, reason))
		}
		return nil
	}

	b := tg.Button
	m.Reply(
		fmt.Sprintf("Warning issued to %s (%d/%d)\nReason: %s",
			userName, count, settings.MaxWarns, reason),
		&tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				b.Data("Remove Warning", fmt.Sprintf("rmwarn_%d_%d", userID, m.SenderID())).Danger(),
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

	if c.SenderID != adminID && !IsUserAdmin(c.Client, c.SenderID, c.ChatID, "ban") {
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
		m.Reply("Warning system is only available in groups")
		return nil
	}

	user, _, err := GetUserFromContext(m)
	if err != nil {
		user, _ = m.Client.ResolvePeer(m.SenderID())
	}

	userID := m.Client.GetPeerID(user)
	warns, err := db.GetWarns(m.ChatID(), userID)
	if err != nil || len(warns) == 0 {
		m.Reply("This user has no warnings on record")
		return nil
	}

	settings, _ := db.GetWarnSettings(m.ChatID())

	userInfo, _ := m.Client.GetUser(userID)
	userName := "User"
	if userInfo != nil {
		userName = userInfo.FirstName
	}

	var resp strings.Builder
	resp.WriteString(fmt.Sprintf("Warning Record for %s: %d/%d\n\n", userName, len(warns), settings.MaxWarns))

	for i, warn := range warns {
		adminInfo, _ := m.Client.GetUser(warn.AdminID)
		adminName := "Unknown"
		if adminInfo != nil {
			adminName = adminInfo.FirstName
		}
		resp.WriteString(fmt.Sprintf("%d. %s\n   By %s on %s\n",
			i+1, warn.Reason, adminName, warn.Timestamp.Format("02 Jan 2006")))
	}

	m.Reply(resp.String())
	return nil
}

func ResetWarnsHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warning system is only available in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		m.Reply("You need Ban Users permission to clear warnings")
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
		m.Reply("This user has no warnings to clear")
		return nil
	}

	if err := db.ResetWarns(m.ChatID(), userID); err != nil {
		m.Reply("Failed to clear warnings")
		return nil
	}

	userInfo, _ := m.Client.GetUser(userID)
	userName := "User"
	if userInfo != nil {
		userName = userInfo.FirstName
	}

	m.Reply(fmt.Sprintf("Cleared %d warning(s) for %s", len(warns), userName))
	return nil
}

func RemoveWarnHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warning system is only available in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		m.Reply("You need Ban Users permission to remove warnings")
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
		m.Reply("This user has no warnings to remove")
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

	m.Reply(fmt.Sprintf("Removed last warning for %s. Current: %d/%d", userName, newCount, settings.MaxWarns))
	return nil
}

func SetWarnLimitHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warning settings can only be changed in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to modify warning settings")
		return nil
	}

	args := strings.TrimSpace(m.Args())
	if args == "" {
		settings, _ := db.GetWarnSettings(m.ChatID())
		m.Reply(fmt.Sprintf("Current warning limit: %d\n\nUsage: /setwarnlimit <number>\nRange: 1-20", settings.MaxWarns))
		return nil
	}

	limit, err := strconv.Atoi(args)
	if err != nil || limit < 1 || limit > 20 {
		m.Reply("Warning limit must be between 1 and 20")
		return nil
	}

	settings, _ := db.GetWarnSettings(m.ChatID())
	settings.MaxWarns = limit

	if err := db.SetWarnSettings(m.ChatID(), settings); err != nil {
		m.Reply("Failed to update warning limit")
		return nil
	}

	m.Reply(fmt.Sprintf("Warning limit updated to %d", limit))
	return nil
}

func SetWarnActionHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warning settings can only be changed in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to modify warning settings")
		return nil
	}

	args := strings.ToLower(strings.TrimSpace(m.Args()))
	settings, _ := db.GetWarnSettings(m.ChatID())

	if args == "" {
		m.Reply(fmt.Sprintf(`Warning Enforcement Action

Current setting: %s
Current limit: %d warnings

Usage: /setwarnaction <action>

Available actions:
• ban - Ban user when limit is reached
• mute - Mute user when limit is reached
• kick - Remove user when limit is reached`, settings.Action, settings.MaxWarns))
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
		m.Reply("Unknown action. Options: ban, mute, kick")
		return nil
	}

	settings.Action = action

	if err := db.SetWarnSettings(m.ChatID(), settings); err != nil {
		m.Reply("Failed to update warning action")
		return nil
	}

	m.Reply(fmt.Sprintf("Warning enforcement action set to: %s", action))
	return nil
}

func WarnSettingsHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Warning settings can only be viewed in groups")
		return nil
	}

	settings, _ := db.GetWarnSettings(m.ChatID())

	m.Reply(fmt.Sprintf(`Warning System Configuration

Limit: %d warnings
Action: %s

Use /setwarnlimit to change the limit
Use /setwarnaction to change the action`, settings.MaxWarns, settings.Action))
	return nil
}

func registerWarnsHandlers() {
	c := Client
	c.On("cmd:warn", WarnUserHandler)
	c.On("cmd:warns", ListWarnsHandler)
	c.On("cmd:rmwarn", RemoveWarnHandler)
	c.On("cmd:resetwarns", ResetWarnsHandler)
	c.On("cmd:setwarnlimit", SetWarnLimitHandler)
	c.On("cmd:setwarnaction", SetWarnActionHandler)
	c.On("cmd:setwarnmode", SetWarnActionHandler)
	c.On("cmd:warnsettings", WarnSettingsHandler)
	c.On("cmd:twarn", TemporaryWarnHandler)
	c.On("callback:rmwarn_", RemoveWarnCallback)
	c.On("callback:undo_", UndoActionHandler)
}

func init() {
	QueueHandlerRegistration(registerWarnsHandlers)

	Mods.AddModule("Warns", `<b>Warning System</b>

<b>Issue Warnings:</b>
/warn [user] [reason] - Issue a warning to user (auto-triggers actions when limit reached)
/twarn [user] [duration] [reason] - Temporary warning that auto-expires

<b>View & Manage:</b>
/warns [user] - Check warnings for a user
/rmwarn [user] - Remove last warning from user
/resetwarns [user] - Clear all warnings (requires admin)
/warnsettings - View current warning configuration

<b>Configuration:</b>
/setwarnlimit [count] - Set how many warnings trigger auto-action (default: 3)
/setwarnaction [action] - Set action: ban, mute, or kick
/setwarnmode [days] - Configure auto-decay (warnings expire after N days)

<i>💡 Click undo buttons within 5 minutes to reverse warning actions</i>

Actions: ban, mute, kick`)
}

type ActionHistory struct {
	ActionID  string
	Type      string
	UserID    int64
	ChatID    int64
	AdminID   int64
	Timestamp time.Time
	Data      map[string]interface{}
}

var actionHistories = make(map[int64][]ActionHistory)

func RecordAction(chatID, userID, adminID int64, actionType string, data map[string]interface{}) {
	action := ActionHistory{
		ActionID:  fmt.Sprintf("%d_%d_%s", chatID, time.Now().UnixNano(), actionType),
		Type:      actionType,
		UserID:    userID,
		ChatID:    chatID,
		AdminID:   adminID,
		Timestamp: time.Now(),
		Data:      data,
	}

	actionHistories[chatID] = append(actionHistories[chatID], action)

	if len(actionHistories[chatID]) > 50 {
		actionHistories[chatID] = actionHistories[chatID][1:]
	}
}

func TemporaryWarnHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Temporary warns can only be used in groups")
		return nil
	}

	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "ban") {
		m.Reply("You need Ban Users permission to use temporary warns")
		return nil
	}

	args := strings.Fields(m.Args())
	if len(args) < 2 {
		m.Reply("Usage: /twarn <user> <duration> [reason]\nExample: /twarn @user 7d spam")
		return nil
	}

	user, err := m.Client.ResolveUsername(args[0])
	if err != nil {
		user, _ = m.Client.ResolvePeer(m.SenderID())
	}

	userID := m.Client.GetPeerID(user)

	if IsUserAdmin(m.Client, userID, m.ChatID(), "") {
		m.Reply("Cannot warn administrators")
		return nil
	}

	duration, err := parseAdminDuration(args[1])
	if err != nil {
		m.Reply("Invalid duration. Examples: 1h, 1d, 1w")
		return nil
	}

	reason := "No reason specified"
	if len(args) > 2 {
		reason = strings.Join(args[2:], " ")
	}

	warn := &db.Warn{
		Reason:    reason,
		AdminID:   m.SenderID(),
		Timestamp: time.Now(),
	}

	count, _ := db.AddWarn(m.ChatID(), userID, warn)
	settings, _ := db.GetWarnSettings(m.ChatID())

	userInfo, _ := m.Client.GetUser(userID)
	userName := "User"
	if userInfo != nil {
		userName = userInfo.FirstName
	}

	RecordAction(m.ChatID(), userID, m.SenderID(), "twarn", map[string]interface{}{
		"reason":   reason,
		"duration": duration.String(),
	})

	b := tg.Button
	m.Reply(
		fmt.Sprintf("Warning issued to %s (%d/%d)\nReason: %s\nAutomatic removal in: %s",
			userName, count, settings.MaxWarns, reason, duration.String()),
		&tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				b.Data("Undo Warning", fmt.Sprintf("undo_twarn_%d_%d", userID, m.SenderID())),
			).Build(),
		},
	)

	go func() {
		<-time.After(duration)
		warns, _ := db.GetWarns(m.ChatID(), userID)
		if len(warns) > 0 && warns[len(warns)-1].Reason == reason {
			db.ResetWarns(m.ChatID(), userID)
			for i := 0; i < len(warns)-1; i++ {
				db.AddWarn(m.ChatID(), userID, warns[i])
			}
		}
	}()

	return nil
}

// UndoActionHandler - Undo recent actions within 5 minutes
func UndoActionHandler(c *tg.CallbackQuery) error {
	data := c.DataString()

	if !strings.HasPrefix(data, "undo_") {
		return nil
	}

	parts := strings.Split(strings.TrimPrefix(data, "undo_"), "_")
	if len(parts) < 3 {
		return nil
	}

	actionType := parts[0]
	userID, _ := strconv.ParseInt(parts[1], 10, 64)
	adminID, _ := strconv.ParseInt(parts[2], 10, 64)

	if c.SenderID != adminID && !IsUserAdmin(c.Client, c.SenderID, c.ChatID, "ban") {
		c.Answer("Only the admin who took this action can undo it", &tg.CallbackOptions{Alert: true})
		return nil
	}

	if actionHistories[c.ChatID] == nil {
		c.Answer("No recent actions to undo", &tg.CallbackOptions{Alert: true})
		return nil
	}

	var targetAction *ActionHistory
	for i := len(actionHistories[c.ChatID]) - 1; i >= 0; i-- {
		action := actionHistories[c.ChatID][i]
		if action.Type == actionType &&
			action.UserID == userID &&
			time.Since(action.Timestamp) < 5*time.Minute {
			targetAction = &action
			break
		}
	}

	if targetAction == nil {
		c.Answer("Action not found or expired (5-minute window)", &tg.CallbackOptions{Alert: true})
		return nil
	}

	// Handle different action types
	switch actionType {
	case "twarn", "warn":
		// Remove the warning
		warns, _ := db.GetWarns(c.ChatID, userID)
		if len(warns) > 0 {
			db.ResetWarns(c.ChatID, userID)
			// Re-add all warnings except the last one
			for i := 0; i < len(warns)-1; i++ {
				db.AddWarn(c.ChatID, userID, warns[i])
			}
			c.Answer("Warning removed", &tg.CallbackOptions{Alert: false})
			c.Edit("Warning has been removed")
		}

	case "ban":
		// Unban the user
		user, err := c.Client.ResolvePeer(userID)
		if err == nil && user != nil {
			_, err := c.Client.EditBanned(c.ChatID, user, &tg.BannedOptions{Unban: true})
			if err == nil {
				c.Answer("Ban removed", &tg.CallbackOptions{Alert: false})
				c.Edit("User has been unbanned")
			} else {
				c.Answer("Failed to unban user: "+err.Error(), &tg.CallbackOptions{Alert: true})
			}
		}

	case "tban":
		// Temporarily ban - unban the user
		user, err := c.Client.ResolvePeer(userID)
		if err == nil && user != nil {
			_, err := c.Client.EditBanned(c.ChatID, user, &tg.BannedOptions{Unban: true})
			if err == nil {
				c.Answer("Temporary ban removed", &tg.CallbackOptions{Alert: false})
				c.Edit("Temporary ban has been reversed")
			} else {
				c.Answer("Failed to remove ban: "+err.Error(), &tg.CallbackOptions{Alert: true})
			}
		}

	case "mute":
		// Unmute the user
		user, err := c.Client.ResolvePeer(userID)
		if err == nil && user != nil {
			_, err := c.Client.EditBanned(c.ChatID, user, &tg.BannedOptions{Unmute: true})
			if err == nil {
				c.Answer("Mute removed", &tg.CallbackOptions{Alert: false})
				c.Edit("User has been unmuted")
			} else {
				c.Answer("Failed to unmute user: "+err.Error(), &tg.CallbackOptions{Alert: true})
			}
		}

	case "tmute":
		// Temporarily mute - unmute the user
		user, err := c.Client.ResolvePeer(userID)
		if err == nil && user != nil {
			_, err := c.Client.EditBanned(c.ChatID, user, &tg.BannedOptions{Unmute: true})
			if err == nil {
				c.Answer("Temporary mute removed", &tg.CallbackOptions{Alert: false})
				c.Edit("Temporary mute has been reversed")
			} else {
				c.Answer("Failed to remove mute: "+err.Error(), &tg.CallbackOptions{Alert: true})
			}
		}

	default:
		c.Answer("Unknown action type", &tg.CallbackOptions{Alert: true})
	}

	return nil
}
