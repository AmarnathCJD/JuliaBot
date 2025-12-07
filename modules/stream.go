package modules

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type QueueItem struct {
	fileName string
	fileSize int64
	media    tg.MessageMedia
	userID   int64
}

type StreamData struct {
	stream    *tg.RTMPStream
	chatID    int64
	userID    int64
	fileName  string
	fileSize  int64
	media     tg.MessageMedia
	client    *tg.Client
	startTime time.Time
	statusMsg *tg.NewMessage
	stopChan  chan struct{}
	isPaused  bool
	queue     []QueueItem
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

	streamsMu.Lock()
	sd, streaming := activeStreams[chatID]
	if streaming {
		sd.queue = append(sd.queue, QueueItem{
			fileName: reply.File.Name,
			fileSize: reply.File.Size,
			media:    reply.Media(),
			userID:   m.SenderID(),
		})
		streamsMu.Unlock()
		m.Reply(fmt.Sprintf("<b>Added to queue</b> (Position: %d)\n<b>File:</b> %s", len(sd.queue), reply.File.Name))
		return nil
	}
	streamsMu.Unlock()

	msg, _ := m.Reply("Initializing stream...")
	startStream(reply, rtmpURL, chatID, m.SenderID(), msg)
	return nil
}

func startStream(m *tg.NewMessage, rtmpURL string, chatID, userID int64, statusMsg *tg.NewMessage) {
	config := tg.DefaultRTMPConfig()
	st, err := m.Client.NewRTMPStream(chatID, config)
	if err != nil {
		statusMsg.Edit(fmt.Sprintf("<b>Error:</b> Failed to create stream\n<code>%s</code>", err.Error()))
		return
	}

	fmt.Println("Starting stream to URL:", rtmpURL)

	if err := st.SetFullURL(rtmpURL); err != nil {
		statusMsg.Edit(fmt.Sprintf("<b>Error:</b> Failed to set RTMP URL\n<code>%s</code>", err.Error()))
		return
	}

	if err := st.StartPipe(); err != nil {
		statusMsg.Edit(fmt.Sprintf("<b>Error:</b> Failed to start stream pipe\n<code>%s</code>", err.Error()))
		return
	}

	streamData := &StreamData{
		stream:    st,
		chatID:    chatID,
		userID:    userID,
		fileName:  m.File.Name,
		fileSize:  m.File.Size,
		media:     m.Media(),
		client:    m.Client,
		startTime: time.Now(),
		statusMsg: statusMsg,
		stopChan:  make(chan struct{}),
		isPaused:  false,
		queue:     []QueueItem{},
	}

	streamsMu.Lock()
	activeStreams[chatID] = streamData
	streamsMu.Unlock()

	updateStreamStatus(streamData)

	go feedStream(streamData)
	go monitorStream(streamData)
}

func feedStream(sd *StreamData) {
	defer func() {
		sd.stream.ClosePipe()
		streamsMu.Lock()
		delete(activeStreams, sd.chatID)
		streamsMu.Unlock()
	}()

	for {
		chunkSize := int64(512 * 1024)

		for offset := int64(0); offset < sd.fileSize; {
			select {
			case <-sd.stopChan:
				return
			default:
			}

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

		streamsMu.Lock()
		if len(sd.queue) == 0 {
			streamsMu.Unlock()
			return
		}

		next := sd.queue[0]
		sd.queue = sd.queue[1:]
		sd.fileName = next.fileName
		sd.fileSize = next.fileSize
		sd.media = next.media
		sd.userID = next.userID
		sd.startTime = time.Now()
		streamsMu.Unlock()

		sd.statusMsg.Edit(fmt.Sprintf("<b>Now streaming:</b> %s\n<b>Queued by:</b> <code>%d</code>", next.fileName, next.userID))
	}
}

func monitorStream(sd *StreamData) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sd.stopChan:
			return
		case <-ticker.C:
			updateStreamStatus(sd)
		}
	}
}

