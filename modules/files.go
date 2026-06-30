package modules

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	downloadCancels = make(map[int32]context.CancelFunc)
	cancelMutex     sync.RWMutex
)

func SendFileByIDHandle(m *tg.NewMessage) error {
	fileId := m.Args()
	if fileId == "" {
		m.Reply("No fileId provided")
		return nil
	}

	file, err := tg.ResolveBotFileID(fileId)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	m.ReplyMedia(file)
	return nil
}

func GetFileIDHandle(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a file to get its fileId")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if r.File == nil {
		m.Reply("No file found in the reply")
		return nil
	}

	m.Reply("<b>FileId:</b> <code>" + r.File.FileID + "</code>")
	return nil
}

const defaultTransferSpeedCap int64 = 20 * 1024 * 1024

func UploadHandle(m *tg.NewMessage) error {
	filename := m.Args()
	if filename == "" {
		m.Reply("No filename provided")
		return nil
	}

	spoiler := false
	if strings.Contains(filename, "-s") {
		spoiler = true
		filename = strings.ReplaceAll(filename, "-s", "")
	}

	speedCap := defaultTransferSpeedCap
	filename = strings.TrimSpace(filename)

	msg, _ := m.Reply("Uploading...")
	uploadStartTimestamp := time.Now()

	if _, err := m.RespondMedia(filename, &tg.MediaOptions{
		ForceDocument: strings.Contains(filename, "--doc"),
		Spoiler:       spoiler,
		Upload: &tg.UploadOptions{
			ProgressManager: tg.NewProgressManager(5).SetMessage(msg),
			MaxBytesPerSec:  speedCap,
		},
	}); err != nil {
		msg.Edit("Error: " + err.Error())
		return nil
	} else {
		msg.Edit("Uploaded <code>" + filename + "</code> in <code>" + time.Since(uploadStartTimestamp).String() + "</code>")
	}

	return nil
}

func DownloadHandle(m *tg.NewMessage) error {
	if !m.IsReply() && m.Args() == "" {
		m.Reply("Reply to a file to download it")
		return nil
	}

	fn := m.Args()

	var r, msg *tg.NewMessage
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		r = reply
		var rerr error
		msg, rerr = m.Reply("Downloading...")
		if rerr != nil || msg == nil {
			return nil
		}
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
			if r == nil || r.File == nil {
				m.Reply("Not a file.")
				return nil
			}
			fn = r.File.Name
			msg, err = m.Reply("Downloading... (from c " + strconv.Itoa(id) + ")")
			if err != nil || msg == nil {
				return nil
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
			if r == nil || r.File == nil {
				m.Reply("Not a file.")
				return nil
			}
			fn = r.File.Name
			msg, err = m.Reply("Downloading... (from " + username + " " + strconv.Itoa(id) + ")")
			if err != nil || msg == nil {
				return nil
			}
		}
	}

	uploadStartTimestamp := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelMutex.Lock()
	downloadCancels[msg.ID] = cancel
	cancelMutex.Unlock()

	defer func() {
		cancelMutex.Lock()
		delete(downloadCancels, msg.ID)
		cancelMutex.Unlock()
	}()

	if fi, err := r.Download(&tg.DownloadOptions{
		ProgressManager: tg.NewProgressManager(5).SetMessage(msg),
		FileName:        fn,
		Ctx:             ctx,
		Delay:           150,
		MaxBytesPerSec:  defaultTransferSpeedCap,
	}); err != nil {
		if err == context.Canceled {
			msg.Edit("Download cancelled.")
		} else {
			msg.Edit("Error: " + err.Error())
		}
		return nil
	} else {
		msg.Edit("Downloaded <code>" + fi + "</code> in <code>" + time.Since(uploadStartTimestamp).String() + "</code>")
	}

	return nil
}

func CancelDownloadHandle(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a download message to cancel it")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	cancelMutex.RLock()
	cancel, exists := downloadCancels[reply.ID]
	cancelMutex.RUnlock()

	if !exists {
		m.Reply("No active download found for this message")
		return nil
	}

	cancel()
	m.Reply("Download cancelled!")
	return nil
}

