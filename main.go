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
		AppID:     int32(appId),
		AppHash:   os.Getenv("APP_HASH"),
		LogLevel:  tg.LogDebug,
		ForceIPv6: true,
	})

	if err != nil {
		panic(err)
	}

	client.Conn()
	return
	// client.SendMedia("roseloverx", "Avatar.The.Last.Airbender.S01E01.720p.WEB-HD.x264-Pahe.in.mkv")
	// return
	client.LoginBot(os.Getenv("BOT_TOKEN"))
	//client.AuthPrompt()
	var pm = tg.NewProgressManager(2)
	pm.Edit(func(total, curr int64) {
		fmt.Println(pm.GetStats(curr))
	})
	// https: //t.me/rzTODO/257
	fi, _ := client.GetMessageByID("rzTODO", 257)

	client.DownloadMedia(fi, &tg.DownloadOptions{
		ProgressManager: pm,
	})

	initFunc(client)
	me, err := client.GetMe()

	if err != nil {
		panic(err)
	}

	client.Logger.Info(fmt.Sprintf("Bot started as @%s, in %s.", me.Username, time.Since(time.Unix(startTimeStamp, 0)).String()))
	client.Idle()
}
