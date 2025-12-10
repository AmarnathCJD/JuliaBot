package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

type Aria2RPC struct {
	url   string
	token string
	mu    sync.Mutex
}

type Aria2Download struct {
	gid           string
	fileName      string
	totalLength   int64
	completed     int64
	downloadSpeed int64
	status        string
	progressMsg   *telegram.NewMessage
	chatID        int64
	userID        int64
	stopChan      chan struct{}
}

var (
	aria2Client    *Aria2RPC
	aria2Cmd       *exec.Cmd
	downloads      = make(map[string]*Aria2Download)
	downloadsMu    sync.RWMutex
	aria2Started   bool
	aria2StartedMu sync.Mutex
)

func initAria2() error {
	aria2StartedMu.Lock()
	defer aria2StartedMu.Unlock()

	if aria2Started {
		return nil
	}

	rpcPort := "6800"
	rpcSecret := "juliabot_secret"

	aria2Cmd = exec.Command("aria2c",
		"--enable-rpc",
		"--rpc-listen-all=false",
		"--rpc-listen-port="+rpcPort,
		"--rpc-secret="+rpcSecret,
		"--max-connection-per-server=16",
		"--max-concurrent-downloads=5",
		"--split=16",
		"--min-split-size=1M",
		"--continue=true",
		"--dir=tmp",
		"--allow-overwrite=true",
		"--auto-file-renaming=false",
		"--enable-mmap=true",
		"--file-allocation=none",
		"--follow-torrent=true",
		"--bt-enable-lpd=true",
		"--bt-max-peers=50",
		"--seed-time=0",
	)

	if err := aria2Cmd.Start(); err != nil {
		return fmt.Errorf("failed to start aria2c: %v", err)
	}

	time.Sleep(2 * time.Second)

	aria2Client = &Aria2RPC{
		url:   "http://localhost:" + rpcPort + "/jsonrpc",
		token: "token:" + rpcSecret,
	}

	aria2Started = true
	return nil
}

func (a *Aria2RPC) call(method string, params []interface{}) (map[string]interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if params == nil {
		params = []interface{}{}
	}
	params = append([]interface{}{a.token}, params...)

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  method,
		"params":  params,
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post(a.url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if errData, ok := result["error"]; ok {
		return nil, fmt.Errorf("aria2 error: %v", errData)
	}

	return result, nil
}

func (a *Aria2RPC) addURI(uri string) (string, error) {
	result, err := a.call("aria2.addUri", []interface{}{[]string{uri}})
	if err != nil {
		return "", err
	}
	gid, _ := result["result"].(string)
	return gid, nil
}

func (a *Aria2RPC) addTorrent(torrentData []byte) (string, error) {
	encoded := base64Encode(torrentData)
	result, err := a.call("aria2.addTorrent", []interface{}{encoded})
	if err != nil {
		return "", err
	}
	gid, _ := result["result"].(string)
	return gid, nil
}

func (a *Aria2RPC) tellStatus(gid string) (map[string]interface{}, error) {
	result, err := a.call("aria2.tellStatus", []interface{}{gid})
	if err != nil {
		return nil, err
	}
	status, ok := result["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid status response")
	}
	return status, nil
}

func (a *Aria2RPC) remove(gid string) error {
	_, err := a.call("aria2.remove", []interface{}{gid})
	return err
}

func (a *Aria2RPC) forceRemove(gid string) error {
	_, err := a.call("aria2.forceRemove", []interface{}{gid})
	return err
}

func (a *Aria2RPC) tellActive() ([]map[string]interface{}, error) {
	result, err := a.call("aria2.tellActive", nil)
	if err != nil {
		return nil, err
	}
	active, ok := result["result"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid active response")
	}
	var downloads []map[string]interface{}
	for _, item := range active {
		if dl, ok := item.(map[string]interface{}); ok {
			downloads = append(downloads, dl)
		}
	}
	return downloads, nil
}

