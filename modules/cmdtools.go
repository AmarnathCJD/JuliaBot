package modules

import (
	"fmt"
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

	defer msg.Delete()

	cmd := exec.Command("ffmpeg", "-i", media, "-vn", "-ab", "128k", "-ar", "44100", "-y", fmt.Sprintf("%s_audio.mp3", media))
	err = cmd.Run()
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	_, err = m.ReplyMedia(fmt.Sprintf("%s_audio.mp3", media), telegram.MediaOptions{
		Caption: "Here is your audio file!",
	})

	return err
}
