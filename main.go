package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

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

	client, err := tg.NewClient(tg.ClientConfig{
		AppID:   int32(appId),
		AppHash: os.Getenv("APP_HASH"),
		Logger:  tg.NewLogger(tg.LogInfo).NoColor(),
		Session: "session.dat",
	})
	client.Conn()

	//x, _ := client.GetMessageByID("")

	// ud, err := client.MessagesCheckChatInvite("gef1CuB_4z01YTk9")
	// x := ud.(*tg.ChatInviteAlready).Chat.(*tg.Channel)
	// msg, _ := client.GetHistory(&tg.InputPeerChannel{ChannelID: x.ID, AccessHash: x.AccessHash}, &tg.HistoryOption{
	// 	Limit: 600,
	// })
	// for _, x := range msg {
	// 	if !x.IsMedia() {
	// 		continue
	// 	}
	// 	f, _ := x.Download()
	// 	client.SendMessage("umbiyasanam", f)
	// }
	// return

	client.Conn()
	client.LoginBot(os.Getenv("BOT_TOKEN"))

	initFunc(client)
	me, err := client.GetMe()

	if err != nil {
		panic(err)
	}

	client.Logger.Info(fmt.Sprintf("Authenticated as @%s, in %s.", me.Username, time.Since(time.Unix(startTimeStamp, 0)).String()))
	client.Idle()
}
