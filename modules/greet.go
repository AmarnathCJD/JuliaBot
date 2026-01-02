package modules

import (
	"fmt"

	"github.com/amarnathcjd/gogram/telegram"
)

var GREET_ENABLED = false

const GREET_MESSAGE = "Hey <b>%s</b>, Welcome to the group!"

func UserJoinHandle(m *telegram.ParticipantUpdate) error {
	// if m.IsBanned() {
	// 	m.Client.SendMessage(m.ChannelID(), "User Boom Banned.")
	// }
	if (m.IsJoined() || m.IsAdded()) && GREET_ENABLED {
		m.Client.SendMessage(m.ChannelID(), fmt.Sprintf(GREET_MESSAGE, m.User.FirstName))
	}

	return nil
}

func ModifyGreetStatus(m *telegram.NewMessage) error {
	if m.Args() == "enable" || m.Args() == "on" {
		GREET_ENABLED = true
		m.Reply("New users will be greeted!")
	} else if m.Args() == "disable" {
		GREET_ENABLED = false
		m.Reply("New users will not be greeted!")
	} else {
		m.Reply("Invalid argument. Use 'enable' or 'disable'")
	}

	return nil
}

func init() {
	Mods.AddModule("Greet", `<b>Here are the commands available in Greet module:</b>
The Greet module is used to greet new users when they join the group.

<code>/greet enable/disable</code> - Enable or disable greeting new users`)
}
