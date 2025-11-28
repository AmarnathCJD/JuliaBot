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

	"github.com/amarnathcjd/gogram/telegram"
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

	url, provider, err = postToSpaceBin(content)
	if err != nil {
		m.Reply("Error posting to " + provider)
		return nil
	}

	b := telegram.Button

	m.Reply(fmt.Sprintf("<b>Pasted to <a href='%s'>%s</a></b>", url, provider), &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().AddRow(
			b.URL("View Paste", url),
		).Build(),
	})

	return nil
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