func updateStreamStatus(sd *StreamData) {
	duration := time.Since(sd.startTime).Round(time.Second)
	position := sd.stream.CurrentPosition()
	progress := float64(position) / float64(sd.fileSize) * 100

	status := fmt.Sprintf(
		"<b>Streaming:</b> %s\n\n"+
			"<b>Duration:</b> %s\n"+
			"<b>Progress:</b> %.1f%%\n"+
			"<b>Position:</b> %s / %s\n"+
			"<b>Queue:</b> %d files",
		sd.fileName,
		duration,
		progress,
		formatBytes(int64(position)),
		formatBytes(sd.fileSize),
		len(sd.queue),
	)

	keyboard := tg.NewKeyboard()
	if sd.isPaused {
		keyboard.AddRow(
			tg.Button.Data("▶️ Resume", fmt.Sprintf("stream_resume_%d", sd.chatID)),
			tg.Button.Data("⏹ Stop", fmt.Sprintf("stream_stop_%d", sd.chatID)),
		)
	} else {
		keyboard.AddRow(
			tg.Button.Data("⏸ Pause", fmt.Sprintf("stream_pause_%d", sd.chatID)),
			tg.Button.Data("⏹ Stop", fmt.Sprintf("stream_stop_%d", sd.chatID)),
		)
	}
	keyboard.AddRow(
		tg.Button.Data("🔄 Refresh", fmt.Sprintf("stream_refresh_%d", sd.chatID)),
		tg.Button.Data("📋 Queue", fmt.Sprintf("stream_queue_%d", sd.chatID)),
	)

	sd.statusMsg.Edit(status, &tg.SendOptions{ReplyMarkup: keyboard.Build()})
}

func StreamCallbackHandler(cb *tg.CallbackQuery) error {
	data := cb.DataString()
	if !strings.HasPrefix(data, "stream_") {
		return nil
	}

	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return nil
	}

	action := parts[1]
	chatID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil
	}

	streamsMu.RLock()
	sd, exists := activeStreams[chatID]
	streamsMu.RUnlock()

	if !exists {
		cb.Answer("Stream not found", &tg.CallbackOptions{Alert: true})
		return nil
	}

	if cb.Sender.ID != sd.userID {
		cb.Answer("Only the stream starter can control it", &tg.CallbackOptions{Alert: true})
		return nil
	}

	switch action {
	case "pause":
		sd.stream.Pause()
		streamsMu.Lock()
		sd.isPaused = true
		streamsMu.Unlock()
		cb.Answer("Stream paused")
		updateStreamStatus(sd)

	case "resume":
		sd.stream.Resume()
		streamsMu.Lock()
		sd.isPaused = false
		streamsMu.Unlock()
		cb.Answer("Stream resumed")
		updateStreamStatus(sd)

	case "stop":
		close(sd.stopChan)
		sd.stream.Stop()
		streamsMu.Lock()
		delete(activeStreams, chatID)
		streamsMu.Unlock()
		cb.Edit("<b>Stream stopped</b>")
		cb.Answer("Stream stopped")

	case "refresh":
		updateStreamStatus(sd)
		cb.Answer("Status refreshed")

	case "queue":
		streamsMu.RLock()
		if len(sd.queue) == 0 {
			streamsMu.RUnlock()
			cb.Answer("Queue is empty", &tg.CallbackOptions{Alert: true})
			return nil
		}
		var queueText strings.Builder
		queueText.WriteString("<b>Queue:</b>\n\n")
		for i, item := range sd.queue {
			queueText.WriteString(fmt.Sprintf("%d. %s\n   Size: %s\n\n", i+1, item.fileName, formatBytes(item.fileSize)))
		}
		streamsMu.RUnlock()
		cb.Answer(queueText.String(), &tg.CallbackOptions{Alert: true})
	}

	return nil
}

func ListStreamsHandler(m *tg.NewMessage) error {
	streamsMu.RLock()
	defer streamsMu.RUnlock()

	if len(activeStreams) == 0 {
		m.Reply("No active streams")
		return nil
	}

	var sb strings.Builder
	sb.WriteString("<b>Active Streams:</b>\n\n")

	for chatID, sd := range activeStreams {
		duration := time.Since(sd.startTime).Round(time.Second)
		position := sd.stream.CurrentPosition()
		progress := float64(position) / float64(sd.fileSize) * 100

		sb.WriteString(fmt.Sprintf("<b>Chat:</b> <code>%d</code>\n", chatID))
		sb.WriteString(fmt.Sprintf("<b>File:</b> %s\n", sd.fileName))
		sb.WriteString(fmt.Sprintf("<b>Duration:</b> %s\n", duration))
		sb.WriteString(fmt.Sprintf("<b>Progress:</b> %.1f%%\n\n", progress))
	}

	m.Reply(sb.String())
	return nil
}

func StopStreamHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("This command can only be used in groups")
		return nil
	}

	chatID := m.ChatID()

	streamsMu.Lock()
	sd, exists := activeStreams[chatID]
	if !exists {
		streamsMu.Unlock()
		m.Reply("No active stream in this chat")
		return nil
	}

	close(sd.stopChan)
	sd.stream.Stop()
	delete(activeStreams, chatID)
	streamsMu.Unlock()

	m.Reply("<b>Stream stopped</b>")
	return nil
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
