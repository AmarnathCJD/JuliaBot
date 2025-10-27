package modules

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func ShellHandle(m *telegram.NewMessage) error {
	cmd := m.Args()
	var cmd_args []string
	if cmd == "" {
		m.Reply("No command provided")
		return nil
	}

	if runtime.GOOS == "windows" {
		cmd = "cmd"
		cmd_args_b := strings.Split(m.Args(), " ")
		cmd_args = []string{"/C"}
		cmd_args = append(cmd_args, cmd_args_b...)
	} else {
		cmd = strings.Split(cmd, " ")[0]
		cmd_args = strings.Split(m.Args(), " ")
		cmd_args = append(cmd_args[:0], cmd_args[1:]...)
	}
	cmx := exec.Command(cmd, cmd_args...)
	var out bytes.Buffer
	cmx.Stdout = &out
	var errx bytes.Buffer
	cmx.Stderr = &errx
	err := cmx.Run()

	if errx.String() == "" && out.String() == "" {
		if err != nil {
			m.Reply("<code>Error:</code> <b>" + err.Error() + "</b>")
			return nil
		}
		m.Reply("<code>No Output</code>")
		return nil
	}

	if out.String() != "" {
		m.Reply(`<pre lang="bash">` + strings.TrimSpace(out.String()) + `</pre>`)
	} else {
		m.Reply(`<pre lang="bash">` + strings.TrimSpace(errx.String()) + `</pre>`)
	}
	return nil
}

// --------- Eval function ------------

const boiler_code_for_eval = `
package main

import "fmt"
import "github.com/amarnathcjd/gogram/telegram"
import "encoding/json"

%s

var msg_id int32 = %d

var client *telegram.Client
var message *telegram.NewMessage
var m *telegram.NewMessage
var r *telegram.NewMessage
` + "var msg = `%s`\nvar snd = `%s`\nvar cht = `%s`\nvar chn = `%s`\nvar cch = `%s`" + `


func evalCode() {
	%s
}

func main() {
	var msg_o *telegram.MessageObj
	var snd_o *telegram.UserObj
	var cht_o *telegram.ChatObj
	var chn_o *telegram.Channel
	json.Unmarshal([]byte(msg), &msg_o)
	json.Unmarshal([]byte(snd), &snd_o)
	json.Unmarshal([]byte(cht), &cht_o)
	json.Unmarshal([]byte(chn), &chn_o)
	client, _ = telegram.NewClient(telegram.ClientConfig{
		StringSession: "%s",
	})

	client.Cache.ImportJSON([]byte(cch))

	client.Conn()

	x := []telegram.User{}
	y := []telegram.Chat{}
	x = append(x, snd_o)
	if chn_o != nil {
		y = append(y, chn_o)
	}
	if cht_o != nil {
		y = append(y, cht_o)
	}
	client.Cache.UpdatePeersToCache(x, y)
	idx := 0
	if cht_o != nil {
		idx = int(cht_o.ID)
	}
	if chn_o != nil {
		idx = int(chn_o.ID)
	}
	if snd_o != nil && idx == 0 {
		idx = int(snd_o.ID)
	}

	messageX, err := client.GetMessages(idx, &telegram.SearchOption{
		IDs: int(msg_id),
	})

	if err != nil {
		fmt.Println(err)
	}

	message = &messageX[0]
	m = message
	r, _ = message.GetReplyMessage()

	fmt.Println("output-start")
	evalCode()
}

func packMessage(c *telegram.Client, message telegram.Message, sender *telegram.UserObj, channel *telegram.Channel, chat *telegram.ChatObj) *telegram.NewMessage {
	var (
		m = &telegram.NewMessage{}
	)
	switch message := message.(type) {
	case *telegram.MessageObj:
		m.ID = message.ID
		m.OriginalUpdate = message
		m.Message = message
		m.Client = c
	default:
		return nil
	}
	m.Sender = sender
	m.Chat = chat
	m.Channel = channel
	if m.Channel != nil && (m.Sender.ID == m.Channel.ID) {
		m.SenderChat = channel
	} else {
		m.SenderChat = &telegram.Channel{}
	}
	m.Peer, _ = c.GetSendablePeer(message.(*telegram.MessageObj).PeerID)

	/*if m.IsMedia() {
		FileID := telegram.PackBotFileID(m.Media())
		m.File = &telegram.CustomFile{
			FileID: FileID,
			Name:   getFileName(m.Media()),
			Size:   getFileSize(m.Media()),
			Ext:    getFileExt(m.Media()),
		}
	}*/
	return m
}
`

