package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	tg "github.com/amarnathcjd/gogram/telegram"
	dotenv "github.com/joho/godotenv"

	_ "net/http"
	_ "net/http/pprof"
)

const LOAD_MODULES = true

var startTimeStamp = time.Now().Unix()
var ownerId int64 = 0

func main() {
	dotenv.Load()
	appId, _ := strconv.Atoi(os.Getenv("APP_ID"))
	ownerId, _ = strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64)
	sess, _ := decodeTelethonSessionString("1BJWap1sBuzrkr_HKaCCJpkrCZQJev9gF_rHROVDh0CbvkL5NNIWzneL7WoI_4jPgr6ruE427nTLkPKxEB__XXBONNHXMo2x3Z0RR8Bp4YMOx_Kffi0ScTXP2idt_dA9ZfQCWep_WbCI9dEBWRrF-mvtJZgueMyys50N-Z2ETi2o44mEah7FXjSySrYx9zVynvBNwMZWL4H2nZfDuP28-M4ovisiKZ2dGqgvf6DfLjkJfHl9qNzBoNbd2qwdEVdpZAUW1uj8M53jZpx__EJ3eFeCSEoMA3KyH7FD5vVY9QjKQ8C6YVWUDenm4sL2DSRvvKFLO3tVg9W94QMxjOkaSCtcOCsREU_A=")

	client, err := tg.NewClient(tg.ClientConfig{
		AppID:   int32(appId),
		AppHash: os.Getenv("APP_HASH"),
		Logger:  tg.NewLogger(tg.LogInfo).NoColor(),
		//Session: "session.datu",
		StringSession: sess.Encode(),
		MemorySession: true,
	})

	fmt.Println(client.JoinChannel("gogrammers"))
	if err != nil {
		panic(err)
	}

	client.Conn()
	//client.OpenChat(&tg.InputChannelObj{ChannelID: 1232792540, AccessHash: 8856309246363801590})
	client.On("message", func(m *tg.NewMessage) error {
		//m.React("👍")
		//fmt.Println("chat:", m.ChatID(), "accessHash:", m.Channel.AccessHash)
		fmt.Println("messageLatency", time.Since(time.Unix(int64(m.Date()), 0)).String(), "chatId", m.ChatID(), "messageId", m.ID)
		return nil
	})
	client.On("command:fuck", func(m *tg.NewMessage) error {
		t := time.Now()
		y, _ := m.Reply("pong")

		y.Edit(fmt.Sprintf("%s", time.Since(t).String()))
		return nil
	})
	client.Idle()
	return

	client.LoginBot(os.Getenv("BOT_TOKEN"))

	initFunc(client)
	me, err := client.GetMe()

	if err != nil {
		panic(err)
	}

	client.Logger.Info(fmt.Sprintf("Authenticated as @%s, in %s.", me.Username, time.Since(time.Unix(startTimeStamp, 0)).String()))
	client.Idle()
}

func decodeTelethonSessionString(sessionString string) (*telegram.Session, error) {
	data, err := base64.URLEncoding.DecodeString(sessionString[1:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}

	ipLen := 4
	if len(data) == 352 {
		ipLen = 16
	}

	expectedLen := 1 + ipLen + 2 + 256
	if len(data) != expectedLen {
		return nil, fmt.Errorf("invalid session string length")
	}

	// ">B{}sH256s"
	offset := 1

	// IP Address (4 or 16 bytes based on IPv4 or IPv6)
	ipData := data[offset : offset+ipLen]
	ip := net.IP(ipData)
	ipAddress := ip.String()
	offset += ipLen

	// Port (2 bytes, Big Endian)
	port := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	// Auth Key (256 bytes)
	var authKey [256]byte
	copy(authKey[:], data[offset:offset+256])

	return &tg.Session{
		Hostname: ipAddress + ":" + fmt.Sprint(port),
		Key:      authKey[:],
	}, nil
}
