package modules

import (
	"bufio"
	"bytes"
	"context"
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
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func ShellHandle(m *telegram.NewMessage) error {
	cmd := m.Args()
	if cmd == "" {
		m.Reply("No command provided")
		return nil
	}

	var cmx *exec.Cmd
	if runtime.GOOS == "windows" {
		cmx = exec.Command("cmd", "/C", cmd)
	} else {
		cmx = exec.Command("bash", "-c", cmd)
	}

	var out bytes.Buffer
	cmx.Stdout = &out
	var errx bytes.Buffer
	cmx.Stderr = &errx
	err := cmx.Run()

	stdout := strings.TrimSpace(out.String())
	stderr := strings.TrimSpace(errx.String())

	var result string
	if stdout != "" && stderr != "" {
		result = stdout + "\n\n<b>stderr:</b>\n" + stderr
	} else if stdout != "" {
		result = stdout
	} else if stderr != "" {
		result = stderr
	} else if err != nil {
		m.Reply("<code>Error:</code> <b>" + err.Error() + "</b>")
		return nil
	} else {
		m.Reply("<code>No Output</code>")
		return nil
	}

	if len(result) > 4000 {
		os.WriteFile("tmp/shell_out.txt", []byte(result), 0644)
		m.ReplyMedia("tmp/shell_out.txt", &telegram.MediaOptions{Caption: "Output"})
		os.Remove("tmp/shell_out.txt")
		return nil
	}

	m.Reply(`<pre language="bash">` + result + `</pre>`)
	return nil
}

func TcpHandler(m *telegram.NewMessage) error {
	pid := os.Getpid()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", fmt.Sprintf("netstat -ano | findstr %d", pid))
	} else {
		cmd = exec.Command("bash", "-c", fmt.Sprintf("ss -tnp 2>/dev/null | grep 'pid=%d' || cat /proc/%d/net/tcp 2>/dev/null", pid, pid))
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Run()

	output := strings.TrimSpace(out.String())
	count := strings.Count(output, "\n") + 1

	if output == "" {
		tcpConns := parseProcNetTcp(pid)
		if len(tcpConns) == 0 {
			m.Reply("<code>No active TCP connections</code>")
			return nil
		}
		output = strings.Join(tcpConns, "\n")
	}

	result := fmt.Sprintf("<b>TCP Connections (PID: %d)</b>\n\n<b>%s</b>", pid, output)
	result += fmt.Sprintf("\n\n<b>Total Connections:</b> %d", count)
	if len(result) > 4000 {
		os.WriteFile("tmp/tcp_out.txt", []byte(output), 0644)
		m.ReplyMedia("tmp/tcp_out.txt", &telegram.MediaOptions{Caption: fmt.Sprintf("TCP Connections (PID: %d)", pid)})
		os.Remove("tmp/tcp_out.txt")
		return nil
	}

	m.Reply(result)
	return nil
}

func parseProcNetTcp(pid int) []string {
	var connections []string

	fdPath := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := os.ReadDir(fdPath)
	if err != nil {
		return connections
	}

	socketInodes := make(map[string]bool)
	for _, fd := range fds {
		link, err := os.Readlink(filepath.Join(fdPath, fd.Name()))
		if err != nil {
			continue
		}
		if strings.HasPrefix(link, "socket:[") {
			inode := strings.TrimPrefix(strings.TrimSuffix(link, "]"), "socket:[")
			socketInodes[inode] = true
		}
	}

	tcpFile, err := os.Open("/proc/net/tcp")
	if err != nil {
		return connections
	}
	defer tcpFile.Close()

	scanner := bufio.NewScanner(tcpFile)
	scanner.Scan()

	stateMap := map[string]string{
		"01": "ESTABLISHED",
		"02": "SYN_SENT",
		"03": "SYN_RECV",
		"04": "FIN_WAIT1",
		"05": "FIN_WAIT2",
		"06": "TIME_WAIT",
		"07": "CLOSE",
		"08": "CLOSE_WAIT",
		"09": "LAST_ACK",
		"0A": "LISTEN",
		"0B": "CLOSING",
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}

		inode := fields[9]
		if !socketInodes[inode] {
			continue
		}

		localAddr := parseHexAddr(fields[1])
		remoteAddr := parseHexAddr(fields[2])
		state := stateMap[fields[3]]
		if state == "" {
			state = fields[3]
		}

		connections = append(connections, fmt.Sprintf("%s → %s [%s]", localAddr, remoteAddr, state))
	}

	return connections
}

