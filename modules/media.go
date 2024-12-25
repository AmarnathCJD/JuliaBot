package modules

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

// media utilities

func convertThumb(thumbFilePath string, width int) string {
	// use ffmpeg, keep aspect ratio
	// ffmpeg -i thumb.jpg -vf scale=512:-1 thumb_512.jpg

	cmd := exec.Command("ffmpeg", "-i", thumbFilePath, "-vf", fmt.Sprintf("scale=%d:-1", width), strings.Replace(thumbFilePath, ".jpg", fmt.Sprintf("_%d.jpg", width), 1))
	err := cmd.Run()
	if err != nil {
		return ""
	}

	return strings.Replace(thumbFilePath, ".jpg", fmt.Sprintf("_%d.jpg", width), 1)
}

func SetThumbHandler(m *telegram.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Error: Reply to a media message")
		return nil
	}

	msg, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if msg.Photo() == nil && msg.Sticker() == nil {
		m.Reply("Error: Not a photo or sticker")
		return nil
	}

	_, err = msg.Download(&telegram.DownloadOptions{FileName: "thumb.jpg"})
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if m.Args() != "" {
		width := m.Args()
		wid, _ := strconv.Atoi(width)
		if wid < 1 {
			m.Reply("Error: Invalid width")
			return nil
		}
		thumb := convertThumb("thumb.jpg", wid)
		if thumb == "" {
			m.Reply("Error: Failed to convert thumbnail")
			return nil
		}

		os.Remove("thumb.jpg")
		os.Rename(thumb, "thumb.jpg")
	}

	m.Reply("Thumbnail set successfully")
	return nil
}

func MirrorFileHandler(m *telegram.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Error: Reply to a media message to mirror")
		return nil
	}

	msg, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if !msg.IsMedia() {
		m.Reply("Error: Not a media message")
		return nil
	}

	msg, _ = m.Reply("<code>Downloading...</code>")

	// pm := telegram.NewProgressManager(7)
	// pm.Edit(mediaDownloadProgress(msg.File.Name, msg, pm))

	file, err := msg.Download(&telegram.DownloadOptions{FileName: msg.File.Name})
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	defer os.Remove(file)

	msg.Edit("<code>Re-Uploading...</code>")
	defer msg.Delete()

	// thumb, _ := os.Open("thumb.jpg_512.jpg")
	// defer thumb.Close()

	// fthumb_bytes, _ := io.ReadAll(thumb)
	a := m.Args()
	if strings.Contains(a, "-f") {
		a = msg.File.Name
	}

	_, err = m.RespondMedia(file, telegram.MediaOptions{
		Caption:       fmt.Sprintf("<b>%s</b>", a),
		Thumb:         "thumb.jpg",
		ForceDocument: true,
	})

	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	return nil
}
