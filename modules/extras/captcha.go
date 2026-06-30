package extras

import (
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/fogleman/gg"
	bolt "go.etcd.io/bbolt"
	modules "main/modules"
)

const (
	captchaBucket   = "captcha_cfg"
	captchaTypeBtn  = "button"
	captchaTypeMath = "math"
	captchaTypeImg  = "image"
	captchaPrefix   = "cap:"
)

type CaptchaConfig struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
	Timeout int    `json:"timeout"`
}

type captchaState struct {
	UserID    int64
	ChatID    int64
	Type      string
	Answer    string
	MessageID int32
	Deadline  time.Time
	Cancel    chan struct{}
	UserName  string
}

var captchaStates sync.Map

func captchaStateKey(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func defaultCaptchaConfig() *CaptchaConfig {
	return &CaptchaConfig{Enabled: false, Type: captchaTypeBtn, Timeout: 60}
}

func getCaptchaConfig(chatID int64) *CaptchaConfig {
	cfg := defaultCaptchaConfig()
	database, err := db.GetDB()
	if err != nil || database == nil {
		return cfg
	}
	_ = database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(captchaBucket))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(strconv.FormatInt(chatID, 10)))
		if data == nil {
			return nil
		}
		_ = json.Unmarshal(data, cfg)
		return nil
	})
	if cfg.Type != captchaTypeBtn && cfg.Type != captchaTypeMath && cfg.Type != captchaTypeImg {
		cfg.Type = captchaTypeBtn
	}
	if cfg.Timeout < 15 || cfg.Timeout > 600 {
		cfg.Timeout = 60
	}
	return cfg
}

func saveCaptchaConfig(chatID int64, cfg *CaptchaConfig) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db unavailable")
	}
	return database.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(captchaBucket))
		if err != nil {
			return err
		}
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return b.Put([]byte(strconv.FormatInt(chatID, 10)), data)
	})
}

func CaptchaEnabled(chatID int64) bool {
	return getCaptchaConfig(chatID).Enabled
}

func captchaLoadFont(dc *gg.Context, size float64) {
	name := modules.GetRandomFont()
	candidates := []string{
		"./assets/" + name,
		"assets/" + name,
		"../assets/" + name,
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "assets", name))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "assets", name),
			filepath.Join(dir, "..", "assets", name),
		)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if err := dc.LoadFontFace(p, size); err == nil {
				return
			}
		}
	}
}

func captchaDrawNoise(dc *gg.Context, w, h int) {
	for i := 0; i < 1200; i++ {
		x := rand.Float64() * float64(w)
		y := rand.Float64() * float64(h)
		dc.SetRGBA(rand.Float64(), rand.Float64(), rand.Float64(), 0.35)
		dc.DrawPoint(x, y, 1.2)
		dc.Fill()
	}
	for i := 0; i < 8; i++ {
		dc.SetRGBA(rand.Float64(), rand.Float64(), rand.Float64(), 0.45)
		dc.SetLineWidth(1 + rand.Float64()*2)
		dc.DrawLine(
			rand.Float64()*float64(w), rand.Float64()*float64(h),
			rand.Float64()*float64(w), rand.Float64()*float64(h),
		)
		dc.Stroke()
	}
}

