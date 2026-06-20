package modules

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var ttsPlayLangCodes = map[string]bool{
	"af": true, "ar": true, "bg": true, "bn": true, "bs": true, "ca": true,
	"cs": true, "cy": true, "da": true, "de": true, "el": true, "en": true,
	"eo": true, "es": true, "et": true, "fi": true, "fr": true, "gu": true,
	"hi": true, "hr": true, "hu": true, "hy": true, "id": true, "is": true,
	"it": true, "ja": true, "jw": true, "km": true, "kn": true, "ko": true,
	"la": true, "lv": true, "mk": true, "ml": true, "mr": true, "my": true,
	"ne": true, "nl": true, "no": true, "pl": true, "pt": true, "ro": true,
	"ru": true, "si": true, "sk": true, "sq": true, "sr": true, "su": true,
	"sv": true, "sw": true, "ta": true, "te": true, "th": true, "tl": true,
	"tr": true, "uk": true, "ur": true, "vi": true, "zh": true,
}

func TTSPlayHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			args = strings.TrimSpace(r.Text())
		}
	}
	if args == "" {
		m.Reply("usage: <code>/tts &lt;text&gt;</code> or <code>/tts &lt;lang&gt; &lt;text&gt;</code>")
		return nil
	}

	lang := "en"
	text := args
	parts := strings.SplitN(args, " ", 2)
	if len(parts) == 2 {
		candidate := strings.ToLower(strings.TrimSpace(parts[0]))
		if len(candidate) == 2 && ttsPlayLangCodes[candidate] {
			lang = candidate
			text = strings.TrimSpace(parts[1])
		}
	}

	if text == "" {
		m.Reply("usage: <code>/tts &lt;text&gt;</code> or <code>/tts &lt;lang&gt; &lt;text&gt;</code>")
		return nil
	}

	if len([]rune(text)) > 200 {
		text = string([]rune(text)[:200])
	}

	endpoint := fmt.Sprintf(
		"https://translate.google.com/translate_tts?ie=UTF-8&q=%s&tl=%s&client=tw-ob",
		url.QueryEscape(text), url.QueryEscape(lang),
	)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		m.Reply("error: " + html.EscapeString(err.Error()))
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
	req.Header.Set("Referer", "https://translate.google.com/")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.Reply("error fetching tts: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.Reply(fmt.Sprintf("tts api returned status %d", resp.StatusCode))
		return nil
	}

	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("tts_%d.mp3", time.Now().UnixNano()))
	out, err := os.Create(tmpPath)
	if err != nil {
		m.Reply("error creating temp file: " + html.EscapeString(err.Error()))
		return nil
	}

	if _, err := io.Copy(out, io.LimitReader(resp.Body, 5*1024*1024)); err != nil {
		out.Close()
		os.Remove(tmpPath)
		m.Reply("error writing tts: " + html.EscapeString(err.Error()))
		return nil
	}
	out.Close()
	defer os.Remove(tmpPath)

	caption := "<b>TTS</b> [<code>" + html.EscapeString(lang) + "</code>]\n" + html.EscapeString(text)
	_, err = m.ReplyMedia(tmpPath, &tg.MediaOptions{Caption: caption})
	if err != nil {
		m.Reply("error sending tts: " + html.EscapeString(err.Error()))
		return nil
	}
	return nil
}

func init() { QueueHandlerRegistration(registerTTSPlayHandlers) }

func registerTTSPlayHandlers() {
	c := Client
	c.On("cmd:tts", TTSPlayHandler)
}
