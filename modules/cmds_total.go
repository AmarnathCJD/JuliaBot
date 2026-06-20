package modules

import (
	"fmt"
	"sort"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var allCmdsCategories = map[string][]string{
	"Admin": {
		"promote", "demote", "ban", "unban", "kick", "mute", "unmute",
		"tban", "tmute", "purge", "del", "pin", "unpin", "unpinall",
		"adminlist", "title", "invitelink", "setchatpic", "setchattitle",
		"setchatdesc",
	},
	"Warns": {
		"warn", "warns", "resetwarn", "warnlimit", "warnmode", "rmwarn",
	},
	"Blacklist": {
		"blacklist", "addblacklist", "rmblacklist", "blacklistmode",
	},
	"Locks": {
		"lock", "unlock", "locks", "locktypes",
	},
	"Filters": {
		"filter", "filters", "stop", "stopall",
	},
	"Notes": {
		"save", "get", "notes", "clear", "clearall",
	},
	"Rules": {
		"rules", "setrules", "clearrules",
	},
	"Welcome": {
		"welcome", "setwelcome", "resetwelcome", "goodbye", "setgoodbye",
		"resetgoodbye", "cleanwelcome", "cleanservice",
	},
	"AFK": {
		"afk", "brb",
	},
	"AI": {
		"ai", "ask", "gpt", "gemini", "imagine", "img",
	},
	"Media": {
		"sticker", "kang", "stickerid", "getsticker", "stickerpack",
		"upscale", "removebg", "enhance",
	},
	"YouTube": {
		"yt", "ytdl", "ytaudio", "ytsearch", "song",
	},
	"Instagram": {
		"insta", "ig", "instadl", "reel",
	},
	"Terabox": {
		"terabox", "tb",
	},
	"Aria2": {
		"mirror", "leech", "status", "cancel", "ariastatus",
	},
	"Files": {
		"upload", "rename", "ls", "rm",
	},
	"Translator": {
		"tr", "translate", "tts", "detectlang",
	},
	"Stickers": {
		"stickers", "stickerinfo", "delsticker",
	},
	"Timer": {
		"timer", "remind", "remindme",
	},
	"Utils": {
		"id", "info", "json", "ping", "echo", "paste", "shorten",
		"weather", "calc", "base64", "hash", "uuid",
	},
	"Misc": {
		"runs", "shrug", "lenny", "flip", "roll", "choose", "decide",
	},
	"Help": {
		"help", "start", "commands", "cmds", "allcmds",
	},
	"Dev": {
		"eval", "exec", "shell", "py", "restart", "logs", "stats",
		"speedtest", "sysinfo",
	},
	"Games": {
		"dice", "rps", "tictactoe", "ttt", "hangman", "wordgame",
		"unscramble",
	},
	"Fun": {
		"meme", "joke", "fortune", "quote", "moviequote", "tarot",
		"mood", "dadjoke", "fact", "numfact", "bored", "reaction",
		"shrugemoji",
	},
	"Anime": {
		"anime", "manga", "waifu", "neko",
	},
	"Food": {
		"food", "recipe", "tipcalc",
	},
	"Image Tools": {
		"avatar", "qr", "qrdecode", "palette", "gradient", "colors",
		"ascii", "placeholder", "memegen", "reverseimage",
	},
	"Text Tools": {
		"binary", "morse", "leet", "emojify", "translateemoji",
		"wordcount", "ispalindrome", "anagram", "makepali", "jsonpretty",
	},
	"Info": {
		"whois", "ipinfo", "truecaller", "timezone", "epoch",
		"covid", "stocks", "space", "pokemon",
	},
	"Inline": {
		"inline",
	},
}

func AllCmdsHandler(m *tg.NewMessage) error {
	cats := make([]string, 0, len(allCmdsCategories))
	for k := range allCmdsCategories {
		cats = append(cats, k)
	}
	sort.Strings(cats)
	var b strings.Builder
	b.WriteString("<b>JuliaBot Commands</b>\n\n")
	total := 0
	for _, cat := range cats {
		cmds := allCmdsCategories[cat]
		sorted := make([]string, len(cmds))
		copy(sorted, cmds)
		sort.Strings(sorted)
		b.WriteString(fmt.Sprintf("<b>%s</b>: ", cat))
		parts := make([]string, 0, len(sorted))
		for _, c := range sorted {
			parts = append(parts, "/"+c)
			total++
		}
		b.WriteString(strings.Join(parts, " "))
		b.WriteString("\n\n")
	}
	b.WriteString(fmt.Sprintf("<b>Total:</b> <code>%d</code> commands across <code>%d</code> categories.", total, len(cats)))
	m.Reply(b.String())
	return nil
}

func registerCmdsTotalHandlers() {
	c := Client
	c.On("cmd:cmds", AllCmdsHandler)
	c.On("cmd:allcmds", AllCmdsHandler)
}

func init() {
	QueueHandlerRegistration(registerCmdsTotalHandlers)
}
