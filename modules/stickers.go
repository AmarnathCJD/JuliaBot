package modules

import (
	"os"
	"os/exec"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func GifToSticker(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("<b>Error:</b> Please reply to a GIF message to convert it to a sticker.")
		return nil
	}

	r, err := m.GetReplyMessage()
	if err != nil {
		m.Reply("<b>Error:</b> Unable to fetch the replied message.")
		return nil
	}

	if !r.IsMedia() || !strings.HasSuffix(r.File.Name, ".gif") {
		m.Reply("<b>Error:</b> The replied message is not a GIF.")
		return nil
	}

	fi, err := r.Download(&tg.DownloadOptions{
		FileName: "gif.gif",
	})
	if err != nil {
		m.Reply("<b>Error:</b> Unable to download the GIF.")
		return nil
	}

	cmd := []string{
		"ffmpeg",
		"-i", fi,
		"sticker.webm",
	}

	defer os.Remove("gif.gif")
	defer os.Remove("sticker.webm")

	exec.Command(cmd[0], cmd[1:]...).Run()
	m.ReplyMedia("sticker.webm", &tg.MediaOptions{
		Attributes: []tg.DocumentAttribute{
			&tg.DocumentAttributeSticker{
				Alt:        "😍",
				Stickerset: &tg.InputStickerSetEmpty{},
			},
		},
	})

	return nil
}
