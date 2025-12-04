package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	//"time"

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

	// Initialize network logger
	networkLogger := NewNetworkLogger("8999")
	if err := networkLogger.Start(); err != nil {
		fmt.Printf("Failed to start network logger: %v\n", err)
	}

	logZap, err := os.OpenFile("log.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer logZap.Close()
	defer networkLogger.Stop()

	// Include network logger in output writers
	wr := io.MultiWriter(os.Stdout, logZap, networkLogger)
	//log.SetOutput(wr)

	appId, _ := strconv.Atoi(os.Getenv("APP_ID"))
	ownerId, _ = strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64)

	cfg := tg.NewClientConfigBuilder(int32(appId), os.Getenv("APP_HASH")).
		WithLogger(tg.NewLogger(tg.LogDebug, tg.LoggerConfig{
			Output: wr,
		})).
		WithReqTimeout(100).
		Build()

	client, err := tg.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	client.Conn()

	client.LoginBot(os.Getenv("BOT_TOKEN"))
	client.Logger.Info("Bot is running as @%s", client.Me().Username)
	initFunc(client)
	//client.FetchDifferenceOnStartup()

	client.Idle()
	client.Logger.Info("Bot stopped")
}
