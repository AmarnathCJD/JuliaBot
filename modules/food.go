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

type foodishResponse struct {
	Image string `json:"image"`
}

type mealDBResponse struct {
	Meals []mealDBItem `json:"meals"`
}

type mealDBItem struct {
	IDMeal       string `json:"idMeal"`
	StrMeal      string `json:"strMeal"`
	StrCategory  string `json:"strCategory"`
	StrArea      string `json:"strArea"`
	StrInstr     string `json:"strInstructions"`
	StrMealThumb string `json:"strMealThumb"`
	StrSource    string `json:"strSource"`
	StrYoutube   string `json:"strYoutube"`
	StrIng1      string `json:"strIngredient1"`
	StrIng2      string `json:"strIngredient2"`
	StrIng3      string `json:"strIngredient3"`
	StrIng4      string `json:"strIngredient4"`
	StrIng5      string `json:"strIngredient5"`
	StrIng6      string `json:"strIngredient6"`
	StrIng7      string `json:"strIngredient7"`
	StrIng8      string `json:"strIngredient8"`
	StrIng9      string `json:"strIngredient9"`
	StrIng10     string `json:"strIngredient10"`
	StrIng11     string `json:"strIngredient11"`
	StrIng12     string `json:"strIngredient12"`
	StrIng13     string `json:"strIngredient13"`
	StrIng14     string `json:"strIngredient14"`
	StrIng15     string `json:"strIngredient15"`
	StrIng16     string `json:"strIngredient16"`
	StrIng17     string `json:"strIngredient17"`
	StrIng18     string `json:"strIngredient18"`
	StrIng19     string `json:"strIngredient19"`
	StrIng20     string `json:"strIngredient20"`
	StrMeas1     string `json:"strMeasure1"`
	StrMeas2     string `json:"strMeasure2"`
	StrMeas3     string `json:"strMeasure3"`
	StrMeas4     string `json:"strMeasure4"`
	StrMeas5     string `json:"strMeasure5"`
	StrMeas6     string `json:"strMeasure6"`
	StrMeas7     string `json:"strMeasure7"`
	StrMeas8     string `json:"strMeasure8"`
	StrMeas9     string `json:"strMeasure9"`
	StrMeas10    string `json:"strMeasure10"`
	StrMeas11    string `json:"strMeasure11"`
	StrMeas12    string `json:"strMeasure12"`
	StrMeas13    string `json:"strMeasure13"`
	StrMeas14    string `json:"strMeasure14"`
	StrMeas15    string `json:"strMeasure15"`
	StrMeas16    string `json:"strMeasure16"`
	StrMeas17    string `json:"strMeasure17"`
	StrMeas18    string `json:"strMeasure18"`
	StrMeas19    string `json:"strMeasure19"`
	StrMeas20    string `json:"strMeasure20"`
}

func (i mealDBItem) ingredients() [][2]string {
	ings := []string{i.StrIng1, i.StrIng2, i.StrIng3, i.StrIng4, i.StrIng5, i.StrIng6, i.StrIng7, i.StrIng8, i.StrIng9, i.StrIng10, i.StrIng11, i.StrIng12, i.StrIng13, i.StrIng14, i.StrIng15, i.StrIng16, i.StrIng17, i.StrIng18, i.StrIng19, i.StrIng20}
	meas := []string{i.StrMeas1, i.StrMeas2, i.StrMeas3, i.StrMeas4, i.StrMeas5, i.StrMeas6, i.StrMeas7, i.StrMeas8, i.StrMeas9, i.StrMeas10, i.StrMeas11, i.StrMeas12, i.StrMeas13, i.StrMeas14, i.StrMeas15, i.StrMeas16, i.StrMeas17, i.StrMeas18, i.StrMeas19, i.StrMeas20}
	var out [][2]string
	for idx, ing := range ings {
		ing = strings.TrimSpace(ing)
		if ing == "" {
			continue
		}
		m := ""
		if idx < len(meas) {
			m = strings.TrimSpace(meas[idx])
		}
		out = append(out, [2]string{ing, m})
	}
	return out
}

func FoodPornHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://foodish-api.com/api/")
	if err != nil {
		m.Reply("couldn't fetch food: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data foodishResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't fetch food: " + html.EscapeString(err.Error()))
		return nil
	}
	if data.Image == "" {
		m.Reply("couldn't fetch food: no image returned")
		return nil
	}
	if _, err := m.ReplyMedia(data.Image, &tg.MediaOptions{}); err != nil {
		m.Reply("couldn't fetch food: " + html.EscapeString(err.Error()))
		return nil
	}
	return nil
}