func captchaRenderText(text string, w, h int, fontSize float64) (string, error) {
	dc := gg.NewContext(w, h)
	dc.SetRGB(0.95, 0.95, 0.95)
	dc.Clear()

	captchaDrawNoise(dc, w, h)

	captchaLoadFont(dc, fontSize)
	chars := []rune(text)
	if len(chars) == 0 {
		return "", fmt.Errorf("empty text")
	}
	cellW := float64(w) / float64(len(chars)+1)
	for i, ch := range chars {
		dc.Push()
		x := cellW*float64(i+1) + (rand.Float64()-0.5)*6
		y := float64(h)/2 + (rand.Float64()-0.5)*10
		angle := (rand.Float64() - 0.5) * 0.6
		dc.RotateAbout(angle, x, y)
		dc.SetRGB(rand.Float64()*0.4, rand.Float64()*0.4, rand.Float64()*0.4)
		dc.DrawStringAnchored(string(ch), x, y, 0.5, 0.5)
		dc.Pop()
	}

	for i := 0; i < 4; i++ {
		dc.SetRGBA(rand.Float64()*0.6, rand.Float64()*0.6, rand.Float64()*0.6, 0.4)
		dc.SetLineWidth(1.2)
		dc.DrawLine(
			rand.Float64()*float64(w), rand.Float64()*float64(h),
			rand.Float64()*float64(w), rand.Float64()*float64(h),
		)
		dc.Stroke()
	}

	out := filepath.Join(os.TempDir(), fmt.Sprintf("captcha_%d.png", time.Now().UnixNano()))
	if err := dc.SavePNG(out); err != nil {
		return "", err
	}
	return out, nil
}

