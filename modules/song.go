package modules

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

type Sptfy struct {
	Artists    string `json:"artists"`
	Title      string `json:"title"`
	Image      string `json:"image"`
	IsPlaying  bool   `json:"is_playing"`
	DurationMs int    `json:"duration_ms"`
	ProgressMs int    `json:"progress_ms"`
	URL        string `json:"url"`
}

func InlineSpotify(m *telegram.InlineQuery) error {
	b := m.Builder()
	svg, _ := http.Get("https://spotify-now-playing-psi-silk.vercel.app/api/current-playing?s=1")
	if svg.StatusCode != 200 {
		b.Article("Error", "Failed to fetch data", "Failed to fetch data", &telegram.ArticleOptions{
			ReplyMarkup: telegram.NewKeyboard().AddRow(
				telegram.Button{}.SwitchInline("Retry", true, "sp"),
			).Build(),
		})
		m.Answer(b.Results())
		return nil
	}

	defer svg.Body.Close()
	var s Sptfy
	json.NewDecoder(svg.Body).Decode(&s)

	var caption string
	if s.IsPlaying {
		caption = fmt.Sprintf(
			"🎵 <b><i>Now Playing:</i></b> <a href=\"%s\">%s</a>\n"+
				"🎤 <b><i>Artist:</i></b> %s\n"+
				"⏱ <b><i>Time:</i></b> %s / %s",
			s.URL,
			s.Title,
			s.Artists,
			fmtDuration(s.ProgressMs),
			fmtDuration(s.DurationMs),
		)
	} else {
		caption = "<i>🚫 No song is currently playing</i>"
	}

	btn := telegram.Button{}

	if s.Image != "" {
		b.Document(s.Image, &telegram.ArticleOptions{
			Title:         "Spotify Now Playing",
			Description:   "Shows the currently playing song on Spotify",
			Caption:       caption,
			ForceDocument: true,
			ReplyMarkup: telegram.NewKeyboard().AddRow(
				btn.URL("Open in Spotify", s.URL),
			).Build(),
		})
	} else {
		b.Article("Spotify Now Playing", "Shows the currently playing song on Spotify", caption, &telegram.ArticleOptions{
			ReplyMarkup: telegram.NewKeyboard().AddRow(
				btn.URL("Open in Spotify", s.URL),
			).Build(),
		})
	}

	m.Answer(b.Results(), telegram.InlineSendOptions{Gallery: true, CacheTime: 0})
	return nil
}

func fmtDuration(ms int) string {
	sec := ms / 1000
	min := sec / 60
	sec = sec % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}

func init() {
	Mods.AddModule("Song", `<b>Here are the commands available in Song module:</b>
The Song module is used to download songs from YouTube.

Its currently Broken!`)
}
