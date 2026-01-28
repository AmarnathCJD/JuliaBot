package main

import (
	"fmt"
	"io"
	"log"
	"main/modules/db"
	"net/http"
	"os"
	"strconv"

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
	// p, err := tg.ProxyFromURL("https://t.me/proxy?server=ultra.transiiantanialnmiomana.info&port=443&secret=eee9a4f23b1d768c04a8d7f39120ca5b6e6D656469612E737465616D706F77657265642E636F6D")
	// if err != nil {
	// 	panic(err)
	// }
	client, err := tg.NewClient(tg.ClientConfig{
		//Session: "userxyz",
		AppID:   int32(appId),
		AppHash: os.Getenv("APP_HASH"),
		//Proxy:      p,
		//DataCenter: 1,
		LogLevel: tg.LogInfo,
	})
	if err != nil {
		panic(err)
	}

	// client.SetProxy(p)
	client.Conn()
	client.Log.SetOutput(wr)
	client.LoginBot(os.Getenv("BOT_TOKEN"))

	client.Logger.Info("Bot is running as @%s", client.Me().Username)
	x := "![😒](tg://emoji?id=6134147226441614765) ɪᴛ ᴡᴀs ᴀ ɢʀᴇᴀᴛ ᴊᴏᴜʀɴᴇʏ ᴡɪᴛʜ ʏ'ᴀʟʟ, ᴛʜᴀɴᴋs ғᴏʀ ᴇᴠᴇʀʏᴛʜɪɴɢ... ![😗](tg://emoji?id=6134366475932141065)![❄️](tg://emoji?id=5231492061718390437) ʜᴀᴘᴘʏ ɴᴇᴡ ʏᴇᴀʀ ᴇᴠᴇ ![❄️](tg://emoji?id=5231492061718390437)"
	client.SendMessage("gogrammers", x, &tg.SendOptions{
		ParseMode: tg.MarkDown,
	})
	go func() {
		log.Println("Pprof server starting on :9009")
		if err := http.ListenAndServe(":9009", nil); err != nil {
			log.Printf("Pprof server error: %v", err)
		}
	}()

	initFunc(client)

	client.OnCommand("senders", func(m *tg.NewMessage) error {
		x := client.GetExportedSendersStatus()
		var result string
		for a, b := range x {
			result += fmt.Sprintf("dc%d: %d senders\n", a, b)
		}
		m.Reply("<b>Exported Senders Status:</b>\n" + result)
		return nil
	})
	client.Idle()
	db.CloseDB()
	client.Logger.Info("Bot stopped")
}
