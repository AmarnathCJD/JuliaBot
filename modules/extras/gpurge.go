package extras

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

const gpurgeMaxN = 200

func gpurgeChunkedDelete(client *tg.Client, chatID int64, ids []int32) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deleted := 0
	for i := 0; i < len(ids); i += 100 {
		end := i + 100
		if end > len(ids) {
			end = len(ids)
		}
		if _, err := client.DeleteMessages(chatID, ids[i:end]); err != nil {
			return deleted, err
		}
		deleted += end - i
		if end < len(ids) {
			time.Sleep(400 * time.Millisecond)
		}
	}
	return deleted, nil
}

func SilentPurgeHandle(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		modules.ReplyTemp(m, "<b>Permission denied.</b> You need Delete Messages permission.", 5)
		m.Delete()
		return nil
	}
	if !modules.CanBot(m.Client, m.Channel, "delete") {
		modules.ReplyTemp(m, "<b>I need Delete Messages permission.</b>", 5)
		m.Delete()
		return nil
	}
	if !m.IsReply() {
		modules.ReplyTemp(m, "<b>Reply to a message to silently purge from.</b>", 5)
		m.Delete()
		return nil
	}
	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Delete()
		return nil
	}

	startID := int32(reply.ID)
	endID := int32(m.ID)
	if startID > endID {
		m.Delete()
		return nil
	}

	total := int(endID - startID + 1)
	ids := make([]int32, 0, total)
	for i := startID; i <= endID; i++ {
		ids = append(ids, i)
	}

	_, _ = gpurgeChunkedDelete(m.Client, m.ChatID(), ids)
	return nil
}

func UserPurgeHandle(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>This command only works in groups.</b>")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		m.Reply("<b>Permission denied.</b> You need Delete Messages permission.")
		return nil
	}
	if !modules.CanBot(m.Client, m.Channel, "delete") {
		m.Reply("<b>I need Delete Messages permission.</b>")
		return nil
	}
	if !m.IsReply() {
		m.Reply("<b>Reply to the user's message and pass N.</b>\n<code>/upurge &lt;N&gt;</code>")
		return nil
	}

	args := m.ArgsList()
	if len(args) < 1 {
		m.Reply("<b>Usage:</b> <code>/upurge &lt;N&gt;</code>")
		return nil
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n <= 0 {
		m.Reply("<b>Invalid N.</b> Provide a positive integer.")
		return nil
	}
	if n > gpurgeMaxN {
		m.Reply(fmt.Sprintf("<b>N too high.</b> Maximum is %d.", gpurgeMaxN))
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Couldn't read the replied message.</b>")
		return nil
	}
	targetID := reply.SenderID()
	if targetID == 0 {
		m.Reply("<b>Couldn't identify the sender of that message.</b>")
		return nil
	}

	scanDepth := int32(n * 2)
	if scanDepth < 20 {
		scanDepth = 20
	}
	if scanDepth > 400 {
		scanDepth = 400
	}

	history, err := m.Client.GetHistory(m.ChatID(), &tg.HistoryOption{
		Limit:            scanDepth,
		Offset:           int32(m.ID),
		SleepThresholdMs: 50,
	})
	if err != nil {
		m.Reply(modules.AdminFriendlyError(err, "scan history"))
		return nil
	}

	var ids []int32
	for i := range history {
		msg := &history[i]
		if msg.SenderID() == targetID {
			ids = append(ids, int32(msg.ID))
			if len(ids) >= n {
				break
			}
		}
	}

	if len(ids) == 0 {
		m.Reply("<b>No messages from that user found in the recent history.</b>")
		return nil
	}

	deleted, derr := gpurgeChunkedDelete(m.Client, m.ChatID(), ids)
	if derr != nil {
		m.Reply(modules.AdminFriendlyError(derr, "delete messages"))
		return nil
	}

	name := fmt.Sprintf("user <code>%d</code>", targetID)
	if peer, perr := m.Client.ResolvePeer(targetID); perr == nil {
		if dn := strings.TrimSpace(modules.GetPeerDisplayName(m.Client, peer)); dn != "" {
			name = "<b>" + dn + "</b>"
		}
	}

	status, _ := m.Reply(fmt.Sprintf("Deleted <b>%d</b> message(s) from %s.", deleted, name))
	if status != nil {
		go func(chatID int64, statusID int32) {
			time.Sleep(5 * time.Second)
			m.Client.DeleteMessages(chatID, []int32{statusID})
		}(m.ChatID(), status.ID)
	}
	return nil
}

func MyPurgeHandle(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>This command only works in groups.</b>")
		return nil
	}

	args := m.ArgsList()
	if len(args) < 1 {
		m.Reply("<b>Usage:</b> <code>/mypurge &lt;N&gt;</code>")
		return nil
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n <= 0 {
		m.Reply("<b>Invalid N.</b> Provide a positive integer.")
		return nil
	}
	if n > gpurgeMaxN {
		m.Reply(fmt.Sprintf("<b>N too high.</b> Maximum is %d.", gpurgeMaxN))
		return nil
	}

	targetID := m.SenderID()
	if targetID == 0 {
		return nil
	}

	scanDepth := int32(n * 2)
	if scanDepth < 20 {
		scanDepth = 20
	}
	if scanDepth > 400 {
		scanDepth = 400
	}

	history, err := m.Client.GetHistory(m.ChatID(), &tg.HistoryOption{
		Limit:            scanDepth,
		Offset:           int32(m.ID),
		SleepThresholdMs: 50,
	})
	if err != nil {
		m.Reply(modules.AdminFriendlyError(err, "scan history"))
		return nil
	}

	var ids []int32
	for i := range history {
		msg := &history[i]
		if msg.SenderID() == targetID {
			ids = append(ids, int32(msg.ID))
			if len(ids) >= n {
				break
			}
		}
	}

	if len(ids) == 0 {
		modules.ReplyTemp(m, "<b>No messages of yours found in the recent history.</b>", 5)
		m.Delete()
		return nil
	}

	ids = append(ids, int32(m.ID))
	_, _ = gpurgeChunkedDelete(m.Client, m.ChatID(), ids)
	return nil
}

func registerGPurgeHandlers() {
	c := modules.Client
	c.On("cmd:spurge", SilentPurgeHandle)
	c.On("cmd:upurge", UserPurgeHandle)
	c.On("cmd:mypurge", MyPurgeHandle)
}

func init() {
	modules.QueueHandlerRegistration(registerGPurgeHandlers)

	modules.Mods.AddModule("GPurge", `<b>Group Purge (Recovery)</b>

Targeted purge tools that go beyond the basic /purge range.

<b>Commands:</b>
 - <code>/spurge</code> - Silent purge. Reply to a message; deletes everything from that message through the command. Command and reply itself are wiped, no status spam.
 - <code>/upurge &lt;N&gt;</code> - Admin only. Reply to a user; deletes their last N messages in the chat (scans up to 2N deep, capped at 400).
 - <code>/mypurge &lt;N&gt;</code> - Any user. Deletes your own last N messages (scans up to 2N deep, capped at 400).

<b>Limits:</b>
 - Maximum N: 200.
 - Bot needs Delete Messages permission for admin commands.
 - For /mypurge, the bot still needs Delete Messages permission to remove messages it didn't author.

<b>Permissions:</b>
 - /spurge and /upurge require the invoker to have Delete Messages admin right.
 - /mypurge is open to any group member.`)
}
