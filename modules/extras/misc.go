package extras

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	modules "main/modules"
	"main/modules/db"

	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
)

func Gban(m *tg.NewMessage) error {
	user, reason, err := modules.GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	message, _ := m.Reply("Enforcing global ban...")
	done := 0
	m.Client.Broadcast(context.Background(), nil, func(c tg.Chat) error {
		_, err := m.Client.EditBanned(c, user, &tg.BannedOptions{Ban: true})
		if err == nil {
			done++
		}
		return nil
	}, 600)

	message.Edit(fmt.Sprintf("Global ban enforced in %d groups.\nReason: %s", done, reason))
	return nil
}

func Ungban(m *tg.NewMessage) error {
	user, _, err := modules.GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	message, _ := m.Reply("Removing global ban...")
	done := 0
	m.Client.Broadcast(context.Background(), nil, func(c tg.Chat) error {
		_, err := m.Client.EditBanned(c, user, &tg.BannedOptions{Ban: false})
		if err == nil {
			done++
		}
		return nil
	}, 600)
	message.Edit(fmt.Sprintf("Global ban removed in %d groups.", done))
	return nil
}

func NightModeHandler(m *tg.NewMessage) error {
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info rights to use this command")
		return nil
	}

	args := m.Args()
	if args == "" {
		m.Reply("Usage: /nightmode on/off")
		return nil
	}

	var enable bool
	switch strings.ToLower(args) {
	case "on":
		enable = true
	case "off":
		enable = false
	default:
		m.Reply("Usage: /nightmode on/off")
		return nil
	}

	chat, err := m.Client.GetChat(m.ChatID())
	if err != nil {
		m.Reply("Error fetching chat info")
		return nil
	}

	current := chat.DefaultBannedRights
	if current == nil {
		current = &tg.ChatBannedRights{}
	}

	current.SendMessages = enable

	_, err = m.Client.MessagesEditChatDefaultBannedRights(m.Peer, current)
	if err != nil {
		m.Reply("Failed to toggle night mode: " + err.Error())
		return nil
	}

	if enable {
		m.Reply("Night mode enabled. Messages are restricted.")
	} else {
		m.Reply("Night mode disabled. Messages allowed.")
	}
	return nil
}

var popularLangCodes = []struct {
	Code string
	Name string
}{
	{"en", "English"}, {"es", "Spanish"}, {"fr", "French"}, {"de", "German"},
	{"it", "Italian"}, {"pt", "Portuguese"}, {"ru", "Russian"}, {"zh", "Chinese"},
	{"ja", "Japanese"}, {"ko", "Korean"}, {"ar", "Arabic"}, {"hi", "Hindi"},
	{"bn", "Bengali"}, {"ta", "Tamil"}, {"te", "Telugu"}, {"ml", "Malayalam"},
	{"kn", "Kannada"}, {"mr", "Marathi"}, {"gu", "Gujarati"}, {"pa", "Punjabi"},
	{"ur", "Urdu"}, {"fa", "Persian"}, {"tr", "Turkish"}, {"nl", "Dutch"},
	{"pl", "Polish"}, {"uk", "Ukrainian"}, {"sv", "Swedish"}, {"no", "Norwegian"},
	{"da", "Danish"}, {"fi", "Finnish"}, {"cs", "Czech"}, {"el", "Greek"},
	{"he", "Hebrew"}, {"th", "Thai"}, {"vi", "Vietnamese"}, {"id", "Indonesian"},
	{"ms", "Malay"}, {"ro", "Romanian"}, {"hu", "Hungarian"}, {"sw", "Swahili"},
}

func googleTranslate(text, target string) (string, string, error) {
	api := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=%s&dt=t&q=%s",
		target, url.QueryEscape(text))
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(api)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}
	if len(result) == 0 {
		return "", "", fmt.Errorf("no result")
	}
	chunks, _ := result[0].([]interface{})
	var sb strings.Builder
	for _, c := range chunks {
		if line, ok := c.([]interface{}); ok && len(line) > 0 {
			if s, ok := line[0].(string); ok {
				sb.WriteString(s)
			}
		}
	}
	src := "unknown"
	if len(result) > 2 {
		if s, ok := result[2].(string); ok {
			src = s
		}
	}
	return sb.String(), src, nil
}

