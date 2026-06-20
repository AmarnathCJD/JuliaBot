package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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

func init() { QueueHandlerRegistration(registerUrbanHandlers) }
func registerUrbanHandlers() {
	c := Client
	c.On("cmd:urban", UrbanHandler)
	c.On("callback:urban:", UrbanCallback)
}