func captchaRandomString(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

func captchaGenMath() (string, string) {
	a := rand.Intn(20) + 1
	b := rand.Intn(20) + 1
	switch rand.Intn(3) {
	case 0:
		return fmt.Sprintf("%d + %d = ?", a, b), strconv.Itoa(a + b)
	case 1:
		if a < b {
			a, b = b, a
		}
		return fmt.Sprintf("%d - %d = ?", a, b), strconv.Itoa(a - b)
	default:
		x := rand.Intn(9) + 2
		y := rand.Intn(9) + 2
		return fmt.Sprintf("%d x %d = ?", x, y), strconv.Itoa(x * y)
	}
}

func captchaRestrictUser(c *tg.Client, chatID int64, userID int64, restrict bool) {
	user, err := c.ResolvePeer(userID)
	if err != nil || user == nil {
		return
	}
	if restrict {
		_, _ = c.EditBanned(chatID, user, &tg.BannedOptions{Mute: true})
	} else {
		_, _ = c.EditBanned(chatID, user, &tg.BannedOptions{Unmute: true})
	}
}

func captchaKickUser(c *tg.Client, chatID int64, userID int64) {
	user, err := c.ResolvePeer(userID)
	if err != nil || user == nil {
		return
	}
	_, _ = c.EditBanned(chatID, user, &tg.BannedOptions{Ban: true})
	_, _ = c.EditBanned(chatID, user, &tg.BannedOptions{Unban: true})
}

func captchaCleanup(chatID, userID int64, c *tg.Client, msgID int32) {
	key := captchaStateKey(chatID, userID)
	captchaStates.Delete(key)
	if msgID > 0 {
		_, _ = c.DeleteMessages(chatID, []int32{msgID})
	}
}

func captchaScheduleTimeout(c *tg.Client, st *captchaState) {
	go func() {
		select {
		case <-time.After(time.Until(st.Deadline)):
			if _, ok := captchaStates.Load(captchaStateKey(st.ChatID, st.UserID)); !ok {
				return
			}
			captchaKickUser(c, st.ChatID, st.UserID)
			captchaCleanup(st.ChatID, st.UserID, c, st.MessageID)
			_, _ = c.SendMessage(st.ChatID, fmt.Sprintf("%s failed the captcha and was removed.", html.EscapeString(st.UserName)))
		case <-st.Cancel:
			return
		}
	}()
}

func CaptchaParticipantHandler(p *tg.ParticipantUpdate) error {
	if !p.IsJoined() && !p.IsAdded() {
		return nil
	}
	if p.User == nil || p.User.Bot {
		return nil
	}

	chatID := p.ChatID()
	cfg := getCaptchaConfig(chatID)
	if !cfg.Enabled {
		return nil
	}

	userID := p.User.ID
	key := captchaStateKey(chatID, userID)
	if _, exists := captchaStates.Load(key); exists {
		return nil
	}

	captchaRestrictUser(p.Client, chatID, userID, true)

	userName := p.User.FirstName
	if userName == "" {
		userName = "user"
	}
	mention := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", userID, html.EscapeString(userName))

	deadline := time.Now().Add(time.Duration(cfg.Timeout) * time.Second)

	st := &captchaState{
		UserID:   userID,
		ChatID:   chatID,
		Type:     cfg.Type,
		Deadline: deadline,
		Cancel:   make(chan struct{}),
		UserName: userName,
	}

	switch cfg.Type {
	case captchaTypeBtn:
		st.Answer = "ok"
		b := tg.Button
		kb := tg.NewKeyboard().AddRow(
			b.Data("I'm not a bot", fmt.Sprintf("%sbtn:%d:%d", captchaPrefix, chatID, userID)),
		).Build()
		msg, err := p.Client.SendMessage(chatID, fmt.Sprintf(
			"Welcome %s! Please click the button below within %d seconds to verify you are human.",
			mention, cfg.Timeout,
		), &tg.SendOptions{ReplyMarkup: kb})
		if err == nil && msg != nil {
			st.MessageID = msg.ID
		}

	case captchaTypeMath:
		question, answer := captchaGenMath()
		st.Answer = answer
		img, err := captchaRenderText(question, 360, 140, 36)
		if err != nil {
			captchaRestrictUser(p.Client, chatID, userID, false)
			return nil
		}
		defer os.Remove(img)

		correct, _ := strconv.Atoi(answer)
		opts := []int{correct}
		for len(opts) < 4 {
			delta := rand.Intn(11) - 5
			if delta == 0 {
				delta = 1
			}
			candidate := correct + delta
			if candidate < 0 {
				candidate = correct + rand.Intn(5) + 1
			}
			dup := false
			for _, o := range opts {
				if o == candidate {
					dup = true
					break
				}
			}
			if !dup {
				opts = append(opts, candidate)
			}
		}
		rand.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })

		b := tg.Button
		kb := tg.NewKeyboard()
		row := []tg.KeyboardButton{}
		for i, o := range opts {
			label := strconv.Itoa(o)
			data := fmt.Sprintf("%smath:%d:%d:%s", captchaPrefix, chatID, userID, label)
			row = append(row, b.Data(label, data))
			if (i+1)%2 == 0 {
				kb.AddRow(row...)
				row = []tg.KeyboardButton{}
			}
		}
		if len(row) > 0 {
			kb.AddRow(row...)
		}

		msg, err := p.Client.SendMedia(chatID, img, &tg.MediaOptions{
			Caption: fmt.Sprintf(
				"Welcome %s! Solve the captcha within %d seconds. Pick the correct answer below.",
				mention, cfg.Timeout,
			),
			FileName:    "captcha.png",
			MimeType:    "image/png",
			ReplyMarkup: kb.Build(),
		})
		if err == nil && msg != nil {
			st.MessageID = msg.ID
		}

	case captchaTypeImg:
		text := captchaRandomString(5)
		st.Answer = text
		img, err := captchaRenderText(text, 320, 120, 44)
		if err != nil {
			captchaRestrictUser(p.Client, chatID, userID, false)
			return nil
		}
		defer os.Remove(img)

		msg, err := p.Client.SendMedia(chatID, img, &tg.MediaOptions{
			Caption: fmt.Sprintf(
				"Welcome %s! Reply to this message with the 5 characters shown within %d seconds (case-insensitive).",
				mention, cfg.Timeout,
			),
			FileName: "captcha.png",
			MimeType: "image/png",
		})
		if err == nil && msg != nil {
			st.MessageID = msg.ID
		}
	}

	captchaStates.Store(key, st)
	captchaScheduleTimeout(p.Client, st)
	return nil
}

