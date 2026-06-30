package extras

import (
	"encoding/json"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"html"
	"io"
	modules "main/modules"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// === from dictionary.go ===
type dictionaryDefinition struct {
	Definition string `json:"definition"`
	Example    string `json:"example"`
}

type dictionaryMeaning struct {
	PartOfSpeech string                 `json:"partOfSpeech"`
	Definitions  []dictionaryDefinition `json:"definitions"`
}

type dictionaryPhonetic struct {
	Text string `json:"text"`
}

type dictionaryEntry struct {
	Word      string               `json:"word"`
	Phonetic  string               `json:"phonetic"`
	Phonetics []dictionaryPhonetic `json:"phonetics"`
	Meanings  []dictionaryMeaning  `json:"meanings"`
}

func DefineHandler(m *tg.NewMessage) error {
	word := strings.TrimSpace(m.Args())
	if word == "" {
		m.Reply("Usage: <code>/define &lt;word&gt;</code>")
		return nil
	}
	client := &http.Client{Timeout: 30 * time.Second}
	url := "https://api.dictionaryapi.dev/api/v2/entries/en/" + word
	resp, err := client.Get(url)
	if err != nil {
		m.Reply("couldn't fetch definition: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		m.Reply("No definition found for <b>" + html.EscapeString(word) + "</b>")
		return nil
	}
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var entries []dictionaryEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		m.Reply("couldn't parse definition: " + err.Error())
		return nil
	}
	if len(entries) == 0 {
		m.Reply("No definition found for <b>" + html.EscapeString(word) + "</b>")
		return nil
	}
	entry := entries[0]
	headword := entry.Word
	if headword == "" {
		headword = word
	}
	phonetic := entry.Phonetic
	if phonetic == "" {
		for _, p := range entry.Phonetics {
			if p.Text != "" {
				phonetic = p.Text
				break
			}
		}
	}
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(headword))
	b.WriteString("</b>")
	if phonetic != "" {
		b.WriteString(" <i>")
		b.WriteString(html.EscapeString(phonetic))
		b.WriteString("</i>")
	}
	b.WriteString("\n")
	count := 0
	for _, meaning := range entry.Meanings {
		if count >= 3 {
			break
		}
		for _, def := range meaning.Definitions {
			if count >= 3 {
				break
			}
			if def.Definition == "" {
				continue
			}
			count++
			b.WriteString(fmt.Sprintf("\n<b>%d.</b> <i>(%s)</i> %s", count, html.EscapeString(meaning.PartOfSpeech), html.EscapeString(def.Definition)))
			if def.Example != "" {
				b.WriteString("\n   <i>e.g. ")
				b.WriteString(html.EscapeString(def.Example))
				b.WriteString("</i>")
			}
		}
	}
	if count == 0 {
		m.Reply("No definition found for <b>" + html.EscapeString(word) + "</b>")
		return nil
	}
	m.Reply(b.String())
	return nil
}

func initFromSrc_dictionary_0_1() { modules.QueueHandlerRegistration(registerDictionaryHandlers) }
func registerDictionaryHandlers() {
	c := modules.Client
	c.On("cmd:define", DefineHandler)
}
// === from kanji_lookup.go ===
type kanjiLookupResponse struct {
	Kanji        string   `json:"kanji"`
	Grade        int      `json:"grade"`
	StrokeCount  int      `json:"stroke_count"`
	Meanings     []string `json:"meanings"`
	KunReadings  []string `json:"kun_readings"`
	OnReadings   []string `json:"on_readings"`
	NameReadings []string `json:"name_readings"`
	JLPT         int      `json:"jlpt"`
	Unicode      string   `json:"unicode"`
	HeisigEn     string   `json:"heisig_en"`
}

