package modules

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type activeStream struct {
	cmd       *exec.Cmd
	chatID    int64
	userID    int64
	fileName  string
	rtmpURL   string
	startTime time.Time
}

var (
	streams   = make(map[int]*activeStream)
	streamsMu sync.RWMutex
	streamID  int
)

func StreamHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("This command can only be used in groups")
		return nil
	}

	if !m.IsReply() {
		m.Reply("<b>Usage:</b> Reply to a video/audio file with:\n<code>/stream &lt;rtmp_url&gt;</code>")
		return nil
	}

	rtmpURL := strings.TrimSpace(m.Args())
	if rtmpURL == "" {
		m.Reply("<b>Error:</b> Please provide an RTMP URL\n<code>/stream rtmp://server/key</code>")
		return nil
	}

	reply, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("Failed to get reply message")
		return err
	}

	if !reply.IsMedia() {
		m.Reply("The replied message is not a media file")
		return nil
	}

	if reply.Audio() == nil && reply.Video() == nil && reply.Document() == nil {
		m.Reply("The replied message is not a video/audio file")
		return nil
	}

	msg, _ := m.Reply("Starting stream...")
	id := startStream(reply, rtmpURL, m.SenderID(), msg)
	if id > 0 {
		msg.Edit(fmt.Sprintf("🔴 <b>Stream started</b> (ID: <code>%d</code>)\n<b>File:</b> %s\n\nUse <code>/stopstream %d</code> to stop", id, reply.File.Name, id))
	}
	return nil
}

func startStream(m *tg.NewMessage, rtmpURL string, userID int64, statusMsg *tg.NewMessage) int {
	var chunkSize int64 = 1024 * 1024
	fileSize := m.File.Size

	cmd := exec.Command("ffmpeg",
		"-stream_loop", "-1",
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-preset", "superfast",
		"-b:v", "2000k",
		"-maxrate", "2000k",
		"-bufsize", "4000k",
		"-pix_fmt", "yuv420p",
		"-g", "30",
		"-threads", "0",
		"-c:a", "aac",
		"-b:a", "96k",
		"-ac", "2",
		"-ar", "44100",
		"-f", "flv",
		"-rtmp_buffer", "100",
		"-rtmp_live", "live",
		rtmpURL,
	)

	ffmpegIn, err := cmd.StdinPipe()
	if err != nil {
		statusMsg.Edit("Failed to initialize ffmpeg")
		return 0
	}

	if err := cmd.Start(); err != nil {
		statusMsg.Edit("Failed to start ffmpeg: " + err.Error())
		return 0
	}

	streamsMu.Lock()
	streamID++
	id := streamID
	streams[id] = &activeStream{
		cmd:       cmd,
		chatID:    m.ChatID(),
		userID:    userID,
		fileName:  m.File.Name,
		rtmpURL:   rtmpURL,
		startTime: time.Now(),
	}
	streamsMu.Unlock()

	go func() {
		defer ffmpegIn.Close()
		defer func() {
			streamsMu.Lock()
			delete(streams, id)
			streamsMu.Unlock()
		}()

		for i := int64(0); i < fileSize; i += chunkSize {
			chunk, _, err := m.Client.DownloadChunk(m.Media(), int(i), int(i+chunkSize), int(chunkSize))
			if err != nil {
				return
			}
			if _, err := ffmpegIn.Write(chunk); err != nil {
				return
			}
		}
	}()

	go func() {
		cmd.Wait()
		streamsMu.Lock()
		delete(streams, id)
		streamsMu.Unlock()
	}()

	return id
}

func ListStreamsHandler(m *tg.NewMessage) error {
	streamsMu.RLock()
	defer streamsMu.RUnlock()

	if len(streams) == 0 {
		m.Reply("No active streams")
		return nil
	}

	var sb strings.Builder
	sb.WriteString("<b>Active Streams:</b>\n\n")

	for id, s := range streams {
		duration := time.Since(s.startTime).Round(time.Second)
		sb.WriteString(fmt.Sprintf("<b>ID:</b> <code>%d</code>\n", id))
		sb.WriteString(fmt.Sprintf("<b>File:</b> %s\n", s.fileName))
		sb.WriteString(fmt.Sprintf("<b>Duration:</b> %s\n", duration))
		sb.WriteString(fmt.Sprintf("<b>Chat:</b> <code>%d</code>\n\n", s.chatID))
	}

	m.Reply(sb.String())
	return nil
}

func StopStreamHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/stopstream &lt;id&gt;</code>\nUse <code>/streams</code> to list active streams")
		return nil
	}

	id, err := strconv.Atoi(args)
	if err != nil {
		m.Reply("<b>Error:</b> Invalid stream ID")
		return nil
	}

	streamsMu.Lock()
	stream, exists := streams[id]
	if !exists {
		streamsMu.Unlock()
		m.Reply("<b>Error:</b> Stream not found")
		return nil
	}

	if stream.cmd != nil && stream.cmd.Process != nil {
		stream.cmd.Process.Kill()
	}
	delete(streams, id)
	streamsMu.Unlock()

	m.Reply(fmt.Sprintf("⏹ Stream <code>%d</code> stopped", id))
	return nil
}
