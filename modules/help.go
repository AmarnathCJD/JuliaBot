package modules

import (
	"fmt"
	"sort"
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
		if strings.EqualFold(v.Name, name) {
			return v.Help
		}
	}
	return ""
}

func (m *Modules) Init(c *telegram.Client) {
	for _, v := range m.Mod {
		modName := v.Name
		modHelp := v.Help
		c.On("callback:help_"+strings.ToLower(modName), func(c *telegram.CallbackQuery) error {
			return HelpModuleCallback(modName, modHelp)(c)
		})
	}
}

var Mods = Modules{}

func HelpHandle(m *telegram.NewMessage) error {
	b := telegram.Button

	if !m.IsPrivate() {
		m.Reply("Use /help in private chat for detailed help.",
			&telegram.SendOptions{
				ReplyMarkup: b.Keyboard(b.Row(b.URL("Open Private Chat", "t.me/"+m.Client.Me().Username+"?start=help"))),
			})
		return nil
	}

	// Sort modules alphabetically
	sortedMods := make([]Mod, len(Mods.Mod))
	copy(sortedMods, Mods.Mod)
	sort.Slice(sortedMods, func(i, j int) bool {
		return sortedMods[i].Name < sortedMods[j].Name
	})

	var buttons []telegram.KeyboardButton
	for _, v := range sortedMods {
		buttons = append(buttons, b.Data(v.Name, "help_"+strings.ToLower(v.Name)))
	}

	helpText := `<b>Julia Bot</b>
<i>A feature-rich Telegram bot built with gogram</i>

Select a module below to view its commands and usage.

<b>Available Modules:</b> ` + fmt.Sprintf("%d", len(Mods.Mod))

	m.Reply(helpText,
		&telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().NewColumn(3, buttons...).AddRow(
				b.URL("Source Code", "https://github.com/amarnathcjd/gogram"),
			).Build(),
		})

	return nil
}

func HelpModuleCallback(name, help string) func(*telegram.CallbackQuery) error {
	return func(c *telegram.CallbackQuery) error {
		c.Answer("Loading " + name + "...")

		b := telegram.Button
		helpWithBack := help + "\n\n<i>Use /help to see all modules</i>"

		c.Edit(helpWithBack, &telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().AddRow(
				b.Data("Back to Menu", "help_back"),
			).Build(),
		})
		return nil
	}
}

func HelpBackCallback(c *telegram.CallbackQuery) error {
	b := telegram.Button

	sortedMods := make([]Mod, len(Mods.Mod))
	copy(sortedMods, Mods.Mod)
	sort.Slice(sortedMods, func(i, j int) bool {
		return sortedMods[i].Name < sortedMods[j].Name
	})

	var buttons []telegram.KeyboardButton
	for _, v := range sortedMods {
		buttons = append(buttons, b.Data(v.Name, "help_"+strings.ToLower(v.Name)))
	}

	helpText := `<b>Julia Bot</b>
<i>A feature-rich Telegram bot built with gogram</i>

Select a module below to view its commands and usage.

<b>Available Modules:</b> ` + fmt.Sprintf("%d", len(Mods.Mod))

	c.Edit(helpText, &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().NewColumn(3, buttons...).AddRow(
			b.URL("Source Code", "https://github.com/amarnathcjd/gogram"),
		).Build(),
	})

	return nil
}