func KanjiLookupHandler(m *tg.NewMessage) error {
	q := strings.TrimSpace(m.Args())
	if q == "" {
		m.Reply("Usage: <code>/kanji &lt;character&gt;</code>")
		return nil
	}
	runes := []rune(q)
	character := string(runes[0])
	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := "https://kanjiapi.dev/v1/kanji/" + url.PathEscape(character)
	resp, err := client.Get(endpoint)
	if err != nil {
		m.Reply("couldn't fetch kanji: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		m.Reply("No kanji found for <b>" + html.EscapeString(character) + "</b>")
		return nil
	}
	if resp.StatusCode != 200 {
		m.Reply(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}
	var data kanjiLookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		m.Reply("couldn't parse kanji: " + err.Error())
		return nil
	}
	if data.Kanji == "" {
		m.Reply("No kanji found for <b>" + html.EscapeString(character) + "</b>")
		return nil
	}
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(data.Kanji))
	b.WriteString("</b>")
	if data.Unicode != "" {
		b.WriteString(" <i>(U+")
		b.WriteString(html.EscapeString(data.Unicode))
		b.WriteString(")</i>")
	}
	b.WriteString("\n")
	if len(data.Meanings) > 0 {
		b.WriteString("\n<b>Meanings:</b> ")
		b.WriteString(html.EscapeString(strings.Join(data.Meanings, ", ")))
	}
	if len(data.OnReadings) > 0 {
		b.WriteString("\n<b>On:</b> ")
		b.WriteString(html.EscapeString(strings.Join(data.OnReadings, ", ")))
	}
	if len(data.KunReadings) > 0 {
		b.WriteString("\n<b>Kun:</b> ")
		b.WriteString(html.EscapeString(strings.Join(data.KunReadings, ", ")))
	}
	if len(data.NameReadings) > 0 {
		b.WriteString("\n<b>Name:</b> ")
		b.WriteString(html.EscapeString(strings.Join(data.NameReadings, ", ")))
	}
	b.WriteString(fmt.Sprintf("\n<b>Strokes:</b> %d", data.StrokeCount))
	if data.JLPT > 0 {
		b.WriteString(fmt.Sprintf("\n<b>JLPT:</b> N%d", data.JLPT))
	}
	if data.Grade > 0 {
		b.WriteString(fmt.Sprintf("\n<b>Grade:</b> %d", data.Grade))
	}
	if data.HeisigEn != "" {
		b.WriteString("\n<b>Heisig:</b> ")
		b.WriteString(html.EscapeString(data.HeisigEn))
	}
	m.Reply(b.String())
	return nil
}

func initFromSrc_kanji_lookup_1_1() { modules.QueueHandlerRegistration(registerKanjiLookupHandlers) }
func registerKanjiLookupHandlers() {
	c := modules.Client
	c.On("cmd:kanji", KanjiLookupHandler)
}
// === from urban.go ===
type urbanDefinition struct {
	Word       string `json:"word"`
	Definition string `json:"definition"`
	Example    string `json:"example"`
	ThumbsUp   int    `json:"thumbs_up"`
	ThumbsDown int    `json:"thumbs_down"`
	Author     string `json:"author"`
	Permalink  string `json:"permalink"`
}

type urbanResponse struct {
	List []urbanDefinition `json:"list"`
}

type urbanState struct {
	Term    string
	Results []urbanDefinition
	Created time.Time
}

var urbanCache sync.Map

const urbanPageSize = 3
const urbanMaxResults = 10
const urbanTTL = 30 * time.Minute

func urbanFetch(term string) ([]urbanDefinition, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := "https://api.urbandictionary.com/v0/define?term=" + url.QueryEscape(term)
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var out urbanResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.List) > urbanMaxResults {
		out.List = out.List[:urbanMaxResults]
	}
	return out.List, nil
}

func urbanCleanText(s string) string {
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "]", "")
	return strings.TrimSpace(s)
}

