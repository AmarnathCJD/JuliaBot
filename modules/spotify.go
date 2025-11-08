package modules

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/tramhao/id3v2"
)

func rebuildOgg(filename string) {
	oggS := []byte("OggS")
	oggStart := []byte{0x00, 0x02}
	zeroes := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	vorbisStart := []byte{0x01, 0x1E, 0x01, 'v', 'o', 'r', 'b', 'i', 's'}
	channels := []byte{0x02}
	sampleRate := []byte{0x44, 0xAC, 0x00, 0x00}
	bitRate := []byte{0x00, 0xE2, 0x04, 0x00}
	packetSizes := []byte{0xB8, 0x01}

	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error opening OGG file:", err)
		return
	}
	defer file.Close()

	_, err = file.WriteAt(oggS, 0)
	if err != nil {
		fmt.Println("Error writing OGGS:", err)
		return
	}

	file.Seek(4, 0)
	file.Write(oggStart)
	file.Seek(6, 0)
	file.Write(zeroes)
	file.Seek(72, 0)

	buffer := make([]byte, 4)
	_, err = file.ReadAt(buffer, 4)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	file.Seek(14, 0)
	file.Write(buffer)
	file.Seek(18, 0)
	file.Write(zeroes)
	file.Seek(26, 0)
	file.Write(vorbisStart)

	file.Seek(35, 0)
	file.Write(zeroes)
	file.Seek(39, 0)
	file.Write(channels)
	file.Seek(40, 0)
	file.Write(sampleRate)
	file.Seek(48, 0)
	file.Write(bitRate)
	file.Seek(56, 0)
	file.Write(packetSizes)
	file.Seek(58, 0)
	file.Write(oggS)
	file.Seek(62, 0)
	file.Write(zeroes)
}

func decryptAudioFile(filePath string, hexKey string) ([]byte, string, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, "", fmt.Errorf("invalid hex key: %v", err)
	}

	buffer, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %v", err)
	}

	audioAesIv, err := hex.DecodeString("72e067fbddcbcf77ebe8bc643f630d93")
	if err != nil {
		return nil, "", fmt.Errorf("invalid AES IV: %v", err)
	}
	ivInt := int64(0)
	for i, b := range audioAesIv {
		ivInt |= int64(b) << (8 * (len(audioAesIv) - i - 1))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create AES cipher: %v", err)
	}

	ctr := cipher.NewCTR(block, audioAesIv)
	startTime := time.Now()
	decryptedBuffer := make([]byte, len(buffer))
	ctr.XORKeyStream(decryptedBuffer, buffer)

	decryptTime := time.Since(startTime).Milliseconds()
	return decryptedBuffer, fmt.Sprintf("%dms", decryptTime), nil
}

