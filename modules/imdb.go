package modules

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	tg "github.com/amarnathcjd/gogram/telegram"
)

type SearchResult struct {
	IMDBID string `json:"imdb_id"`
	Title  string `json:"title"`
	Year   string `json:"year"`
	Poster string `json:"poster"`
}

func SearchIMDB(query string) ([]SearchResult, error) {
	url := fmt.Sprintf("https://www.imdb.com/search/title/?title=%s", strings.ReplaceAll(query, " ", "+"))
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	doc.Find(".dli-parent").Each(func(i int, s *goquery.Selection) {
		imdbID, exists := s.Find("a.ipc-title-link-wrapper").Attr("href")
		if !exists {
			return
		}
		imdbID = strings.Split(strings.TrimPrefix(imdbID, "/title/"), "/")[0]
		title := s.Find("h3.ipc-title__text").Text()
		meta := s.Find("span.dli-title-metadata-item")

		posters := s.Find("img.ipc-image").AttrOr("src", "")
		if posters != "" {
			posters = strings.Split(posters, "@._V")[0] + "@._V1_.jpg"
		}
		if meta.Length() == 0 {
			return
		}
		year := meta.First().Text()
		results = append(results, SearchResult{IMDBID: imdbID, Title: title, Year: year, Poster: posters})
	})

	return results, nil
}

