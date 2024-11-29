package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
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
		LogLevel: tg.LogDebug,
		Session:  "session.dat",
	})
	client.Log.NoColor()

	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		goroutines := runtime.NumGoroutine()
		w.Write([]byte(fmt.Sprintf("Bot is running since %s, Goroutines: %d", time.Since(time.Unix(startTimeStamp, 0)).String(), goroutines)))
	})
	go http.ListenAndServe(":80", nil)
	//time.Sleep(10 * time.Second)
	// https://t.me/rzTODO/263

	client.Logger.Info("Bot started, Loading modules...")
	m, _ := client.GetMessageByID("rztodo", 266)
	fmt.Println(m)
	p := tg.NewProgressManager(2)
	p.Edit(func(a, b int64) {
		fmt.Println(p.GetStats(b))
	})
	m.Download(&tg.DownloadOptions{
		ProgressManager: p,
	})
	client.Logger.Info("done")

	client.Idle()
	return

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
