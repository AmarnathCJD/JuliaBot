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

type MirrorOptions struct {
	Destination   interface{}
	NoProgress    bool
	ForceDocument bool
	NoThumb       bool
	Delay         int
	FileName      string
	SourceLink    string
}

func parseMirrorArgs(args string) *MirrorOptions {
	opts := &MirrorOptions{}
	parts := strings.Fields(args)

	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "-c":
			if i+1 < len(parts) {
				i++
				dest := parts[i]
				if strings.HasPrefix(dest, "@") {
					opts.Destination = dest[1:]
				} else if strings.Contains(dest, "t.me/c/") {
					reg := regexp.MustCompile(`t.me/c/(\d+)`)
					match := reg.FindStringSubmatch(dest)
					if len(match) == 2 {
						chatID, _ := strconv.ParseInt("-100"+match[1], 10, 64)
						opts.Destination = chatID
					}
				} else if strings.Contains(dest, "t.me/") {
					reg := regexp.MustCompile(`t.me/(\w+)`)
					match := reg.FindStringSubmatch(dest)
					if len(match) == 2 {
						opts.Destination = match[1]
					}
				} else {
					if id, err := strconv.ParseInt(dest, 10, 64); err == nil {
						opts.Destination = id
					} else {
						opts.Destination = dest
					}
				}
			}
		case "-nop":
			opts.NoProgress = true
		case "-doc":
			opts.ForceDocument = true
		case "-not":
			opts.NoThumb = true
		case "-d":
			if i+1 < len(parts) {
				i++
				opts.Delay, _ = strconv.Atoi(parts[i])
			}
		case "-fn":
			if i+1 < len(parts) {
				i++
				opts.FileName = parts[i]
			}
		default:
			if strings.Contains(parts[i], "t.me/") {
				opts.SourceLink = parts[i]
			} else if opts.FileName == "" && !strings.HasPrefix(parts[i], "-") {
				opts.FileName = parts[i]
			}
		}
	}

	return opts
}

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
		m.Reply("Reply to a file to download it\n\nOptions:\n-c <dest> : destination chat (@user, t.me/user, t.me/c/id)\n-nop : no progress\n-doc : force document\n-not : no thumbnail\n-d <sec> : delay in seconds\n-fn <name> : custom filename")
		return nil
	}

	opts := parseMirrorArgs(m.Args())

	var r, msg *telegram.NewMessage
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}
		r = reply
		if !opts.NoProgress {
			msg, _ = m.Reply("Downloading...")
		}
	} else if opts.SourceLink != "" {
		// Parse source link
		reg := regexp.MustCompile(`t.me/(\w+)/(\d+)`)
		match := reg.FindStringSubmatch(opts.SourceLink)
		if len(match) != 3 || match[1] == "c" {
			// https://t.me/c/2183493392/8
			reg = regexp.MustCompile(`t.me/c/(\d+)/(\d+)`)
			match = reg.FindStringSubmatch(opts.SourceLink)
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
			if opts.FileName == "" {
				opts.FileName = r.File.Name
			}
			if !opts.NoProgress {
				msg, _ = m.Reply("Downloading... (from c " + strconv.Itoa(id) + ")")
			}
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
			if opts.FileName == "" {
				opts.FileName = r.File.Name
			}
			if !opts.NoProgress {
				msg, _ = m.Reply("Downloading... (from " + username + " " + strconv.Itoa(id) + ")")
			}
		}
	} else {
		m.Reply("No source specified. Reply to a message or provide a t.me link")
		return nil
	}

	// Set up download options
	dlOpts := &telegram.DownloadOptions{FileName: opts.FileName}
	if !opts.NoProgress && msg != nil {
		dlOpts.ProgressManager = telegram.NewProgressManager(5).SetMessage(msg)
	}

	fi, err := r.Download(dlOpts)
	if err != nil {
		if msg != nil {
			msg.Edit("Error: " + err.Error())
		} else {
			m.Reply("Error: " + err.Error())
		}
		return nil
	}

	// Set up upload options
	var mediaOpts = &telegram.MediaOptions{
		ForceDocument: opts.ForceDocument,
		Spoiler:       true,
	}

	if !opts.NoProgress && msg != nil {
		mediaOpts.Upload.ProgressManager = telegram.NewProgressManager(5).SetMessage(msg)
	}

	if !opts.NoThumb {
		if _, err := os.Stat("thumb.jpg"); err == nil {
			mediaOpts.Thumb = "thumb.jpg"
		}
	}

	// Determine destination
	var dest interface{} = m.ChatID()
	if opts.Destination != nil {
		dest = opts.Destination
	}

	_, err = m.Client.SendMedia(dest, fi, mediaOpts)
	if err != nil {
		if msg != nil {
			msg.Edit("Upload Error: " + err.Error())
		} else {
			m.Reply("Upload Error: " + err.Error())
		}
	}

	defer os.Remove(fi)
	if msg != nil {
		defer msg.Delete()
	}
	return nil
}