func parseHexAddr(hexAddr string) string {
	parts := strings.Split(hexAddr, ":")
	if len(parts) != 2 {
		return hexAddr
	}

	ip := parts[0]
	if len(ip) == 8 {
		b1, _ := strconv.ParseUint(ip[6:8], 16, 8)
		b2, _ := strconv.ParseUint(ip[4:6], 16, 8)
		b3, _ := strconv.ParseUint(ip[2:4], 16, 8)
		b4, _ := strconv.ParseUint(ip[0:2], 16, 8)
		port, _ := strconv.ParseUint(parts[1], 16, 16)
		return fmt.Sprintf("%d.%d.%d.%d:%d", b1, b2, b3, b4, port)
	}

	return hexAddr
}

// --------- Eval function ------------

const boiler_code_for_eval = `
package main

import (
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
)

%s

var msgID int32 = %d
var chatID int64 = %d
var cacheJSON = %q

var client *tg.Client
var message *tg.NewMessage
var m *tg.NewMessage
var r *tg.NewMessage

func evalCode() {
	%s
}

func main() {
	var err error
	client, err = tg.NewClient(tg.ClientConfig{
		StringSession: %q,
		LogLevel:      tg.LogDisable,
	})
	if err != nil {
		fmt.Println("Client error:", err)
		return
	}

	if cacheJSON != "" {
		client.Cache.ImportJSON([]byte(cacheJSON))
	}

	if _, err := client.Conn(); err != nil {
		fmt.Println("Connection error:", err)
		return
	}

	msgs, err := client.GetMessages(chatID, &tg.SearchOption{
		IDs: int(msgID),
	})
	if err != nil || len(msgs) == 0 {
		fmt.Println("Failed to get message:", err)
		return
	}

	message = &msgs[0]
	m = message
	r, _ = message.GetReplyMessage()

	fmt.Println("output-start")
	evalCode()
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

	os.MkdirAll("tmp", 0755)
	msg, _ := m.Reply("Evaluating...")
	defer msg.Delete()

	resp, isfile := perfomEval(code, m, imports)
	if isfile {
		if _, err := m.ReplyMedia(resp, &telegram.MediaOptions{Caption: "Output"}); err != nil {
			m.Reply("Error: " + err.Error())
		}
		os.Remove(resp)
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

const goModContents = `module main

go 1.25.0

