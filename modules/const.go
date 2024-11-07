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