func formatMeal(meal mealDBItem) string {
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(meal.StrMeal))
	b.WriteString("</b>")
	meta := []string{}
	if strings.TrimSpace(meal.StrCategory) != "" {
		meta = append(meta, html.EscapeString(meal.StrCategory))
	}
	if strings.TrimSpace(meal.StrArea) != "" {
		meta = append(meta, html.EscapeString(meal.StrArea))
	}
	if len(meta) > 0 {
		b.WriteString("\n<i>")
		b.WriteString(strings.Join(meta, " | "))
		b.WriteString("</i>")
	}
	ings := meal.ingredients()
	if len(ings) > 0 {
		b.WriteString("\n\n<b>Ingredients:</b>")
		for _, pair := range ings {
			b.WriteString("\n- ")
			b.WriteString(html.EscapeString(pair[0]))
			if pair[1] != "" {
				b.WriteString(" (")
				b.WriteString(html.EscapeString(pair[1]))
				b.WriteString(")")
			}
		}
	}
	instr := strings.TrimSpace(meal.StrInstr)
	if instr != "" {
		b.WriteString("\n\n<b>Instructions:</b>\n")
		if len(instr) > 2500 {
			instr = instr[:2500] + "..."
		}
		b.WriteString(html.EscapeString(instr))
	}
	if strings.TrimSpace(meal.StrYoutube) != "" {
		b.WriteString("\n\n<a href=\"")
		b.WriteString(html.EscapeString(meal.StrYoutube))
		b.WriteString("\">Video</a>")
	}
	if strings.TrimSpace(meal.StrSource) != "" {
		b.WriteString(" | <a href=\"")
		b.WriteString(html.EscapeString(meal.StrSource))
		b.WriteString("\">Source</a>")
	}
	return b.String()
}

func RecipeHandler(m *tg.NewMessage) error {
	query := strings.TrimSpace(m.Args())
	if query == "" {
		m.Reply("<b>Usage:</b> <code>/recipe &lt;query&gt;</code>")
		return nil
	}
	status, _ := m.Reply("Searching <code>" + html.EscapeString(query) + "</code>...")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://www.themealdb.com/api/json/v1/1/search.php?s=" + url.QueryEscape(query))
	if err != nil {
		status.Edit("couldn't fetch recipe: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		status.Edit(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data mealDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		status.Edit("couldn't fetch recipe: " + html.EscapeString(err.Error()))
		return nil
	}
	if len(data.Meals) == 0 {
		status.Edit("<b>No recipes found for:</b> <code>" + html.EscapeString(query) + "</code>")
		return nil
	}
	meal := data.Meals[0]
	caption := formatMeal(meal)
	if strings.TrimSpace(meal.StrMealThumb) != "" {
		if _, err := m.ReplyMedia(meal.StrMealThumb, &tg.MediaOptions{Caption: caption}); err != nil {
			status.Edit(caption, &tg.SendOptions{LinkPreview: false})
			return nil
		}
		status.Delete()
		return nil
	}
	status.Edit(caption, &tg.SendOptions{LinkPreview: false})
	return nil
}

func RandomFoodHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://www.themealdb.com/api/json/v1/1/random.php")
	if err != nil {
		m.Reply("couldn't fetch meal: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data mealDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't fetch meal: " + html.EscapeString(err.Error()))
		return nil
	}
	if len(data.Meals) == 0 {
		m.Reply("couldn't fetch meal: empty response")
		return nil
	}
	meal := data.Meals[0]
	caption := formatMeal(meal)
	if strings.TrimSpace(meal.StrMealThumb) != "" {
		if _, err := m.ReplyMedia(meal.StrMealThumb, &tg.MediaOptions{Caption: caption}); err != nil {
			m.Reply(caption, &tg.SendOptions{LinkPreview: false})
			return nil
		}
		return nil
	}
	m.Reply(caption, &tg.SendOptions{LinkPreview: false})
	return nil
}

func init() { QueueHandlerRegistration(registerFoodHandlers) }
func registerFoodHandlers() {
	c := Client
	c.On("cmd:foodporn", FoodPornHandler)
	c.On("cmd:recipe", RecipeHandler)
	c.On("cmd:randomfood", RandomFoodHandler)
}
