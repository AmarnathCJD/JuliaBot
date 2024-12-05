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
	m.ReplyMedia("song.ogg", telegram.MediaOptions{
		Attributes: []telegram.DocumentAttribute{
			&telegram.DocumentAttributeFilename{
				FileName: "song.ogg",
			},
			&telegram.DocumentAttributeAudio{
				Title:     response.Name,
				Performer: response.Aritst,
			},
		},
		Caption: "<b>Decryption Time: <code>" + decryptTime + "</code></b>",
	})

	return nil
}