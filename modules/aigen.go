package modules

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

var imageAPIURL = "http://localhost:9998"

func EditImageCustomHandler(m *telegram.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Please reply to an image to edit it")
		return nil
	}

	prompt := m.Args()
	if prompt == "" {
		m.Reply("Provide edit instructions\n")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if !r.IsMedia() {
		m.Reply("Please reply to an image")
		return nil
	}

	msg, _ := m.Reply("-- Editing image... --")

	fi, err := m.Client.DownloadMedia(r.Media())
	if err != nil {
		msg.Edit("Error downloading image: " + err.Error())
		return nil
	}
	defer os.Remove(fi)

	editedImage, err := sendImageToAPI(fi, prompt, "/edit")
	if err != nil {
		msg.Edit("Error editing image: " + err.Error())
		return nil
	}
	defer os.Remove(editedImage)

	_, err = m.ReplyMedia(editedImage, telegram.MediaOptions{
		Caption: fmt.Sprintf("✨ <b>Image Edited</b>"),
	})
	if err != nil {
		msg.Edit("Error uploading edited image: " + err.Error())
		return nil
	}

	msg.Delete()
	return nil
}

func GenerateImageHandler(m *telegram.NewMessage) error {
	prompt := m.Args()
	if prompt == "" {
		m.Reply("Provide a prompt to generate an image\n")
		return nil
	}

	msg, _ := m.Reply("-- Generating image... --")

	generatedImage, err := generateImageFromPrompt(prompt)
	if err != nil {
		msg.Edit("Error generating image: " + err.Error())
		return nil
	}
	defer os.Remove(generatedImage)

	_, err = m.ReplyMedia(generatedImage, telegram.MediaOptions{
		Caption: fmt.Sprintf("✨ <b>Generated Image</b>\n<code>%s</code>", prompt),
	})
	if err != nil {
		msg.Edit("Error uploading generated image: " + err.Error())
		return nil
	}

	msg.Delete()
	return nil
}

func sendImageToAPI(imagePath, prompt, endpoint string) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("image", "image.png")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}

	if prompt != "" {
		err = writer.WriteField("prompt", prompt)
		if err != nil {
			return "", err
		}
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	url := imageAPIURL + endpoint
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s", string(body))
	}

	outputPath := "tmp/edited_" + strings.TrimPrefix(imagePath, "tmp/")
	os.MkdirAll("tmp", 0755)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, resp.Body)
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func generateImageFromPrompt(prompt string) (string, error) {
	jsonData := fmt.Sprintf(`{"prompt": "%s"}`, strings.ReplaceAll(prompt, `"`, `\"`))

	url := imageAPIURL + "/gen"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s", string(body))
	}

	outputPath := "tmp/generated_image.png"
	os.MkdirAll("tmp", 0755)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, resp.Body)
	if err != nil {
		return "", err
	}

	return outputPath, nil
}
