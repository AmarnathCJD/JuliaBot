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
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
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

func SpotifyInlineHandler(i *telegram.InlineQuery) error {
	if i.Query == "" || strings.Contains(i.Query, "pin") {
		return nil
	}
	//a := time.Now()

	b := i.Builder()
	args := i.Args()
	if args == "" {
		b.Article("No query", "Please enter a spotify song id or query to search for", "No query")
		i.Answer(b.Results())
		return nil
	}

	fmt.Println("Searching for:", args)

	req, _ := http.NewRequest("GET", "http://localhost:5000/get_track/"+args, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.Article("Error", "Failed to search for song", "Error")
		i.Answer(b.Results())
		return nil
	}

	defer resp.Body.Close()
	var response struct {
		CDNURL string `json:"cdnurl"`
		Key    string `json:"key"`
		Name   string `json:"name"`
		Aritst string `json:"artist"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Println("Error:", err)
		dt, _ := json.Marshal(response)
		fmt.Println(string(dt))
		b.Article("Error", "Failed to decode response", err.Error())
		i.Answer(b.Results())
		return nil
	}

	if response.CDNURL == "" || response.Key == "" {
		b.Article("No song found", "No song found for the query", "No song found")
		i.Answer(b.Results())
		return nil
	}

	rawFile, err := http.Get(response.CDNURL)
	if err != nil {
		b.Article("Error", "Failed to download song", "Error")
		i.Answer(b.Results())
		return nil
	}

	defer rawFile.Body.Close()
	buffer, err := io.ReadAll(rawFile.Body)
	if err != nil {
		b.Article("Error", "Failed to read song", "Error")
		i.Answer(b.Results())
		return nil
	}

	os.WriteFile("song.encrypted", buffer, 0644)
	defer os.Remove("song.encrypted")

	decryptedBuffer, decryptTime, err := decryptAudioFile("song.encrypted", response.Key)
	if err != nil {
		b.Article("Error", "Failed to decrypt song", "Error")
		i.Answer(b.Results())
		return nil
	}

	os.WriteFile("song.ogg", decryptedBuffer, 0644)
	defer os.Remove("song.ogg")

	rebuildOgg("song.ogg")
	fixedFile, err := RepairOGG("song.ogg")
	if err != nil {
		b.Article("Error", "Failed to repair song", "Error")
		i.Answer(b.Results())
		return nil
	}

	defer os.Remove(fixedFile)
	fi, err := i.Client.UploadFile(fixedFile)
	if err != nil {
		b.Article("Error", "Failed to upload song", "Error")
		i.Answer(b.Results())
		return nil
	}

	ul, err := i.Client.MessagesUploadMedia("", &telegram.InputPeerSelf{}, &telegram.InputMediaUploadedDocument{
		File: fi,
		Attributes: []telegram.DocumentAttribute{
			&telegram.DocumentAttributeFilename{
				FileName: "song.ogg",
			},
			&telegram.DocumentAttributeAudio{
				Title:     response.Name,
				Performer: response.Aritst,
			},
		},
		MimeType: "audio/mpeg",
	})
	if err != nil {
		b.Article("Error", "Failed to upload song", "Error")
		i.Answer(b.Results())
		return nil
	}

	dc := ul.(*telegram.MessageMediaDocument).Document.(*telegram.DocumentObj)
	bt := telegram.Button{}

	res := &telegram.InputBotInlineResultDocument{
		ID:   "song_1",
		Type: "audio",
		Document: &telegram.InputDocumentObj{
			ID:            dc.ID,
			AccessHash:    dc.AccessHash,
			FileReference: dc.FileReference,
		},
		Title:       response.Name,
		Description: response.Aritst,
		SendMessage: &telegram.InputBotInlineMessageMediaAuto{
			Message: "<b>Decryption Time: <code>" + decryptTime + "</code></b>",
			ReplyMarkup: telegram.NewKeyboard().AddRow(
				bt.SwitchInline("Search Again", true, "sp"),
			).Build(),
		},
	}

	b.InlineResults = append(b.InlineResults, res)
	i.Answer(b.Results())
	return nil
}

func SpotifyHandler(m *telegram.NewMessage) error {
	if m.Args() == "" {
		m.Reply("Usage: /spot <spotify-song-id / query>")
		return nil
	}

	req, _ := http.NewRequest("GET", "http://localhost:5000/get_track/"+m.Args(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	defer resp.Body.Close()
	var response struct {
		CDNURL string `json:"cdnurl"`
		Key    string `json:"key"`
		Name   string `json:"name"`
		Aritst string `json:"artist"`
		Tc     string `json:"tc"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		m.Reply("Error: " + err.Error())
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
	fixedFile, err := RepairOGG("song.ogg")
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	b := telegram.Button{}

	defer os.Remove(fixedFile)
	m.ReplyMedia(fixedFile, telegram.MediaOptions{
		Attributes: []telegram.DocumentAttribute{
			&telegram.DocumentAttributeFilename{
				FileName: "song.ogg",
			},
			&telegram.DocumentAttributeAudio{
				Title:     response.Name,
				Performer: response.Aritst,
			},
		},
		Caption:  "<b>Decryption Time: <code>" + decryptTime + "</code></b>",
		MimeType: "audio/mpeg",
		ReplyMarkup: telegram.NewKeyboard().AddRow(
			b.URL("Spotify Link", fmt.Sprintf("https://open.spotify.com/track/%s", response.Tc)),
		).Build(),
	})

	return nil
}

func RepairOGG(inputFile string) (string, error) {
	// ffmpeg -i song.ogg -c copy fixed_song.ogg

	outputFile := inputFile[:len(inputFile)-len(".ogg")] + "_repaired.ogg"
	cmd := exec.Command("ffmpeg", "-i", inputFile, "-c", "copy", outputFile)
	err := cmd.Run()
	if err != nil {
		return inputFile, fmt.Errorf("failed to repair file: %w", err)
	}

	return outputFile, nil
}
