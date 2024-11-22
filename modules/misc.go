package modules

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
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

	url, err := postToSpaceBin(content)
	if err != nil {
		m.Reply("Error posting to spacebin")
		return nil
	}

	b := telegram.Button{}

	m.Reply(fmt.Sprintf("<b>Pasted to <a href='%s'>SpaceBin</a></b>", url), telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().AddRow(
			b.URL("View Paste", url),
		).Build(),
	})

	return nil
}

func postToSpaceBin(content string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("content", content); err != nil {
		return "", fmt.Errorf("error writing field: %w", err)
	}

	writer.Close()
	req, err := http.NewRequest("POST", "https://spaceb.in/", &body)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("location header not found")
	}

	return "https://spaceb.in" + location, nil
}
