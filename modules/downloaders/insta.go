package downloaders

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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

// convertMediaFormat converts unsupported media formats to supported ones using ffmpeg
func convertMediaFormat(inputPath, inputExt string) (string, error) {
	outputExt := ".jpg"
	outputPath := strings.TrimSuffix(inputPath, inputExt) + outputExt

	// Determine output format based on input
	if strings.ToLower(inputExt) == ".heic" || strings.ToLower(inputExt) == ".heif" {
		outputExt = ".jpg"
		outputPath = strings.TrimSuffix(inputPath, inputExt) + outputExt
	} else if strings.ToLower(inputExt) == ".mp2" || strings.ToLower(inputExt) == ".m2ts" || strings.ToLower(inputExt) == ".mts" {
		outputExt = ".mp4"
		outputPath = strings.TrimSuffix(inputPath, inputExt) + outputExt
	}

	// Run ffmpeg conversion
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-y", outputPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	err := cmd.Run()
	if err != nil {
		fmt.Printf("FFmpeg conversion failed for %s: %v\n", inputPath, err)
		return "", err
	}

	// Remove input file after successful conversion
	os.Remove(inputPath)
	fmt.Printf("Converted %s to %s\n", inputPath, outputPath)

	return outputPath, nil
}

// detectMediaByMagicBytes detects media type by checking file magic bytes
func detectMediaByMagicBytes(data []byte) string {
	if len(data) < 12 {
		return ""
	}

	// Check for HEIF/HEIC (ftyp at offset 4)
	if len(data) >= 12 {
		if string(data[4:8]) == "ftyp" {
			// Check the brand after ftyp
			brand := string(data[8:12])
			if strings.Contains(brand, "heic") || strings.Contains(brand, "heix") ||
				strings.Contains(brand, "hevc") || strings.Contains(brand, "hevx") ||
				strings.Contains(brand, "mif1") {
				return ".heif"
			}
		}
	}

	// Check for JPEG (FFD8FF)
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ".jpg"
	}

	// Check for PNG (89504E47)
	if len(data) >= 4 && string(data[0:4]) == "\x89PNG" {
		return ".png"
	}

	// Check for GIF (474946)
	if len(data) >= 3 && string(data[0:3]) == "GIF" {
		return ".gif"
	}

	// Check for MP4 (ftyp at offset 4)
	if len(data) >= 12 && string(data[4:8]) == "ftyp" {
		return ".mp4"
	}

	// Check for WebP (RIFF...WEBP)
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return ".webp"
	}

	return ""
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
		req, _ := http.NewRequest("GET", result.url, nil)
		//fmt.Printf("Downloading from URL: %s\n", result.url)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

		resp, err := http.DefaultClient.Do(req)
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

		mimeType := http.DetectContentType(body)
		extensions, err := mime.ExtensionsByType(mimeType)
		if err == nil && len(extensions) > 0 {
			ext = extensions[0]
		} else {
			magicExt := detectMediaByMagicBytes(body)
			if magicExt != "" {
				ext = magicExt
			} else {
				ext = filepath.Ext(result.url)
			}
		}

		if ext == ".jfif" {
			ext = ".jpg"
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

		if ext == ".mp2" || ext == ".m2ts" || ext == ".mts" || ext == ".heic" || ext == ".heif" {
			convertedPath, err := convertMediaFormat(filePath, ext)
			if err != nil {
				fmt.Printf("Skipping file due to conversion failure: %s\n", filePath)
				os.Remove(filePath)
				continue
			}
			filePath = convertedPath
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

	imageFilePaths, err := downloadMediaFiles(resp.Images, false)
	if err != nil {
		m.Reply("Failed to download image files")
		return nil
	}

	videoFilePaths, err := downloadMediaFiles(resp.Videos, true)
	if err != nil {
		m.Reply("Failed to download video files")
		return nil
	}

	if len(imageFilePaths) == 0 && len(videoFilePaths) == 0 {
		m.Reply("No media files could be downloaded")
		return nil
	}

	defer func() {
		for _, file := range imageFilePaths {
			os.Remove(file)
		}
		for _, file := range videoFilePaths {
			os.Remove(file)
		}
	}()

	if len(imageFilePaths) > 0 {
		if len(imageFilePaths) == 1 {
			_, err := m.ReplyMedia(imageFilePaths[0])
			if err != nil {
				m.Client.Log.Error(err)
			}
		} else {
			_, err := m.ReplyAlbum(imageFilePaths)
			if err != nil {
				m.Client.Log.Error(err)
			}
		}
	}

	if len(videoFilePaths) > 0 {
		if len(videoFilePaths) == 1 {
			_, err := m.ReplyMedia(videoFilePaths[0])
			if err != nil {
				m.Client.Log.Error(err)
			}
		} else {
			_, err := m.ReplyAlbum(videoFilePaths)
			if err != nil {
				m.Client.Log.Error(err)
			}
		}
	}

	return nil
}
