package extras

import (
	"encoding/json"
	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
	"net/http"
	"time"
)

// === from cat_dog.go ===
type catApiResponse struct {
	URL string `json:"url"`
}

type dogApiResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

func CatHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://api.thecatapi.com/v1/images/search")
	if err != nil {
		m.Reply("Failed to fetch a cat. Try again later.")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("Cat API returned an error. Try again later.")
		return nil
	}
	var data []catApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse cat response.")
		return nil
	}
	if len(data) == 0 || data[0].URL == "" {
		m.Reply("No cat image found. Try again.")
		return nil
	}
	if _, err := m.ReplyMedia(data[0].URL, &tg.MediaOptions{Caption: "Meow!"}); err != nil {
		m.Reply("<a href=\"" + data[0].URL + "\">Cat</a>")
	}
	return nil
}

func DogHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://dog.ceo/api/breeds/image/random")
	if err != nil {
		m.Reply("Failed to fetch a dog. Try again later.")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("Dog API returned an error. Try again later.")
		return nil
	}
	var data dogApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse dog response.")
		return nil
	}
	if data.Status != "success" || data.Message == "" {
		m.Reply("No dog image found. Try again.")
		return nil
	}
	if _, err := m.ReplyMedia(data.Message, &tg.MediaOptions{Caption: "Woof!"}); err != nil {
		m.Reply("<a href=\"" + data.Message + "\">Dog</a>")
	}
	return nil
}

func initFromSrc_cat_dog_0_1() { modules.QueueHandlerRegistration(registerCatDogHandlers) }
func registerCatDogHandlers() {
	c := modules.Client
	c.On("cmd:cat", CatHandler)
	c.On("cmd:dog", DogHandler)
}
// === from duck_image.go ===
type duckApiResponse struct {
	Message string `json:"message"`
	URL     string `json:"url"`
}

func DuckHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://random-d.uk/api/v2/random")
	if err != nil {
		m.Reply("Failed to fetch a duck. Try again later.")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("Duck API returned an error. Try again later.")
		return nil
	}
	var data duckApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse duck response.")
		return nil
	}
	if data.URL == "" {
		m.Reply("No duck image found. Try again.")
		return nil
	}
	if _, err := m.ReplyMedia(data.URL, &tg.MediaOptions{Caption: "Quack!"}); err != nil {
		m.Reply("<a href=\"" + data.URL + "\">Duck</a>")
	}
	return nil
}

func initFromSrc_duck_image_1_1() { modules.QueueHandlerRegistration(registerDuckHandlers) }
func registerDuckHandlers() {
	c := modules.Client
	c.On("cmd:duck", DuckHandler)
}
// === from fox_image.go ===
type foxApiResponse struct {
	Image string `json:"image"`
	Link  string `json:"link"`
}

func FoxHandler(m *tg.NewMessage) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://randomfox.ca/floof/")
	if err != nil {
		m.Reply("Failed to fetch a fox. Try again later.")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		m.Reply("Fox API returned an error. Try again later.")
		return nil
	}
	var data foxApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("Failed to parse fox response.")
		return nil
	}
	if data.Image == "" {
		m.Reply("No fox image found. Try again.")
		return nil
	}
	if _, err := m.ReplyMedia(data.Image, &tg.MediaOptions{Caption: "What does the fox say?"}); err != nil {
		m.Reply("<a href=\"" + data.Image + "\">Fox</a>")
	}
	return nil
}

func initFromSrc_fox_image_2_1() { modules.QueueHandlerRegistration(registerFoxHandlers) }
func registerFoxHandlers() {
	c := modules.Client
	c.On("cmd:fox", FoxHandler)
}

func init() {
	initFromSrc_cat_dog_0_1()
	initFromSrc_duck_image_1_1()
	initFromSrc_fox_image_2_1()
}
