package main

import (
	"main/modules"
	"os"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func FilterOwner(m *telegram.NewMessage) bool {
	if m.SenderID() == ownerId {
		return true
	}
	m.Reply("You are not allowed to use this command")
	return false
}

func FilterOwnerAndAuth(m *telegram.NewMessage) bool {
	auths := os.Getenv("AUTH_USERS")
	if auths == "" {
		return FilterOwner(m)
	} else {
		if m.SenderID() == ownerId {
			return true
		}
		au := strings.Split(auths, ",")
		for _, user := range au {
			if strings.TrimSpace(user) == strconv.Itoa(int(m.SenderID())) {
				return true
			}
		}
	}

	m.Reply("You are not allowed to use this command")
	return false
}

func FilterOwnerNoReply(m *telegram.NewMessage) bool {
	return m.SenderID() == ownerId
}

func initFunc(c *telegram.Client) {
	c.UpdatesGetState()

	if LoadModules {
		// adminCMD
		c.On("message:^/rspot", modules.RestartSpotify, telegram.FilterFunc(FilterOwner))
		c.On("message:^/rproxy", modules.RestartProxy, telegram.FilterFunc(FilterOwnerAndAuth))
		c.On("message:^/promote", modules.PromoteUserHandle)
		c.On("message:^/restart", modules.RestartHandle, telegram.FilterFunc(FilterOwner))
		c.On("message:^/id", modules.IDHandle)

		c.On("message:^/mz", modules.YtSongDL)
		c.On("message:^/spot(:?ify)? (.*)", modules.SpotifyHandler)
		c.On("message:^/spots", modules.SpotifySearchHandler)
		c.On("message:^/sh", modules.ShellHandle, telegram.FilterFunc(FilterOwner))
		c.On("message:^/bash", modules.ShellHandle, telegram.FilterFunc(FilterOwner))
		c.On("message:^/ls", modules.LsHandler, telegram.FilterFunc(FilterOwner))
		c.On("message:^/ul", modules.UploadHandle, telegram.FilterFunc(FilterOwnerNoReply))
		c.On("message:^/upd", modules.UpdateSourceCodeHandle, telegram.FilterFunc(FilterOwnerNoReply))
		c.On("message:^/gban", modules.GbanMeme, telegram.FilterFunc(FilterOwner))
		c.On("message:^/greet", modules.ModifyGreetStatus)
		c.On("message:^/start", modules.StartHandle)
		c.On("message:^/help", modules.HelpHandle)
		c.On("message:^/sys", modules.GatherSystemInfo)
		c.On("message:^/info", modules.UserHandle)
		c.On("message:^/json", modules.JsonHandle)
		c.On("message:^.ping", modules.PingHandle)
		c.On("command:eval", modules.EvalHandle, telegram.FilterFunc(FilterOwnerNoReply))

		c.On("message:^/sessgen", modules.GenStringSessionHandler)

		c.On("message:^/file", modules.SendFileByIDHandle)
		c.On("message:^/fid", modules.GetFileIDHandle)
		c.On("message:^/ldl", modules.DownloadHandle, telegram.FilterFunc(FilterOwnerNoReply))

		c.On("inline:pin", modules.PinterestInlineHandle)
		c.On("inline:doge", modules.DogeStickerInline)

		c.On("message:^/stream", modules.StreamHandler)

		//c.AddRawHandler(&telegram.UpdateBotInlineSend{}, modules.SpotifyInlineHandler)
		c.On(telegram.OnInline, modules.SpotifyInlineSearch)
		c.On(telegram.OnChoosenInline, modules.SpotifyInlineHandler)

		c.On("command:paste", modules.PasteBinHandler)
		c.On("command:timer", modules.SetTimerHandler)
		c.On("command:math", modules.MathHandler)

		c.On("command:ai", modules.AIImageGEN)
		c.On("command:snap", modules.SnapSaveHandler)
		c.On("command:insta", modules.SnapSaveHandler)
		//c.On("command:truec", modules.TruecallerHandle)
		c.On("command:doge", modules.DogeSticker)

		c.On("callback:spot_(.*)_(.*)", modules.SpotifyHandlerCallback)
		c.On("message:^/midj", modules.MidjHandler)
		c.On("message:^/vid", modules.YtVideoDL)

		c.On(telegram.OnParticipant, modules.UserJoinHandle)

		// media-utils

		c.On("message:^/setthumb", modules.SetThumbHandler)
		c.On("message:^/mirror", modules.MirrorFileHandler)

		c.On("message:^/setpfp", modules.SetBotPfpHandler, telegram.FilterFunc(FilterOwner))

		c.On("message:^/media", modules.MediaInfoHandler)
		c.On("message:^/imdb", modules.ImdbHandler)
		c.On("inline:imdb", modules.ImDBInlineSearchHandler)
		c.On("callback:imdb_(.*)_(.*)", modules.ImdbCallbackHandler)

		// c.On("message:/color", modules.ColorizeHandler)
		// c.On("message:/upscale", modules.UpscaleHandler)
		// c.On("message:/expand", modules.ExpandHandler)
		// c.On("message:/edit", modules.ReplaceHandler)

		//c.On(telegram.OnNewMessage, modules.AIHandler)

		c.On(telegram.OnNewMessage, modules.AFKHandler)

		modules.Mods.Init(c)
	}
}
