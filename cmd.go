package main

import (
	"main/modules"

	"github.com/amarnathcjd/gogram/telegram"
)

func FilterOwner(m *telegram.NewMessage) bool {
	if m.SenderID() == ownerId {
		return true
	}
	m.Reply("You are not allowed to use this command")
	return false
}

func FilterOwnerNoReply(m *telegram.NewMessage) bool {
	if m.SenderID() == ownerId {
		return true
	}
	return false
}

func initFunc(c *telegram.Client) {
	c.UpdatesGetState()

	if LOAD_MODULES {
		c.On("message:/mz", modules.YtSongDL)
		c.On("message:/spot(:?ify)? (.*)", modules.SpotifyHandler)
		c.On("message:/spots", modules.SpotifySearchHandler)
		c.On("message:/sh", modules.ShellHandle, telegram.FilterFunc(FilterOwner))
		c.On("message:/bash", modules.ShellHandle, telegram.FilterFunc(FilterOwner))
		c.On("message:/ls", modules.LsHandler, telegram.FilterFunc(FilterOwner))
		c.On("message:/ul", modules.UploadHandle, telegram.FilterFunc(FilterOwnerNoReply))
		c.On("message:/upd", modules.UpdateSourceCodeHandle, telegram.FilterFunc(FilterOwnerNoReply))
		c.On("message:/gban", modules.GbanMeme, telegram.FilterFunc(FilterOwner))

		c.On("message:/start", modules.StartHandle)
		c.On("message:/help", modules.HelpHandle)
		c.On("message:/sys", modules.GatherSystemInfo)
		c.On("message:/info", modules.UserHandle)
		c.On("message:/json", modules.JsonHandle)
		c.On("message:.ping", modules.PingHandle)
		c.On("command:eval", modules.EvalHandle, telegram.FilterFunc(FilterOwnerNoReply))

		c.On("message:/sessgen", modules.GenStringSessionHandler)

		c.On("message:/file", modules.SendFileByIDHandle)
		c.On("message:/fid", modules.GetFileIDHandle)
		c.On("message:.dl", modules.DownloadHandle, telegram.FilterFunc(FilterOwnerNoReply))

		c.On("inline:pin", modules.PinterestInlineHandle)
		c.On("inline:doge", modules.DogeStickerInline)

		c.On("message:/stream", modules.StreamHandler)

		c.AddRawHandler(&telegram.UpdateBotInlineSend{}, modules.SpotifyInlineHandler)
		//c.AddInlineHandler(telegram.OnInlineQuery, modules.SpotifyInlineSearch)
		c.On(telegram.OnInline, modules.SpotifyInlineSearch)

		c.On("command:paste", modules.PasteBinHandler)
		c.On("command:timer", modules.SetTimerHandler)

		c.On("command:ai", modules.AIImageGEN)
		//c.On("command:truec", modules.TruecallerHandle)
		c.On("command:doge", modules.DogeSticker)

		c.On("callback:spot_(.*)_(.*)", modules.SpotifyHandlerCallback)

		c.On(telegram.OnParticipant, modules.UserJoinHandle)

		// media-utils

		c.On("message:/setthumb", modules.SetThumbHandler)
		c.On("message:/mirror", modules.MirrorFileHandler)

		modules.Mods.Init(c)
	}
}