func AddDLHandler(m *telegram.NewMessage) error {
	if err := initAria2(); err != nil {
		m.Reply(fmt.Sprintf("Failed to initialize aria2: %v", err))
		return nil
	}

	uri := strings.TrimSpace(m.Args())
	var gid string
	var err error

	if m.IsReply() && uri == "" {
		reply, rerr := m.GetReplyMessage()
		if rerr == nil {
			if doc := reply.Document(); doc != nil {
				fileName := reply.File.Name
				if strings.HasSuffix(strings.ToLower(fileName), ".torrent") {
					fileBytes, _ := m.Client.DownloadMedia(doc, nil)
					gid, err = aria2Client.addTorrent([]byte(fileBytes))
				}
			} else if reply.Text() != "" {
				uri = reply.Text()
			}
		}
	}

	if gid == "" {
		if uri == "" {
			m.Reply("<b>Usage:</b> <code>/adddl &lt;url/magnet&gt;</code>\n\nSupports:\n• HTTP/HTTPS\n• Magnet links\n• Torrent files (reply to .torrent)")
			return nil
		}

		gid, err = aria2Client.addURI(uri)
	}

	if err != nil {
		m.Reply(fmt.Sprintf("Failed to add download: %v", err))
		return nil
	}

	msg, _ := m.Reply(fmt.Sprintf("Download added\nGID: <code>%s</code>\n\nFetching info...", gid))

	download := &Aria2Download{
		gid:         gid,
		progressMsg: msg,
		chatID:      m.ChatID(),
		userID:      m.SenderID(),
		stopChan:    make(chan struct{}),
	}

	downloadsMu.Lock()
	downloads[gid] = download
	downloadsMu.Unlock()

	go updateProgress(download)

	return nil
}

func updateProgress(dl *Aria2Download) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dl.stopChan:
			return
		case <-ticker.C:
			status, err := aria2Client.tellStatus(dl.gid)
			if err != nil {
				continue
			}

			dl.status = status["status"].(string)
			if completedLength, ok := status["completedLength"].(string); ok {
				dl.completed, _ = strconv.ParseInt(completedLength, 10, 64)
			}
			if totalLength, ok := status["totalLength"].(string); ok {
				dl.totalLength, _ = strconv.ParseInt(totalLength, 10, 64)
			}
			if downloadSpeed, ok := status["downloadSpeed"].(string); ok {
				dl.downloadSpeed, _ = strconv.ParseInt(downloadSpeed, 10, 64)
			}

			if files, ok := status["files"].([]interface{}); ok && len(files) > 0 {
				if file, ok := files[0].(map[string]interface{}); ok {
					if path, ok := file["path"].(string); ok {
						parts := strings.Split(path, "/")
						dl.fileName = parts[len(parts)-1]
					}
				}
			}

			if dl.status == "complete" {
				dl.progressMsg.Edit(fmt.Sprintf("✅ <b>Download Complete</b>\n\n<b>File:</b> <code>%s</code>\n<b>Size:</b> <code>%s</code>\n<b>GID:</b> <code>%s</code>",
					dl.fileName, formatBytes(dl.totalLength), dl.gid))
				close(dl.stopChan)
				return
			}

			if dl.status == "error" || dl.status == "removed" {
				dl.progressMsg.Edit(fmt.Sprintf("❌ <b>Download Failed</b>\n\n<b>Status:</b> <code>%s</code>\n<b>GID:</b> <code>%s</code>",
					dl.status, dl.gid))
				close(dl.stopChan)
				return
			}

			if dl.totalLength > 0 {
				progress := float64(dl.completed) / float64(dl.totalLength) * 100
				eta := calculateETA(dl.completed, dl.totalLength, dl.downloadSpeed)
				progressBar := createProgressBar(progress)

				text := fmt.Sprintf("<b>Downloading</b>\n\n<b>File:</b> <code>%s</code>\n<b>Size:</b> <code>%s</code>\n<b>Downloaded:</b> <code>%s</code>\n<b>Speed:</b> <code>%s/s</code>\n<b>ETA:</b> <code>%s</code>\n<b>Progress:</b> %s <code>%.1f%%</code>\n<b>GID:</b> <code>%s</code>",
					dl.fileName,
					formatBytes(dl.totalLength),
					formatBytes(dl.completed),
					formatBytes(dl.downloadSpeed),
					eta,
					progressBar,
					progress,
					dl.gid,
				)

				dl.progressMsg.Edit(text)
			}
		}
	}
}

func ListDLsHandler(m *telegram.NewMessage) error {
	if err := initAria2(); err != nil {
		m.Reply(fmt.Sprintf("Failed to initialize aria2: %v", err))
		return nil
	}

	active, err := aria2Client.tellActive()
	if err != nil {
		m.Reply(fmt.Sprintf("Failed to get downloads: %v", err))
		return nil
	}

	if len(active) == 0 {
		m.Reply("No active downloads")
		return nil
	}

	text := "<b>Active Downloads:</b>\n\n"
	for i, dl := range active {
		gid := dl["gid"].(string)
		status := dl["status"].(string)
		completed, _ := strconv.ParseInt(dl["completedLength"].(string), 10, 64)
		total, _ := strconv.ParseInt(dl["totalLength"].(string), 10, 64)

		fileName := "Unknown"
		if files, ok := dl["files"].([]interface{}); ok && len(files) > 0 {
			if file, ok := files[0].(map[string]interface{}); ok {
				if path, ok := file["path"].(string); ok {
					parts := strings.Split(path, "/")
					fileName = parts[len(parts)-1]
				}
			}
		}

		progress := 0.0
		if total > 0 {
			progress = float64(completed) / float64(total) * 100
		}

		text += fmt.Sprintf("%d. <code>%s</code>\n   Status: <b>%s</b>\n   Progress: <code>%.1f%%</code>\n   GID: <code>%s</code>\n\n",
			i+1, fileName, status, progress, gid)
	}

	m.Reply(text)
	return nil
}

