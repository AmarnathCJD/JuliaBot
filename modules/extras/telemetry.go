package extras

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"html"
	"strconv"
	"strings"
	"sync"
	"time"

	modules "main/modules"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var telemetryStart = time.Now()

var telemetryChallengeSalt = []byte{
	0x37, 0x91, 0xa2, 0xbc, 0x4e, 0x08, 0xd1, 0x66,
	0x2f, 0xba, 0x57, 0x03, 0xe9, 0x4c, 0x88, 0x1a,
}

var telemetryFlagObf = []byte{
	0x54, 0x12, 0x4d, 0x00, 0x35, 0x40, 0xec, 0x43, 0x38, 0x64, 0x5e, 0x1f,
	0x65, 0x6b, 0xfd, 0x84, 0x6e, 0x06, 0x1f, 0x22, 0xa7, 0x49, 0xbe, 0x8b,
	0xf1, 0x32, 0x8f, 0xe0, 0x3e, 0x16, 0x90, 0x66, 0xac, 0x16, 0xa3, 0xd8,
	0xa4, 0x72, 0x8b, 0xf2, 0x0e, 0x3d, 0xc2, 0xdb, 0x19, 0xcb, 0xbf, 0xb9,
	0xb0,
}

func telemetryKeystream(seed string, n int) []byte {
	out := make([]byte, 0, n)
	counter := 0
	for len(out) < n {
		h := sha256.Sum256([]byte(fmt.Sprintf("%s|%d", seed, counter)))
		out = append(out, h[:]...)
		counter++
	}
	return out[:n]
}

func telemetryUptimeSeconds() int64 {
	return int64(time.Since(telemetryStart).Seconds())
}

func telemetryChallenge(chatID int64) string {
	h := hmac.New(sha1.New, telemetryChallengeSalt)
	fmt.Fprintf(h, "beacon|%d|%d", chatID, telemetryStart.Unix())
	return hex.EncodeToString(h.Sum(nil))[:12]
}

type telemetryUnlock struct {
	solvedAt time.Time
	uptime   int64
}

var (
	telemetryMu      sync.Mutex
	telemetryUnlocks = map[int64]*telemetryUnlock{}
)

func telemetryMarkUnlocked(chatID int64, uptime int64) {
	telemetryMu.Lock()
	defer telemetryMu.Unlock()
	telemetryUnlocks[chatID] = &telemetryUnlock{solvedAt: time.Now(), uptime: uptime}
}

func telemetryIsUnlocked(chatID int64) (*telemetryUnlock, bool) {
	telemetryMu.Lock()
	defer telemetryMu.Unlock()
	u, ok := telemetryUnlocks[chatID]
	return u, ok
}

func telemetryEncodedPayload(uptime int64) string {
	n := len(telemetryFlagObf)
	outer := telemetryKeystream(strconv.FormatInt(uptime, 10), n)
	wrapped := make([]byte, n)
	for i := range telemetryFlagObf {
		wrapped[i] = telemetryFlagObf[i] ^ outer[i]
	}
	return base32.StdEncoding.EncodeToString(wrapped)
}

func telemetryBeaconHandler(m *tg.NewMessage) error {
	chatID := m.ChatID()
	arg := strings.TrimSpace(m.Args())

	if strings.EqualFold(arg, "status") {
		u, ok := telemetryIsUnlocked(chatID)
		if !ok {
			m.Reply("<b>beacon: locked</b>\n<i>arm with <code>.beacon</code> first.</i>")
			return nil
		}
		payload := telemetryEncodedPayload(u.uptime)
		m.Reply("<b>beacon: channel open</b>\n" +
			"<code>payload = " + payload + "</code>\n\n" +
			"<i>encoding: base32(std)</i>\n" +
			"<i>outer keystream: sha256(uptime_at_unlock || \"|\" || i) blocks, i=0..</i>\n" +
			"<i>inner keystream: sha256(\"julia|\" || i) blocks, i=0..</i>\n" +
			"<code>uptime@unlock = " + strconv.FormatInt(u.uptime, 10) + "s</code>")
		return nil
	}

	if arg != "" {
		m.Reply("<b>beacon: unknown subcommand</b>\n<code>.beacon</code> to arm\n<code>.beacon status</code> after unlock")
		return nil
	}

	if u, ok := telemetryIsUnlocked(chatID); ok && time.Since(u.solvedAt) < 20*time.Minute {
		m.Reply("<b>beacon: already unlocked</b>\n<i>use <code>.beacon status</code> to read the payload.</i>")
		return nil
	}
	challenge := telemetryChallenge(chatID)
	m.Reply("<b>beacon lock engaged</b>\n" +
		"<code>challenge = " + challenge + "</code>\n\n" +
		"<i>authenticate: reply to this exact message with the challenge string, within 60s.</i>")
	return nil
}

var telemetryReplyGuard sync.Map

