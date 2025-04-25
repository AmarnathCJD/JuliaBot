package modules

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
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

	m.Reply("Thumbnail set successfully!!")

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
	return nil
}

func MirrorFileHandler(m *telegram.NewMessage) error {
	if !m.IsReply() && m.Args() == "" {
		m.Reply("Reply to a file to download it")
		return nil
	}

	fn := m.Args()

	var r, msg *telegram.NewMessage
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		r = reply
		msg, _ = m.Reply("Downloading...")
	} else {
		reg := regexp.MustCompile(`t.me/(\w+)/(\d+)`)
		match := reg.FindStringSubmatch(m.Args())
		if len(match) != 3 || match[1] == "c" {
			// https://t.me/c/2183493392/8
			reg = regexp.MustCompile(`t.me/c/(\d+)/(\d+)`)
			match = reg.FindStringSubmatch(m.Args())
			if len(match) != 3 {
				m.Reply("Invalid link")
				return nil
			}

			id, err := strconv.Atoi(match[2])
			if err != nil {
				m.Reply("Invalid link: " + err.Error())
				return nil
			}

			chatID, err := strconv.Atoi(match[1])
			if err != nil {
				m.Reply("Invalid link: " + err.Error())
				return nil
			}

			msgX, err := m.Client.GetMessageByID(chatID, int32(id))
			if err != nil {
				m.Reply("Error: " + err.Error())
				return nil
			}
			r = msgX
			fn = r.File.Name
			msg, _ = m.Reply("Downloading... (from c " + strconv.Itoa(id) + ")")
		} else {
			username := match[1]
			id, err := strconv.Atoi(match[2])
			if err != nil {
				m.Reply("Invalid link")
				return nil
			}

			msgX, err := m.Client.GetMessageByID(username, int32(id))
			if err != nil {
				m.Reply("Error: " + err.Error())
				return nil
			}
			r = msgX
			fn = r.File.Name
			msg, _ = m.Reply("Downloading... (from " + username + " " + strconv.Itoa(id) + ")")
		}
	}

	fi, err := r.Download(&telegram.DownloadOptions{FileName: fn, ProgressManager: telegram.NewProgressManager(5).SetMessage(msg)})
	if err != nil {
		msg.Edit("Error: " + err.Error())
		return nil
	}

	var opt = telegram.MediaOptions{
		ForceDocument:   true,
		ProgressManager: telegram.NewProgressManager(5).SetMessage(msg),
		Spoiler:         true,
	}

	if _, err := os.Stat("thumb.jpg"); err == nil {
		opt.Thumb = "thumb.jpg"
	}

	m.RespondMedia(fi, opt)
	defer os.Remove(fn)
	defer msg.Delete()
	return nil
}
