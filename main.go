package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	//"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	_ "github.com/joho/godotenv/autoload"

	_ "net/http/pprof"
)

var ownerId int64 = 0
var LoadModules = os.Getenv("ENV") != "development"

func main() {
	logZap, err := os.OpenFile("log.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer logZap.Close()
	wr := io.MultiWriter(os.Stdout, logZap)

	appId, _ := strconv.Atoi(os.Getenv("APP_ID"))
	ownerId, _ = strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64)

	client, err := tg.QuickBot(int32(appId), os.Getenv("APP_HASH"), os.Getenv("BOT_TOKEN"))
	if err != nil {
		panic(err)
	}
	client.Log.SetOutput(wr)

	client.Logger.Info("Bot is running as @%s", client.Me().Username)

	go func() {
		log.Println("Pprof server starting on :9009")
		if err := http.ListenAndServe(":9009", nil); err != nil {
			log.Printf("Pprof server error: %v", err)
		}
	}()

	initFunc(client)

	client.Idle()
	client.Logger.Info("Bot stopped")
}
