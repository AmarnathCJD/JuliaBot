package modules

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func PasteBinHandler(m *telegram.NewMessage) error {
	if m.Args() == "" && !m.IsReply() {
		m.Reply("Please provide some text to paste")
		return nil
	}

	content := m.Args()

	if m.IsReply() {
		r, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error getting reply message")
			return nil
		}

		if r.IsMedia() {
			if r.Photo() != nil {
				m.Reply("<code>Photo</code> is not supported")
				return nil
			}

			if r.File.Size > 50*1024*200 { // 10MB
				m.Reply("File size too large, max 10MB")
				return nil
			}

			doc, err := r.Download()
			if err != nil {
				m.Reply("Error downloading file")
				return nil
			}

			f, err := os.ReadFile(doc)
			if err != nil {
				m.Reply("Error reading file")
				return nil
			}

			content = string(f)
		} else {
			content = r.Text()
		}
	}

	var (
		url      string
		provider string
		err      error
	)

	url, provider, err = postToPatbin(content)
	if err != nil {
		url, provider, err = postToSpaceBin(content)
		if err != nil {
			m.Reply("Error posting to paste services")
			return nil
		}
	}

	b := telegram.Button

	m.Reply(fmt.Sprintf("<b>Pasted to <a href='%s'>%s</a></b>", url, provider), &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().AddRow(
			b.URL("View Paste", url),
		).Build(),
	})

	return nil
}

// postToPatbin posts content to patbin.fun
func postToPatbin(content string) (string, string, error) {
	payload := fmt.Sprintf(`{"content":%q,"title":"","language":"text","is_public":true}`, content)

	req, err := http.NewRequest("POST", "https://patbin.fun/api/paste", bytes.NewBufferString(payload))
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error reading response: %w", err)
	}

	// Parse the ID from JSON response {"id":"xxx",...}
	// Simple extraction without full JSON parsing
	idStart := bytes.Index(body, []byte(`"id":"`))
	if idStart == -1 {
		return "", "", fmt.Errorf("id not found in response")
	}
	idStart += 6
	idEnd := bytes.Index(body[idStart:], []byte(`"`))
	if idEnd == -1 {
		return "", "", fmt.Errorf("id end not found in response")
	}

	pasteID := string(body[idStart : idStart+idEnd])
	return "https://patbin.fun/" + pasteID, "Patbin", nil
}

func postToSpaceBin(content string) (string, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("content", content); err != nil {
		return "", "", fmt.Errorf("error writing field: %w", err)
	}

	writer.Close()
	req, err := http.NewRequest("POST", "https://spaceb.in/", &body)
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	if location == "" {
		return "", "", fmt.Errorf("location header not found")
	}

	return "https://spaceb.in" + location, "SpaceBin", nil
}

func Gban(m *telegram.NewMessage) error {
	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	message, _ := m.Reply("Enforcing global ban...")
	done := 0
	m.Client.Broadcast(context.Background(), nil, func(c telegram.Chat) error {
		_, err := m.Client.EditBanned(c, user, &telegram.BannedOptions{Ban: true})
		if err == nil {
			done++
		}
		return nil
	}, 600)

	message.Edit(fmt.Sprintf("Global ban enforced in %d groups.\nReason: %s", done, reason))
	return nil
}

func Ungban(m *telegram.NewMessage) error {
	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	message, _ := m.Reply("Removing global ban...")
	done := 0
	m.Client.Broadcast(context.Background(), nil, func(c telegram.Chat) error {
		_, err := m.Client.EditBanned(c, user, &telegram.BannedOptions{Ban: false})
		if err == nil {
			done++
		}
		return nil
	}, 600)
	message.Edit(fmt.Sprintf("Global ban removed in %d groups.", done))
	return nil
}

func mathQuery(query string) (string, error) {
	c := &http.Client{}
	url := "https://evaluate-expression.p.rapidapi.com/?expression=" + url.QueryEscape(query)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-rapidapi-host", "evaluate-expression.p.rapidapi.com")
	req.Header.Add("x-rapidapi-key", "cf9e67ea99mshecc7e1ddb8e93d1p1b9e04jsn3f1bb9103c3f")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) == "" {
		return "", fmt.Errorf("invalid math expression")
	}

	return string(body), nil
}

func MathHandler(m *telegram.NewMessage) error {
	q := m.Args()
	if q == "" {
		m.Reply("please provide a mathematical expression")
		return nil
	}

	result, err := mathQuery(q)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	m.Reply(fmt.Sprintf("Evaluated: <code>%s</code>", result))
	return nil
}

func NightModeHandler(m *telegram.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
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
		current = &telegram.ChatBannedRights{}
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

func registerMiscHandlers() {
	c := Client
	c.On("command:paste", PasteBinHandler)
	c.On("command:math", MathHandler)
	c.On("command:audio", ConvertToAudioHandle)
	c.On("cmd:help", HelpHandle)
	c.On("cmd:nightmode", NightModeHandler)
	c.On("cmd:tempnote", SaveTempNoteHandler)
	c.On("callback:verify_op_", AdminVerifyCallback)
	c.On("callback:help_back", HelpBackCallback)
	c.On("cmd:adddl", AddDLHandler, tg.CustomFilter(FilterOwnerAndAuth))
	c.On("cmd:listdls", ListDLsHandler, tg.CustomFilter(FilterOwnerAndAuth))
	c.On("cmd:rmdl", RmDLHandler, tg.CustomFilter(FilterOwnerAndAuth))
	c.On("cmd:listdl", ListDLHandler, tg.CustomFilter(FilterOwnerAndAuth))
}

func init() {
	QueueHandlerRegistration(registerMiscHandlers)
}
