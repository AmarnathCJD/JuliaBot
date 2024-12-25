package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
)

func DogeSticker(m *telegram.NewMessage) error {
	Args := m.Args()

	im, err := gg.LoadImage("./assets/IMG_20220227_202434_649_cleanup.jpg")
	if err != nil {
		m.Reply("failed to load base image")
		return err
	}

	const width, height = 461, 512
	dc := gg.NewContext(width, height)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	fontPath := "./assets/Inter_28pt-Medium.ttf"
	if err := dc.LoadFontFace(fontPath, 85); err != nil {
		m.Reply("failed to load font")
		return err
	}

	fontSize := FitTextToBox(dc, Args, width-20, height/3)
	if err := dc.LoadFontFace(fontPath, fontSize); err != nil {
		m.Reply("failed to load font")
		return err
	}

	lines := WrapText(dc, Args, width-20)
	dc.DrawImage(im, 0, 0)

	dc.SetRGB(0, 0, 0)
	y := float64(height / 3 * 2 / 3)
	lineHeight := fontSize * 1.2
	for _, line := range lines {
		dc.DrawStringAnchored(line, width/2, y, 0.5, 0.5)
		y += lineHeight
	}

	dc.SavePNG("out.webp")
	m.RespondMedia("out.webp")
	return nil
}

func FitTextToBox(dc *gg.Context, text string, maxWidth, maxHeight int) float64 {
	fontSize := 85.0
	for fontSize > 10 {
		dc.LoadFontFace("./assets/Inter_28pt-Medium.ttf", fontSize)
		wrapped := WrapText(dc, text, maxWidth)
		totalHeight := float64(len(wrapped)) * fontSize * 1.2
		if totalHeight <= float64(maxHeight) {
			return fontSize
		}
		fontSize -= 2
	}
	return 10.0
}

func WrapText(dc *gg.Context, text string, maxWidth int) []string {
	words := dc.WordWrap(text, float64(maxWidth))
	return words
}

func PinterestInlineHandle(i *telegram.InlineQuery) error {
	b := i.Builder()
	button := telegram.Button{}
	if i.Args() == "" {
		b.Article("No query", "Please enter a query to search for", "No query", &telegram.ArticleOptions{
			ReplyMarkup: button.Keyboard(
				button.Row(
					button.SwitchInline("Search!!!", true, "pin "),
				),
			),
		})

		i.Answer(b.Results())
		return nil
	}

	offset := 0
	if i.Offset != "" {
		offset, _ = strconv.Atoi(i.Offset)
	}

	images, err := fetchPinterestImages(i.Query, 5, offset)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		b.Article("No images found", "No images found for the query", "No images found", &telegram.ArticleOptions{
			ReplyMarkup: button.Keyboard(
				button.Row(
					button.SwitchInline("Search again", true, "pin "),
				),
			),
		})

		i.Answer(b.Results())
	} else {
		for im := range images {
			b.Photo(images[im], &telegram.ArticleOptions{
				ID:    fmt.Sprintf("%d", im),
				Title: fmt.Sprintf("pinterest-image-%d", im+1),
				ReplyMarkup: button.Keyboard(
					button.Row(
						button.SwitchInline("Search again", true, "pin "),
					),
				),
			})
		}

		i.Answer(b.Results(), telegram.InlineSendOptions{
			Gallery:    true,
			NextOffset: strconv.Itoa(offset + 1),
		})
	}

	return nil
}

func fetchPinterestImages(query string, lim int, offset int) ([]string, error) {
	headers := map[string]string{
		"Accept":                  "application/json, text/javascript, */*; q=0.01",
		"Accept-Language":         "en-IN,en-GB;q=0.9,en-US;q=0.8,en;q=0.7,ml;q=0.6,bn;q=0.5",
		"Cache-Control":           "no-cache",
		"User-Agent":              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
		"X-App-Version":           "e5cf318",
		"X-Pinterest-Appstate":    "active",
		"X-Pinterest-Pws-Handler": "www/index.js",
		"X-Pinterest-Source-Url":  "/",
		"X-Requested-With":        "XMLHttpRequest",
	}

	params := url.Values{}
	params.Set("source_url", "/search/pins/?eq=World&etslf=67&len=2&q=world%20map&rs=ac")
	params.Set("data", fmt.Sprintf(`{"options":{"query":"%s","redux_normalize_feed":true,"scope":"pins","source_url":"/search/pins/?eq=World&etslf=67&len=2&q=world%%20map&rs=ac"},"context":{}}`, query))

	baseURL := "https://in.pinterest.com/resource/BaseSearchResource/get/"
	reqURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type ImageData struct {
		Images struct {
			Orig struct {
				URL string `json:"url"`
			} `json:"orig"`
		} `json:"images"`
	}
	var parsedResponse struct {
		ResourceResponse struct {
			Data struct {
				Results []ImageData `json:"results"`
			} `json:"data"`
		} `json:"resource_response"`
	}

	err = json.Unmarshal(body, &parsedResponse)
	if err != nil {
		return nil, err
	}

	var imageUrls []string
	for _, result := range parsedResponse.ResourceResponse.Data.Results {
		imageUrls = append(imageUrls, result.Images.Orig.URL)
	}

	if len(imageUrls) > lim && offset == 0 {
		imageUrls = imageUrls[:lim]
	} else if len(imageUrls) > lim && offset > 0 {
		offset = offset * lim
		if offset > len(imageUrls) {
			return nil, fmt.Errorf("No more images")
		}
		imageUrls = imageUrls[offset : offset+lim]
	}

	return imageUrls, nil
}

func init() {
	Mods.AddModule("Inline", `<b>Here are the commands available in Inline module:</b>

- <code>@botusername pin &lt;query&gt;</code> - Search for images on Pinterest`)
}
