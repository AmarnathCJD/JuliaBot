package modules

import (
	"fmt"
	"strings"
	"sync"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type StreamData struct {
	stream   *tg.RTMPStream
	chatID   int64
	media    tg.MessageMedia
	client   *tg.Client
	fileSize int64
}

var (
	activeStreams = make(map[int64]*StreamData)
	streamsMu     sync.RWMutex
	rtmpConfigs   = make(map[int64]string)
	rtmpConfigsMu sync.RWMutex
)

func SetRTMPHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("This command can only be used in groups")
		return nil
	}

	rtmpURL := strings.TrimSpace(m.Args())
	if rtmpURL == "" {
		m.Reply("<b>Usage:</b> <code>/setrtmp rtmp://server/app/key</code>")
		return nil
	}

	if !strings.HasPrefix(rtmpURL, "rtmp://") && !strings.HasPrefix(rtmpURL, "rtmps://") {
		m.Reply("<b>Error:</b> Invalid RTMP URL format")
		return nil
	}

	chatID := m.ChatID()
	rtmpConfigsMu.Lock()
	rtmpConfigs[chatID] = rtmpURL
	rtmpConfigsMu.Unlock()

	m.Reply(fmt.Sprintf("<b>RTMP URL set for this chat</b>\n<code>%s</code>", rtmpURL))
	return nil
}

func StreamHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("This command can only be used in groups")
		return nil
	}

	chatID := m.ChatID()

	rtmpConfigsMu.RLock()
	rtmpURL, exists := rtmpConfigs[chatID]
	rtmpConfigsMu.RUnlock()

	if !exists {
		m.Reply("<b>Error:</b> RTMP URL not configured\nUse <code>/setrtmp rtmp://server/app/key</code>")
		return nil
	}

	if !m.IsReply() {
		m.Reply("<b>Usage:</b> Reply to a video/audio file with <code>/stream</code>")
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
		m.Reply("Please reply to a video or audio file")
		return nil
	}

	streamsMu.RLock()
	_, streaming := activeStreams[chatID]
	streamsMu.RUnlock()

	if streaming {
		m.Reply("Stream is already active")
		return nil
	}

	go startStream(reply, rtmpURL, chatID)
	m.Reply("Started playing")
	return nil
}

func startStream(m *tg.NewMessage, rtmpURL string, chatID int64) {
	config := tg.DefaultRTMPConfig()
	st, err := m.Client.NewRTMPStream(chatID, config)
	if err != nil {
		return
	}

	if err := st.SetFullURL(rtmpURL); err != nil {
		return
	}

	if err := st.StartPipe(); err != nil {
		return
	}

	streamData := &StreamData{
		stream:   st,
		chatID:   chatID,
		media:    m.Media(),
		client:   m.Client,
		fileSize: m.File.Size,
	}

	streamsMu.Lock()
	activeStreams[chatID] = streamData
	streamsMu.Unlock()

	feedStream(streamData)
}

func feedStream(sd *StreamData) {
	defer func() {
		sd.stream.ClosePipe()
		streamsMu.Lock()
		delete(activeStreams, sd.chatID)
		streamsMu.Unlock()
	}()

	chunkSize := int64(512 * 1024)

	for offset := int64(0); offset < sd.fileSize; {
		endOffset := offset + chunkSize
		if endOffset > sd.fileSize {
			endOffset = sd.fileSize
		}

		chunk, _, err := sd.client.DownloadChunk(sd.media, int(offset), int(endOffset), int(chunkSize))
		if err != nil {
			return
		}

		if err := sd.stream.FeedChunk(chunk); err != nil {
			return
		}

		offset = endOffset
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
