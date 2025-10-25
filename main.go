package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	dotenv "github.com/joho/godotenv"

	_ "net/http"
	_ "net/http/pprof"
)

func init() {
	dotenv.Load()
}

var ownerId int64 = 0
var LoadModules = os.Getenv("ENV") != "development"

func main() {
	fmt.Println("Starting JuliaBot...")
	logZap, err := os.OpenFile("log.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer logZap.Close()
	wr := io.MultiWriter(os.Stdout, logZap)
	log.SetOutput(wr)

	appId, _ := strconv.Atoi(os.Getenv("APP_ID"))
	ownerId, _ = strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64)
	var sessionName = "rusty.dat"
	if os.Getenv("SESSION_NAME") != "" {
		sessionName = os.Getenv("SESSION_NAME")
	}

	fmt.Println("Session Name:", sessionName)

	cfg := tg.NewClientConfigBuilder(int32(appId), os.Getenv("APP_HASH")).
		WithSession(sessionName).
		WithLogger(tg.NewLogger(tg.LogDebug).NoColor()).
		WithReqTimeout(100000 * time.Millisecond).
		Build()

	client, err := tg.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	client.Start()

	fmt.Println("Connecting to Telegram...")

	client.Conn()
	client.LoginBot(os.Getenv("BOT_TOKEN"))
	client.Logger.Info("Bot is running..., Press Ctrl+C to stop it.")
	initFunc(client)

	client.Idle()
}