func LangsHandler(m *tg.NewMessage) error {
	var sb strings.Builder
	sb.WriteString("<b>Popular Language Codes (ISO 639-1)</b>\n\n")
	for _, l := range popularLangCodes {
		fmt.Fprintf(&sb, "<code>%s</code> - %s\n", l.Code, html.EscapeString(l.Name))
	}
	sb.WriteString("\n<i>Use with /tr &lt;code&gt; replying to a message.</i>")
	m.Reply(sb.String())
	return nil
}

func DetectHandler(m *tg.NewMessage) error {
	text := m.Args()
	if text == "" && m.IsReply() {
		r, _ := m.GetReplyMessage()
		text = r.Text()
	}
	if strings.TrimSpace(text) == "" {
		m.Reply("Provide text or reply to a message: <code>/detect &lt;text&gt;</code>")
		return nil
	}
	translated, code, err := googleTranslate(text, "en")
	if err != nil {
		m.Reply("Detection failed")
		return nil
	}
	name := code
	for _, l := range popularLangCodes {
		if l.Code == code {
			name = l.Name + " (" + code + ")"
			break
		}
	}
	m.Reply(fmt.Sprintf("<b>Detected Language:</b> %s\n\n<b>English:</b>\n<code>%s</code>",
		html.EscapeString(name), html.EscapeString(translated)))
	return nil
}

func TranslateHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a message to translate it")
		return nil
	}
	args := m.Args()
	targetLang := "en"
	replaceMode := false
	if args != "" {
		for _, p := range strings.Fields(args) {
			if p == "-r" {
				replaceMode = true
			} else {
				targetLang = p
			}
		}
	}
	r, _ := m.GetReplyMessage()
	text := r.Text()
	if text == "" {
		m.Reply("No text to translate")
		return nil
	}
	translated, src, err := googleTranslate(text, targetLang)
	if err != nil {
		m.Reply("Translation failed")
		return nil
	}
	if replaceMode && modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		r.Delete()
		m.Delete()
		m.Respond(fmt.Sprintf("<b>Translated from %s:</b>\n%s", src, translated))
	} else {
		m.Reply(fmt.Sprintf("<b>Translated (%s -> %s):</b>\n<code>%s</code>", src, targetLang, translated))
	}
	return nil
}

type autotrConfig struct {
	Enabled bool   `json:"e"`
	Lang    string `json:"l"`
	Min     int    `json:"m"`
}

var (
	autotrBucket = []byte("autotr")
	autotrCache  = make(map[int64]*autotrConfig)
	autotrMu     sync.RWMutex
)

func autotrChatKey(chatID int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(chatID))
	return b
}

func autotrLoad(chatID int64) *autotrConfig {
	autotrMu.RLock()
	if c, ok := autotrCache[chatID]; ok {
		autotrMu.RUnlock()
		return c
	}
	autotrMu.RUnlock()
	cfg := &autotrConfig{Enabled: false, Lang: "en", Min: 4}
	database, err := db.GetDB()
	if err != nil || database == nil {
		return cfg
	}
	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(autotrBucket)
		if b == nil {
			return nil
		}
		raw := b.Get(autotrChatKey(chatID))
		if raw == nil {
			return nil
		}
		var c autotrConfig
		if err := json.Unmarshal(raw, &c); err == nil {
			if c.Lang == "" {
				c.Lang = "en"
			}
			if c.Min <= 0 {
				c.Min = 4
			}
			cfg = &c
		}
		return nil
	})
	autotrMu.Lock()
	autotrCache[chatID] = cfg
	autotrMu.Unlock()
	return cfg
}

func autotrSave(chatID int64, cfg *autotrConfig) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db unavailable")
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	err = database.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(autotrBucket)
		if err != nil {
			return err
		}
		return b.Put(autotrChatKey(chatID), data)
	})
	if err == nil {
		autotrMu.Lock()
		autotrCache[chatID] = cfg
		autotrMu.Unlock()
	}
	return err
}

func AutoTrHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Auto-translate works in groups only.</b>")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		return nil
	}
	args := strings.TrimSpace(m.Args())
	cfg := autotrLoad(m.ChatID())
	if args == "" || args == "status" {
		state := "off"
		if cfg.Enabled {
			state = "on"
		}
		m.Reply(fmt.Sprintf("<b>Auto-Translate</b>\n • State: <code>%s</code>\n • Lang: <code>%s</code>\n • Min chars: <code>%d</code>\n\n<i>Usage:</i>\n <code>/autotr on|off</code>\n <code>/autotr lang &lt;iso&gt;</code>\n <code>/autotr min &lt;chars&gt;</code>",
			state, html.EscapeString(cfg.Lang), cfg.Min))
		return nil
	}
	parts := strings.Fields(args)
	switch strings.ToLower(parts[0]) {
	case "on", "enable":
		cfg.Enabled = true
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Auto-translate enabled.</b> Target: <code>%s</code>", html.EscapeString(cfg.Lang)))
	case "off", "disable":
		cfg.Enabled = false
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply("<b>Auto-translate disabled.</b>")
	case "lang", "language":
		if len(parts) < 2 {
			m.Reply("<b>Usage:</b> <code>/autotr lang &lt;iso&gt;</code>")
			return nil
		}
		lang := strings.ToLower(strings.TrimSpace(parts[1]))
		if len(lang) < 2 || len(lang) > 8 {
			m.Reply("<b>Invalid language code.</b>")
			return nil
		}
		cfg.Lang = lang
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Target language set to</b> <code>%s</code>", html.EscapeString(lang)))
	case "min":
		if len(parts) < 2 {
			m.Reply("<b>Usage:</b> <code>/autotr min &lt;chars&gt;</code>")
			return nil
		}
		var n int
		if _, err := fmt.Sscanf(parts[1], "%d", &n); err != nil || n < 1 || n > 4096 {
			m.Reply("<b>Invalid number.</b> Must be between 1 and 4096.")
			return nil
		}
		cfg.Min = n
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Minimum length set to</b> <code>%d</code>", n))
	default:
		m.Reply("<b>Unknown subcommand.</b> Use <code>on</code>, <code>off</code>, <code>lang</code>, <code>min</code>, or <code>status</code>.")
	}
	return nil
}

func AutoTrWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	if m.Sender != nil && m.Sender.Bot {
		return nil
	}
	if m.Message != nil && m.Message.ViaBotID != 0 {
		return nil
	}
	text := strings.TrimSpace(m.Text())
	if text == "" {
		return nil
	}
	if t := text; len(t) > 0 && (t[0] == '/' || t[0] == '!' || t[0] == '.') {
		return nil
	}
	cfg := autotrLoad(m.ChatID())
	if !cfg.Enabled {
		return nil
	}
	if len([]rune(text)) < cfg.Min {
		return nil
	}
	translated, src, err := googleTranslate(text, cfg.Lang)
	if err != nil || translated == "" {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(src), strings.TrimSpace(cfg.Lang)) {
		return nil
	}
	if strings.TrimSpace(translated) == strings.TrimSpace(text) {
		return nil
	}
	m.Reply(fmt.Sprintf("<blockquote><i>%s→%s</i> %s</blockquote>",
		html.EscapeString(src), html.EscapeString(cfg.Lang), html.EscapeString(translated)))
	return nil
}

type weatherGeocodeResult struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Country   string  `json:"country"`
	Admin1    string  `json:"admin1"`
	Timezone  string  `json:"timezone"`
}

type weatherCurrent struct {
	Time             string  `json:"time"`
	Temperature2m    float64 `json:"temperature_2m"`
	RelativeHumidity int     `json:"relative_humidity_2m"`
	WeatherCode      int     `json:"weather_code"`
	WindSpeed10m     float64 `json:"wind_speed_10m"`
	IsDay            int     `json:"is_day"`
}

type weatherForecastResponse struct {
	Latitude  float64        `json:"latitude"`
	Longitude float64        `json:"longitude"`
	Timezone  string         `json:"timezone"`
	Current   weatherCurrent `json:"current"`
}

func weatherCodeDescription(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Fog"
	case 51, 53, 55:
		return "Drizzle"
	case 56, 57:
		return "Freezing drizzle"
	case 61, 63, 65:
		return "Rain"
	case 66, 67:
		return "Freezing rain"
	case 71, 73, 75, 77:
		return "Snow"
	case 80, 81, 82:
		return "Rain showers"
	case 85, 86:
		return "Snow showers"
	case 95, 96, 99:
		return "Thunderstorm"
	}
	return "Unknown"
}

