package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	//"time"

	"github.com/amarnathcjd/gogram/telegram"
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

	client, err := tg.NewClient(tg.ClientConfig{
		//Session: "user1",
		AppID:   int32(appId),
		AppHash: os.Getenv("APP_HASH"),
	})
	if err != nil {
		panic(err)
	}
	client.Conn()
	//client.AuthPrompt()
	client.Log.SetOutput(wr)
	client.LoginBot(os.Getenv("BOT_TOKEN"))

	client.Logger.Info("Bot is running as @%s", client.Me().Username)

	go func() {
		log.Println("Pprof server starting on :9009")
		if err := http.ListenAndServe(":9009", nil); err != nil {
			log.Printf("Pprof server error: %v", err)
		}
	}()

	initFunc(client)
	// client.OnRaw(&tg.UpdateGroupCall{}, func(m tg.Update, c *tg.Client) error {
	// 	fmt.Println(client.JSON(m))
	// 	return nil
	// })
	client.OnCommand("senders", func(m *tg.NewMessage) error {
		x := client.GetExportedSendersStatus()
		var result string
		for a, b := range x {
			result += fmt.Sprintf("dc%d: %d senders\n", a, b)
		}
		m.Reply("<b>Exported Senders Status:</b>\n" + result)
		return nil
	})
	client.OnCommand("wizard", advancedWizardExample, telegram.FromUser(ownerId))
	client.Idle()
	client.Logger.Info("Bot stopped")
}

func advancedWizardExample(m *telegram.NewMessage) error {

	return nil
}