func resolveImports(code string) (string, []string) {
	var imports []string
	importsRegex := regexp.MustCompile(`import\s*\(([\s\S]*?)\)|import\s*\"([\s\S]*?)\"`)
	importsMatches := importsRegex.FindAllStringSubmatch(code, -1)
	for _, v := range importsMatches {
		if v[1] != "" {
			imports = append(imports, v[1])
		} else {
			imports = append(imports, v[2])
		}
	}
	code = importsRegex.ReplaceAllString(code, "")
	return code, imports
}

func EvalHandle(m *telegram.NewMessage) error {
	code := m.Args()
	code, imports := resolveImports(code)

	if code == "" {
		return nil
	}

	defer os.Remove("tmp/eval.go")
	defer os.Remove("tmp/eval_out.txt")
	defer os.Remove("tmp")

	resp, isfile := perfomEval(code, m, imports)
	if isfile {
		if _, err := m.ReplyMedia(resp, telegram.MediaOptions{Caption: "Output"}); err != nil {
			m.Reply("Error: " + err.Error())
		}
		return nil
	}
	resp = strings.TrimSpace(resp)

	if resp != "" {
		if _, err := m.Reply(resp); err != nil {
			m.Reply(err)
		}
	}
	return nil
}

func perfomEval(code string, m *telegram.NewMessage, imports []string) (string, bool) {
	msg_b, _ := json.Marshal(m.Message)
	snd_b, _ := json.Marshal(m.Sender)
	cnt_b, _ := json.Marshal(m.Chat)
	chn_b, _ := json.Marshal(m.Channel)
	cache_b, _ := m.Client.Cache.ExportJSON()
	var importStatement string = ""
	if len(imports) > 0 {
		importStatement = "import (\n"
		for _, v := range imports {
			importStatement += `"` + v + `"` + "\n"
		}
		importStatement += ")\n"
	}

	code_file := fmt.Sprintf(boiler_code_for_eval, importStatement, m.ID, msg_b, snd_b, cnt_b, chn_b, cache_b, code, m.Client.ExportSession())
	tmp_dir := "tmp"
	_, err := os.ReadDir(tmp_dir)
	if err != nil {
		err = os.Mkdir(tmp_dir, 0755)
		if err != nil {
			fmt.Println(err)
		}
	}

	//defer os.Remove(tmp_dir)

	os.WriteFile(tmp_dir+"/eval.go", []byte(code_file), 0644)
	cmd := exec.Command("go", "run", "tmp/eval.go")
	var stdOut bytes.Buffer
	cmd.Stdout = &stdOut
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	err = cmd.Run()
	if stdOut.String() == "" && stdErr.String() == "" {
		if err != nil {
			return fmt.Sprintf("<b>#EVALERR:</b> <code>%s</code>", err.Error()), false
		}
		return "<b>#EVALOut:</b> <code>No Output</code>", false
	}

	if stdOut.String() != "" {
		if len(stdOut.String()) > 4095 {
			os.WriteFile("tmp/eval_out.txt", stdOut.Bytes(), 0644)
			return "tmp/eval_out.txt", true
		}

		strDou := strings.Split(stdOut.String(), "output-start")

		return fmt.Sprintf("<b>#EVALOut:</b> <code>%s</code>", strings.TrimSpace(strDou[1])), false
	}

	if stdErr.String() != "" {
		var regexErr = regexp.MustCompile(`eval.go:\d+:\d+:`)
		errMsg := regexErr.Split(stdErr.String(), -1)
		if len(errMsg) > 1 {
			if len(errMsg[1]) > 4095 {
				os.WriteFile("tmp/eval_out.txt", []byte(errMsg[1]), 0644)
				return "tmp/eval_out.txt", true
			}
			return fmt.Sprintf("<b>#EVALERR:</b> <code>%s</code>", strings.TrimSpace(errMsg[1])), false
		}
		return fmt.Sprintf("<b>#EVALERR:</b> <code>%s</code>", stdErr.String()), false
	}

	return "<b>#EVALOut:</b> <code>No Output</code>", false
}

