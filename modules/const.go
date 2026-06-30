package modules

import (
	"math/rand"
)

var random_emoji_list = []string{
	"❤️",
	"👍",
	"😍",
	"😂",
	"😊",
	"🔥",
	"😘",
	"💕",
}

func getRandomEmoticon() string {
	return random_emoji_list[rand.Intn(len(random_emoji_list))]
}

var fonts = []string{
	"Inter_28pt-Bold.ttf",
	"Swiss 721 Black Extended BT.ttf",
}

func GetRandomFont() string {
	return fonts[rand.Intn(len(fonts))]
}
