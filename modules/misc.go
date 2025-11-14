package modules

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func PasteBinHandler(m *telegram.NewMessage) error {
	if m.Args() == "" && !m.IsReply() {
		m.Reply("Please provide some text to paste")
		return nil
	}

	content := m.Args()

	isKatBin := false
	if strings.Contains(content, "-k") {
		isKatBin = true
		content = strings.Replace(content, "-k", "", -1)
	}

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
	if isKatBin {
		url, provider, err = postToKatBin(content)
	} else {
		url, provider, err = postToSpaceBin(content)
	}
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

func postToKatBin(content string) (string, string, error) {
	var body = `{"paste": {"content": "%s"}}`
	body = fmt.Sprintf(body, content)

	req, err := http.NewRequest("POST", "https://katb.in/api/paste", strings.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "deflate, gzip")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Host", "katb.in")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return "", "", fmt.Errorf("status code not 200: %d", resp.StatusCode)
	}

	var bodyReader io.ReadCloser = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		bodyReader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return "", "", fmt.Errorf("error creating gzip reader: %w", err)
		}
		defer bodyReader.Close()
	}

	var result map[string]interface{}
	if err := json.NewDecoder(bodyReader).Decode(&result); err != nil {
		return "", "", fmt.Errorf("error decoding response: %w", err)
	}

	if result["id"] == nil {
		return "", "", fmt.Errorf("id not found in response")
	}

	return fmt.Sprintf("https://katb.in/%s", result["id"]), "Katb.in", nil
}

func GbanMeme(m *telegram.NewMessage) error {
	randTime := rand.Intn(100)
	randChatCount := rand.Intn(1000)

	msg, _ := m.Reply(fmt.Sprintf("⚡ Enforcing Global Ban on %d chats", randChatCount))

	time.Sleep(time.Duration(randTime) * time.Second)

	msg.Reply(fmt.Sprintf("⚒️ Global Ban enforced on %d chats", randChatCount))
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
