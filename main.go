package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	dotenv "github.com/joho/godotenv"
)

const LOAD_MODULES = true

var startTimeStamp = time.Now().Unix()
var ownerId int64 = 0

func main() {
	dotenv.Load()
	ownerId, _ = strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64)

	appId, _ := strconv.Atoi(os.Getenv("APP_ID"))
	client, err := tg.NewClient(tg.ClientConfig{
		AppID:    int32(appId),
		AppHash:  os.Getenv("APP_HASH"),
		LogLevel: tg.LogDebug,
		Session:  "session.dat",
	})
	if err != nil {
		panic(err)
	}

	client.Logger.Info("Bot started, Loading modules...")

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
