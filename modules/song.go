package modules

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func YtSongDL(m *telegram.NewMessage) error {
	m.Reply("<code>Feature Under Maintainance!!!</code>")
	return nil
	args := m.Args()
	if args == "" {
		m.Reply("Provide song name!")
		return nil
	}

	// Get the video ID
	cmd_to_get_id := exec.Command("yt-dlp", "ytsearch:"+args, "--get-id")
	output, err := cmd_to_get_id.Output()
	if err != nil {
		log.Println(err)
		return err
	}
	videoID := strings.TrimSpace(string(output))

	// Download the song
	cmd := exec.Command("yt-dlp", "https://www.youtube.com/watch?v="+videoID, "--embed-metadata", "--embed-thumbnail", "-f", "bestaudio", "-x", "--audio-format", "mp3", "-o", "%(id)s.mp3")
	err = cmd.Run()
	if err != nil {
		log.Println(err)
		return err
	}

	fmt.Println("Downloaded the song")

	m.RespondMedia(videoID + ".mp3")

	return nil
}

func init() {
	Mods.AddModule("Song", `<b>Here are the commands available in Song module:</b>
The Song module is used to download songs from YouTube.

Its currently Broken!`)
}