func urbanRender(term string, results []urbanDefinition, page int) (string, int) {
	if len(results) == 0 {
		return "<b>No definitions found for</b> <i>" + html.EscapeString(term) + "</i>", 0
	}
	total := len(results)
	pages := (total + urbanPageSize - 1) / urbanPageSize
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * urbanPageSize
	end := start + urbanPageSize
	if end > total {
		end = total
	}
	var b strings.Builder
	b.WriteString("<b>Urban Dictionary:</b> <i>")
	b.WriteString(html.EscapeString(term))
	b.WriteString("</i>\n")
	for i := start; i < end; i++ {
		d := results[i]
		def := urbanCleanText(d.Definition)
		ex := urbanCleanText(d.Example)
		if len(def) > 600 {
			def = def[:600] + "..."
		}
		if len(ex) > 400 {
			ex = ex[:400] + "..."
		}
		b.WriteString(fmt.Sprintf("\n<b>%d.</b> 👍 %d  👎 %d\n", i+1, d.ThumbsUp, d.ThumbsDown))
		b.WriteString("<blockquote>")
		b.WriteString(html.EscapeString(def))
		b.WriteString("</blockquote>")
		if ex != "" {
			b.WriteString("\n<i>Example:</i>\n<blockquote>")
			b.WriteString(html.EscapeString(ex))
			b.WriteString("</blockquote>")
		}
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("\n<i>Page %d / %d</i>", page+1, pages))
	return b.String(), pages
}

func urbanKeyboard(term string, page, pages int) *tg.ReplyInlineMarkup {
	if pages <= 1 {
		return nil
	}
	b := tg.Button
	kb := tg.NewKeyboard()
	var row []tg.KeyboardButton
	if page > 0 {
		row = append(row, b.Data("« Prev", fmt.Sprintf("urban:%s:%d", term, page-1)))
	}
	if page < pages-1 {
		row = append(row, b.Data("Next »", fmt.Sprintf("urban:%s:%d", term, page+1)))
	}
	if len(row) > 0 {
		kb.AddRow(row...)
	}
	return kb.Build()
}

func urbanGC() {
	now := time.Now()
	urbanCache.Range(func(k, v any) bool {
		st, ok := v.(*urbanState)
		if !ok || now.Sub(st.Created) > urbanTTL {
			urbanCache.Delete(k)
		}
		return true
	})
}

func UrbanHandler(m *tg.NewMessage) error {
	term := strings.TrimSpace(m.Args())
	if term == "" {
		m.Reply("Usage: <code>/urban &lt;term&gt;</code>")
		return nil
	}
	urbanGC()
	results, err := urbanFetch(term)
	if err != nil {
		m.Reply("couldn't fetch urban dictionary: " + html.EscapeString(err.Error()))
		return nil
	}
	if len(results) == 0 {
		m.Reply("<b>No definitions found for</b> <i>" + html.EscapeString(term) + "</i>")
		return nil
	}
	text, pages := urbanRender(term, results, 0)
	opts := &tg.SendOptions{}
	if kb := urbanKeyboard(term, 0, pages); kb != nil {
		opts.ReplyMarkup = kb
	}
	sent, err := m.Reply(text, opts)
	if err != nil {
		return nil
	}
	key := fmt.Sprintf("%d:%d", sent.ChatID(), sent.ID)
	urbanCache.Store(key, &urbanState{Term: term, Results: results, Created: time.Now()})
	return nil
}

