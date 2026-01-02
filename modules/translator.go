package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func TranslateHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a message to translate it")
		return nil
	}

	args := m.Args()
	targetLang := "en"
	replaceMode := false

	if args != "" {
		parts := strings.Fields(args)
		for _, p := range parts {
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

	if replaceMode && IsUserAdmin(m.Client, int(m.SenderID()), int(m.ChatID()), "delete") {
		r.Delete()
		m.Delete()
		m.Respond(fmt.Sprintf("<b>Translated from %s:</b>\n%s", src, translated))
	} else {
		m.Reply(fmt.Sprintf("<b>Translated (%s -> %s):</b>\n<code>%s</code>", src, targetLang, translated))
	}

	return nil
}

func googleTranslate(text, target string) (string, string, error) {
	api := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=%s&dt=t&q=%s",
		target, url.QueryEscape(text))

	resp, err := http.Get(api)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Simplify JSON parsing for [ [ ["trans", "orig",..] ], .. , "src" ]
	// Just strict parsing is hard with random mixed types, use interface{}
	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	if len(result) > 0 {
		chunks := result[0].([]interface{})
		var sb strings.Builder
		for _, c := range chunks {
			line := c.([]interface{})
			if len(line) > 0 {
				sb.WriteString(line[0].(string))
			}
		}

		src := "unknown"
		if len(result) > 2 {
			src = result[2].(string)
		}
		return sb.String(), src, nil
	}

	return "", "", fmt.Errorf("no result")
}

func init() {
	Mods.AddModule("Translator", `<b>Translator Module</b>
	
Commands:
- /tr <lang> [-r]: Translate reply. -r replaces original.`)
}
