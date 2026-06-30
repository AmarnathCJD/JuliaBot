package modules

import (
	"math/rand"
)



var fonts = []string{
	"Inter_28pt-Bold.ttf",
	"Swiss 721 Black Extended BT.ttf",
}

func GetRandomFont() string {
	return fonts[rand.Intn(len(fonts))]
}
