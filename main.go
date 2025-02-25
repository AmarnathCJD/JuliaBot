package main

import (
	"fmt"
	"main/modules"
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
		Session: "session.dat",
	})

	if err != nil {
		panic(err)
	}

	client.Conn()
	client.LoginBot(os.Getenv("BOT_TOKEN"))
	client.Logger.Info("Bot is running...")
	initFunc(client)
	me, err := client.GetMe()

	if err != nil {
		panic(err)
	}

	client.On("message:/media", modules.MediaInfoHandler)
	client.On("message:/imdb", modules.ImdbHandler)
	client.On("callback:imdb_(.*)_(.*)", modules.ImdbCallbackHandler)

	client.Logger.Info(fmt.Sprintf("Authenticated as -> @%s, in %s.", me.Username, time.Since(time.Unix(startTimeStamp, 0)).String()))
	client.Idle()
}
