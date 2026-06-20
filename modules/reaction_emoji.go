package modules

import (
	"html"
	"math/rand"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var maxReactPool = []string{
	"❤", "\U0001F44D", "\U0001F44E", "\U0001F525", "\U0001F970",
	"\U0001F44F", "\U0001F60A", "\U0001F914", "\U0001F92F", "\U0001F389",
	"\U0001F60D", "\U0001F921", "\U0001F92E", "\U0001F4A9", "\U0001F64F",
	"\U0001F44C", "\U0001F54A", "\U0001F921", "\U0001F971", "\U0001F60E",
	"\U0001F633", "\U0001F480", "\U0001F914", "⚡", "\U0001F34C",
	"\U0001F3C6", "\U0001F494", "\U0001F92A", "\U0001F61E", "\U0001F44C",
}

func ReactCmdHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a message with <code>/react &lt;emoji&gt;</code>")
		return nil
	}

	emoji := strings.TrimSpace(m.Args())
	if emoji == "" {
		m.Reply("<b>Provide an emoji.</b> Example: <code>/react \U0001F525</code>")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil || reply == nil {
		m.Reply("<b>Could not fetch the replied message.</b>")
		return nil
	}

	if err := reply.React(emoji); err != nil {
		m.Reply("<b>Failed to react:</b> <code>" + html.EscapeString(err.Error()) + "</code>")
		return nil
	}

	return nil
}

func MaxReactHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Usage:</b> reply to a message with <code>/maxreact</code> for hype.")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil || reply == nil {
		m.Reply("<b>Could not fetch the replied message.</b>")
		return nil
	}

	count := rand.Intn(8) + 3
	if count > 10 {
		count = 10
	}
	if count > len(maxReactPool) {
		count = len(maxReactPool)
	}

	idxs := rand.Perm(len(maxReactPool))[:count]
	picks := make([]string, 0, count)
	seen := map[string]struct{}{}
	for _, i := range idxs {
		e := maxReactPool[i]
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		picks = append(picks, e)
	}

	if err := reply.React(picks); err != nil {
		for _, e := range picks {
			if err2 := reply.React(e); err2 == nil {
				return nil
			}
		}
		m.Reply("<b>Failed to react:</b> <code>" + html.EscapeString(err.Error()) + "</code>")
		return nil
	}

	return nil
}

func registerReactionEmojiHandlers() {
	c := Client
	c.On("cmd:react", ReactCmdHandler)
	c.On("cmd:maxreact", MaxReactHandler)
}

func init() {
	QueueHandlerRegistration(registerReactionEmojiHandlers)

	Mods.AddModule("ReactionEmoji", `<b>Reaction Emoji</b>

React to messages with emojis.

<b>Commands:</b>
/react &lt;emoji&gt; - Reply to a message; bot reacts with that emoji.
/maxreact - Reply to a message; bot reacts with multiple random emojis for hype (up to 10).

<b>Notes:</b>
Telegram may restrict some emojis based on chat settings. Errors are caught gracefully.`)
}