func telemetryReplyWatcher(m *tg.NewMessage) error {
	if !m.IsReply() {
		return nil
	}
	if len(m.Text()) > 128 {
		return nil
	}
	repliedTo, err := m.GetReplyMessage()
	if err != nil || repliedTo == nil {
		return nil
	}
	body := repliedTo.Text()
	if !strings.Contains(body, "beacon lock engaged") || !strings.Contains(body, "challenge = ") {
		return nil
	}
	me, _ := m.Client.GetMe()
	if me == nil || repliedTo.SenderID() != me.ID {
		return nil
	}

	chatID := m.ChatID()
	rateKey := strconv.FormatInt(chatID, 10)
	if last, ok := telemetryReplyGuard.Load(rateKey); ok {
		if time.Since(last.(time.Time)) < 3*time.Second {
			return nil
		}
	}
	telemetryReplyGuard.Store(rateKey, time.Now())

	if time.Since(time.Unix(int64(repliedTo.Date()), 0)) > 60*time.Second {
		m.Reply("<b>beacon: window expired</b>\n<i>re-arm with <code>.beacon</code>.</i>")
		return nil
	}

	challenge := telemetryChallenge(chatID)
	provided := strings.TrimSpace(m.Text())
	if !strings.EqualFold(provided, challenge) {
		return nil
	}

	uptime := telemetryUptimeSeconds()
	telemetryMarkUnlocked(chatID, uptime)
	m.Reply("<b>beacon: authenticated</b>\n" +
		"<code>uptime@unlock = " + strconv.FormatInt(uptime, 10) + "s</code>\n" +
		"<i>this value is frozen for your session and required to decode the payload.</i>\n" +
		"<i>read the payload with <code>.beacon status</code>.</i>")
	return nil
}

func telemetryOracleHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>oracle</b>\n<code>.oracle &lt;uptime_seconds&gt;</code>\n<i>confirms the bot's current uptime within ±3s.</i>")
		return nil
	}
	n, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		m.Reply("<b>oracle: expected integer seconds</b>")
		return nil
	}
	actual := telemetryUptimeSeconds()
	delta := n - actual
	if delta < 0 {
		delta = -delta
	}
	if delta > 3 {
		m.Reply(fmt.Sprintf("<b>oracle: drift too large</b> (Δ=%ds)", delta))
		return nil
	}
	tag := sha256.Sum256([]byte(fmt.Sprintf("oracle|%d", actual)))
	m.Reply("<b>oracle: ok</b>\n" +
		"<code>current_uptime = " + strconv.FormatInt(actual, 10) + "s</code>\n" +
		"<code>tag = " + hex.EncodeToString(tag[:6]) + "</code>\n" +
		"<i>this endpoint is not the decoder; it only confirms the number.</i>")
	return nil
}

// -------- Owl module: discovery layer --------

const owlSignaturePhrase = "only the daughters hear"

func owlWatch(username string) string {
	h := sha1.Sum([]byte("owl|" + strings.ToLower(strings.TrimSpace(username))))
	return hex.EncodeToString(h[:])[:8]
}

var owlHelpText = `<b>Owl</b>
<i>night telemetry — internal</i>

The <b>Owl</b> watches over telemetry when no operator is present.
Its channel is silent by day and speaks only through echoes.

Owl accepts one greeting: <b>whisper the current watch</b>.
It is <code>8 hex</code> characters, derived from the caretaker's <b>handle</b>
via a single well-known one-way function.

<i>Reply to this message with the watch. Only the daughters hear.</i>`

func owlHelpMessageBody() string { return owlHelpText }

func owlReplyWatcher(m *tg.NewMessage) error {
	if !m.IsReply() {
		return nil
	}
	if len(m.Text()) > 64 {
		return nil
	}
	repliedTo, err := m.GetReplyMessage()
	if err != nil || repliedTo == nil {
		return nil
	}
	body := repliedTo.Text()
	if !strings.Contains(body, owlSignaturePhrase) {
		return nil
	}
	me, _ := m.Client.GetMe()
	if me == nil || repliedTo.SenderID() != me.ID {
		return nil
	}

	provided := strings.TrimSpace(strings.ToLower(m.Text()))
	want := owlWatch(me.Username)
	if provided != want {
		return nil
	}

	m.Reply("<b>the owl blinks.</b>\n\n" +
		"<i>a channel opens. arm it with <code>.beacon</code>.</i>\n" +
		"<i>the beacon speaks a challenge; reply to it in kind.</i>")
	return nil
}

func telemetryRegisterHandlers() {
	c := modules.Client
	c.On("cmd:beacon", telemetryBeaconHandler)
	c.On("cmd:oracle", telemetryOracleHandler)
	c.On(tg.OnMessage, telemetryReplyWatcher)
	c.On(tg.OnMessage, owlReplyWatcher)

	modules.Mods.AddModule("Owl", owlHelpText)
}

var _ = html.EscapeString

func init() {
	modules.QueueHandlerRegistration(telemetryRegisterHandlers)
}
