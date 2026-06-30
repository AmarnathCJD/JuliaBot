package extras

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

type pasteService struct {
	id     string
	name   string
	upload func(string) (string, error)
}

var pasteServices = []pasteService{
	{id: "katbin", name: "katb.in", upload: pasteUploadKatbin},
	{id: "spacebin", name: "spaceb.in", upload: pasteUploadSpacebin},
	{id: "dpaste", name: "dpaste.com", upload: pasteUploadDpaste},
}

func pasteHTTPClient() *http.Client {
	return &http.Client{Timeout: 25 * time.Second}
}

func pasteUploadKatbin(content string) (string, error) {
	body, err := json.Marshal(map[string]map[string]string{
		"paste": {"content": content},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", "https://katb.in/api/paste", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := pasteHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("katb.in HTTP %d", resp.StatusCode)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("katb.in: empty id")
	}
	return "https://katb.in/" + out.ID, nil
}

func pasteUploadSpacebin(content string) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("content", content); err != nil {
		return "", err
	}
	w.Close()
	req, err := http.NewRequest("POST", "https://spaceb.in/", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := &http.Client{
		Timeout: 25 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("spaceb.in: no Location (HTTP %d)", resp.StatusCode)
	}
	if strings.HasPrefix(loc, "/") {
		loc = "https://spaceb.in" + loc
	}
	return loc, nil
}

func pasteUploadDpaste(content string) (string, error) {
	form := url.Values{
		"content": {content},
		"syntax":  {"text"},
	}
	req, err := http.NewRequest("POST", "https://dpaste.com/api/v2/", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := pasteHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return "", fmt.Errorf("dpaste HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if loc := resp.Header.Get("Location"); loc != "" {
		return strings.TrimSpace(loc), nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	u := strings.TrimSpace(string(b))
	if strings.HasPrefix(u, "http") {
		return u, nil
	}
	return "", fmt.Errorf("dpaste: no URL in response")
}

func pasteParseFlag(args string) (string, string) {
	args = strings.TrimSpace(args)
	for _, flag := range [][2]string{
		{"-s", "spacebin"},
		{"-d", "dpaste"},
		{"-k", "katbin"},
	} {
		if strings.HasPrefix(args, flag[0]+" ") {
			return flag[1], strings.TrimSpace(args[len(flag[0]):])
		}
		if args == flag[0] {
			return flag[1], ""
		}
	}
	return "", args
}

func pasteOrderedServices(preferredID string) []pasteService {
	if preferredID == "" {
		return pasteServices
	}
	out := make([]pasteService, 0, len(pasteServices))
	for _, s := range pasteServices {
		if s.id == preferredID {
			out = append(out, s)
			break
		}
	}
	for _, s := range pasteServices {
		if s.id != preferredID {
			out = append(out, s)
		}
	}
	return out
}

func PasteHandler(m *tg.NewMessage) error {
	preferred, rest := pasteParseFlag(m.Args())
	content := rest

	if content == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil && r != nil {
			if t := r.Text(); strings.TrimSpace(t) != "" {
				content = t
			}
		}
	}

	if strings.TrimSpace(content) == "" {
		m.Reply("usage: <code>/paste [-s|-d|-k] &lt;text&gt;</code> or reply to a message\n\n" +
			"<b>backends:</b>\n" +
			"  default → katb.in\n" +
			"  -s → spaceb.in\n" +
			"  -d → dpaste.com\n" +
			"  -k → katb.in (explicit)\n\n" +
			"<i>autofalls back to the others if the chosen service is down</i>")
		return nil
	}

	status, _ := m.Reply("uploading paste...")

	services := pasteOrderedServices(preferred)
	var errs []string
	for _, svc := range services {
		urlStr, err := svc.upload(content)
		if err == nil && urlStr != "" {
			msg := "paste: " + html.EscapeString(urlStr) + "\n<i>via " + svc.name + "</i>"
			if len(errs) > 0 {
				msg += "\n<i>(fallback after: " + html.EscapeString(strings.Join(errs, ", ")) + ")</i>"
			}
			if status != nil {
				status.Edit(msg)
			} else {
				m.Reply(msg)
			}
			return nil
		}
		errs = append(errs, svc.name+": "+truncErr(err))
	}

	msg := "all paste services failed:\n<code>" + html.EscapeString(strings.Join(errs, "\n")) + "</code>"
	if status != nil {
		status.Edit(msg)
	} else {
		m.Reply(msg)
	}
	return nil
}

func truncErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if len(s) > 120 {
		s = s[:120] + "..."
	}
	return s
}

func init() {
	modules.QueueHandlerRegistration(func() {
		c := modules.Client
		c.On("cmd:paste", PasteHandler)
	})
}
