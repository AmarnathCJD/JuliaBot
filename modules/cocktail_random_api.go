package modules

import (
	tg "github.com/amarnathcjd/gogram/telegram"
)

func CocktailRouletteHandler(m *tg.NewMessage) error {
	drink, err := fetchCocktail("https://www.thecocktaildb.com/api/json/v1/1/random.php")
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch a random cocktail.")
		return nil
	}
	if drink == nil || drink.StrDrink == "" {
		m.Reply("<b>Error:</b> no cocktail returned.")
		return nil
	}
	caption := "🍸 " + formatCocktail(*drink)
	if len(caption) > 1024 {
		caption = caption[:1020] + "..."
	}
	thumb := derefStr(drink.StrDrinkThumb)
	if thumb != "" {
		if _, err := m.ReplyMedia(thumb, &tg.MediaOptions{Caption: caption}); err == nil {
			return nil
		}
	}
	m.Reply(caption)
	return nil
}

func registerCocktailRandomAPIHandlers() {
	c := Client
	c.On("cmd:cocktailroulette", CocktailRouletteHandler)
}

func init() {
	QueueHandlerRegistration(registerCocktailRandomAPIHandlers)
}
