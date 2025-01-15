package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func YtSongDL(m *telegram.NewMessage) error {
	args := m.Args()
	if args == "" {
		m.Reply("Provide song url~")
		return nil
	}

	var channelId string = "(unknown)"
	var thumbImage string

	if !strings.Contains(args, "youtube.com") {
		vidId, channel, thumb, err := searchYouTube(args)
		if err != nil {
			m.Reply("<code>video not found.</code>")
			return nil
		}

		args = "https://www.youtube.com/watch?v=" + vidId
		channelId = channel
		thumbImage = thumb
	}

	vid, err := getVid(args)
	if err != nil {
		m.Reply("<code>video not found.</code>")
		return nil
	}

	re := regexp.MustCompile(`onVideoOptionSelected\('(.+?)', '(.+?)', '(.+?)', (\d+), '(.+?)', '(.+?)'\)`)
	matches := re.FindAllStringSubmatch(vid, -1)
	for _, match := range matches {

		if match[5] == "mp4a" {
			fi, _ := http.Get(match[2])
			if fi.StatusCode != 200 {
				m.Reply("<code>video not found.</code>")
				return nil
			}

			defer fi.Body.Close()
			body, _ := io.ReadAll(fi.Body)
			os.WriteFile("song.mp3", body, 0644)
			defer os.Remove("song.mp3")

			m.ReplyMedia("song.mp3", telegram.MediaOptions{
				Attributes: []telegram.DocumentAttribute{
					&telegram.DocumentAttributeFilename{
						FileName: strings.Split(match[3], "', '")[1] + ".mp3",
					},
					&telegram.DocumentAttributeAudio{
						Title:     strings.Split(match[3], "', '")[1],
						Performer: channelId,
					},
				},
				Thumb: thumbImage,
			})
		}
	}
	return nil
}

func searchYouTube(query string) (string, string, string, error) {
	searchQuery := strings.ReplaceAll(query, " ", "+")
	url := "https://www.youtube.com/results?search_query=" + searchQuery
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf("error reading response: %w", err)
	}

	bodyText := string(body)
	urlRegex := regexp.MustCompile(`https://i\.ytimg\.com/vi/[\w-]+/`)
	urls := urlRegex.FindAllString(bodyText, -1)

	channelRegex := regexp.MustCompile(`"\/@[\w-]+"`)
	channels := channelRegex.FindAllString(bodyText, -1)

	if len(urls) == 0 || len(channels) == 0 {
		return "", "", "", fmt.Errorf("no results found")
	}

	videoIDs := []string{}
	for _, url := range urls {
		parts := strings.Split(url, "/")
		if len(parts) >= 5 {
			videoIDs = append(videoIDs, parts[4])
		}
	}

	if len(videoIDs) == 0 {
		return "", "", "", fmt.Errorf("no video IDs found")
	}

	if len(channels) == 0 {
		return "", "", "", fmt.Errorf("no channels found")
	}

	return videoIDs[0], channels[0][2 : len(channels[0])-1], "https://i.ytimg.com/vi/" + videoIDs[0] + "/default.jpg", nil
}

func getVid(videoURL string) (string, error) {
	payload := []byte(`videoURL=` + videoURL)

	req, err := http.NewRequest("POST", "https://ssyoutube.online/yt-video-detail/", bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	return string(body), nil
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
				telegram.Button.SwitchInline("Retry", true, "sp"),
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

	btn := telegram.Button

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
