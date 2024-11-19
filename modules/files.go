package modules

import (
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func SendFileByIDHandle(m *telegram.NewMessage) error {
	fileId := m.Args()
	if fileId == "" {
		m.Reply("No fileId provided")
		return nil
	}

	file, err := telegram.ResolveBotFileID(fileId)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	m.ReplyMedia(file)
	return nil
}

func GetFileIDHandle(m *telegram.NewMessage) error {
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

func UploadHandle(m *telegram.NewMessage) error {
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

	msg, _ := m.Reply("Uploading...")
	uploadStartTimestamp := time.Now()

	//var pm *telegram.ProgressManager

	if _, err := m.RespondMedia(filename, telegram.MediaOptions{
		Spoiler: spoiler,
		// ProgressCallback: func(total, curr int64) {
		// 	if pm == nil {
		// 		//pm = telegram.NewProgressManager(total, 5)
		// 	}
		// 	// if pm.ShouldEdit() {
		// 	// 	m.Client.EditMessage(m.ChatID(), msg.ID, pm.GetStats(curr))
		// 	// }
		// },
	}); err != nil {
		msg.Edit("Error: " + err.Error())
		return nil
	} else {

		msg.Edit("Uploaded " + filename + " in <code>" + time.Since(uploadStartTimestamp).String() + "</code>")
	}

	return nil
}

func DownloadHandle(m *telegram.NewMessage) error {
	if !m.IsReply() && m.Args() == "" {
		m.Reply("Reply to a file to download it")
		return nil
	}

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
		if len(match) != 3 {
			m.Reply("Invalid link")
			return nil
		}

		username := match[1]
		id, err := strconv.Atoi(match[2])
		if err != nil {
			m.Reply("Invalid link")
			return nil
		}

		msg, err := m.Client.GetMessageByID(username, int32(id))
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		r = msg
		msg, _ = m.Reply("Downloading... (from " + username + " " + strconv.Itoa(id) + ")")
	}

	uploadStartTimestamp := time.Now()

	//var pm *telegram.ProgressManager

	if fi, err := r.Download(&telegram.DownloadOptions{
		// ProgressCallback: func(total, curr int64) {
		// 	if pm == nil {
		// 	//	pm = telegram.NewProgressManager(total, 2)
		// 	}
		// 	if pm.ShouldEdit() {
		// 		//m.Client.EditMessage(m.ChatID(), msg.ID, pm.GetStats(curr))
		// 		fmt.Println(pm.GetStats(curr))
		// 	}
		// },
	}); err != nil {
		msg.Edit("Error: " + err.Error())
		return nil
	} else {
		runtime.GC()
		msg.Edit("Downloaded " + fi + " in <code>" + time.Since(uploadStartTimestamp).String() + "</code>")
	}

	return nil
}

func init() {
	Mods.AddModule("Files", `<b>Here are the commands available in Files module:</b>

<code>/file &lt;fileId&gt;</code> - Send a file by its fileId
<code>/fid</code> - Reply to a file to get its fileId
<code>/ul &lt;filename&gt; [-s]</code> - Upload a file
<code>/dl</code> - Reply to a file to download it`)
}
