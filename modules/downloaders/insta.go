package downloaders

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
	"sync"

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

// checkContentType checks if URL is actually an image by HEAD request
func checkContentType(url string) (bool, string) {
	resp, err := http.Head(url)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		if strings.Contains(strings.ToLower(contentDisposition), ".jpg") ||
			strings.Contains(strings.ToLower(contentDisposition), ".jpeg") ||
			strings.Contains(strings.ToLower(contentDisposition), ".png") ||
			strings.Contains(strings.ToLower(contentDisposition), ".gif") {
			return true, contentDisposition
		}
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(contentType), "image") {
		return true, contentType
	}

	return false, ""
}

// downloadMediaFiles downloads URLs and returns local file paths
func downloadMediaFiles(mediaURLs []string, isVideoList bool) ([]string, error) {
	var filePaths []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	type urlCheck struct {
		index   int
		url     string
		isImage bool
		info    string
	}

	checkResults := make([]urlCheck, len(mediaURLs))

	for i, mediaURL := range mediaURLs {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()
			isImage, info := checkContentType(url)
			checkResults[idx] = urlCheck{
				index:   idx,
				url:     url,
				isImage: isImage,
				info:    info,
			}
		}(i, mediaURL)
	}

	wg.Wait()

	for _, result := range checkResults {
		resp, err := http.Get(result.url)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		ext := ".mp4"
		if result.isImage {
			ext = ".jpg"
			if strings.Contains(strings.ToLower(result.info), ".png") {
				ext = ".png"
			} else if strings.Contains(strings.ToLower(result.info), ".gif") {
				ext = ".gif"
			}
		}

		filename := fmt.Sprintf("insta_%d%s", result.index, ext)
		filePath := filepath.Join(os.TempDir(), filename)

		file, err := os.Create(filePath)
		if err != nil {
			continue
		}

		_, err = io.Copy(file, bytes.NewReader(body))
		file.Close()
		if err != nil {
			continue
		}

		mu.Lock()
		filePaths = append(filePaths, filePath)
		mu.Unlock()
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

	filePaths, err := downloadMediaFiles(allURLs, false)
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
