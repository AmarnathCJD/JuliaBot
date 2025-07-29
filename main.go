package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

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
	// ;logging setup
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
		WithLogger(tg.NewLogger(tg.LogInfo).NoColor()).
		Build()

	client, err := tg.NewClient(cfg)
	client.Start()

	fmt.Println("Connecting to Telegram...")
	fmt.Println(client.GetMe())
	tk, err := client.AccountInitTakeoutSession(&tg.AccountInitTakeoutSessionParams{
		MessageChats: true,
	})
	if err != nil {
		panic(err)
	}

	req := &tg.ChannelsGetLeftChannelsParams{
		Offset: 0,
	}

	resp, err := client.InvokeWithTakeout(int(tk.ID), req)

	if err != nil {
		panic("Error invoking with takeout: " + err.Error())
	}
	fmt.Println("Left Channels:", client.JSON(resp))
	client.Conn()
	client.LoginBot(os.Getenv("BOT_TOKEN"))
	client.Logger.Info("Bot is running..., Press Ctrl+C to stop it.")

	initFunc(client)

	client.Idle()
}
