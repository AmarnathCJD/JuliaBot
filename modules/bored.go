package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"math/rand"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type boredActivityResponse struct {
	Activity      string  `json:"activity"`
	Type          string  `json:"type"`
	Participants  int     `json:"participants"`
	Price         float64 `json:"price"`
	Link          string  `json:"link"`
	Key           string  `json:"key"`
	Accessibility float64 `json:"accessibility"`
	Error         string  `json:"error"`
}

var boredFallbackList = []boredActivityResponse{
	{Activity: "Learn a new programming language", Type: "education", Participants: 1, Price: 0, Accessibility: 0.1},
	{Activity: "Go for a walk in your neighborhood", Type: "relaxation", Participants: 1, Price: 0, Accessibility: 0.0},
	{Activity: "Cook a new recipe you have never tried before", Type: "cooking", Participants: 1, Price: 0.2, Accessibility: 0.15},
	{Activity: "Write a short story about a random object near you", Type: "relaxation", Participants: 1, Price: 0, Accessibility: 0.0},
	{Activity: "Call a family member you have not spoken to in a while", Type: "social", Participants: 2, Price: 0, Accessibility: 0.05},
	{Activity: "Organize your closet and donate clothes you no longer wear", Type: "busywork", Participants: 1, Price: 0, Accessibility: 0.1},
	{Activity: "Start a journal and write today's entry", Type: "relaxation", Participants: 1, Price: 0.05, Accessibility: 0.1},
	{Activity: "Plan a future vacation, even if you cannot go yet", Type: "relaxation", Participants: 1, Price: 0, Accessibility: 0.1},
	{Activity: "Learn to juggle with three balls or socks", Type: "recreational", Participants: 1, Price: 0, Accessibility: 0.2},
	{Activity: "Watch a classic film you have never seen", Type: "relaxation", Participants: 1, Price: 0.1, Accessibility: 0.05},
	{Activity: "Do a 20 minute yoga or stretching session", Type: "relaxation", Participants: 1, Price: 0, Accessibility: 0.15},
	{Activity: "Make a playlist of your favorite songs from a specific year", Type: "music", Participants: 1, Price: 0, Accessibility: 0.05},
	{Activity: "Build a paper airplane and test different designs", Type: "diy", Participants: 1, Price: 0, Accessibility: 0.05},
	{Activity: "Try meditating for ten minutes", Type: "relaxation", Participants: 1, Price: 0, Accessibility: 0.1},
	{Activity: "Read a chapter of a book you have been putting off", Type: "education", Participants: 1, Price: 0, Accessibility: 0.05},
	{Activity: "Draw a self portrait without looking at the paper", Type: "diy", Participants: 1, Price: 0.05, Accessibility: 0.1},
	{Activity: "Plant a seed in a small pot and care for it", Type: "diy", Participants: 1, Price: 0.15, Accessibility: 0.2},
	{Activity: "Solve a crossword or sudoku puzzle", Type: "recreational", Participants: 1, Price: 0.05, Accessibility: 0.1},
	{Activity: "Take a series of photos around your home", Type: "diy", Participants: 1, Price: 0, Accessibility: 0.05},
	{Activity: "Try writing a poem in a style you do not usually use", Type: "relaxation", Participants: 1, Price: 0, Accessibility: 0.1},
	{Activity: "Reorganize your desktop and clean up old files", Type: "busywork", Participants: 1, Price: 0, Accessibility: 0.05},
	{Activity: "Have a board game night with friends or family", Type: "social", Participants: 3, Price: 0.1, Accessibility: 0.3},
	{Activity: "Bake cookies and share them with neighbors", Type: "cooking", Participants: 1, Price: 0.2, Accessibility: 0.25},
	{Activity: "Volunteer for a local cause for a few hours", Type: "charity", Participants: 1, Price: 0, Accessibility: 0.4},
	{Activity: "Learn a magic trick and perform it for someone", Type: "social", Participants: 2, Price: 0, Accessibility: 0.2},
	{Activity: "Make a list of 100 things that make you happy", Type: "relaxation", Participants: 1, Price: 0, Accessibility: 0.0},
	{Activity: "Take an online class on a topic that interests you", Type: "education", Participants: 1, Price: 0.1, Accessibility: 0.1},
	{Activity: "Rearrange the furniture in one room of your home", Type: "diy", Participants: 1, Price: 0, Accessibility: 0.15},
	{Activity: "Learn the basics of origami and fold a crane", Type: "diy", Participants: 1, Price: 0.05, Accessibility: 0.15},
	{Activity: "Write a thank you note to someone who helped you", Type: "social", Participants: 2, Price: 0.05, Accessibility: 0.1},
}

func boredPriceLabel(p float64) string {
	switch {
	case p <= 0:
		return "Free"
	case p < 0.2:
		return "Low"
	case p < 0.5:
		return "Medium"
	default:
		return "High"
	}
}

func boredAccessibilityLabel(a float64) string {
	switch {
	case a <= 0.15:
		return "Very Easy"
	case a <= 0.35:
		return "Easy"
	case a <= 0.6:
		return "Moderate"
	case a <= 0.8:
		return "Hard"
	default:
		return "Very Hard"
	}
}

func formatBoredActivity(data boredActivityResponse, source string) string {
	out := "<b>Bored? Try This</b>\n\n"
	out += "<b>Activity:</b> " + html.EscapeString(data.Activity) + "\n"
	if data.Type != "" {
		out += "<b>Type:</b> " + html.EscapeString(data.Type) + "\n"
	}
	if data.Participants > 0 {
		out += fmt.Sprintf("<b>Participants:</b> %d\n", data.Participants)
	}
	out += fmt.Sprintf("<b>Accessibility:</b> %s (%.2f)\n", boredAccessibilityLabel(data.Accessibility), data.Accessibility)
	out += fmt.Sprintf("<b>Price:</b> %s (%.2f)\n", boredPriceLabel(data.Price), data.Price)
	if data.Link != "" {
		out += "\n<a href=\"" + html.EscapeString(data.Link) + "\">More info</a>\n"
	}
	out += "\n<i>Source: " + source + "</i>"
	return out
}

func boredFallback(m *tg.NewMessage) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pick := boredFallbackList[r.Intn(len(boredFallbackList))]
	m.Reply(formatBoredActivity(pick, "offline list"))
}

func BoredHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("http://www.boredapi.com/api/activity")
	if err != nil {
		boredFallback(m)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		boredFallback(m)
		return nil
	}
	var data boredActivityResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		boredFallback(m)
		return nil
	}
	if data.Activity == "" || data.Error != "" {
		boredFallback(m)
		return nil
	}
	m.Reply(formatBoredActivity(data, "boredapi.com"))
	return nil
}

func init() { QueueHandlerRegistration(registerBoredHandlers) }
func registerBoredHandlers() {
	c := Client
	c.On("cmd:bored", BoredHandler)
}