func JsonHandle(m *telegram.NewMessage) error {
	var jsonString []byte
	if !m.IsReply() {
		if strings.Contains(m.Args(), "-s") {
			jsonString, _ = json.MarshalIndent(m.Sender, "", "  ")
		} else if strings.Contains(m.Args(), "-m") {
			jsonString, _ = json.MarshalIndent(m.Media(), "", "  ")
		} else if strings.Contains(m.Args(), "-c") {
			jsonString, _ = json.MarshalIndent(m.Channel, "", "  ")
		} else {
			jsonString, _ = json.MarshalIndent(m.OriginalUpdate, "", "  ")
		}
	} else {
		r, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("<code>Error:</code> <b>" + err.Error() + "</b>")
			return nil
		}
		if strings.Contains(m.Args(), "-s") {
			jsonString, _ = json.MarshalIndent(r.Sender, "", "  ")
		} else if strings.Contains(m.Args(), "-m") {
			jsonString, _ = json.MarshalIndent(r.Media(), "", "  ")
		} else if strings.Contains(m.Args(), "-c") {
			jsonString, _ = json.MarshalIndent(r.Channel, "", "  ")
		} else if strings.Contains(m.Args(), "-f") {
			jsonString, _ = json.MarshalIndent(r.File, "", "  ")
		} else {
			jsonString, _ = json.MarshalIndent(r.OriginalUpdate, "", "  ")
		}
	}

	// find all "Data": "<base64>" and decode and replace with actual data
	dataFieldRegex := regexp.MustCompile(`"Data": "([a-zA-Z0-9+/]+={0,2})"`)
	dataFields := dataFieldRegex.FindAllStringSubmatch(string(jsonString), -1)
	for _, v := range dataFields {
		decoded, err := base64.StdEncoding.DecodeString(v[1])
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}
		jsonString = []byte(strings.ReplaceAll(string(jsonString), v[0], `"Data": "`+string(decoded)+`"`))
	}

	if len(jsonString) > 4095 {
		defer os.Remove("message.json")
		tmpFile, err := os.Create("message.json")
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		_, err = tmpFile.Write(jsonString)
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		_, err = m.ReplyMedia(tmpFile.Name(), telegram.MediaOptions{Caption: "Message JSON"})
		if err != nil {
			m.Reply("Error: " + err.Error())
		}
	} else {
		m.Reply("<pre lang='json'>" + string(jsonString) + "</pre>")
	}

	return nil
}

func formatMediaInfo(info string) string {
	lines := strings.Split(info, "\n")
	var formatted strings.Builder
	formatted.WriteString("<b>📊 Media Information</b>\n\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			formatted.WriteString("\n")
			continue
		}

		// Check if line is a section header (no colon, usually all caps or title case)
		if !strings.Contains(line, ":") && len(line) > 0 {
			// Section headers in bold
			formatted.WriteString("<b>" + line + "</b>\n")
		} else if strings.Contains(line, ":") {
			// Split key-value pairs
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Format key in bold, value in regular text
				if value != "" {
					formatted.WriteString("<b>" + key + ":</b> <code>" + value + "</code>\n")
				} else {
					formatted.WriteString("<b>" + key + ":</b>\n")
				}
			} else {
				formatted.WriteString(line + "\n")
			}
		} else {
			formatted.WriteString(line + "\n")
		}
	}

	return formatted.String()
}

func MediaInfoHandler(m *telegram.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a message to get media info")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if !r.IsMedia() {
		m.Reply("This message is not a media")
		return nil
	}

	msg, _ := m.Reply("<code>Gathering media info...</code>")

	var downloadedFileName string
	if r.File.Size > 40*1024*1024 { // 20MB
		// download first 40MB of the file

		bytes, _, err := m.Client.DownloadChunk(r.Media(), 0, 40*1024*1024, 512*1024)
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		os.WriteFile("tmp/media", bytes, 0644)
		downloadedFileName = "tmp/media"
	} else {
		fi, err := m.Client.DownloadMedia(r.Media())
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		downloadedFileName = fi
	}
	defer os.Remove(downloadedFileName)

	cmd := exec.Command("mediainfo", downloadedFileName)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	mediaInfoOutput := strings.Trim(out.String(), "\n")

	// If output is less than 3000 characters, format and send as message
	if len(mediaInfoOutput) < 3000 {
		formattedOutput := formatMediaInfo(mediaInfoOutput)
		msg.Edit(formattedOutput)
		return nil
	}

	// Otherwise, post to pastebin
	url, _, err := postToSpaceBin(mediaInfoOutput)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	msg.Edit("<b><a href='"+url+"'>Media Info Pasted</a></b>", telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().AddRow(
			telegram.Button.URL("View", url),
		).Build(),
		LinkPreview: true,
	})
	return nil
}