type SpotifyResponse struct {
	CDNURL string `json:"cdnurl"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Aritst string `json:"artist"`
	Tc     string `json:"tc"`
	Cover  string `json:"cover"`
	Lyrics string `json:"lyrics"`
}

type SpotifyPlaylistResponse struct {
	Tracks []SpotifyResponse `json:"tracks"`
}

type SpotifySearchResponse struct {
	Results []struct {
		Name       string `json:"name"`
		Artist     string `json:"artist"`
		ID         string `json:"id"`
		Year       string `json:"year"`
		Cover      string `json:"cover"`
		CoverSmall string `json:"cover_small"`
	} `json:"results"`
}

func SpotifyInlineSearch(i *telegram.InlineQuery) error {
	if strings.Contains(i.Query, "pin") || strings.Contains(i.Query, "doge") || strings.Contains(i.Query, "imdb") {
		return nil
	}

	b := i.Builder()
	args := i.Query
	if args == "" {
		b.Article("No query", "Please enter a spotify song id or query to search for", "No query")
		i.Answer(b.Results())
		return nil
	}

	req, _ := http.NewRequest("GET", "http://localhost:5000/search_track/"+args+"?lim=12", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.Article("Error", "Failed to search for song", "Error")
		i.Answer(b.Results())
		return nil
	}

	defer resp.Body.Close()
	var response SpotifySearchResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		b.Article("Error", "Failed to decode response", err.Error())
		i.Answer(b.Results())
		return nil
	}

	if len(response.Results) == 0 {
		b.Article("No song found", "No song found for the query", "No song found")
		i.Answer(b.Results())
		return nil
	}

	var bt = telegram.Button
	for _, r := range response.Results {
		b.Article(fmt.Sprintf("%s - %s", r.Name, r.Artist), r.Year, fmt.Sprintf("<b>Spotify Song - Ripping...</b>\n\n<b>Name:</b> %s\n<b>Artist:</b> %s\n<b>Year:</b> %s\n\n<b>Spotify ID:</b> <code>%s</code>", r.Name, r.Artist, r.Year, r.ID), &telegram.ArticleOptions{
			ID: r.ID,
			ReplyMarkup: telegram.NewKeyboard().AddRow(
				bt.SwitchInline("Search Again", true, ""),
			).Build(),
			Thumb: telegram.InputWebDocument{
				URL:      r.CoverSmall,
				Size:     1500,
				MimeType: "image/jpeg",
			},
		})
	}

	i.Answer(b.Results())
	return nil
}

func SpotifyInlineHandler(u *telegram.InlineSend) error {
	if strings.Contains(u.OriginalUpdate.Query, "pin") || strings.Contains(u.OriginalUpdate.Query, "doge") {
		return nil
	}
	if strings.Contains(u.OriginalUpdate.Query, "imdb") {
		ImdbInlineOnSendHandler(u)
		return nil
	}

	req, _ := http.NewRequest("GET", "http://localhost:5000/get_track/"+u.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()
	var response SpotifyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil
	}

	if response.CDNURL == "" || response.Key == "" {
		u.Edit("Spotify song not found.")
		return nil
	}

	rawFile, err := http.Get(response.CDNURL)
	if err != nil {
		return nil
	}

	defer rawFile.Body.Close()
	buffer, _ := io.ReadAll(rawFile.Body)

	os.WriteFile("song.encrypted", buffer, 0644)
	defer os.Remove("song.encrypted")

	decryptedBuffer, decryptTime, err := decryptAudioFile("song.encrypted", response.Key)
	if err != nil {
		return nil
	}

	os.WriteFile("song.ogg", decryptedBuffer, 0644)
	defer os.Remove("song.ogg")

	rebuildOgg("song.ogg")
	fixedFile, thumb, err := RepairOGG("song.ogg", response)
	if err != nil {
		return nil
	}

	defer os.Remove(fixedFile)
	u.Edit("<b>Decryption Time: <code>"+decryptTime+"</code></b>", &telegram.SendOptions{
		Media:    fixedFile,
		MimeType: "audio/mpeg",
		Attributes: []telegram.DocumentAttribute{
			&telegram.DocumentAttributeFilename{
				FileName: "song.ogg",
			},
			&telegram.DocumentAttributeAudio{
				Title:     response.Name,
				Performer: response.Aritst,
			},
		},
		ProgressManager: telegram.NewProgressManager(3).SetInlineMessage(u.Client, &u.MsgID),
		Thumb:           thumb,
		Spoiler:         true,
		ReplyMarkup: telegram.NewKeyboard().AddRow(
			telegram.Button.URL("Spotify Link", fmt.Sprintf("https://open.spotify.com/track/%s", response.Tc)),
		).Build(),
	})
	return nil
}

func SpotifySearchHandler(m *telegram.NewMessage) error {
	args := m.Args()

	if args == "" {
		m.Reply("Usage: /spots &lt;query&gt;")
		return nil
	}

	req, _ := http.NewRequest("GET", "http://localhost:5000/search_track/"+args, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	defer resp.Body.Close()
	var response SpotifySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	if len(response.Results) == 0 {
		m.Reply("No songs found for the query")
	}

	var b = telegram.Button
	var kb = telegram.NewKeyboard()
	for _, r := range response.Results {
		kb.AddRow(b.Data(fmt.Sprintf("%s - %s", r.Name, r.Artist), fmt.Sprintf("spot_%s_%d", r.ID, m.SenderID())))
	}
	m.Reply("<b>Select a song from below:</b>", telegram.SendOptions{
		ReplyMarkup: kb.Build(),
	})
	return nil
}

func SpotifyHandler(m *telegram.NewMessage) error {
	args := m.Args()

	if args == "" {
		m.Reply("Usage: /spot <code>&lt;song_id&gt;</code> or <code>&lt;spotify_url&gt;</code>")
		return nil
	}

	force := true
	if strings.Contains(args, "-s") {
		force = false
		args = strings.ReplaceAll(args, "-s", "")
	}

	if strings.Contains(args, "open.spotify.com") {
		if strings.Contains(args, "playlist") || strings.Contains(args, "album") {
			args = extractPlaylistIdFromURL(args)
			if args == "" {
				m.Reply("Invalid Spotify Playlist URL")
				return nil
			}

			m.Reply("This feature is not available yet.")
			return nil
		}
		args = extractTrackIdFromURL(args)
		if args == "" {
			m.Reply("Invalid Spotify URL")
			return nil
		}
		force = true
	}
	if !force {
		req, _ := http.NewRequest("GET", "http://localhost:5000/search_track/"+args, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		defer resp.Body.Close()
		var response SpotifySearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		if len(response.Results) == 0 {
			m.Reply("No songs found for the query")
		}

		var b = telegram.Button
		var kb = telegram.NewKeyboard()
		for _, r := range response.Results {
			kb.AddRow(b.Data(fmt.Sprintf("%s - %s", r.Name, r.Artist), fmt.Sprintf("spot_%s_%d", r.ID, m.SenderID())))
		}
		m.Reply("<b>Select a song from below:</b>", telegram.SendOptions{
			ReplyMarkup: kb.Build(),
		})
		return nil
	}

	req, _ := http.NewRequest("GET", "http://localhost:5000/get_track/"+args, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	var response SpotifyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		m.Reply("We couldn't find the song. (JSON Decode Error)")
		return nil
	}

	if response.CDNURL == "" || response.Key == "" {
		m.Reply("Spotify song not found.")
		return nil
	}

	rawFile, err := http.Get(response.CDNURL)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	defer rawFile.Body.Close()
	buffer, err := io.ReadAll(rawFile.Body)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	os.WriteFile("song.encrypted", buffer, 0644)
	defer os.Remove("song.encrypted")

	decryptedBuffer, decryptTime, err := decryptAudioFile("song.encrypted", response.Key)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	os.WriteFile("song.ogg", decryptedBuffer, 0644)
	defer os.Remove("song.ogg")

	rebuildOgg("song.ogg")
	fixedFile, thumb, err := RepairOGG("song.ogg", response)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	b := telegram.Button

	defer os.Remove(fixedFile)
	m.ReplyMedia(fixedFile, telegram.MediaOptions{
		Attributes: []telegram.DocumentAttribute{
			&telegram.DocumentAttributeFilename{
				FileName: fmt.Sprintf("%s.ogg", response.Name),
			},
			&telegram.DocumentAttributeAudio{
				Title:     response.Name,
				Performer: response.Aritst,
			},
		},
		Thumb:    thumb,
		Spoiler:  true,
		Caption:  "<b>Decryption Time: <code>" + decryptTime + "</code></b>",
		MimeType: "audio/mpeg",
		ReplyMarkup: telegram.NewKeyboard().AddRow(
			b.URL("Spotify Link", fmt.Sprintf("https://open.spotify.com/track/%s", response.Tc)),
		).Build(),
	})

	return nil
}

func SpotifyHandlerCallback(cb *telegram.CallbackQuery) error {
	payload := strings.Split(cb.DataString(), "_")
	if len(payload) != 3 {
		return nil
	}
	if !strings.EqualFold(payload[2], fmt.Sprintf("%d", cb.SenderID)) {
		cb.Answer("Not for you :)", &telegram.CallbackOptions{Alert: true})
		return nil
	}
	cb.Answer("Processing...")
	songId := payload[1]

	req, _ := http.NewRequest("GET", "http://localhost:5000/get_track/"+songId, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cb.Answer("Error: "+err.Error(), &telegram.CallbackOptions{Alert: true})
	}
	defer resp.Body.Close()
	var response SpotifyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		cb.Answer("We couldn't find the song. (JSON Decode Error)", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	if response.CDNURL == "" || response.Key == "" {
		cb.Answer("Spotify song not found.", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	rawFile, err := http.Get(response.CDNURL)
	if err != nil {
		cb.Answer("Error: "+err.Error(), &telegram.CallbackOptions{Alert: true})
		return nil
	}

	defer rawFile.Body.Close()
	buffer, err := io.ReadAll(rawFile.Body)
	if err != nil {
		cb.Answer("Error: " + err.Error())
		return nil
	}

	os.WriteFile("song.encrypted", buffer, 0644)
	defer os.Remove("song.encrypted")

	decryptedBuffer, decryptTime, err := decryptAudioFile("song.encrypted", response.Key)
	if err != nil {
		cb.Answer("Error: " + err.Error())
		return nil
	}

	os.WriteFile("song.ogg", decryptedBuffer, 0644)
	defer os.Remove("song.ogg")

	rebuildOgg("song.ogg")
	fixedFile, thumb, err := RepairOGG("song.ogg", response)
	if err != nil {
		cb.Answer("Error: " + err.Error())
		return nil
	}

	b := telegram.Button

	defer os.Remove(fixedFile)
	cb.Edit("<b>Decryption Time: <code>"+decryptTime+"</code></b>", &telegram.SendOptions{
		Media: fixedFile,
		Attributes: []telegram.DocumentAttribute{
			&telegram.DocumentAttributeFilename{
				FileName: fmt.Sprintf("%s.ogg", response.Name),
			},
			&telegram.DocumentAttributeAudio{
				Title:     response.Name,
				Performer: response.Aritst,
			},
		},
		Thumb:    thumb,
		Spoiler:  true,
		MimeType: "audio/mpeg",
		ReplyMarkup: telegram.NewKeyboard().AddRow(
			b.URL("Spotify Link", fmt.Sprintf("https://open.spotify.com/track/%s", response.Tc)),
		).Build(),
	})

	return nil
}

func RepairOGG(inputFile string, r SpotifyResponse) (string, []byte, error) {
	cov, err := http.Get(r.Cover)
	if err != nil {
		return inputFile, nil, fmt.Errorf("failed to download cover: %w", err)
	}
	defer cov.Body.Close()
	coverData, err := io.ReadAll(cov.Body)
	if err != nil {
		return inputFile, nil, fmt.Errorf("failed to read cover: %w", err)
	}
	outputFile := fmt.Sprintf("%s.ogg", r.Tc)
	cmd := exec.Command("ffmpeg", "-i", inputFile, "-c", "copy", "-metadata", fmt.Sprintf("lyrics=%s", r.Lyrics), outputFile)

	err = cmd.Run()
	if err != nil {
		return inputFile, nil, fmt.Errorf("failed to repair file: %w", err)
	}

	// if vorbiscomment is available, use it to add metadata
	_, err = exec.LookPath("vorbiscomment")
	if err == nil {
		vorbisFi := "METADATA_BLOCK_PICTURE=" + createVorbisImageBlock(coverData) + "\n"
		vorbisFi += "ALBUM=Spotify\n"
		vorbisFi += "ARTIST=" + r.Aritst + "\n"
		vorbisFi += "TITLE=" + r.Name + "\n"
		vorbisFi += "GENRE=Spotify, Music, Gogram, RoseLoverX\n"
		vorbisFi += "DATE=" + fmt.Sprintf("%d", time.Now().Year()) + "\n"
		// vorbisFi += "LYRICS=" + strings.ReplaceAll(r.Lyrics, "\n", " ") + "\n"
		os.WriteFile("vorbis.txt", []byte(vorbisFi), 0644)
		defer os.Remove("vorbis.txt")
		cmd = exec.Command("vorbiscomment", "-a", outputFile, "-c", "vorbis.txt")
		err = cmd.Run()
		if err != nil {
			return inputFile, coverData, fmt.Errorf("failed to add metadata: %w", err)
		}

		return outputFile, coverData, nil
	}

	tag, _ := id3v2.Open(outputFile, id3v2.Options{Parse: true})
	tag.SetArtist(r.Aritst)
	tag.SetTitle(r.Name)
	tag.SetVersion(4)
	tag.SetYear(fmt.Sprintf("%d", time.Now().Year()))
	tag.SetGenre("Spotify, Music, Gogram, Telegram")
	tag.SetAlbum("Spotify")
	tag.AddAttachedPicture(id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     coverData,
	})

	var lyr []id3v2.SyncedText
	for _, l := range strings.Split(r.Lyrics, "\n") {
		if l == "" {
			continue
		}
		re := regexp.MustCompile(`\[(\d+):(\d+).(\d+)\](.*)`)
		matches := re.FindStringSubmatch(l)
		if len(matches) != 5 {
			continue
		}

		minutes, _ := strconv.Atoi(matches[1])
		seconds, _ := strconv.Atoi(matches[2])
		millis, _ := strconv.Atoi(matches[3])
		totalMillis := (minutes*60 + seconds) * 1000
		totalMillis += millis

		lyr = append(lyr, id3v2.SyncedText{
			Text:      matches[4],
			Timestamp: uint32(totalMillis),
		})
	}

	tag.AddSynchronisedLyricsFrame(id3v2.SynchronisedLyricsFrame{
		ContentType:       1,
		Encoding:          id3v2.EncodingUTF8,
		TimestampFormat:   2,
		Language:          "eng",
		ContentDescriptor: "Musixmatch",
		SynchronizedTexts: lyr,
	})

	return outputFile, coverData, tag.Save()
}

func createVorbisImageBlock(imageBytes []byte) string {
	os.WriteFile("cover.jpg", imageBytes, 0644)
	defer os.Remove("cover.jpg")
	exec.Command("./cover_gen.sh", "cover.jpg").Run()
	coverData, _ := os.ReadFile("cover.base64")
	defer os.Remove("cover.base64")
	return string(coverData)
}

func extractTrackIdFromURL(url string) string {
	re := regexp.MustCompile(`track/(\w+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}

func extractPlaylistIdFromURL(url string) string {
	re := regexp.MustCompile(`playlist/(\w+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}