require github.com/amarnathcjd/gogram v1.6.10-0.20251206151850-63c357afc3a5
`

func perfomEval(code string, m *telegram.NewMessage, imports []string) (string, bool) {
	var importStatement string = ""
	if len(imports) > 0 {
		importStatement = "import (\n"
		for _, v := range imports {
			importStatement += "\t\"" + v + "\"\n"
		}
		importStatement += ")\n"
	}

	var chatID int64
	if m.Channel != nil {
		chatID = m.Channel.ID
	} else if m.Chat != nil {
		chatID = m.Chat.ID
	} else if m.Sender != nil {
		chatID = m.Sender.ID
	}

	cacheJSON, _ := m.Client.Cache.ExportJSON()
	code_file := fmt.Sprintf(boiler_code_for_eval, importStatement, m.ID, chatID, cacheJSON, code, m.Client.ExportSession())
	tmp_dir := "tmp"
	if _, err := os.ReadDir(tmp_dir); err != nil {
		if err := os.Mkdir(tmp_dir, 0755); err != nil {
			return fmt.Sprintf("❌ <b>System Error</b>\n<code>%s</code>", err.Error()), false
		}
	}

	evalFile := tmp_dir + "/eval.go"
	os.WriteFile(evalFile, []byte(code_file), 0644)
	defer os.Remove(evalFile)

	tmpGoMod := tmp_dir + "/go.mod"
	tmpGoSum := tmp_dir + "/go.sum"

	if _, err := os.Stat(tmpGoMod); os.IsNotExist(err) {
		os.WriteFile(tmpGoMod, []byte(goModContents), 0644)
	}

	if _, err := os.Stat(tmpGoSum); os.IsNotExist(err) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, "go", "mod", "download", "github.com/amarnathcjd/gogram@dev")
		cmd.Dir = tmp_dir
		cmd.Run()
		cancel()
	}

	binaryPath := tmp_dir + "/eval_bin"
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	defer os.Remove(binaryPath)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", filepath.Base(binaryPath), "eval.go")
	buildCmd.Dir = tmp_dir
	var buildErr bytes.Buffer
	buildCmd.Stderr = &buildErr

	if err := buildCmd.Run(); err != nil {
		stderr := strings.TrimSpace(buildErr.String())
		if stderr != "" {
			regexErr := regexp.MustCompile(`eval\.go:(\d+):(\d+):\s*`)
			errMsg := regexErr.ReplaceAllString(stderr, "Line $1: ")
			errMsg = strings.ReplaceAll(errMsg, "# command-line-arguments\n", "")
			errMsg = strings.TrimSpace(errMsg)
			return fmt.Sprintf("❌ <b>Compilation Error</b>\n<pre>%s</pre>", errMsg), false
		}
		return fmt.Sprintf("❌ <b>Build Error</b>\n<code>%s</code>", err.Error()), false
	}

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = tmp_dir
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	err := cmd.Run()
	stdout := strings.TrimSpace(stdOut.String())
	stderr := strings.TrimSpace(stdErr.String())

	if ctx.Err() == context.DeadlineExceeded {
		return "⏱️ <b>Timeout</b>\n<i>Execution exceeded 60 seconds</i>", false
	}

	if stdout == "" && stderr == "" {
		if err != nil {
			return fmt.Sprintf("❌ <b>Execution Error</b>\n<pre>%s</pre>", err.Error()), false
		}
		return "✅ <b>Eval Complete</b>\n<i>No output returned</i>", false
	}

	if stdout != "" {
		parts := strings.Split(stdout, "output-start")
		output := stdout
		if len(parts) > 1 {
			output = strings.TrimSpace(parts[1])
		}

		if len(output) > 4000 {
			os.WriteFile("tmp/eval_out.txt", []byte(output), 0644)
			return "tmp/eval_out.txt", true
		}

		return fmt.Sprintf("✅ <b>Eval Output</b>\n<pre>%s</pre>", output), false
	}

	if stderr != "" {
		regexErr := regexp.MustCompile(`eval\.go:(\d+):(\d+):\s*`)
		errMsg := regexErr.ReplaceAllString(stderr, "Line $1: ")

		errMsg = strings.ReplaceAll(errMsg, "# command-line-arguments\n", "")
		errMsg = strings.TrimSpace(errMsg)

		if len(errMsg) > 4000 {
			os.WriteFile("tmp/eval_out.txt", []byte(errMsg), 0644)
			return "tmp/eval_out.txt", true
		}

		return fmt.Sprintf("❌ <b>Compilation Error</b>\n<pre>%s</pre>", errMsg), false
	}

	return "✅ <b>Eval Complete</b>\n<i>No output returned</i>", false
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

		_, err = m.ReplyMedia(tmpFile.Name(), &telegram.MediaOptions{Caption: "Message JSON"})
		if err != nil {
			m.Reply("Error: " + err.Error())
		}
	} else {
		m.Reply("<pre language='json'>" + string(jsonString) + "</pre>")
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

	msg.Edit("<b><a href='"+url+"'>Media Info Pasted</a></b>", &telegram.SendOptions{
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

func GoHandler(m *telegram.NewMessage) error {
	m.Reply(fmt.Sprintf(`➜ <b>Current Go Routines: %d</b>`, runtime.NumGoroutine()))
	return nil
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

	_, phoneNum, err := m.Ask("Please enter your phone number")
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if ok, err := client.Login(phoneNum.Text(), &telegram.LoginOptions{
		CodeCallback: func() (string, error) {
			_, code, err := m.Ask("Please enter the code")
			if err != nil {
				m.Reply("Error: " + err.Error())
				return "", err
			}
			return code.Text(), nil
		},
		PasswordCallback: func() (string, error) {
			_, password, err := m.Ask("Please enter the @FA password")
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

func SpectrogramHandler(m *telegram.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to an audio file to generate spectrogram")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if !r.IsMedia() {
		m.Reply("This message is not a media file")
		return nil
	}

	msg, _ := m.Reply("<code>Generating spectrogram...</code>")
	fi, err := m.Client.DownloadMedia(r.Media())
	if err != nil {
		msg.Edit("Error downloading file: " + err.Error())
		return nil
	}
	defer os.Remove(fi)

	outputFile := "tmp/spectrogram_" + strconv.FormatInt(int64(m.ID), 10) + ".png"
	defer os.Remove(outputFile)

	wavFile := "tmp/audio_" + strconv.FormatInt(int64(m.ID), 10) + ".wav"
	defer os.Remove(wavFile)

	os.MkdirAll("tmp", 0755)

	convertCmd := exec.Command("ffmpeg", "-i", fi, "-ar", "44100", "-ac", "2", wavFile, "-y")
	var convertErr bytes.Buffer
	convertCmd.Stderr = &convertErr

	err = convertCmd.Run()
	if err != nil {
		errMsg := convertErr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		msg.Edit("<code>Error converting to WAV:</code> <b>" + errMsg + "</b>")
		return nil
	}

	cmd := exec.Command("sox", wavFile, "-n", "spectrogram", "-o", outputFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		msg.Edit("<code>Error generating spectrogram:</code> <b>" + errMsg + "</b>")
		return nil
	}
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		msg.Edit("<code>Error: Spectrogram file was not generated</code>")
		return nil
	}

	_, err = m.ReplyMedia(outputFile, &telegram.MediaOptions{
		Caption: "🎵 Audio Spectrogram",
	})
	if err != nil {
		msg.Edit("Error uploading spectrogram: " + err.Error())
		return nil
	}

	msg.Delete()
	return nil
}

var (
	spotifyRestartCMD = "sudo systemctl restart spotdl.service"
	proxyRestartCMD   = "sudo systemctl restart wireproxy.service"
	selfRestartCMD    = "sudo go run ."
)

func RestartSpotify(m *tg.NewMessage) error {
	msg, _ := m.Reply("Restarting Spotify...")
	if err := execCommand(spotifyRestartCMD); err != nil {
		return err
	}
	msg.Edit("Spotify restarted successfully.")
	return nil
}

func RestartProxy(m *tg.NewMessage) error {
	msg, _ := m.Reply("Restarting WProxy...")
	if err := execCommand(proxyRestartCMD); err != nil {
		return err
	}
	msg.Edit("Proxy restarted successfully.")
	return nil
}

func RestartHandle(m *tg.NewMessage) error {
	msg, _ := m.Reply("Restarting bot...")
	if err := execCommand(selfRestartCMD); err != nil {
		return err
	}
	defer os.Exit(0)
	msg.Edit("Bot restarted successfully.")
	return nil
}

func execCommand(cmd string) error {
	command := exec.Command("bash", "-c", cmd)
	_, err := command.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

func init() {
	Mods.AddModule("Dev", `<b>Here are the commands available in Dev module:</b>

- <code>/sh &lt;command&gt;</code> - Execute shell commands
- <code>/eval &lt;code&gt;</code> - Evaluate Go code
- <code>/json [-s | -m | -c] &lt;message&gt;</code> - Get JSON of a message`)
}