func LsHandler(m *telegram.NewMessage) error {
	dir := m.Args()
	if dir == "" {
		dir = "."
	}
	cmd := exec.Command("ls", dir)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	fileTypeEmoji := map[string]string{
		"file":   "📄",
		"dir":    "📁",
		"video":  "🎥",
		"audio":  "🎵",
		"image":  "🖼️",
		"go":     "📜",
		"python": "🐍",
		"txt":    "📝",
	}

	if err != nil {
		m.Reply("<code>Error:</code> <b>" + err.Error() + "</b>")
		return nil
	}

	files := strings.Split(strings.TrimSpace(out.String()), "\n")
	var sizeTotal int64

	var resp string
	for _, file := range files {
		fileType := "file"
		if strings.Contains(file, ".") {
			fp := strings.Split(file, ".")
			fileType = fp[len(fp)-1]
		}
		switch fileType {
		case "mp4", "mkv", "webm", "avi", "flv", "mov", "wmv", "3gp":
			fileType = "video"
		case "mp3", "wav", "flac", "ogg", "m4a", "wma":
			fileType = "audio"
		case "jpg", "jpeg", "png", "gif", "webp", "bmp", "tiff":
			fileType = "image"
		case "go":
			fileType = "go"
		case "py":
			fileType = "python"
		case "txt":
			fileType = "txt"
		default:
			fileType = "file"
		}
		size := calcFileOrDirSize(filepath.Join(dir, file))
		sizeTotal += size
		resp += fileTypeEmoji[fileType] + " " + file + " " + "(" + sizeToHuman(size) + ")" + "\n"
	}

	resp += "\nTotal: " + sizeToHuman(sizeTotal)

	m.Reply("<pre lang='bash'>" + resp + "</pre>")
	return nil
}

func sizeToHuman(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
}

func calcFileOrDirSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}

	if !fi.IsDir() {
		return fi.Size()
	}

	var size int64
	walker := func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fi, err := info.Info()
			if err != nil {
				return err
			}
			size += fi.Size()
		}
		return nil
	}

	err = filepath.WalkDir(path, walker)
	if err != nil {
		return 0
	}

	return size
}

func GenStringSessionHandler(m *telegram.NewMessage) error {
	if !m.IsPrivate() {
		m.Reply("This command can only be used in private chat")
		return nil
	}

	var appId = os.Getenv("APP_ID")
	appIdInt, _ := strconv.Atoi(appId)

	client, _ := telegram.NewClient(telegram.ClientConfig{
		AppID:         int32(appIdInt),
		AppHash:       os.Getenv("APP_HASH"),
		LogLevel:      telegram.LogDisable,
		MemorySession: true,
	})
	defer client.Terminate()

	phoneNum, err := m.Ask("Please enter your phone number")
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if ok, err := client.Login(phoneNum.Text(), &telegram.LoginOptions{
		CodeCallback: func() (string, error) {
			code, err := m.Ask("Please enter the code")
			if err != nil {
				m.Reply("Error: " + err.Error())
				return "", err
			}
			return code.Text(), nil
		},
		PasswordCallback: func() (string, error) {
			password, err := m.Ask("Please enter the @FA password")
			if err != nil {
				m.Reply("Error: " + err.Error())
				return "", err
			}
			return password.Text(), nil
		},
	}); !ok {
		if _, err := client.GetMe(); err == nil {
			m.Respond("Your string session is: <code>" + client.ExportSession() + "</code>")
			return nil
		}

		m.Reply("Error: " + err.Error())
		return nil
	}

	m.Respond("Your string session is: <code>" + client.ExportSession() + "</code>")
	return nil
}

func SetBotPfpHandler(m *telegram.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a photo to set it as bot profile picture")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if !r.IsMedia() {
		m.Reply("This message is not a media")
		return nil
	}

	fi, err := m.Client.DownloadMedia(r.Media())
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	defer os.Remove(fi)
	fiup, err := m.Client.UploadFile(fi)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	_, err = m.Client.PhotosUploadProfilePhoto(&telegram.PhotosUploadProfilePhotoParams{
		File: fiup,
	})

	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	m.Reply("Profile picture updated")
	return nil
}

func init() {
	Mods.AddModule("Dev", `<b>Here are the commands available in Dev module:</b>

- <code>/sh &lt;command&gt;</code> - Execute shell commands
- <code>/eval &lt;code&gt;</code> - Evaluate Go code
- <code>/json [-s | -m | -c] &lt;message&gt;</code> - Get JSON of a message`)
}
