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

func convertMediaFormat(inputPath, inputExt string) (string, error) {
	ext := strings.ToLower(inputExt)
	outputPath := strings.TrimSuffix(inputPath, inputExt)

	var cmd *exec.Cmd
	switch ext {
	case ".heic", ".heif":
		outputPath += ".jpg"
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-y", outputPath,
		)

	case ".f4v", ".m4v", ".flv", ".wmv", ".avi", ".mkv", ".webm", ".mov", ".m2ts", ".mts", ".mp2":
		outputPath += ".mp4"
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "128k",
			"-movflags", "+faststart",
			"-pix_fmt", "yuv420p",
			"-y", outputPath,
		)

	default:
		outputPath += ".mp4"
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-y", outputPath,
		)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = io.Discard

	err := cmd.Run()
	if err != nil {
		fmt.Printf("FFmpeg conversion failed for %s: %v\nFFmpeg stderr: %s\n", inputPath, err, stderr.String())

		if ext == ".f4v" || ext == ".m4v" || ext == ".mov" {
			fallbackPath := strings.TrimSuffix(inputPath, inputExt) + "_fallback.mp4"
			fallbackCmd := exec.Command("ffmpeg",
				"-i", inputPath,
				"-c:v", "copy",
				"-c:a", "aac",
				"-movflags", "+faststart",
				"-y", fallbackPath,
			)

			var fallbackStderr bytes.Buffer
			fallbackCmd.Stderr = &fallbackStderr
			fallbackCmd.Stdout = io.Discard

			fallbackErr := fallbackCmd.Run()
			if fallbackErr == nil {
				os.Remove(inputPath)
				return fallbackPath, nil
			}

			recoveryPath := strings.TrimSuffix(inputPath, inputExt) + "_recovery.mp4"
			recoveryCmd := exec.Command("ffmpeg",
				"-err_detect", "ignore_err",
				"-i", inputPath,
				"-c:v", "libx264",
				"-preset", "ultrafast",
				"-crf", "28",
				"-c:a", "aac",
				"-b:a", "96k",
				"-movflags", "+faststart",
				"-pix_fmt", "yuv420p",
				"-y", recoveryPath,
			)

			var recoveryStderr bytes.Buffer
			recoveryCmd.Stderr = &recoveryStderr
			recoveryCmd.Stdout = io.Discard

			recoveryErr := recoveryCmd.Run()
			if recoveryErr == nil {
				os.Remove(inputPath)
				return recoveryPath, nil
			}
			fmt.Printf("FFmpeg fallback/recovery also failed for %s: %v\nStderr: %s\n", inputPath, recoveryErr, recoveryStderr.String())
		}
		return "", err
	}

	os.Remove(inputPath)

	return outputPath, nil
}

func detectMediaByMagicBytes(data []byte) string {
	if len(data) < 12 {
		return ""
	}

	if len(data) >= 12 && string(data[4:8]) == "ftyp" {
		brand := strings.ToLower(string(data[8:12]))
		if strings.Contains(brand, "heic") || strings.Contains(brand, "heix") ||
			strings.Contains(brand, "hevc") || strings.Contains(brand, "hevx") ||
			strings.Contains(brand, "mif1") {
			return ".heif"
		}
		if strings.Contains(brand, "f4v") || strings.Contains(brand, "f4p") ||
			strings.Contains(brand, "f4a") || strings.Contains(brand, "f4b") {
			return ".f4v"
		}
		if strings.Contains(brand, "m4v") || strings.Contains(brand, "m4vh") ||
			strings.Contains(brand, "m4vp") {
			return ".m4v"
		}
		if strings.Contains(brand, "flv") {
			return ".flv"
		}
		if strings.Contains(brand, "qt") || brand == "mqt " {
			return ".mov"
		}
		return ".mp4"
	}

	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ".jpg"
	}

	if len(data) >= 4 && string(data[0:4]) == "\x89PNG" {
		return ".png"
	}

	if len(data) >= 3 && string(data[0:3]) == "GIF" {
		return ".gif"
	}

	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return ".webp"
	}

	if len(data) >= 4 && string(data[0:3]) == "FLV" && data[3] == 0x01 {
		return ".flv"
	}

	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:11]) == "AVI" {
		return ".avi"
	}

	if len(data) >= 4 && data[0] == 0x47 {
		return ".mts"
	}

	return ""
}

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

		needsConversion := func(e string) bool {
			convertFormats := []string{".mp2", ".m2ts", ".mts", ".heic", ".heif", ".f4v", ".m4v", ".flv", ".wmv", ".avi", ".mkv", ".webm", ".mov"}
			for _, f := range convertFormats {
				if e == f {
					return true
				}
			}
			return false
		}

		if needsConversion(ext) {
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
