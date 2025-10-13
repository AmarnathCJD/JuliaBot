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
		c.On("command:rspot", modules.RestartSpotify, telegram.FilterFunc(FilterOwner))
		c.On("command:rproxy", modules.RestartProxy, telegram.FilterFunc(FilterOwnerAndAuth))
		c.On("command:promote", modules.PromoteUserHandle)
		c.On("command:restart", modules.RestartHandle, telegram.FilterFunc(FilterOwner))
		c.On("command:id", modules.IDHandle)

		c.On("command:mz", modules.YtSongDL)
		c.On("command:spotify", modules.SpotifyHandler)
		c.On("command:spot", modules.SpotifyHandler)
		c.On("command:spots", modules.SpotifySearchHandler)
		c.On("command:sh", modules.ShellHandle, telegram.FilterFunc(FilterOwner))
		c.On("command:bash", modules.ShellHandle, telegram.FilterFunc(FilterOwner))
		c.On("command:ls", modules.LsHandler, telegram.FilterFunc(FilterOwner))
		c.On("command:ul", modules.UploadHandle, telegram.FilterFunc(FilterOwnerNoReply))
		c.On("command:upd", modules.UpdateSourceCodeHandle, telegram.FilterFunc(FilterOwnerNoReply))
		c.On("command:gban", modules.GbanMeme, telegram.FilterFunc(FilterOwner))
		c.On("command:greet", modules.ModifyGreetStatus)
		c.On("command:start", modules.StartHandle)
		c.On("command:help", modules.HelpHandle)
		c.On("command:sys", modules.GatherSystemInfo)
		c.On("command:info", modules.UserHandle)
		c.On("command:json", modules.JsonHandle)
		c.On("command:ping", modules.PingHandle)
		c.On("command:eval", modules.EvalHandle, telegram.FilterFunc(FilterOwnerNoReply))

		c.On("command:sessgen", modules.GenStringSessionHandler)

		c.On("command:file", modules.SendFileByIDHandle)
		c.On("command:fid", modules.GetFileIDHandle)
		c.On("command:ldl", modules.DownloadHandle, telegram.FilterFunc(FilterOwnerNoReply))

		c.On("inline:pin", modules.PinterestInlineHandle)
		c.On("inline:doge", modules.DogeStickerInline)

		c.On("command:stream", modules.StreamHandler)

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
		c.On("command:midj", modules.MidjHandler)
		c.On("command:vid", modules.YtVideoDL)

		c.On(telegram.OnParticipant, modules.UserJoinHandle)

		// media-utils

		c.On("command:setthumb", modules.SetThumbHandler)
		c.On("command:mirror", modules.MirrorFileHandler)

		c.On("command:setpfp", modules.SetBotPfpHandler, telegram.FilterFunc(FilterOwner))

		c.On("command:media", modules.MediaInfoHandler)
		c.On("command:imdb", modules.ImdbHandler)
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
