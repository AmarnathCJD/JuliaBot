package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/amarnathcjd/gogram/telegram"
)

func ConvertToAudioHandle(m *telegram.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Please reply to a video message to convert it to audio.")
		return nil
	}

	vidMsg, ok := m.GetReplyMessage()
	if ok != nil {
		m.Reply("Error fetching the replied message.")
		return nil
	}

	msg, _ := m.Reply("<code>Converting to audio...</code>")

	media, err := vidMsg.Download(&telegram.DownloadOptions{
		ProgressManager: telegram.NewProgressManager(5).SetMessage(msg),
	})
	if err != nil {
		m.Reply("Error downloading the video.")
		return nil
	}
	msg.Edit("<code>Downloaded, converting...</code>")
	defer msg.Delete()

	thumbPath := fmt.Sprintf("%s_thumb.jpg", media)
	thumbCmd := exec.Command("ffmpeg", "-i", media, "-ss", "00:00:01", "-vframes", "1", "-y", thumbPath)
	thumbCmd.Run()
	defer os.Remove(thumbPath)

	var title, performer string
	metaCmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", media)
	if output, err := metaCmd.Output(); err == nil {
		var metadata struct {
			Format struct {
				Tags map[string]string `json:"tags"`
			} `json:"format"`
		}
		if err := json.Unmarshal(output, &metadata); err == nil {
			if t, ok := metadata.Format.Tags["title"]; ok {
				title = t
			}
			if a, ok := metadata.Format.Tags["artist"]; ok {
				performer = a
			} else if a, ok := metadata.Format.Tags["album_artist"]; ok {
				performer = a
			} else if a, ok := metadata.Format.Tags["ARTIST"]; ok {
				performer = a
			}
		}
	}

	audioPath := fmt.Sprintf("%s_audio.mp3", media)
	cmd := exec.Command("ffmpeg", "-i", media, "-vn", "-c:a", "libmp3lame", "-q:a", "2", "-y", audioPath)
	err = cmd.Run()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	attrs := []telegram.DocumentAttribute{
		&telegram.DocumentAttributeFilename{
			FileName: func() string {
				if title != "" {
					return title + ".mp3"
				}
				return "audio.mp3"
			}(),
		},
		&telegram.DocumentAttributeAudio{
			Title:     title,
			Performer: performer,
		},
	}

	_, err = m.ReplyMedia(audioPath, &telegram.MediaOptions{
		Caption:    "Here is your audio file",
		Thumb:      thumbPath,
		Attributes: attrs,
	})

	os.Remove(media)
	os.Remove(audioPath)

	return err
}