// https://v3.sg.media-imdb.com/suggestion/x/avatar.json?includeVideos=1
func quickSearchImdb(query string) ([]SearchResult, error) {
	url := fmt.Sprintf("https://v3.sg.media-imdb.com/suggestion/x/%s.json?includeVideos=1", query)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results struct {
		D []struct {
			ID string `json:"id"`
			L  string `json:"l"`
			Y  int    `json:"y"`
			I  struct {
				URL string `json:"imageUrl"`
			}
		} `json:"d"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	var searchResults []SearchResult
	for _, result := range results.D {
		searchResults = append(searchResults, SearchResult{
			IMDBID: result.ID,
			Title:  result.L,
			Year:   fmt.Sprintf("%d", result.Y),
			Poster: result.I.URL,
		})
	}

	return searchResults, nil
}

type IMDBTitle struct {
	ID                  string              `json:"id"`
	Title               string              `json:"title"`
	OgTitle             string              `json:"og_title"`
	Poster              string              `json:"poster"`
	AltTitle            string              `json:"alt_title"`
	Description         string              `json:"description"`
	Rating              float64             `json:"rating"`
	ViewerClass         string              `json:"viewer_class"`
	Duration            string              `json:"duration"`
	Genres              []string            `json:"genres"`
	ReleaseDate         string              `json:"release_date"`
	Actors              []string            `json:"actors"`
	Trailer             string              `json:"trailer"`
	CountryOfOrigin     string              `json:"country_of_origin"`
	Languages           string              `json:"languages"`
	AlsoKnownAs         string              `json:"also_known_as"`
	FilmingLocations    string              `json:"filming_locations"`
	ProductionCompanies string              `json:"production_companies"`
	RatingCount         string              `json:"rating_count"`
	MetaScore           string              `json:"meta_score"`
	MoreLikeThis        []MoreLikeThisEntry `json:"more_like_this"`
}

type MoreLikeThisEntry struct {
	IMDBID string `json:"imdb_id"`
	Title  string `json:"title"`
}

func GetIMDBTitle(titleID string) (*IMDBTitle, error) {
	url := fmt.Sprintf("https://www.imdb.com/title/%s/", titleID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch IMDb page: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	jsonMeta := doc.Find("script[type='application/ld+json']").First().Text()
	var jsonObj map[string]any
	if err := json.Unmarshal([]byte(jsonMeta), &jsonObj); err != nil {
		return nil, err
	}

	title := doc.Find("h1[data-testid=hero__pageTitle]").First().Text()
	poster, _ := jsonObj["image"].(string)
	description := getObjValue(jsonObj, "description")
	var rating = 0.0
	if jsonObj["aggregateRating"] != nil {
		rating = jsonObj["aggregateRating"].(map[string]any)["ratingValue"].(float64)
	}
	viewerClass, isViewerClass := jsonObj["contentRating"].(string)
	duration := doc.Find("li[data-testid=title-techspec_runtime] div").Text()
	genres := []string{}
	if genresArr, isGenres := jsonObj["genre"].([]any); isGenres {
		for _, genre := range genresArr {
			genres = append(genres, genre.(string))
		}
	}
	releaseDate := doc.Find("li[data-testid=title-details-releasedate] a").Text()
	actors := []string{}
	doc.Find("a[data-testid=title-cast-item__actor]").Each(func(i int, s *goquery.Selection) {
		actors = append(actors, s.Text())
	})
	trailer := ""
	if trailerObj, isTrailer := jsonObj["trailer"].(map[string]any); isTrailer {
		trailer = trailerObj["embedUrl"].(string)
	}
	countryOfOrigin := ""
	doc.Find("li[data-testid=title-details-origin] a").Each(func(i int, s *goquery.Selection) {
		countryOfOrigin += s.Text() + ", "
	})
	countryOfOrigin = strings.TrimSuffix(countryOfOrigin, ", ")
	languages := ""
	doc.Find("li[data-testid=title-details-languages] a").Each(func(i int, s *goquery.Selection) {
		languages += s.Text() + ", "
	})
	languages = strings.TrimSuffix(languages, ", ")
	alsoKnownAs := doc.Find("li[data-testid=title-details-akas] div").First().Text()
	filmingLocations := ""
	doc.Find("li[data-testid=title-details-filminglocations] a").Each(func(i int, s *goquery.Selection) {
		filmingLocations += s.Text() + ", "
	})
	filmingLocations = strings.ReplaceAll(filmingLocations, "Filming locations, ", "")
	filmingLocations = strings.TrimSuffix(filmingLocations, ", ")
	productionCompanies := ""
	doc.Find("li[data-testid=title-details-companies] a").Each(func(i int, s *goquery.Selection) {
		productionCompanies += s.Text() + ", "
	})
	productionCompanies = strings.ReplaceAll(productionCompanies, "Production companies, ", "")
	productionCompanies = strings.TrimSuffix(productionCompanies, ", ")
	ratingCount := strings.ReplaceAll(doc.Find("div.sc-eb51e184-3").First().Text(), ",", "")
	altTitle, isAltTitle := jsonObj["alternateName"].(string)
	metaScore := doc.Find("span.metacritic-score-box").Text()

	moreLikeThis := []MoreLikeThisEntry{}
	doc.Find("section[data-testid=MoreLikeThis] div.ipc-poster-card").Each(func(i int, s *goquery.Selection) {
		mId, _ := s.Find("a.ipc-lockup-overlay").Attr("href")
		mId = strings.TrimPrefix(mId, "/title/")
		mId = strings.Split(mId, "/")[0]
		mTitle := s.Find("img.ipc-image").AttrOr("alt", "")
		moreLikeThis = append(moreLikeThis, MoreLikeThisEntry{
			IMDBID: mId,
			Title:  mTitle,
		})
	})

	var tt = &IMDBTitle{
		ID:                  titleID,
		Title:               title,
		OgTitle:             doc.Find("div.sc-ec65ba05-1").First().Text(),
		Poster:              poster,
		Description:         description,
		Rating:              rating,
		Duration:            duration,
		Genres:              genres,
		ReleaseDate:         strings.Replace(releaseDate, "Release date", "", 1),
		Actors:              actors,
		Trailer:             trailer,
		CountryOfOrigin:     countryOfOrigin,
		Languages:           languages,
		AlsoKnownAs:         alsoKnownAs,
		FilmingLocations:    filmingLocations,
		ProductionCompanies: productionCompanies,
		RatingCount:         ratingCount,
		MetaScore:           metaScore,
		MoreLikeThis:        moreLikeThis,
	}

	if isAltTitle {
		tt.AltTitle = altTitle
	}
	if isViewerClass {
		tt.ViewerClass = viewerClass
	}

	return tt, nil
}

func getObjValue(obj map[string]any, key string) string {
	if val, exists := obj[key]; exists {
		return val.(string)
	}
	return ""
}

func FormatIMDBDataToHTML(data *IMDBTitle) string {
	parseYearFromString := func(s string) string {
		re := regexp.MustCompile(`\((\d{4})\)`)
		matches := re.FindStringSubmatch(s)
		if len(matches) > 1 {
			return matches[1]
		}
		return ""
	}

	var sb strings.Builder

	// Title with year
	year := parseYearFromString(data.ReleaseDate)
	if year != "" {
		sb.WriteString(fmt.Sprintf("🎬 <b>%s</b> (%s)\n\n", data.Title, year))
	} else {
		sb.WriteString(fmt.Sprintf("🎬 <b>%s</b>\n\n", data.Title))
	}

	// Rating and Metascore on one line
	if data.Rating != 0 {
		stars := "⭐"
		if data.Rating >= 8.0 {
			stars = "⭐⭐⭐"
		} else if data.Rating >= 6.0 {
			stars = "⭐⭐"
		}
		sb.WriteString(fmt.Sprintf("➜ <b>Rating:</b> 📊 %.1f/10 %s <i>(%s votes)</i>", data.Rating, stars, data.RatingCount))
		if data.MetaScore != "" {
			sb.WriteString(fmt.Sprintf(" | <b>Meta:</b> 🏆 %s", data.MetaScore))
		}
		sb.WriteString("\n")
	}

	// Release date and duration on one line
	if data.ReleaseDate != "" || data.Duration != "" {
		sb.WriteString("➜ ")
		if data.ReleaseDate != "" {
			sb.WriteString(fmt.Sprintf("📅 <b>Released:</b> %s", data.ReleaseDate))
		}
		if data.Duration != "" {
			if data.ReleaseDate != "" {
				sb.WriteString(" | ")
			}
			sb.WriteString(fmt.Sprintf("⏱️ <b>Runtime:</b> %s", data.Duration))
		}
		sb.WriteString("\n")
	}

	// Viewer class
	if data.ViewerClass != "" {
		sb.WriteString(fmt.Sprintf("➜ 🔞 <b>Rated:</b> %s\n", data.ViewerClass))
	}

	// Description
	if data.Description != "" {
		sb.WriteString(fmt.Sprintf("\n💬 <i>%s</i>\n\n", data.Description))
	}

	// Genres as hashtags
	if len(data.Genres) > 0 {
		sb.WriteString("➜ 🎭 <b>Genres:</b> ")
		var genres []string
		for _, genre := range data.Genres {
			genres = append(genres, fmt.Sprintf("<a href='https://www.imdb.com/search/title/?genres=%s'>#%s</a>", genre, genre))
		}
		sb.WriteString(strings.Join(genres, " "))
		sb.WriteString("\n")
	}

	// Cast (top 5)
	if len(data.Actors) > 0 {
		sb.WriteString("➜ 👥 <b>Cast:</b> ")
		var actors []string
		for i, actor := range data.Actors {
			if i >= 5 {
				break
			}
			actors = append(actors, fmt.Sprintf("<a href='https://www.imdb.com/find?q=%s'>%s</a>", actor, actor))
		}
		sb.WriteString(strings.Join(actors, ", "))
		sb.WriteString("\n")
	}

	// Countries and Languages on one line
	if data.CountryOfOrigin != "" || data.Languages != "" {
		sb.WriteString("➜ ")
		if data.CountryOfOrigin != "" {
			sb.WriteString(fmt.Sprintf("🌍 <b>Country:</b> %s", data.CountryOfOrigin))
		}
		if data.Languages != "" {
			if data.CountryOfOrigin != "" {
				sb.WriteString(" | ")
			}
			sb.WriteString(fmt.Sprintf("🗣️ <b>Language:</b> %s", data.Languages))
		}
		sb.WriteString("\n")
	}

	// Production companies
	if data.ProductionCompanies != "" {
		sb.WriteString(fmt.Sprintf("➜ 🎥 <b>Production:</b> %s\n", strings.TrimSuffix(data.ProductionCompanies, `, `)))
	}

	// AKA
	if data.AlsoKnownAs != "" {
		sb.WriteString(fmt.Sprintf("\n➜ 📝 <b>AKA:</b> <i>%s</i>\n", data.AlsoKnownAs))
	}

	// Trailer link
	if data.Trailer != "" {
		sb.WriteString(fmt.Sprintf("\n<a href='%s'>▶️ Watch Trailer</a>", data.Trailer))
	}

	// More like this (show up to 10, comma separated)
	if len(data.MoreLikeThis) > 0 {
		sb.WriteString("\n\n🍿 <b>More Like This:</b> ")
		var similar []string
		for i, entry := range data.MoreLikeThis {
			if i >= 10 {
				break
			}
			similar = append(similar, fmt.Sprintf("<a href='https://www.imdb.com/title/%s/'>%s</a>", entry.IMDBID, entry.Title))
		}
		sb.WriteString(strings.Join(similar, ", "))
		sb.WriteString("\n")
	}

	return sb.String()
}

func ImDBInlineSearchHandler(m *tg.InlineQuery) error {
	b := m.Builder()
	if m.Args() == "" {
		b.Article("No query", "Please enter a query to search for", "No query", &tg.ArticleOptions{
			ReplyMarkup: tg.Button.Keyboard(
				tg.Button.Row(
					tg.Button.SwitchInline("Search!!!", true, "imdb "),
				),
			),
		})
		m.Answer(b.Results())
		return nil
	}

	results, err := quickSearchImdb(m.Args())
	if err != nil {
		b.Article("Error", "Failed to fetch IMDb search results", "Error", &tg.ArticleOptions{
			ReplyMarkup: tg.Button.Keyboard(
				tg.Button.Row(
					tg.Button.SwitchInline("Search again", true, "imdb "),
				),
			),
		})
		m.Answer(b.Results())
		return err
	}

	if len(results) == 0 {
		b.Article("No results found", "No results found for the query", "No results found", &tg.ArticleOptions{
			ReplyMarkup: tg.Button.Keyboard(
				tg.Button.Row(
					tg.Button.SwitchInline("Search again", true, "imdb "),
				),
			),
		})
		m.Answer(b.Results())
		return nil
	}

	for i, result := range results {
		if i >= 10 {
			break
		}
		b.Article(fmt.Sprintf("%d. %s", i, result.Title), result.Year, result.IMDBID, &tg.ArticleOptions{
			ID:            result.IMDBID,
			ForceDocument: true,
			ReplyMarkup: tg.Button.Keyboard(
				tg.Button.Row(
					tg.Button.SwitchInline("Search again", true, "imdb "),
				),
			),
			Thumb: tg.InputWebDocument{
				URL: result.Poster,
				Attributes: []tg.DocumentAttribute{
					&tg.DocumentAttributeImageSize{
						W: 100,
						H: 100,
					},
				},
				Size:     100,
				MimeType: "image/jpeg",
			},
		})
	}

	m.Answer(b.Results(), &tg.InlineSendOptions{
		Gallery: false,
	})
	return nil
}

func ImdbHandler(m *tg.NewMessage) error {
	if m.Args() == "" {
		m.Reply("Please provide a search query.", &tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				tg.Button.SwitchInline("Go >> Search IMDb", true, "imdb "),
			).Build(),
		})
		return nil
	}

	if strings.HasPrefix(m.Args(), "tt") {
		data, err := GetIMDBTitle(m.Args())
		if err != nil {
			m.Reply("Failed to fetch IMDb data.")
			return err
		}
		if data.Poster != "" {
			m.ReplyMedia(data.Poster, &tg.MediaOptions{
				Caption: FormatIMDBDataToHTML(data),
				Spoiler: true,
				ReplyMarkup: tg.NewKeyboard().AddRow(
					tg.Button.URL("🔗 IMDb Link", fmt.Sprintf("https://www.imdb.com/title/%s/", m.Args())),
				).Build(),
			})
		} else {
			m.Reply(FormatIMDBDataToHTML(data), &tg.SendOptions{
				ReplyMarkup: tg.NewKeyboard().AddRow(
					tg.Button.URL("🔗 IMDb Link", fmt.Sprintf("https://www.imdb.com/title/%s/", m.Args())),
				).Build(),
			})
		}
	} else {
		results, err := SearchIMDB(m.Args())
		if err != nil {
			m.Reply("Failed to fetch IMDb search results.")
			return err
		}

		if len(results) == 0 {
			m.Reply("No results found.")
			return nil
		}

		var kyb = tg.NewKeyboard()
		for i, result := range results {
			if i >= 10 {
				break
			}
			kyb.AddRow(
				tg.Button.Data(fmt.Sprintf("%s (%s)", result.Title, result.Year), fmt.Sprintf("imdb_%s_%d", result.IMDBID, m.SenderID())),
			)
		}

		m.Reply("🔍 <b>Search Results:</b>", &tg.SendOptions{
			ReplyMarkup: kyb.Build(),
		})
		return nil
	}

	return nil
}

func ImdbInlineOnSendHandler(u *tg.InlineSend) error {
	titleId := u.ID
	data, err := GetIMDBTitle(titleId)
	if err != nil {
		return err
	}

	if data.Poster != "" {
		u.Edit(FormatIMDBDataToHTML(data), &tg.SendOptions{
			Media:   data.Poster,
			Spoiler: true,
			ReplyMarkup: tg.NewKeyboard().AddRow(
				tg.Button.URL("🔗 IMDb Link", fmt.Sprintf("https://www.imdb.com/title/%s/", titleId)),
			).AddRow(
				tg.Button.SwitchInline("Search again", true, "imdb "),
			).Build(),
		})
	} else {
		u.Edit(FormatIMDBDataToHTML(data), &tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				tg.Button.URL("🔗 IMDb Link", fmt.Sprintf("https://www.imdb.com/title/%s/", titleId)),
			).AddRow(
				tg.Button.SwitchInline("Search again", true, "imdb "),
			).Build(),
		})
	}

	return nil
}

func ImdbCallbackHandler(m *tg.CallbackQuery) error {
	dt := strings.Split(m.DataString(), "_")
	if len(dt) != 3 {
		m.Answer("Invalid IMDb data.")
		return nil
	}
	if strings.Compare(dt[2], fmt.Sprintf("%d", m.SenderID)) != 0 {
		m.Answer("You are not allowed to view this data.")
		return nil
	}

	data, err := GetIMDBTitle(dt[1])
	if err != nil {
		m.Answer("Failed to fetch IMDb data.")
		return err
	}

	if data.Poster != "" {
		m.Edit(FormatIMDBDataToHTML(data), &tg.SendOptions{
			Media:   data.Poster,
			Spoiler: true,
			ReplyMarkup: tg.NewKeyboard().AddRow(
				tg.Button.URL("🔗 IMDb Link", fmt.Sprintf("https://www.imdb.com/title/%s/", dt[1])),
			).Build(),
		})
	} else {
		m.Edit(FormatIMDBDataToHTML(data), &tg.SendOptions{
			ReplyMarkup: tg.NewKeyboard().AddRow(
				tg.Button.URL("🔗 IMDb Link", fmt.Sprintf("https://www.imdb.com/title/%s/", dt[1])),
			).Build(),
		})
	}

	return nil
}

func registerIMDBHandlers() {
	c := Client
	c.On("cmd:imdb", ImdbHandler)
	c.On("inline:imdb", ImDBInlineSearchHandler)
	c.On("callback:imdb_(.*) _(.*)", ImdbCallbackHandler)
}

func init() {
	QueueHandlerRegistration(registerIMDBHandlers)
}