func RmDLHandler(m *telegram.NewMessage) error {
	if err := initAria2(); err != nil {
		m.Reply(fmt.Sprintf("Failed to initialize aria2: %v", err))
		return nil
	}

	gid := strings.TrimSpace(m.Args())
	if gid == "" {
		m.Reply("<b>Usage:</b> <code>/rmdl &lt;gid&gt;</code>")
		return nil
	}

	if err := aria2Client.forceRemove(gid); err != nil {
		m.Reply(fmt.Sprintf("Failed to remove download: %v", err))
		return nil
	}

	downloadsMu.Lock()
	if dl, ok := downloads[gid]; ok {
		close(dl.stopChan)
		delete(downloads, gid)
	}
	downloadsMu.Unlock()

	m.Reply(fmt.Sprintf("Download removed\nGID: <code>%s</code>", gid))
	return nil
}

func ListDLHandler(m *telegram.NewMessage) error {
	if err := initAria2(); err != nil {
		m.Reply(fmt.Sprintf("Failed to initialize aria2: %v", err))
		return nil
	}

	gid := strings.TrimSpace(m.Args())
	if gid == "" {
		m.Reply("<b>Usage:</b> <code>/listdl &lt;gid&gt;</code>")
		return nil
	}

	status, err := aria2Client.tellStatus(gid)
	if err != nil {
		m.Reply(fmt.Sprintf("Failed to get download info: %v", err))
		return nil
	}

	completed, _ := strconv.ParseInt(status["completedLength"].(string), 10, 64)
	total, _ := strconv.ParseInt(status["totalLength"].(string), 10, 64)
	speed, _ := strconv.ParseInt(status["downloadSpeed"].(string), 10, 64)
	dlStatus := status["status"].(string)

	fileName := "Unknown"
	if files, ok := status["files"].([]interface{}); ok && len(files) > 0 {
		if file, ok := files[0].(map[string]interface{}); ok {
			if path, ok := file["path"].(string); ok {
				parts := strings.Split(path, "/")
				fileName = parts[len(parts)-1]
			}
		}
	}

	progress := 0.0
	if total > 0 {
		progress = float64(completed) / float64(total) * 100
	}

	eta := calculateETA(completed, total, speed)
	progressBar := createProgressBar(progress)

	text := fmt.Sprintf("<b>Download Info</b>\n\n<b>File:</b> <code>%s</code>\n<b>Status:</b> <code>%s</code>\n<b>Size:</b> <code>%s</code>\n<b>Downloaded:</b> <code>%s</code>\n<b>Speed:</b> <code>%s/s</code>\n<b>ETA:</b> <code>%s</code>\n<b>Progress:</b> %s <code>%.1f%%</code>\n<b>GID:</b> <code>%s</code>",
		fileName,
		dlStatus,
		formatBytes(total),
		formatBytes(completed),
		formatBytes(speed),
		eta,
		progressBar,
		progress,
		gid,
	)

	m.Reply(text)
	return nil
}

func calculateETA(completed, total, speed int64) string {
	if speed == 0 || completed >= total {
		return "N/A"
	}
	remaining := total - completed
	seconds := remaining / speed
	return formatDuration(time.Duration(seconds) * time.Second)
}

func createProgressBar(progress float64) string {
	bars := int(progress / 10)
	filled := strings.Repeat("■", bars)
	empty := strings.Repeat("□", 10-bars)
	return filled + empty
}

func base64Encode(data []byte) string {
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder
	for i := 0; i < len(data); i += 3 {
		b1, b2, b3 := data[i], byte(0), byte(0)
		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}

		result.WriteByte(base64Table[b1>>2])
		result.WriteByte(base64Table[((b1&0x03)<<4)|(b2>>4)])
		if i+1 < len(data) {
			result.WriteByte(base64Table[((b2&0x0f)<<2)|(b3>>6)])
		} else {
			result.WriteByte('=')
		}
		if i+2 < len(data) {
			result.WriteByte(base64Table[b3&0x3f])
		} else {
			result.WriteByte('=')
		}
	}
	return result.String()
}
