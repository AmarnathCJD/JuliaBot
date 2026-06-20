package modules

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func uploadToCatbox(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("reqtype", "fileupload"); err != nil {
		return "", err
	}

	fw, err := writer.CreateFormFile("fileToUpload", "image.jpg")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, "https://catbox.moe/user/api.php", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*64))
	if err != nil {
		return "", err
	}

	link := strings.TrimSpace(string(data))
	if !strings.HasPrefix(link, "https://") && !strings.HasPrefix(link, "http://") {
		return "", fmt.Errorf("unexpected response: %s", link)
	}
	return link, nil
}

func uploadToUguu(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fw, err := writer.CreateFormFile("files[]", "image.jpg")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, "https://uguu.se/upload?output=text", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*64))
	if err != nil {
		return "", err
	}

	link := strings.TrimSpace(string(data))
	if !strings.HasPrefix(link, "https://") && !strings.HasPrefix(link, "http://") {
		return "", fmt.Errorf("unexpected response: %s", link)
	}
	return link, nil
}

func ReverseImageHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("reply to a photo with <code>/reverse</code> to get reverse-image search links")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("error: " + html.EscapeString(err.Error()))
		return nil
	}

	if r.Photo() == nil {
		m.Reply("the replied message is not a photo")
		return nil
	}

	status, _ := m.Reply("<code>preparing reverse search...</code>")

	path, err := m.Client.DownloadMedia(r.Media(), &tg.DownloadOptions{FileName: "reverse_" + fmt.Sprint(time.Now().UnixNano()) + ".jpg"})
	if err != nil {
		msg := "error downloading photo: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(path)

	if status != nil {
		status.Edit("<code>uploading image to host...</code>")
	}

	link, err := uploadToCatbox(path)
	if err != nil {
		link, err = uploadToUguu(path)
	}
	if err != nil {
		msg := "error uploading image: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	enc := url.QueryEscape(link)
	googleURL := "https://lens.google.com/uploadbyurl?url=" + enc
	yandexURL := "https://yandex.com/images/search?rpt=imageview&url=" + enc
	tineyeURL := "https://www.tineye.com/search?url=" + enc
	bingURL := "https://www.bing.com/images/search?view=detailv2&iss=sbi&form=SBIVSP&sbisrc=UrlPaste&q=imgurl:" + enc

	b := tg.Button
	keyb := tg.NewKeyboard().
		AddRow(
			b.URL("Google Lens", googleURL),
			b.URL("Yandex", yandexURL),
		).
		AddRow(
			b.URL("TinEye", tineyeURL),
			b.URL("Bing", bingURL),
		).
		AddRow(
			b.URL("Direct Image", link),
		)

	text := "<b>Reverse Image Search</b>\n<i>Tap a button to search on that engine.</i>\n\n<b>Image:</b> <a href=\"" + html.EscapeString(link) + "\">" + html.EscapeString(link) + "</a>"

	if status != nil {
		status.Edit(text, &tg.SendOptions{
			ReplyMarkup: keyb.Build(),
			LinkPreview: false,
		})
	} else {
		m.Reply(text, &tg.SendOptions{
			ReplyMarkup: keyb.Build(),
			LinkPreview: false,
		})
	}
	return nil
}

func init() { QueueHandlerRegistration(registerReverseImageHandlers) }

func registerReverseImageHandlers() {
	c := Client
	c.On("cmd:reverse", ReverseImageHandler)
}
