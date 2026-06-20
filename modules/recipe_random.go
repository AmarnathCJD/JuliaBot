package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type cocktailDrink struct {
	IDDrink          string  `json:"idDrink"`
	StrDrink         string  `json:"strDrink"`
	StrCategory      *string `json:"strCategory"`
	StrAlcoholic     *string `json:"strAlcoholic"`
	StrGlass         *string `json:"strGlass"`
	StrInstructions  *string `json:"strInstructions"`
	StrDrinkThumb    *string `json:"strDrinkThumb"`
	StrIngredient1   *string `json:"strIngredient1"`
	StrIngredient2   *string `json:"strIngredient2"`
	StrIngredient3   *string `json:"strIngredient3"`
	StrIngredient4   *string `json:"strIngredient4"`
	StrIngredient5   *string `json:"strIngredient5"`
	StrIngredient6   *string `json:"strIngredient6"`
	StrIngredient7   *string `json:"strIngredient7"`
	StrIngredient8   *string `json:"strIngredient8"`
	StrIngredient9   *string `json:"strIngredient9"`
	StrIngredient10  *string `json:"strIngredient10"`
	StrIngredient11  *string `json:"strIngredient11"`
	StrIngredient12  *string `json:"strIngredient12"`
	StrIngredient13  *string `json:"strIngredient13"`
	StrIngredient14  *string `json:"strIngredient14"`
	StrIngredient15  *string `json:"strIngredient15"`
	StrMeasure1      *string `json:"strMeasure1"`
	StrMeasure2      *string `json:"strMeasure2"`
	StrMeasure3      *string `json:"strMeasure3"`
	StrMeasure4      *string `json:"strMeasure4"`
	StrMeasure5      *string `json:"strMeasure5"`
	StrMeasure6      *string `json:"strMeasure6"`
	StrMeasure7      *string `json:"strMeasure7"`
	StrMeasure8      *string `json:"strMeasure8"`
	StrMeasure9      *string `json:"strMeasure9"`
	StrMeasure10     *string `json:"strMeasure10"`
	StrMeasure11     *string `json:"strMeasure11"`
	StrMeasure12     *string `json:"strMeasure12"`
	StrMeasure13     *string `json:"strMeasure13"`
	StrMeasure14     *string `json:"strMeasure14"`
	StrMeasure15     *string `json:"strMeasure15"`
}

type cocktailResponse struct {
	Drinks []cocktailDrink `json:"drinks"`
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func collectCocktailIngredients(d cocktailDrink) []string {
	ings := []*string{d.StrIngredient1, d.StrIngredient2, d.StrIngredient3, d.StrIngredient4, d.StrIngredient5, d.StrIngredient6, d.StrIngredient7, d.StrIngredient8, d.StrIngredient9, d.StrIngredient10, d.StrIngredient11, d.StrIngredient12, d.StrIngredient13, d.StrIngredient14, d.StrIngredient15}
	meas := []*string{d.StrMeasure1, d.StrMeasure2, d.StrMeasure3, d.StrMeasure4, d.StrMeasure5, d.StrMeasure6, d.StrMeasure7, d.StrMeasure8, d.StrMeasure9, d.StrMeasure10, d.StrMeasure11, d.StrMeasure12, d.StrMeasure13, d.StrMeasure14, d.StrMeasure15}
	var out []string
	for i := 0; i < len(ings); i++ {
		ing := derefStr(ings[i])
		if ing == "" {
			continue
		}
		m := derefStr(meas[i])
		if m != "" {
			out = append(out, fmt.Sprintf("• %s %s", html.EscapeString(m), html.EscapeString(ing)))
		} else {
			out = append(out, fmt.Sprintf("• %s", html.EscapeString(ing)))
		}
	}
	return out
}

func formatCocktail(d cocktailDrink) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<b>%s</b>\n", html.EscapeString(d.StrDrink)))
	var meta []string
	if c := derefStr(d.StrCategory); c != "" {
		meta = append(meta, html.EscapeString(c))
	}
	if a := derefStr(d.StrAlcoholic); a != "" {
		meta = append(meta, html.EscapeString(a))
	}
	if g := derefStr(d.StrGlass); g != "" {
		meta = append(meta, html.EscapeString(g))
	}
	if len(meta) > 0 {
		b.WriteString("<i>" + strings.Join(meta, " • ") + "</i>\n")
	}
	ings := collectCocktailIngredients(d)
	if len(ings) > 0 {
		b.WriteString("\n<b>Ingredients:</b>\n")
		b.WriteString(strings.Join(ings, "\n"))
		b.WriteString("\n")
	}
	if instr := derefStr(d.StrInstructions); instr != "" {
		b.WriteString("\n<b>Instructions:</b>\n")
		b.WriteString(html.EscapeString(instr))
	}
	return b.String()
}

func fetchCocktail(endpoint string) (*cocktailDrink, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var data cocktailResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Drinks) == 0 {
		return nil, nil
	}
	return &data.Drinks[0], nil
}

func CocktailHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("usage: <code>/cocktail &lt;name&gt;</code>")
		return nil
	}
	endpoint := "https://thecocktaildb.com/api/json/v1/1/search.php?s=" + url.QueryEscape(query)
	drink, err := fetchCocktail(endpoint)
	if err != nil {
		m.Reply("couldn't fetch cocktail: " + err.Error())
		return nil
	}
	if drink == nil {
		m.Reply("no cocktail found for: <code>" + html.EscapeString(query) + "</code>")
		return nil
	}
	caption := formatCocktail(*drink)
	thumb := derefStr(drink.StrDrinkThumb)
	if thumb != "" {
		if _, err := m.ReplyMedia(thumb, &tg.MediaOptions{Caption: caption}); err != nil {
			m.Reply(caption)
		}
		return nil
	}
	m.Reply(caption)
	return nil
}

func RandomCocktailHandler(m *tg.NewMessage) error {
	drink, err := fetchCocktail("https://thecocktaildb.com/api/json/v1/1/random.php")
	if err != nil {
		m.Reply("couldn't fetch cocktail: " + err.Error())
		return nil
	}
	if drink == nil {
		m.Reply("no cocktail returned")
		return nil
	}
	caption := formatCocktail(*drink)
	thumb := derefStr(drink.StrDrinkThumb)
	if thumb != "" {
		if _, err := m.ReplyMedia(thumb, &tg.MediaOptions{Caption: caption}); err != nil {
			m.Reply(caption)
		}
		return nil
	}
	m.Reply(caption)
	return nil
}

func init() { QueueHandlerRegistration(registerRecipeRandomHandlers) }
func registerRecipeRandomHandlers() {
	c := Client
	c.On("cmd:cocktail", CocktailHandler)
	c.On("cmd:randomcocktail", RandomCocktailHandler)
}