func WeatherHandler(m *tg.NewMessage) error {
	city := strings.TrimSpace(m.Args())
	if city == "" {
		m.Reply("<b>Usage:</b> <code>/weather &lt;city&gt;</code>")
		return nil
	}
	status, _ := m.Reply("Fetching weather for <code>" + html.EscapeString(city) + "</code>...")

	client := &http.Client{Timeout: 30 * time.Second}

	geoResp, err := client.Get(fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1", url.QueryEscape(city)))
	if err != nil {
		status.Edit("Geocode failed: " + html.EscapeString(err.Error()))
		return nil
	}
	defer geoResp.Body.Close()
	var geoData struct {
		Results []weatherGeocodeResult `json:"results"`
	}
	if err := json.NewDecoder(geoResp.Body).Decode(&geoData); err != nil || len(geoData.Results) == 0 {
		status.Edit("City <code>" + html.EscapeString(city) + "</code> not found.")
		return nil
	}
	geo := &geoData.Results[0]

	forecastResp, err := client.Get(fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m,is_day&timezone=auto", geo.Latitude, geo.Longitude))
	if err != nil {
		status.Edit("Forecast failed: " + html.EscapeString(err.Error()))
		return nil
	}
	defer forecastResp.Body.Close()
	var forecast weatherForecastResponse
	if err := json.NewDecoder(forecastResp.Body).Decode(&forecast); err != nil {
		status.Edit("Forecast parse failed: " + html.EscapeString(err.Error()))
		return nil
	}

	var locParts []string
	if geo.Name != "" {
		locParts = append(locParts, geo.Name)
	}
	if geo.Admin1 != "" && geo.Admin1 != geo.Name {
		locParts = append(locParts, geo.Admin1)
	}
	if geo.Country != "" {
		locParts = append(locParts, geo.Country)
	}

	var sb strings.Builder
	sb.WriteString("<b>Weather</b>\n\n")
	sb.WriteString("<b>Location:</b> ")
	sb.WriteString(html.EscapeString(strings.Join(locParts, ", ")))
	sb.WriteString("\n")
	sb.WriteString("<b>Condition:</b> ")
	sb.WriteString(html.EscapeString(weatherCodeDescription(forecast.Current.WeatherCode)))
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "<b>Temperature:</b> <code>%.1f°C</code>\n", forecast.Current.Temperature2m)
	fmt.Fprintf(&sb, "<b>Humidity:</b> <code>%d%%</code>\n", forecast.Current.RelativeHumidity)
	fmt.Fprintf(&sb, "<b>Wind:</b> <code>%.1f km/h</code>\n", forecast.Current.WindSpeed10m)
	fmt.Fprintf(&sb, "<b>Coords:</b> <code>%.4f, %.4f</code>\n", geo.Latitude, geo.Longitude)
	if forecast.Timezone != "" {
		sb.WriteString("<b>Timezone:</b> <code>")
		sb.WriteString(html.EscapeString(forecast.Timezone))
		sb.WriteString("</code>\n")
	}
	status.Edit(sb.String())
	return nil
}

func registerMiscHandlers() {
	c := modules.Client
	c.On("cmd:help", modules.HelpHandle)
	c.On("cmd:nightmode", NightModeHandler)
	c.On("cmd:tempnote", SaveTempNoteHandler)
	c.On("callback:verify_op_", modules.AdminVerifyCallback)
	c.On("callback:help_back", modules.HelpBackCallback)
	c.On("cmd:gban", Gban, tg.CustomFilter(modules.FilterOwner))
	c.On("cmd:ungban", Ungban, tg.CustomFilter(modules.FilterOwner))
	c.On("cmd:tr", TranslateHandler)
	c.On("cmd:langs", LangsHandler)
	c.On("cmd:detect", DetectHandler)
	c.On("cmd:autotr", AutoTrHandler)
	c.On(tg.OnNewMessage, AutoTrWatcher)
	c.On("cmd:weather", WeatherHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerMiscHandlers)
}
