package modules

import (
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

	tg "github.com/amarnathcjd/gogram/telegram"
)

func ShellHandle(m *tg.NewMessage) error {
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
		m.ReplyMedia("tmp/shell_out.txt", &tg.MediaOptions{Caption: "Output"})
		os.Remove("tmp/shell_out.txt")
		return nil
	}

	m.Reply(`<pre language="bash">` + result + `</pre>`)
	return nil
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

func EvalHandle(m *tg.NewMessage) error {
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
		if _, err := m.ReplyMedia(resp, &tg.MediaOptions{Caption: "Output"}); err != nil {
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

func perfomEval(code string, m *tg.NewMessage, imports []string) (string, bool) {
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

	if _, err := os.Stat(tmpGoMod); os.IsNotExist(err) {
		os.WriteFile(tmpGoMod, []byte(goModContents), 0644)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
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
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	err := cmd.Run()
	stdout := strings.TrimSpace(stdOut.String())
	stderr := strings.TrimSpace(stdErr.String())

	if ctx.Err() == context.DeadlineExceeded {
		return "Timeout", false
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

func JsonHandle(m *tg.NewMessage) error {
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

		_, err = m.ReplyMedia(tmpFile.Name(), &tg.MediaOptions{Caption: "Message JSON"})
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

func MediaInfoHandler(m *tg.NewMessage) error {
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

	msg.Edit("<b><a href='"+url+"'>Media Info Pasted</a></b>", &tg.SendOptions{
		ReplyMarkup: tg.NewKeyboard().AddRow(
			tg.Button.URL("View", url),
		).Build(),
		LinkPreview: true,
	})
	return nil
}

func LsHandler(m *tg.NewMessage) error {
	dir := m.Args()
	if dir == "" {
		dir = "."
	}

	absPath, _ := filepath.Abs(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		m.Reply("❌ <b>Error:</b> <code>" + err.Error() + "</code>")
		return nil
	}

	fileTypeEmoji := map[string]string{
		"file":   "📄",
		"dir":    "📁",
		"video":  "🎬",
		"audio":  "🎵",
		"image":  "🖼",
		"go":     "📄",
		"python": "🐍",
		"txt":    "📝",
		"json":   "📋",
		"zip":    "📦",
		"exe":    "⚙️",
	}

	var resp strings.Builder
	resp.WriteString("📂 <b>" + absPath + "</b>\n")
	resp.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")

	var sizeTotal int64
	var fileCount, dirCount int

	var dirs, files []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	for _, entry := range dirs {
		name := entry.Name()
		size := calcFileOrDirSize(filepath.Join(dir, name))
		sizeTotal += size
		dirCount++
		resp.WriteString(fileTypeEmoji["dir"] + " <code>" + name + "/</code>  <i>" + sizeToHuman(size) + "</i>\n")
	}

	for _, entry := range files {
		name := entry.Name()
		fileType := "file"

		if idx := strings.LastIndex(name, "."); idx != -1 {
			ext := strings.ToLower(name[idx+1:])
			switch ext {
			case "mp4", "mkv", "webm", "avi", "flv", "mov", "wmv", "3gp":
				fileType = "video"
			case "mp3", "wav", "flac", "ogg", "m4a", "wma", "opus":
				fileType = "audio"
			case "jpg", "jpeg", "png", "gif", "webp", "bmp", "tiff", "svg":
				fileType = "image"
			case "go", "mod", "sum":
				fileType = "go"
			case "py", "pyw":
				fileType = "python"
			case "txt", "md", "log":
				fileType = "txt"
			case "json", "yaml", "yml", "toml", "xml":
				fileType = "json"
			case "zip", "rar", "7z", "tar", "gz":
				fileType = "zip"
			case "exe", "msi", "dll":
				fileType = "exe"
			}
		}

		emoji := fileTypeEmoji[fileType]
		if emoji == "" {
			emoji = fileTypeEmoji["file"]
		}

		info, _ := entry.Info()
		var size int64
		if info != nil {
			size = info.Size()
		}
		sizeTotal += size
		fileCount++
		resp.WriteString(emoji + " <code>" + name + "</code>  <i>" + sizeToHuman(size) + "</i>\n")
	}

	resp.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(&resp, "📊 <b>%d</b> files, <b>%d</b> folders • <b>%s</b> total", fileCount, dirCount, sizeToHuman(sizeTotal))

	m.Reply(resp.String())
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

func GoHandler(m *tg.NewMessage) error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	heapAlloc := float64(memStats.HeapAlloc) / 1024 / 1024
	heapSys := float64(memStats.HeapSys) / 1024 / 1024
	heapInuse := float64(memStats.HeapInuse) / 1024 / 1024
	stackInuse := float64(memStats.StackInuse) / 1024 / 1024
	numGC := memStats.NumGC

	resp := fmt.Sprintf(`<b>Go Runtime Stats</b>
━━━━━━━━━━━━━━━━━━━━

<b>Goroutines</b>: <code>%d</code>

<b>Heap Memory</b>
  • Allocated: <code>%.2f MB</code>
  • System: <code>%.2f MB</code>	
  • In Use: <code>%.2f MB</code>

<b>Stack</b>: <code>%.2f MB</code>
<b>GC Cycles</b>: <code>%d</code>`,
		runtime.NumGoroutine(),
		heapAlloc,
		heapSys,
		heapInuse,
		stackInuse,
		numGC,
	)

	m.Reply(resp)
	return nil
}

func GenStringSessionHandler(m *tg.NewMessage) error {
	if !m.IsPrivate() {
		m.Reply("This command can only be used in private chat")
		return nil
	}

	var appId = os.Getenv("APP_ID")
	appIdInt, _ := strconv.Atoi(appId)

	client, _ := tg.NewClient(tg.ClientConfig{
		AppID:         int32(appIdInt),
		AppHash:       os.Getenv("APP_HASH"),
		LogLevel:      tg.LogDisable,
		MemorySession: true,
	})
	defer client.Terminate()

	_, phoneNum, err := m.Ask("Please enter your phone number")
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if ok, err := client.Login(phoneNum.Text(), &tg.LoginOptions{
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

func SetBotPfpHandler(m *tg.NewMessage) error {
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

	_, err = m.Client.PhotosUploadProfilePhoto(&tg.PhotosUploadProfilePhotoParams{
		File: fiup,
	})

	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	m.Reply("Profile picture updated")
	return nil
}

func SpectrogramHandler(m *tg.NewMessage) error {
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

	_, err = m.ReplyMedia(outputFile, &tg.MediaOptions{
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

func HandlePostCommand(m *tg.NewMessage) error {
	args := strings.Fields(m.Args())

	var targetChannel string
	var dropMedia bool
	var forwardTag bool
	var contentText string
	var contentArgs []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c":
			if i+1 < len(args) {
				targetChannel = args[i+1]
				i++
			}
		case "-nm":
			dropMedia = true
		case "-fw":
			forwardTag = true
		default:
			contentArgs = append(contentArgs, args[i])
		}
	}

	var media tg.MessageMedia
	var hasMedia bool

	if m.IsReply() {
		replyMsg, err := m.GetReplyMessage()
		if err == nil {
			contentText = replyMsg.Text()
			if !dropMedia && replyMsg.Media() != nil {
				hasMedia = true
				media = replyMsg.Media()
			}
		}
	}

	if contentText == "" && len(contentArgs) > 0 {
		contentText = strings.Join(contentArgs, " ")
	}

	if contentText == "" {
		m.Reply("Please provide content to post (via reply or arguments)")
		return nil
	}

	if targetChannel == "" {
		m.Reply("Please specify target channel with -c flag")
		return nil
	}

	var targetChannelID any

	if _, err := fmt.Sscanf(targetChannel, "%d", &targetChannelID); err != nil {
		user, err := m.Client.ResolveUsername(targetChannel)
		if err != nil {
			m.Reply("Could not resolve channel: " + targetChannel)
			return nil
		}
		targetChannelID = user
	}

	if forwardTag {
		contentText += "\n\n📌 <i>Forwarded</i>"
	}

	opts := &tg.SendOptions{}

	if hasMedia {
		_, err := m.Client.SendMedia(targetChannelID, media, &tg.MediaOptions{
			Caption: contentText,
		})
		if err != nil {
			m.Reply("Failed to post media: " + err.Error())
			return nil
		}
	} else {
		_, err := m.Client.SendMessage(targetChannelID, contentText, opts)
		if err != nil {
			m.Reply("Failed to post message: " + err.Error())
			return nil
		}
	}

	m.Reply("✓ Posted to " + targetChannel)
	return nil
}

func init() {
	QueueHandlerRegistration(registerDevHandlers)

	Mods.AddModule("Dev", `<b>Here are the commands available in Dev module:</b>

- <code>/sh &lt;command&gt;</code> - Execute shell commands
- <code>/eval &lt;code&gt;</code> - Evaluate Go code
- <code>/json [-s | -m | -c] &lt;message&gt;</code> - Get JSON of a message
- <code>/mediainfo</code> - Get media information of a replied media
- <code>/ls [directory]</code> - List files in a directory
- <code>/go</code> - Get Go runtime stats
- <code>/gensession</code> - Generate a new string session
- <code>/setpfp</code> - Set bot profile picture
- <code>/spectrogram</code> - Generate spectrogram of an audio file
- <code>/restart</code> - Restart the bot
- <code>/post -c &lt;channel&gt; [-nm] [-fw] &lt;content&gt;</code> - Post content to a channel
`)
}

func registerDevHandlers() {
	c := Client
	c.On("cmd:sh", ShellHandle, tg.CustomFilter(FilterOwner))
	c.On("cmd:bash", ShellHandle, tg.CustomFilter(FilterOwner))
	c.On("cmd:eval", EvalHandle, tg.CustomFilter(FilterOwnerNoReply))
	c.On("cmd:go", GoHandler)
	c.On("cmd:json", JsonHandle)
	c.On("cmd:ls", LsHandler, tg.CustomFilter(FilterOwner))
	c.On("cmd:sessgen", GenStringSessionHandler)
	c.On("cmd:setpfp", SetBotPfpHandler, tg.CustomFilter(FilterOwner))
	c.On("cmd:spec", SpectrogramHandler)
	c.On("cmd:upd", UpdateSourceCodeHandle, tg.CustomFilter(FilterOwnerNoReply))
	c.On("cmd:post", HandlePostCommand, tg.CustomFilter(FilterOwner))
	c.On("cmd:mediainfo", MediaInfoHandler)
}
