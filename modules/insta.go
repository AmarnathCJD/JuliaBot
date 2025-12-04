package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

type SnapResponse struct {
	Images []string `json:"images"`
	Videos []string `json:"videos"`
	Error  string   `json:"error,omitempty"`
}

func callSnapServer(downloadURL, accessToken string) (*SnapResponse, error) {
	snapServerURL := "https://insta.gogram.fun"

	data := url.Values{}
	data.Set("url", downloadURL)
	data.Set("token", accessToken)

	resp, err := http.PostForm(snapServerURL+"/download", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SnapResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// downloadMediaFiles downloads URLs and returns local file paths
func downloadMediaFiles(mediaURLs []string) ([]string, error) {
	var filePaths []string

	for i, mediaURL := range mediaURLs {
		resp, err := http.Get(mediaURL)
		if err != nil {
			fmt.Printf("Error downloading %s: %v\n", mediaURL, err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response body: %v\n", err)
			continue
		}

		ext := ".mp4"
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(strings.ToLower(mediaURL), ".jpg") ||
			strings.Contains(strings.ToLower(mediaURL), ".jpeg") ||
			strings.Contains(strings.ToLower(mediaURL), ".png") ||
			strings.Contains(strings.ToLower(contentType), "image") {
			ext = ".jpg"
		}

		filename := fmt.Sprintf("insta_%d%s", i, ext)
		filePath := filepath.Join(os.TempDir(), filename)

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("Error creating file: %v\n", err)
			continue
		}

		_, err = io.Copy(file, bytes.NewReader(body))
		file.Close()
		if err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
			continue
		}

		filePaths = append(filePaths, filePath)
	}

	return filePaths, nil
}

func InstaHandler(m *telegram.NewMessage) error {
	if m.Args() == "" {
		m.Reply("Please provide a URL")
		return nil
	}

	msg, _ := m.Reply("Processing your request...")
	defer msg.Delete()

	url := m.Args()
	accessToken := os.Getenv("INSTA_ACCESS")
	if accessToken == "" {
		return fmt.Errorf("INSTA_ACCESS environment variable not set")
	}

	resp, err := callSnapServer(url, accessToken)
	if err != nil {
		m.Reply(fmt.Sprintf("Failed to process the URL: %v", err))
		return nil
	}

	if resp.Error != "" {
		m.Reply(fmt.Sprintf("Error: %s", resp.Error))
		return nil
	}

	if len(resp.Images) == 0 && len(resp.Videos) == 0 {
		m.Reply("No media files found")
		return nil
	}

	// Download all media from URLs
	var allURLs []string
	allURLs = append(allURLs, resp.Images...)
	allURLs = append(allURLs, resp.Videos...)

	filePaths, err := downloadMediaFiles(allURLs)
	if err != nil {
		m.Reply("Failed to download media files")
		return nil
	}

	if len(filePaths) == 0 {
		m.Reply("No media files could be downloaded")
		return nil
	}

	// Cleanup on defer
	defer func() {
		for _, file := range filePaths {
			os.Remove(file)
		}
	}()

	// Send media
	if len(filePaths) == 1 {
		_, err := m.ReplyMedia(filePaths[0])
		if err != nil {
			m.Client.Log.Error(err)
		}
	} else {
		_, err := m.ReplyAlbum(filePaths)
		if err != nil {
			m.Client.Log.Error(err)
		}
	}

	return nil
}