func UrbanCallback(c *tg.CallbackQuery) error {
	data := c.DataString()
	rest := strings.TrimPrefix(data, "urban:")
	idx := strings.LastIndex(rest, ":")
	if idx <= 0 {
		c.Answer("Bad data.", &tg.CallbackOptions{Alert: false})
		return nil
	}
	term := rest[:idx]
	pageStr := rest[idx+1:]
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.Answer("Bad page.", &tg.CallbackOptions{Alert: false})
		return nil
	}
	urbanGC()
	key := fmt.Sprintf("%d:%d", c.ChatID, c.MessageID)
	var results []urbanDefinition
	if v, ok := urbanCache.Load(key); ok {
		if st, ok2 := v.(*urbanState); ok2 && time.Since(st.Created) <= urbanTTL {
			results = st.Results
		}
	}
	if results == nil {
		fresh, ferr := urbanFetch(term)
		if ferr != nil || len(fresh) == 0 {
			c.Answer("Session expired. Run /urban again.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		results = fresh
		urbanCache.Store(key, &urbanState{Term: term, Results: results, Created: time.Now()})
	}
	text, pages := urbanRender(term, results, page)
	opts := &tg.SendOptions{}
	if kb := urbanKeyboard(term, page, pages); kb != nil {
		opts.ReplyMarkup = kb
	}
	c.Edit(text, opts)
	c.Answer("", &tg.CallbackOptions{Alert: false})
	return nil
}

func initFromSrc_urban_2_1() { modules.QueueHandlerRegistration(registerUrbanHandlers) }
func registerUrbanHandlers() {
	c := modules.Client
	c.On("cmd:urban", UrbanHandler)
	c.On("callback:urban:", UrbanCallback)
}
// === from definewiki.go ===
type wikiThumbnail struct {
	Source string `json:"source"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type wikiContentURLs struct {
	Page string `json:"page"`
}

type wikiURLBundle struct {
	Desktop wikiContentURLs `json:"desktop"`
	Mobile  wikiContentURLs `json:"mobile"`
}

type wikiSummaryResponse struct {
	Type         string         `json:"type"`
	Title        string         `json:"title"`
	DisplayTitle string         `json:"displaytitle"`
	Description  string         `json:"description"`
	Extract      string         `json:"extract"`
	Thumbnail    *wikiThumbnail `json:"thumbnail"`
	Originalimg  *wikiThumbnail `json:"originalimage"`
	ContentURLs  wikiURLBundle  `json:"content_urls"`
	Detail       string         `json:"detail"`
	Title404     string         `json:"title"`
}

func wikiHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func wikiTrimExtract(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if idx := strings.Index(s, "\n"); idx > 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	cut := string(runes[:max])
	if i := strings.LastIndex(cut, " "); i > max/2 {
		cut = cut[:i]
	}
	return strings.TrimSpace(cut) + "..."
}

func wikiFetchSummary(term string) (*wikiSummaryResponse, int, error) {
	endpoint := "https://en.wikipedia.org/api/rest_v1/page/summary/" + url.PathEscape(strings.ReplaceAll(term, " ", "_"))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0 (https://github.com/amarnathcjd)")
	req.Header.Set("Accept", "application/json")
	resp, err := wikiHTTPClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var out wikiSummaryResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, resp.StatusCode, err
	}
	return &out, resp.StatusCode, nil
}

func wikiDownloadThumb(src string) (string, error) {
	if src == "" {
		return "", nil
	}
	req, err := http.NewRequest(http.MethodGet, src, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0 (https://github.com/amarnathcjd)")
	resp, err := wikiHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", nil
	}
	ext := strings.ToLower(filepath.Ext(src))
	if ext == "" || len(ext) > 5 {
		ext = ".jpg"
	}
	tmp, err := os.CreateTemp("", "wiki_thumb_*"+ext)
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func WikiHandler(m *tg.NewMessage) error {
	term := strings.TrimSpace(m.Args())
	if term == "" {
		m.Reply("<b>Usage:</b> <code>/wiki &lt;term&gt;</code>")
		return nil
	}
	status, _ := m.Reply("<i>Searching Wikipedia for <code>" + html.EscapeString(term) + "</code>...</i>")
	data, code, err := wikiFetchSummary(term)
	if err != nil {
		msg := "<b>Wikipedia lookup failed:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if code == 404 {
		msg := "<b>No Wikipedia article found for</b> <code>" + html.EscapeString(term) + "</code>."
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if code >= 400 || data == nil {
		detail := ""
		if data != nil {
			detail = data.Detail
		}
		if detail == "" {
			detail = "request failed"
		}
		msg := "<b>Wikipedia lookup failed:</b> <code>" + html.EscapeString(detail) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	title := data.Title
	if title == "" {
		title = data.DisplayTitle
	}
	if title == "" {
		title = term
	}
	pageURL := data.ContentURLs.Desktop.Page
	if pageURL == "" {
		pageURL = data.ContentURLs.Mobile.Page
	}
	if pageURL == "" {
		pageURL = "https://en.wikipedia.org/wiki/" + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	}

	var sb strings.Builder
	sb.WriteString("<b>")
	sb.WriteString(html.EscapeString(title))
	sb.WriteString("</b>")
	if data.Description != "" {
		sb.WriteString("\n<i>")
		sb.WriteString(html.EscapeString(data.Description))
		sb.WriteString("</i>")
	}
	sb.WriteString("\n\n")

	if strings.EqualFold(data.Type, "disambiguation") {
		sb.WriteString("<i>This term refers to multiple things. Please be more specific.</i>\n\n")
	}

	extract := wikiTrimExtract(data.Extract, 600)
	if extract != "" {
		sb.WriteString(html.EscapeString(extract))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("<i>No summary available.</i>\n\n")
	}

	sb.WriteString("<a href=\"")
	sb.WriteString(html.EscapeString(pageURL))
	sb.WriteString("\">Read on Wikipedia</a>")

	out := sb.String()

	thumbURL := ""
	if data.Thumbnail != nil {
		thumbURL = data.Thumbnail.Source
	}
	if thumbURL == "" && data.Originalimg != nil {
		thumbURL = data.Originalimg.Source
	}

	if thumbURL != "" && !strings.EqualFold(data.Type, "disambiguation") {
		path, err := wikiDownloadThumb(thumbURL)
		if err == nil && path != "" {
			defer os.Remove(path)
			if status != nil {
				status.Delete()
			}
			m.ReplyMedia(path, &tg.MediaOptions{Caption: out})
			return nil
		}
	}

	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func initFromSrc_definewiki_3_1() { modules.QueueHandlerRegistration(registerWikiHandlers) }

func registerWikiHandlers() {
	c := modules.Client
	c.On("cmd:wiki", WikiHandler)
}
// === from numfact.go ===
var numFactClient = &http.Client{Timeout: 30 * time.Second}

func fetchNumFact(endpoint string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := numFactClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("numbersapi returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func NumFactHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("Usage: <code>/numfact &lt;number&gt;</code>")
		return nil
	}
	if _, err := strconv.Atoi(arg); err != nil {
		m.Reply("<b>Error:</b> please provide a valid integer.")
		return nil
	}
	endpoint := "http://numbersapi.com/" + arg + "/trivia"
	fact, err := fetchNumFact(endpoint)
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch number fact.")
		return nil
	}
	if fact == "" {
		m.Reply("<b>Error:</b> no fact received.")
		return nil
	}
	m.Reply(fmt.Sprintf("<b>Number Fact (%s):</b>\n<blockquote>%s</blockquote>", html.EscapeString(arg), html.EscapeString(fact)))
	return nil
}

func DateFactHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("Usage: <code>/datefact MM/DD</code>")
		return nil
	}
	parts := strings.Split(arg, "/")
	if len(parts) != 2 {
		m.Reply("<b>Error:</b> format must be <code>MM/DD</code>.")
		return nil
	}
	month, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || month < 1 || month > 12 {
		m.Reply("<b>Error:</b> invalid month (1-12).")
		return nil
	}
	day, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || day < 1 || day > 31 {
		m.Reply("<b>Error:</b> invalid day (1-31).")
		return nil
	}
	endpoint := fmt.Sprintf("http://numbersapi.com/%d/%d/date", month, day)
	fact, err := fetchNumFact(endpoint)
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch date fact.")
		return nil
	}
	if fact == "" {
		m.Reply("<b>Error:</b> no fact received.")
		return nil
	}
	label := fmt.Sprintf("%02d/%02d", month, day)
	m.Reply(fmt.Sprintf("<b>Date Fact (%s):</b>\n<blockquote>%s</blockquote>", html.EscapeString(label), html.EscapeString(fact)))
	return nil
}

func YearFactHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("Usage: <code>/yearfact &lt;year&gt;</code>")
		return nil
	}
	year, err := strconv.Atoi(arg)
	if err != nil {
		m.Reply("<b>Error:</b> please provide a valid year.")
		return nil
	}
	endpoint := fmt.Sprintf("http://numbersapi.com/%d/year", year)
	fact, err := fetchNumFact(endpoint)
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch year fact.")
		return nil
	}
	if fact == "" {
		m.Reply("<b>Error:</b> no fact received.")
		return nil
	}
	m.Reply(fmt.Sprintf("<b>Year Fact (%d):</b>\n<blockquote>%s</blockquote>", year, html.EscapeString(fact)))
	return nil
}

func registerNumFactHandlers() {
	c := modules.Client
	c.On("cmd:numfact", NumFactHandler)
	c.On("cmd:datefact", DateFactHandler)
	c.On("cmd:yearfact", YearFactHandler)
}

func initFromSrc_numfact_4_1() {
	modules.QueueHandlerRegistration(registerNumFactHandlers)
}
// === from random_fact_api.go ===
var randomFactAPIClient = &http.Client{Timeout: 30 * time.Second}

type uselessFactResp struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Source    string `json:"source"`
	SourceURL string `json:"source_url"`
	Language  string `json:"language"`
	Permalink string `json:"permalink"`
}

type zenQuoteResp struct {
	Q    string `json:"q"`
	A    string `json:"a"`
	I    string `json:"i"`
	H    string `json:"h"`
	Date string `json:"date"`
}

func fetchRandomFactAPIJSON(url string, out interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "JuliaBot (https://github.com/amarnathcjd)")
	resp, err := randomFactAPIClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func RandomFact2Handler(m *tg.NewMessage) error {
	var f uselessFactResp
	if err := fetchRandomFactAPIJSON("https://uselessfacts.jsph.pl/api/v2/facts/random?language=en", &f); err != nil {
		m.Reply("<b>Error:</b> failed to fetch random fact.")
		return nil
	}
	if f.Text == "" {
		m.Reply("<b>Error:</b> empty fact received.")
		return nil
	}
	source := f.Source
	if source == "" {
		source = "unknown"
	}
	msg := fmt.Sprintf("<b>Random Fact:</b>\n<i>%s</i>\n\n<b>Source:</b> <code>%s</code>", html.EscapeString(f.Text), html.EscapeString(source))
	if f.Permalink != "" {
		msg += fmt.Sprintf("\n<a href=\"%s\">permalink</a>", html.EscapeString(f.Permalink))
	}
	m.Reply(msg)
	return nil
}

func QuoteDailyHandler(m *tg.NewMessage) error {
	var arr []zenQuoteResp
	if err := fetchRandomFactAPIJSON("https://zenquotes.io/api/today", &arr); err != nil {
		m.Reply("<b>Error:</b> failed to fetch daily quote.")
		return nil
	}
	if len(arr) == 0 || arr[0].Q == "" {
		m.Reply("<b>Error:</b> empty quote received.")
		return nil
	}
	q := arr[0]
	author := q.A
	if author == "" {
		author = "Unknown"
	}
	msg := fmt.Sprintf("<b>Quote of the Day:</b>\n<i>%s</i>\n\n<b>— %s</b>", html.EscapeString(q.Q), html.EscapeString(author))
	if q.Date != "" {
		msg += fmt.Sprintf("\n<code>%s</code>", html.EscapeString(q.Date))
	}
	m.Reply(msg)
	return nil
}

func RandomQuoteHandler(m *tg.NewMessage) error {
	var arr []zenQuoteResp
	if err := fetchRandomFactAPIJSON("https://zenquotes.io/api/random", &arr); err != nil {
		m.Reply("<b>Error:</b> failed to fetch random quote.")
		return nil
	}
	if len(arr) == 0 || arr[0].Q == "" {
		m.Reply("<b>Error:</b> empty quote received.")
		return nil
	}
	q := arr[0]
	author := q.A
	if author == "" {
		author = "Unknown"
	}
	m.Reply(fmt.Sprintf("<b>Random Quote:</b>\n<i>%s</i>\n\n<b>— %s</b>", html.EscapeString(q.Q), html.EscapeString(author)))
	return nil
}

func registerRandomFactAPIHandlers() {
	c := modules.Client
	c.On("cmd:randomfact2", RandomFact2Handler)
	c.On("cmd:quotedaily", QuoteDailyHandler)
	c.On("cmd:randomquote", RandomQuoteHandler)
}

func initFromSrc_random_fact_api_5_1() {
	modules.QueueHandlerRegistration(registerRandomFactAPIHandlers)
}

func init() {
	initFromSrc_dictionary_0_1()
	initFromSrc_kanji_lookup_1_1()
	initFromSrc_urban_2_1()
	initFromSrc_definewiki_3_1()
	initFromSrc_numfact_4_1()
	initFromSrc_random_fact_api_5_1()
}