func CaptchaCallbackHandler(c *tg.CallbackQuery) error {
	data := c.DataString()
	if !strings.HasPrefix(data, captchaPrefix) {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(data, captchaPrefix), ":")
	if len(parts) < 3 {
		return nil
	}
	kind := parts[0]
	chatID, _ := strconv.ParseInt(parts[1], 10, 64)
	userID, _ := strconv.ParseInt(parts[2], 10, 64)

	if c.SenderID != userID {
		c.Answer("This captcha is not for you.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	key := captchaStateKey(chatID, userID)
	raw, ok := captchaStates.Load(key)
	if !ok {
		c.Answer("Captcha expired.", &tg.CallbackOptions{Alert: true})
		return nil
	}
	st := raw.(*captchaState)

	switch kind {
	case "btn":
		captchaRestrictUser(c.Client, chatID, userID, false)
		close(st.Cancel)
		captchaCleanup(chatID, userID, c.Client, st.MessageID)
		c.Answer("Verified!", &tg.CallbackOptions{Alert: false})
	case "math":
		if len(parts) < 4 {
			return nil
		}
		choice := parts[3]
		if choice == st.Answer {
			captchaRestrictUser(c.Client, chatID, userID, false)
			close(st.Cancel)
			captchaCleanup(chatID, userID, c.Client, st.MessageID)
			c.Answer("Verified!", &tg.CallbackOptions{Alert: false})
		} else {
			captchaKickUser(c.Client, chatID, userID)
			close(st.Cancel)
			captchaCleanup(chatID, userID, c.Client, st.MessageID)
			c.Answer("Wrong answer. Removed.", &tg.CallbackOptions{Alert: true})
			_, _ = c.Client.SendMessage(chatID, fmt.Sprintf("%s failed the captcha and was removed.", html.EscapeString(st.UserName)))
		}
	}
	return nil
}

func CaptchaMessageWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	senderID := m.SenderID()
	if senderID == 0 {
		return nil
	}
	key := captchaStateKey(m.ChatID(), senderID)
	raw, ok := captchaStates.Load(key)
	if !ok {
		return nil
	}
	st := raw.(*captchaState)
	if st.Type != captchaTypeImg {
		return nil
	}
	text := strings.ToUpper(strings.TrimSpace(m.Text()))
	if text == "" {
		return nil
	}
	if text == strings.ToUpper(st.Answer) {
		captchaRestrictUser(m.Client, m.ChatID(), senderID, false)
		close(st.Cancel)
		captchaCleanup(m.ChatID(), senderID, m.Client, st.MessageID)
		_, _ = m.Client.DeleteMessages(m.ChatID(), []int32{int32(m.ID)})
		reply, _ := m.Client.SendMessage(m.ChatID(), fmt.Sprintf("%s verified!", html.EscapeString(st.UserName)))
		if reply != nil {
			go func(id int32) {
				time.Sleep(10 * time.Second)
				_, _ = m.Client.DeleteMessages(m.ChatID(), []int32{id})
			}(reply.ID)
		}
	} else {
		captchaKickUser(m.Client, m.ChatID(), senderID)
		close(st.Cancel)
		captchaCleanup(m.ChatID(), senderID, m.Client, st.MessageID)
		_, _ = m.Client.DeleteMessages(m.ChatID(), []int32{int32(m.ID)})
		_, _ = m.Client.SendMessage(m.ChatID(), fmt.Sprintf("%s failed the captcha and was removed.", html.EscapeString(st.UserName)))
	}
	return nil
}

func CaptchaCommandHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Captcha is only available in groups")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info permission to manage captcha")
		return nil
	}

	args := strings.Fields(m.Args())
	cfg := getCaptchaConfig(m.ChatID())

	if len(args) == 0 {
		status := "off"
		if cfg.Enabled {
			status = "on"
		}
		m.Reply(fmt.Sprintf(`<b>Captcha Settings</b>

Status: <b>%s</b>
Type: <b>%s</b>
Timeout: <b>%d seconds</b>

<b>Usage:</b>
/captcha on|off|status
/captcha type &lt;button|math|image&gt;
/captcha timeout &lt;seconds&gt;`, status, html.EscapeString(cfg.Type), cfg.Timeout))
		return nil
	}

	switch strings.ToLower(args[0]) {
	case "on", "enable", "yes", "1":
		cfg.Enabled = true
		if err := saveCaptchaConfig(m.ChatID(), cfg); err != nil {
			m.Reply("Failed to enable captcha")
			return nil
		}
		m.Reply(fmt.Sprintf("Captcha <b>enabled</b>\nType: <b>%s</b>\nTimeout: <b>%d s</b>", html.EscapeString(cfg.Type), cfg.Timeout))
	case "off", "disable", "no", "0":
		cfg.Enabled = false
		if err := saveCaptchaConfig(m.ChatID(), cfg); err != nil {
			m.Reply("Failed to disable captcha")
			return nil
		}
		m.Reply("Captcha <b>disabled</b>")
	case "status":
		status := "off"
		if cfg.Enabled {
			status = "on"
		}
		m.Reply(fmt.Sprintf(`<b>Captcha Status</b>

State: <b>%s</b>
Type: <b>%s</b>
Timeout: <b>%d seconds</b>`, status, html.EscapeString(cfg.Type), cfg.Timeout))
	case "type":
		if len(args) < 2 {
			m.Reply("Usage: /captcha type &lt;button|math|image&gt;")
			return nil
		}
		t := strings.ToLower(args[1])
		if t != captchaTypeBtn && t != captchaTypeMath && t != captchaTypeImg {
			m.Reply("Invalid type. Options: button, math, image")
			return nil
		}
		cfg.Type = t
		if err := saveCaptchaConfig(m.ChatID(), cfg); err != nil {
			m.Reply("Failed to update captcha type")
			return nil
		}
		m.Reply(fmt.Sprintf("Captcha type set to <b>%s</b>", html.EscapeString(t)))
	case "timeout":
		if len(args) < 2 {
			m.Reply("Usage: /captcha timeout &lt;seconds&gt;")
			return nil
		}
		secs, err := strconv.Atoi(args[1])
		if err != nil || secs < 15 || secs > 600 {
			m.Reply("Invalid timeout. Must be between 15 and 600 seconds")
			return nil
		}
		cfg.Timeout = secs
		if err := saveCaptchaConfig(m.ChatID(), cfg); err != nil {
			m.Reply("Failed to update captcha timeout")
			return nil
		}
		m.Reply(fmt.Sprintf("Captcha timeout set to <b>%d seconds</b>", secs))
	default:
		m.Reply("Usage: /captcha on|off|status | type &lt;button|math|image&gt; | timeout &lt;seconds&gt;")
	}
	return nil
}

func registerCaptchaHandlers() {
	c := modules.Client
	c.On("cmd:captcha", CaptchaCommandHandler)
	c.On("callback:"+captchaPrefix, CaptchaCallbackHandler)
	c.On(tg.OnParticipant, CaptchaParticipantHandler)
	c.On(tg.OnNewMessage, CaptchaMessageWatcher)
}

func init() {
	modules.QueueHandlerRegistration(registerCaptchaHandlers)

	modules.Mods.AddModule("Captcha", `<b>Captcha Module</b>

Challenges new members with a captcha to verify they are human. Failed or timed-out users are removed.

<b>Commands:</b>
 /captcha on|off|status - Toggle or view captcha
 /captcha type &lt;button|math|image&gt; - Choose captcha challenge style
 /captcha timeout &lt;seconds&gt; - Set verification window (15-600)

<b>Types:</b>
 button - Click a single inline button
 math   - Solve an arithmetic question via 4 inline buttons
 image  - Reply with the 5-character code shown in a noisy image

<b>Defaults:</b>
 Type: button, Timeout: 60 s`)
}
