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
	ownerId, _ = strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64)

	appId, _ := strconv.Atoi(os.Getenv("APP_ID"))
	client, err := tg.NewClient(tg.ClientConfig{
		AppID:    int32(appId),
		AppHash:  os.Getenv("APP_HASH"),
		LogLevel: tg.LogInfo,
		Session:  "session.dat",
	})
	client.Log.NoColor()

	// m, _ := client.GetMessageByID("rztodo", 278)
	// pm := tg.NewProgressManager(3)
	// pm.Edit(func(totalSize, currentSize int64) {
	// 	fmt.Println(fmt.Sprintf("%d/%d", currentSize, totalSize))
	// })
	// m.Download(&tg.DownloadOptions{
	// 	ProgressManager: pm,
	// })
	// return

	if err != nil {
		panic(err)
	}

	client.LogColor(false)

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
