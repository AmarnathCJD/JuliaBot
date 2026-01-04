package main

import (
	"main/modules"
	"main/modules/downloaders"
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
	_, _ = c.UpdatesGetState()
	c.SetCommandPrefixes("./!-?")

	if LoadModules {
		c.Logger.Info("Loading Modules...")
		// adminCMD
		c.On("cmd:rspot", modules.RestartSpotify, telegram.FromUser(ownerId))
		c.On("cmd:rproxy", modules.RestartProxy, telegram.Custom(FilterOwnerAndAuth))

		c.On("cmd:promote", modules.PromoteUserHandle)
		c.On("cmd:demote", modules.DemoteUserHandle)
		c.On("cmd:ban", modules.BanUserHandle)
		c.On("cmd:unban", modules.UnbanUserHandle)
		c.On("cmd:kick", modules.KickUserHandle)
		c.On("cmd:fullpromote", modules.FullPromoteHandle)
		c.On("cmd:tban", modules.TbanUserHandle)
		c.On("cmd:tmute", modules.TmuteUserHandle)
		c.On("cmd:mute", modules.MuteUserHandle)
		c.On("cmd:unmute", modules.UnmuteUserHandle)
		c.On("cmd:sban", modules.SbanUserHandle)
		c.On("cmd:smute", modules.SmuteUserHandle)
		c.On("cmd:skick", modules.SkickUserHandle)

		c.On("cmd:restart", modules.RestartHandle, telegram.Custom(FilterOwner))
		c.On("cmd:id", modules.IDHandle)

		c.On("cmd:vid", downloaders.YTCustomHandler)
		c.On("cmd:spot(:?ify)? (.*)", modules.SpotifyHandler)
		c.On("cmd:spots", modules.SpotifySearchHandler)
		c.On("cmd:sh", modules.ShellHandle, telegram.Custom(FilterOwner))
		c.On("cmd:bash", modules.ShellHandle, telegram.Custom(FilterOwner))
		c.On("cmd:ls", modules.LsHandler, telegram.Custom(FilterOwner))
		c.On("cmd:ul", modules.UploadHandle, telegram.Custom(FilterOwnerNoReply))
		c.On("cmd:upd", modules.UpdateSourceCodeHandle, telegram.Custom(FilterOwnerNoReply))
		c.On("cmd:gban", modules.Gban, telegram.Custom(FilterOwner))
		c.On("cmd:ungban", modules.Ungban, telegram.Custom(FilterOwner))
		c.On("cmd:greet", modules.WelcomeToggleHandler)
		c.On("cmd:welcome", modules.WelcomeToggleHandler)
		c.On("cmd:start", modules.StartHandle)
		c.On("cmd:help", modules.HelpHandle)
		c.On("cmd:sys", modules.GatherSystemInfo)
		c.On("cmd:info", modules.UserHandle)
		c.On("cmd:json", modules.JsonHandle)
		c.On("cmd:ping", modules.PingHandle)
		c.On("cmd:new", modules.NewYearHandle)
		c.On("cmd:fileinfo", modules.FileInfoHandle)
		c.On("cmd:eval", modules.EvalHandle, telegram.Custom(FilterOwnerNoReply))
		c.On("cmd:go", modules.GoHandler)
		c.On("cmd:cancel", modules.CancelDownloadHandle, telegram.Custom(FilterOwnerNoReply))

		c.On("cmd:adddl", modules.AddDLHandler, telegram.Custom(FilterOwnerAndAuth))
		c.On("cmd:listdls", modules.ListDLsHandler, telegram.Custom(FilterOwnerAndAuth))
		c.On("cmd:rmdl", modules.RmDLHandler, telegram.Custom(FilterOwnerAndAuth))
		c.On("cmd:listdl", modules.ListDLHandler, telegram.Custom(FilterOwnerAndAuth))

		c.On("cmd:sessgen", modules.GenStringSessionHandler)

		c.On("cmd:file", modules.SendFileByIDHandle)
		c.On("cmd:fid", modules.GetFileIDHandle)
		c.On("cmd:ldl", modules.DownloadHandle, telegram.Custom(FilterOwnerNoReply))

		c.On("inline:pin", modules.PinterestInlineHandle)
		c.On("inline:doge", modules.DogeStickerInline)

		c.On("cmd:finfo", modules.FileInfoHandle)
		c.On("cmd:setrtmp", modules.SetRTMPHandler)
		c.On("cmd:stream", modules.StreamHandler)
		c.On("cmd:streams", modules.ListStreamsHandler)
		c.On("cmd:stopstream", modules.StopStreamHandler)
		c.On("callback:stream_", modules.StreamCallbackHandler)

		// c.On(telegram.OnNewMessage, modules.HandleAIMessage) // Old AI disabled
		c.On(telegram.OnNewMessage, modules.HandleAskCommand)
		c.On("cmd:model", modules.HandleModelCommand)
		c.On("callback:model_", modules.HandleModelCallback)
		//c.AddRawHandler(&telegram.UpdateBotInlineSend{}, modules.SpotifyInlineHandler)
		c.On(telegram.OnInline, modules.SpotifyInlineSearch)
		c.On(telegram.OnChosenInline, modules.SpotifyInlineHandler)

		c.On("command:paste", modules.PasteBinHandler)
		c.On("command:timer", modules.SetTimerHandler)
		c.On("callback:snooze_", modules.TimerCallbackHandler)
		c.On("callback:dismiss_", modules.TimerCallbackHandler)

		c.On("command:math", modules.MathHandler)

		// c.On("command:ai", modules.AIImageGEN)
		c.On("command:snap", downloaders.InstaHandler)
		c.On("command:insta", downloaders.InstaHandler)
		c.On("command:tera", downloaders.TeraboxHandler)
		//c.On("command:truec", modules.TruecallerHandle)
		c.On("command:doge", modules.DogeSticker)

		c.On("callback:spot_(.*)_(.*)", modules.SpotifyHandlerCallback)
		//c.On("cmd:vid", modules.YtVideoDL)
		c.On("cmd:ytc", downloaders.YTCustomHandler)
		c.On("callback:ytdl_(.*)", downloaders.YTCallbackHandler)

		c.On(telegram.OnParticipant, modules.WelcomeHandler)
		c.On(telegram.OnParticipant, modules.GoodbyeHandler)
		c.OnCommand("gif", modules.GifToSticker)
		c.OnCommand("kang", modules.KangSticker)
		c.OnCommand("rmkang", modules.RemoveKangedSticker)

		// media-utils

		c.On("cmd:setthumb", modules.SetThumbHandler)
		c.On("cmd:mirror", modules.MirrorFileHandler)

		c.On("cmd:setpfp", modules.SetBotPfpHandler, telegram.Custom(FilterOwner))

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
		c.On(telegram.OnNewMessage, modules.SedHandler)
		c.On("command:audio", modules.ConvertToAudioHandle)

		// Notes module
		c.On("cmd:save", modules.SaveNoteHandler)
		c.On("cmd:note", modules.GetNoteHandler)
		c.On("cmd:notes", modules.ListNotesHandler)
		c.On("cmd:listnotes", modules.ListNotesHandler)
		c.On("cmd:clear", modules.ClearNoteHandler)
		c.On("cmd:clearallnotes", modules.ClearAllNotesHandler)
		c.On("callback:clearallnotes_", modules.ClearAllNotesCallback)
		c.On("callback:cancelnotes_", modules.ClearAllNotesCallback)
		c.On("message:^#", modules.NoteHashHandler)

		// Blacklist module
		c.On("cmd:addbl", modules.AddBlacklistHandler)
		c.On("cmd:addblacklist", modules.AddBlacklistHandler)
		c.On("cmd:rmbl", modules.RemoveBlacklistHandler)
		c.On("cmd:rmblacklist", modules.RemoveBlacklistHandler)
		c.On("cmd:listbl", modules.ListBlacklistHandler)
		c.On("cmd:blacklist", modules.ListBlacklistHandler)
		c.On("cmd:setblaction", modules.SetBlacklistActionHandler)
		c.On("cmd:clearbl", modules.ClearBlacklistHandler)
		c.On("callback:clearbl_", modules.ClearBlacklistCallback)
		c.On("callback:cancelbl_", modules.ClearBlacklistCallback)
		c.On(telegram.OnNewMessage, modules.BlacklistWatcher)

		// Filters module
		c.On("cmd:filter", modules.FilterHandler)
		c.On("cmd:stop", modules.StopFilterHandler)
		c.On("cmd:filters", modules.ListFiltersHandler)
		c.On("cmd:stopall", modules.StopAllFiltersHandler)
		c.On("callback:stopall_", modules.StopAllFiltersCallback)
		c.On("callback:cancelfilters_", modules.StopAllFiltersCallback)
		c.On(telegram.OnNewMessage, modules.FilterWatcher)

		// Rules module
		c.On("cmd:setrules", modules.SetRulesHandler)
		c.On("cmd:rules", modules.GetRulesHandler)
		c.On("cmd:clearrules", modules.ClearRulesHandler)
		c.On("callback:rules_", modules.RulesButtonCallback)

		// Warns module
		c.On("cmd:warn", modules.WarnUserHandler)
		c.On("cmd:warns", modules.ListWarnsHandler)
		c.On("cmd:rmwarn", modules.RemoveWarnHandler)
		c.On("cmd:resetwarns", modules.ResetWarnsHandler)
		c.On("cmd:setwarnlimit", modules.SetWarnLimitHandler)
		c.On("cmd:setwarnaction", modules.SetWarnActionHandler)
		c.On("cmd:warnsettings", modules.WarnSettingsHandler)
		c.On("callback:rmwarn_", modules.RemoveWarnCallback)

		// Welcome/Goodbye module
		c.On("cmd:setwelcome", modules.SetWelcomeHandler)
		c.On("cmd:setgoodbye", modules.SetGoodbyeHandler)
		c.On("cmd:goodbye", modules.GoodbyeToggleHandler)
		c.On("cmd:clearwelcome", modules.ClearWelcomeHandler)
		c.On("cmd:cleargoodbye", modules.ClearGoodbyeHandler)
		c.On("cmd:cleanwelcome", modules.CleanServiceHandler)
		c.On("cmd:wautodelete", modules.WelcomeAutoDeleteHandler)
		c.On("cmd:welcomesettings", modules.WelcomeSettingsHandler)
		c.On("cmd:greetings", modules.WelcomeSettingsHandler)

		// Captcha
		c.On("cmd:captcha", modules.SetCaptchaHandler)
		c.On("cmd:setcaptchamode", modules.SetCaptchaModeHandler)
		c.On("cmd:setcaptchatime", modules.SetCaptchaTimeHandler)
		c.On("cmd:captchamute", modules.CaptchaMuteHandler)
		c.On("cmd:captchakick", modules.CaptchaKickHandler)
		c.On("callback:captcha_verify_", modules.CaptchaVerifyCallback)
		c.On(telegram.OnNewMessage, modules.CaptchaMathHandler)

		c.On("callback:help_back", modules.HelpBackCallback)
		c.On("cmd:tr", modules.TranslateHandler)
		c.On("cmd:ud", modules.UDHandler)

		modules.Mods.Init(c)
	}
}
