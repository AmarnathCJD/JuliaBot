package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var popularLangCodes = []struct {
	Code string
	Name string
}{
	{"en", "English"},
	{"es", "Spanish"},
	{"fr", "French"},
	{"de", "German"},
	{"it", "Italian"},
	{"pt", "Portuguese"},
	{"ru", "Russian"},
	{"zh", "Chinese"},
	{"ja", "Japanese"},
	{"ko", "Korean"},
	{"ar", "Arabic"},
	{"hi", "Hindi"},
	{"bn", "Bengali"},
	{"ta", "Tamil"},
	{"te", "Telugu"},
	{"ml", "Malayalam"},
	{"kn", "Kannada"},
	{"mr", "Marathi"},
	{"gu", "Gujarati"},
	{"pa", "Punjabi"},
	{"ur", "Urdu"},
	{"fa", "Persian"},
	{"tr", "Turkish"},
	{"nl", "Dutch"},
	{"pl", "Polish"},
	{"uk", "Ukrainian"},
	{"sv", "Swedish"},
	{"no", "Norwegian"},
	{"da", "Danish"},
	{"fi", "Finnish"},
	{"cs", "Czech"},
	{"el", "Greek"},
	{"he", "Hebrew"},
	{"th", "Thai"},
	{"vi", "Vietnamese"},
	{"id", "Indonesian"},
	{"ms", "Malay"},
	{"ro", "Romanian"},
	{"hu", "Hungarian"},
	{"sw", "Swahili"},
}

func LangsHandler(m *tg.NewMessage) error {
	var sb strings.Builder
	sb.WriteString("<b>Popular Language Codes (ISO 639-1)</b>\n\n")
	for _, l := range popularLangCodes {
		sb.WriteString(fmt.Sprintf("<code>%s</code> - %s\n", l.Code, html.EscapeString(l.Name)))
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

	code, confidence, translated, err := googleDetectLang(text)
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

	confPct := fmt.Sprintf("%.1f%%", confidence*100)
	out := fmt.Sprintf("<b>Detected Language:</b> %s\n<b>Confidence:</b> %s\n\n<b>English:</b>\n<code>%s</code>",
		html.EscapeString(name), confPct, html.EscapeString(translated))
	m.Reply(out)
	return nil
}

func googleDetectLang(text string) (string, float64, string, error) {
	api := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=en&dt=t&dt=ld&q=%s",
		url.QueryEscape(text))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(api)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, "", err
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", 0, "", err
	}

	var translated strings.Builder
	if len(result) > 0 {
		if chunks, ok := result[0].([]interface{}); ok {
			for _, c := range chunks {
				if line, ok := c.([]interface{}); ok && len(line) > 0 {
					if s, ok := line[0].(string); ok {
						translated.WriteString(s)
					}
				}
			}
		}
	}

	code := "unknown"
	if len(result) > 2 {
		if s, ok := result[2].(string); ok {
			code = s
		}
	}

	confidence := 0.0
	for i := len(result) - 1; i >= 0; i-- {
		if arr, ok := result[i].([]interface{}); ok && len(arr) >= 2 {
			codesArr, okA := arr[0].([]interface{})
			confsArr, okB := arr[len(arr)-1].([]interface{})
			if okA && okB && len(codesArr) > 0 && len(confsArr) > 0 {
				if _, ok := codesArr[0].(string); ok {
					if f, ok := confsArr[0].(float64); ok {
						confidence = f
						break
					}
				}
			}
		}
	}

	return code, confidence, translated.String(), nil
}

func registerTranslate2Handlers() {
	c := Client
	c.On("cmd:langs", LangsHandler)
	c.On("cmd:detect", DetectHandler)
}

func init() {
	QueueHandlerRegistration(registerTranslate2Handlers)
}
