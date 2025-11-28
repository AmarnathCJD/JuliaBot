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
		au := strings.SplitSeq(auths, ",")
		for user := range au {
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
	c.SetCommandPrefixes("./!-?")

	if LoadModules {
		// adminCMD
		c.On("cmd:rspot", modules.RestartSpotify, telegram.NewFilter().FromUsers(ownerId))
		c.On("cmd:rproxy", modules.RestartProxy, telegram.NewFilter().Custom(FilterOwnerAndAuth))

		c.On("cmd:promote", modules.PromoteUserHandle)
		c.On("cmd:demote", modules.DemoteUserHandle)
		c.On("cmd:ban", modules.BanUserHandle)
		c.On("cmd:unban", modules.UnbanUserHandle)
		c.On("cmd:kick", modules.KickUserHandle)

		c.On("cmd:restart", modules.RestartHandle, telegram.NewFilter().Custom(FilterOwner))
		c.On("cmd:id", modules.IDHandle)

		c.On("cmd:mz", modules.YtSongDL)
		c.On("cmd:spot(:?ify)? (.*)", modules.SpotifyHandler)
		c.On("cmd:spots", modules.SpotifySearchHandler)
		c.On("cmd:sh", modules.ShellHandle, telegram.NewFilter().Custom(FilterOwner))
		c.On("cmd:bash", modules.ShellHandle, telegram.NewFilter().Custom(FilterOwner))
		c.On("cmd:ls", modules.LsHandler, telegram.NewFilter().Custom(FilterOwner))
		c.On("cmd:ul", modules.UploadHandle, telegram.NewFilter().Custom(FilterOwnerNoReply))
		c.On("cmd:upd", modules.UpdateSourceCodeHandle, telegram.NewFilter().Custom(FilterOwnerNoReply))
		c.On("cmd:gban", modules.Gban, telegram.NewFilter().Custom(FilterOwner))
		c.On("cmd:ungban", modules.Ungban, telegram.NewFilter().Custom(FilterOwner))
		c.On("cmd:greet", modules.ModifyGreetStatus)
		c.On("cmd:start", modules.StartHandle)
		c.On("cmd:help", modules.HelpHandle)
		c.On("cmd:sys", modules.GatherSystemInfo)
		c.On("cmd:info", modules.UserHandle)
		c.On("cmd:json", modules.JsonHandle)
		c.On("cmd:ping", modules.PingHandle)
		c.On("cmd:fileinfo", modules.FileInfoHandle)
		c.On("cmd:eval", modules.EvalHandle, telegram.FilterFunc(FilterOwnerNoReply))
		c.On("cmd:go", modules.GoHandler)
		c.On("cmd:cancel", modules.CancelDownloadHandle, telegram.FilterFunc(FilterOwnerNoReply))

		c.On("cmd:sessgen", modules.GenStringSessionHandler)

		c.On("cmd:file", modules.SendFileByIDHandle)
		c.On("cmd:fid", modules.GetFileIDHandle)
		c.On("cmd:ldl", modules.DownloadHandle, telegram.FilterFunc(FilterOwnerNoReply))

		c.On("inline:pin", modules.PinterestInlineHandle)
		c.On("inline:doge", modules.DogeStickerInline)

		c.On("cmd:stream", modules.StreamHandler)
		c.On("cmd:streams", modules.ListStreamsHandler)
		c.On("cmd:stopstream", modules.StopStreamHandler)

		//c.AddRawHandler(&telegram.UpdateBotInlineSend{}, modules.SpotifyInlineHandler)
		c.On(telegram.OnInline, modules.SpotifyInlineSearch)
		c.On(telegram.OnChosenInline, modules.SpotifyInlineHandler)

		c.On("command:paste", modules.PasteBinHandler)
		c.On("command:timer", modules.SetTimerHandler)
		c.On("callback:snooze_", modules.TimerCallbackHandler)
		c.On("callback:dismiss_", modules.TimerCallbackHandler)

		c.On("command:math", modules.MathHandler)

		// c.On("command:ai", modules.AIImageGEN)
		c.On("command:snap", modules.SnapSaveHandler)
		c.On("command:insta", modules.SnapSaveHandler)
		//c.On("command:truec", modules.TruecallerHandle)
		c.On("command:doge", modules.DogeSticker)

		c.On("callback:spot_(.*)_(.*)", modules.SpotifyHandlerCallback)
		c.On("cmd:vid", modules.YtVideoDL)

		c.On(telegram.OnParticipant, modules.UserJoinHandle)

		// media-utils

		c.On("cmd:setthumb", modules.SetThumbHandler)
		c.On("cmd:mirror", modules.MirrorFileHandler)

		c.On("cmd:setpfp", modules.SetBotPfpHandler, telegram.FilterFunc(FilterOwner))

		c.On("cmd:media", modules.MediaInfoHandler)
		c.On("cmd:spec", modules.SpectrogramHandler)
		c.On("cmd:imdb", modules.ImdbHandler)
		c.On("inline:imdb", modules.ImDBInlineSearchHandler)
		c.On("callback:imdb_(.*)_(.*)", modules.ImdbCallbackHandler)
		//c.On("cmd:cancel", modules.CancelDownloadHandle)

		// c.On("cmd:edit", modules.EditImageCustomHandler)
		// c.On("cmd:gen", modules.GenerateImageHandler)

		// c.On("cmd:color", modules.ColorizeHandler)
		// c.On("cmd:upscale", modules.UpscaleHandler)
		// c.On("cmd:expand", modules.ExpandHandler)
		// c.On("message:/edit", modules.ReplaceHandler)

		//c.On(telegram.OnNewMessage, modules.AIHandler)

		c.On(telegram.OnNewMessage, modules.AFKHandler)
		c.On("command:audio", modules.ConvertToAudioHandle)

		modules.Mods.Init(c)
	}
}
