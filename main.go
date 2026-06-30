package main

import (
	"io"
	"log"
	"main/modules"
	"main/modules/db"
	_ "main/modules/extras"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

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

	socks := buildSocksProxy()

	appId, _ := strconv.Atoi(os.Getenv("APP_ID"))
	ownerId, _ = strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64)
	clientCfg := tg.ClientConfig{
		AppID:    int32(appId),
		AppHash:  os.Getenv("APP_HASH"),
		LogLevel: tg.LogInfo,
		Session:  "xyumi.dat",
	}
	if socks != nil {
		clientCfg.Proxy = socks
	}
	client, err := tg.NewClient(clientCfg)
	if err != nil {
		panic(err)
	}

	client.Conn()
	client.Log.SetOutput(wr)
	client.LoginBot(os.Getenv("BOT_TOKEN"))

	client.Logger.Info("Bot is running as @%s", client.Me().Username)
	go func() {
		log.Println("Pprof server starting on :9009")
		if err := http.ListenAndServe(":9009", nil); err != nil {
			log.Printf("Pprof server error: %v", err)
		}
	}()

	modules.InitClient(client)
	modules.SetupFilters(ownerId, LoadModules)
	modules.RegisterHandlers()

	client.Idle()
	db.CloseDB()
	client.Logger.Info("Bot stopped")
}


func buildSocksProxy() *tg.Socks5Proxy {
	raw := strings.TrimSpace(os.Getenv("PROXY"))
	if raw == "" {
		return nil
	}
	host, portStr, err := net.SplitHostPort(raw)
	if err != nil {
		log.Printf("[proxy] invalid PROXY=%q (expected host:port): %v", raw, err)
		return nil
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Printf("[proxy] invalid port in PROXY=%q: %v", raw, err)
		return nil
	}
	log.Printf("[proxy] using socks5 %s:%d", host, port)
	return &tg.Socks5Proxy{
		BaseProxy: tg.BaseProxy{Host: host, Port: port},
		Username:  os.Getenv("PROXY_USERNAME"),
		Password:  os.Getenv("PROXY_PASSWORD"),
	}
}
