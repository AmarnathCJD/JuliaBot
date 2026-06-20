package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type recipeFullResponse struct {
	Meals []recipeFullMeal `json:"meals"`
}

type recipeFullMeal struct {
	IDMeal       string `json:"idMeal"`
	StrMeal      string `json:"strMeal"`
	StrCategory  string `json:"strCategory"`
	StrArea      string `json:"strArea"`
	StrInstr     string `json:"strInstructions"`
	StrMealThumb string `json:"strMealThumb"`
	StrTags      string `json:"strTags"`
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

func (i recipeFullMeal) pairs() [][2]string {
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

func recipeFullSteps(instr string) []string {
	instr = strings.ReplaceAll(instr, "\r\n", "\n")
	raw := strings.Split(instr, "\n")
	var steps []string
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		steps = append(steps, line)
	}
	return steps
}

func formatRecipeFullCaption(meal recipeFullMeal) string {
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(meal.StrMeal))
	b.WriteString("</b>\n")

	var meta []string
	if c := strings.TrimSpace(meal.StrCategory); c != "" {
		meta = append(meta, html.EscapeString(c))
	}
	if a := strings.TrimSpace(meal.StrArea); a != "" {
		meta = append(meta, html.EscapeString(a))
	}
	if len(meta) > 0 {
		b.WriteString("<i>")
		b.WriteString(strings.Join(meta, " | "))
		b.WriteString("</i>\n")
	}

	if tags := strings.TrimSpace(meal.StrTags); tags != "" {
		parts := strings.Split(tags, ",")
		var tagged []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			tagged = append(tagged, "#"+html.EscapeString(p))
		}
		if len(tagged) > 0 {
			b.WriteString(strings.Join(tagged, " "))
			b.WriteString("\n")
		}
	}

	pairs := meal.pairs()
	if len(pairs) > 0 {
		b.WriteString("\n<b>Ingredients:</b>\n")
		for _, p := range pairs {
			b.WriteString("• ")
			if p[1] != "" {
				b.WriteString("<code>")
				b.WriteString(html.EscapeString(p[1]))
				b.WriteString("</code> ")
			}
			b.WriteString(html.EscapeString(p[0]))
			b.WriteString("\n")
		}
	}

	var links []string
	if y := strings.TrimSpace(meal.StrYoutube); y != "" {
		links = append(links, "<a href=\""+html.EscapeString(y)+"\">Video</a>")
	}
	if s := strings.TrimSpace(meal.StrSource); s != "" {
		links = append(links, "<a href=\""+html.EscapeString(s)+"\">Source</a>")
	}
	if len(links) > 0 {
		b.WriteString("\n")
		b.WriteString(strings.Join(links, " | "))
		b.WriteString("\n")
	}

	return b.String()
}

func formatRecipeFullInstructions(meal recipeFullMeal) string {
	steps := recipeFullSteps(meal.StrInstr)
	if len(steps) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<b>Instructions — ")
	b.WriteString(html.EscapeString(meal.StrMeal))
	b.WriteString("</b>\n")
	for idx, s := range steps {
		b.WriteString("\n<b>")
		b.WriteString(fmt.Sprintf("%d.", idx+1))
		b.WriteString("</b> ")
		b.WriteString(html.EscapeString(s))
		b.WriteString("\n")
	}
	return b.String()
}

func splitRecipeFullChunks(text string, limit int) []string {
	if len(text) <= limit {
		return []string{text}
	}
	var chunks []string
	remaining := text
	for len(remaining) > limit {
		cut := strings.LastIndex(remaining[:limit], "\n")
		if cut <= 0 {
			cut = limit
		}
		chunks = append(chunks, remaining[:cut])
		remaining = strings.TrimLeft(remaining[cut:], "\n")
	}
	if remaining != "" {
		chunks = append(chunks, remaining)
	}
	return chunks
}

func RecipeFullHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://www.themealdb.com/api/json/v1/1/random.php")
	if err != nil {
		m.Reply("couldn't fetch recipe: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data recipeFullResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't fetch recipe: " + html.EscapeString(err.Error()))
		return nil
	}
	if len(data.Meals) == 0 {
		m.Reply("couldn't fetch recipe: empty response")
		return nil
	}
	meal := data.Meals[0]
	caption := formatRecipeFullCaption(meal)
	if len(caption) > 1000 {
		caption = caption[:1000] + "..."
	}
	if strings.TrimSpace(meal.StrMealThumb) != "" {
		if _, err := m.ReplyMedia(meal.StrMealThumb, &tg.MediaOptions{Caption: caption}); err != nil {
			m.Reply(caption, &tg.SendOptions{LinkPreview: false})
		}
	} else {
		m.Reply(caption, &tg.SendOptions{LinkPreview: false})
	}
	instructions := formatRecipeFullInstructions(meal)
	if instructions == "" {
		return nil
	}
	for _, chunk := range splitRecipeFullChunks(instructions, 4000) {
		m.Reply(chunk, &tg.SendOptions{LinkPreview: false})
	}
	return nil
}

func init() { QueueHandlerRegistration(registerRecipeFullHandlers) }
func registerRecipeFullHandlers() {
	c := Client
	c.On("cmd:recipefull", RecipeFullHandler)
}
