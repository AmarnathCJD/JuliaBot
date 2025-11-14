package modules

import (
	"fmt"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

type Modules struct {
	Mod []Mod
}

type Mod struct {
	Name string
	Help string
}

func (m *Modules) AddModule(name, help string) {
	m.Mod = append(m.Mod, Mod{name, help})
}

func (m *Modules) GetHelp(name string) string {
	for _, v := range m.Mod {
		if v.Name == name {
			return v.Help
		}
	}
	return ""
}

func (m *Modules) Init(c *telegram.Client) {
	for _, v := range m.Mod {
		c.On("callback:help_"+v.Name, func(c *telegram.CallbackQuery) error {
			return HelpModule(v.Name, v.Help)(c)
		})
	}
}

var Mods = Modules{}

func HelpHandle(m *telegram.NewMessage) error {
	var b = telegram.Button

	if !m.IsPrivate() {
		m.Reply("DM me for help!",
			&telegram.SendOptions{
				ReplyMarkup: b.Keyboard(b.Row(b.URL("Click Here", "t.me/"+m.Client.Me().Username+"?start=help"))),
			})
		return nil
	}

	var buttons []telegram.KeyboardButton
	for _, v := range Mods.Mod {
		buttons = append(buttons, b.Data(v.Name+" "+getRandomEmoticon(), strings.ToLower("help_"+v.Name)))
	}

	m.Reply("Hello! I'm <b>Julia</b> created by <b>@amarnathcjd</b>. To demonstrate the capabilities of <b><a href='github.com/amarnathcjd/gogram'>gogram</a></b> library. Here are the available commands:\n\n",
		&telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().NewColumn(2, buttons...).Build(),
		})

	return nil
}

func HelpModule(name, help string) func(*telegram.CallbackQuery) error {
	fmt.Println("HelpModule: ", name)
	return func(c *telegram.CallbackQuery) error {
		c.Answer("Loading...")
		c.Edit(help)
		return nil
	}
}