func FileInfoHandle(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a file to get its info")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	var fi struct {
		FileName   string
		Type       string
		Size       int64
		FileID     string
		Attributes map[string]string
	}
	fi.Attributes = make(map[string]string)

	if r.File != nil {
		fi.FileName = r.File.Name
		fi.Size = r.File.Size
		fi.FileID = r.File.FileID
	}

	switch m := r.Message.Media.(type) {
	case *tg.MessageMediaDocument:
		fi.Type = "Document"
		for _, attr := range m.Document.(*tg.DocumentObj).Attributes {
			switch a := attr.(type) {
			case *tg.DocumentAttributeVideo:
				fi.Type = "Video"
				fi.Attributes["Duration"] = strconv.Itoa(int(a.Duration)) + " seconds"
				fi.Attributes["Width"] = strconv.Itoa(int(a.W)) + " px"
				fi.Attributes["Height"] = strconv.Itoa(int(a.H)) + " px"
			case *tg.DocumentAttributeAudio:
				fi.Type = "Audio"
				fi.Attributes["Duration"] = strconv.Itoa(int(a.Duration)) + " seconds"
				fi.Attributes["Title"] = a.Title
				fi.Attributes["Performer"] = a.Performer
				fi.Attributes["Voice"] = strconv.FormatBool(a.Voice)
			case *tg.DocumentAttributeAnimated:
				fi.Type = "Animated"
			case *tg.DocumentAttributeSticker:
				fi.Type = "Sticker"
				fi.Attributes["Alt"] = a.Alt
			}
		}
	case *tg.MessageMediaPhoto:
		fi.Type = "Photo"
	case *tg.MessageMediaPoll:
		fi.Type = "Poll"
	case *tg.MessageMediaGeo:
		fi.Type = "Geo"
		fi.Attributes["AccuracyRadius"] = strconv.Itoa(int(m.Geo.(*tg.GeoPointObj).AccuracyRadius)) + " meters"
		fi.Attributes["Latitude"] = strconv.FormatFloat(m.Geo.(*tg.GeoPointObj).Lat, 'f', 6, 64)
		fi.Attributes["Longitude"] = strconv.FormatFloat(m.Geo.(*tg.GeoPointObj).Long, 'f', 6, 64)
	default:
		fi.Type = "Unknown"
	}

	var output strings.Builder
	output.WriteString("<b>File Information</b>\n")
	output.WriteString("────────────────────\n")
	output.WriteString("<b>FileName</b>: <code>" + fi.FileName + "</code>\n")
	output.WriteString("<b>Type</b>: <code>" + fi.Type + "</code>\n")
	output.WriteString("<b>Size</b>: <code>" + HumanBytes(uint64(fi.Size)) + "</code>\n")
	output.WriteString("<b>FileID</b>: <code>" + fi.FileID + "</code>\n")
	if len(fi.Attributes) > 0 {
		output.WriteString("<b>Attributes</b>:\n")
		for k, v := range fi.Attributes {
			output.WriteString("   • <b>" + k + "</b>: <code>" + v + "</code>\n")
		}
	}

	m.Reply(output.String())
	return nil
}

func init() {
	QueueHandlerRegistration(registerFileHandlers)

	Mods.AddModule("Files", `<b>Here are the commands available in Files module:</b>

<code>/file &lt;fileId&gt;</code> - Send a file by its fileId
<code>/fid</code> - Reply to a file to get its fileId
<code>/ul &lt;filename&gt; [-s]</code> - Upload a file
<code>/dl</code> - Reply to a file to download it
<code>/cancel</code> - Reply to a download message to cancel it`)
}

func registerFileHandlers() {
	c := Client
	c.On("cmd:file", SendFileByIDHandle)
	c.On("cmd:fid", GetFileIDHandle)
	c.On("cmd:ul", UploadHandle, tg.CustomFilter(FilterOwnerNoReply))
	c.On("cmd:ldl", DownloadHandle, tg.CustomFilter(FilterOwnerNoReply))
	c.On("cmd:cancel", CancelDownloadHandle, tg.CustomFilter(FilterOwnerNoReply))
	c.On("cmd:fileinfo", FileInfoHandle)
	c.On("cmd:finfo", FileInfoHandle)
}
