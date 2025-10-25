package modules

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
)

var startTime = time.Now()

func StartHandle(m *telegram.NewMessage) error {
	greeting := "✨ <b>Hello there!</b> ✨\n\n"
	greeting += "I'm <b>Julia</b>, your friendly bot companion! 🤖💙\n\n"
	greeting += "Here's what I can help you with:\n"
	greeting += "➜ 🎬 <b>Media Magic:</b> Search movies, download videos, convert files\n"
	greeting += "➜ 🎵 <b>Music Vibes:</b> Get songs, Spotify info, and more\n"
	greeting += "➜ 👤 <b>User Info:</b> Discover details about Telegram users\n"
	greeting += "➜ 🔧 <b>System Stats:</b> Check bot performance and health\n"
	greeting += "➜ 🎨 <b>Fun Stuff:</b> Memes, inline queries, and surprises!\n\n"
	greeting += "Type <code>/help</code> to see all my commands! 💫\n\n"
	greeting += "<i>Let's make something awesome together!</i> ✨"

	m.Reply(greeting)
	m.React("❤")
	return nil
}

func GatherSystemInfo(m *telegram.NewMessage) error {
	m.ChatType()

	msg, _ := m.Reply("<code>...System Information...</code>")

	if !IsImageDepsInstalled() {
		renderedImage, err := FillAndRenderSVG(false)
		if err != nil {
			msg.Edit("❌ Failed to render image: " + err.Error())
			return err
		}

		_, err = msg.Edit(
			"",
			telegram.SendOptions{Spoiler: true, Media: renderedImage, Caption: ""},
		)
		if err != nil {
			return err
		}
		return nil
	}

	system, err := gatherSystemInfo()
	if err != nil {
		return err
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get additional system info
	hostInfo, _ := host.Info()
	loadAvg, _ := load.Avg()

	info := "╭─ <b>System Information</b>\n\n"

	// Highlighted metrics at top
	info += fmt.Sprintf("⚡ <b>Goroutines:</b> <code>%d</code> | <b>Process Memory:</b> <code>%s</code>\n\n", runtime.NumGoroutine(), system.ProcessMemory)

	// Performance Metrics
	info += "➜ <b><i>Performance</i></b>\n"
	info += fmt.Sprintf("   ├ <b>CPU Usage:</b> <code>%.2f%%</code>\n", system.CPUPerc)
	if loadAvg != nil {
		info += fmt.Sprintf("   ├ <b>Load Average:</b> <code>%.2f, %.2f, %.2f</code>\n", loadAvg.Load1, loadAvg.Load5, loadAvg.Load15)
	}
	info += fmt.Sprintf("   ├ <b>Heap Allocated:</b> <code>%s</code>\n", HumanBytes(memStats.Alloc))
	info += fmt.Sprintf("   ├ <b>Heap System:</b> <code>%s</code>\n", HumanBytes(memStats.Sys))
	info += fmt.Sprintf("   └ <b>Uptime:</b> <i>%s</i>\n\n", system.Uptime)

	// System Resources
	info += "➜ <b><i>Hardware</i></b>\n"
	info += fmt.Sprintf("   ├ <b>CPU:</b> <i>%s</i>\n", system.CPUName)
	info += fmt.Sprintf("   ├ <b>Cores:</b> <code>%d</code>\n", runtime.NumCPU())
	info += fmt.Sprintf("   ├ <b>Memory:</b> <code>%s</code> / <code>%s</code> <i>(%.1f%%)</i>\n", system.MemUsed, system.MemTotal, system.MemPerc)
	info += fmt.Sprintf("   └ <b>Disk:</b> <code>%s</code> / <code>%s</code> <i>(%.1f%%)</i>\n\n", system.DiskUsed, system.DiskTotal, system.DiskPerc)

	// Runtime Information
	info += "➜ <b><i>Runtime</i></b>\n"
	info += fmt.Sprintf("   ├ <b>Go Version:</b> <code>%s</code>\n", runtime.Version())
	info += fmt.Sprintf("   ├ <b>Platform:</b> <code>%s/%s</code>\n", runtime.GOOS, runtime.GOARCH)
	if hostInfo != nil {
		info += fmt.Sprintf("   ├ <b>Hostname:</b> <code>%s</code>\n", hostInfo.Hostname)
		info += fmt.Sprintf("   ├ <b>Boot Time:</b> <i>%s</i>\n", time.Unix(int64(hostInfo.BootTime), 0).Format("2006-01-02 15:04:05"))
	}
	info += fmt.Sprintf("   ├ <b>GC Cycles:</b> <code>%d</code> | <b>Pauses:</b> <code>%s</code>\n", memStats.NumGC, time.Duration(memStats.PauseTotalNs).Round(time.Millisecond))
	info += fmt.Sprintf("   └ <b>PID:</b> <code>%d</code>\n\n", system.ProcessID)

	info += "╰─────────────────"

	_, err = msg.Edit(
		"",
		telegram.SendOptions{Caption: info},
	)
	if err != nil {
		msg.Edit(info)
	}
	return err
}

var dcLocationMap = map[int]string{
	1: "Miami, US",
	2: "Amsterdam, NL",
	3: "Miami, US",
	4: "Amsterdam, NL",
	5: "Singapore, SG",
}

func UserHandle(m *telegram.NewMessage) error {
	var userID int64 = 0
	var userHash int64 = 0
	if m.IsReply() {
		r, _ := m.GetReplyMessage()
		userID = r.SenderID()
		if r.Sender == nil {
			m.Reply("Error: User not found")
			return nil
		}
		userHash = r.Sender.AccessHash
	} else if len(m.Args()) > 0 {
		i, ok := strconv.Atoi(m.Args())
		if ok != nil {
			user, err := m.Client.ResolveUsername(m.Args())
			if err != nil {
				m.Reply("Error: " + err.Error())
				return nil
			}
			ux, ok := user.(*telegram.UserObj)
			if !ok {
				m.Reply("Error: User not found")
				return nil
			}
			userID = ux.ID
			userHash = ux.AccessHash
		} else {
			userID = int64(i)
			user, err := m.Client.GetUser(int64(i))
			if err != nil {
				m.Reply("Error: " + err.Error())
				return nil
			}
			userHash = user.AccessHash
		}
	} else {
		userID = m.SenderID()
		if m.Sender == nil {
			m.Reply("Error: User not found")
			return nil
		}
		userHash = m.Sender.AccessHash
	}
	user, err := m.Client.UsersGetFullUser(&telegram.InputUserObj{
		UserID:     userID,
		AccessHash: userHash,
	})

	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	uf := user.FullUser
	un := user.Users[0].(*telegram.UserObj)

	var userString string

	// Header with name
	name := un.FirstName
	if un.LastName != "" {
		name += " " + un.LastName
	}
	userString += "👤 <b>" + name + "</b>"

	// Status badges
	if un.Verified {
		userString += " ✓"
	}
	if un.Premium {
		userString += " ⭐"
	}
	if un.Bot {
		userString += " 🤖"
	}
	userString += "\n\n"

	// Username
	if un.Username != "" {
		userString += "➜ 📧 <b>Username:</b> @" + un.Username + "\n"
	}

	// ID
	userString += "➜ 🆔 <b>ID:</b> <code>" + strconv.FormatInt(un.ID, 10) + "</code>\n"

	// DC Location
	userString += "➜ 🌐 <b>DC:</b> {{dcId}}\n"

	// Phone visibility
	if un.Phone != "" {
		userString += "➜ 📱 <b>Phone:</b> +" + un.Phone + "\n"
	}

	// Account restrictions
	if un.Restricted {
		userString += "➜ 🚫 <b>Restricted:</b> Yes\n"
	}
	if un.Scam {
		userString += "➜ ⚠️ <b>Scam:</b> Yes\n"
	}
	if un.Fake {
		userString += "➜ ⚠️ <b>Fake:</b> Yes\n"
	}

	// Support/official status
	if un.Support {
		userString += "➜ 🛟 <b>Support:</b> Yes\n"
	}

	// Bot specific features
	if un.Bot {
		if un.BotChatHistory {
			userString += "➜ 📜 <b>Can Read History:</b> Yes\n"
		}
		if un.BotInlineGeo {
			userString += "➜ 📍 <b>Inline Geo:</b> Yes\n"
		}
		if un.BotAttachMenu {
			userString += "➜ 📎 <b>Attach Menu:</b> Yes\n"
		}
		if un.BotInlinePlaceholder != "" {
			userString += "➜ 💭 <b>Inline Placeholder:</b> " + un.BotInlinePlaceholder + "\n"
		}
	}

	// Common chats count
	if uf.CommonChatsCount > 0 {
		userString += "➜ 👥 <b>Common Groups:</b> " + strconv.Itoa(int(uf.CommonChatsCount)) + "\n"
	}

	// Reserved usernames
	if len(un.Usernames) > 0 {
		var usernames []string
		for _, v := range un.Usernames {
			usernames = append(usernames, "@"+v.Username)
		}
		userString += "\n📌 <b>Also known as:</b> " + strings.Join(usernames, ", ") + "\n"
	}

	// Birthday
	if uf.Birthday != nil {
		userString += "\n➜ 🎂 <b>Birthday:</b> " + parseBirthday(uf.Birthday.Day, uf.Birthday.Month, uf.Birthday.Year) + "\n"
	}

	// Bio
	if uf.About != "" {
		userString += "\n💬 <b>Bio:</b> <i>" + uf.About + "</i>\n"
	}

	// Profile link
	userString += "\n<a href=\"tg://user?id=" + strconv.FormatInt(un.ID, 10) + "\">🔗 View Full Profile</a>"

	var keyb = telegram.NewKeyboard()
	sendableUser, err := m.Client.GetSendableUser(un)
	if err == nil {
		keyb.AddRow(
			telegram.Button.Mention("View Profile", sendableUser),
		)
	} else {
		keyb.AddRow(
			telegram.Button.URL("View Profile", "tg://user?id="+strconv.FormatInt(un.ID, 10)),
		)
	}

	var dcId = 0

	if uf.ProfilePhoto == nil {
		if uf.PersonalPhoto != nil {
			uf.ProfilePhoto = uf.PersonalPhoto
		} else if uf.FallbackPhoto != nil {
			uf.ProfilePhoto = uf.FallbackPhoto
		}
	}

	var buisnessSent bool = false

	if uf.BusinessIntro != nil && uf.BusinessIntro.Sticker != nil {
		stick := uf.BusinessIntro.Sticker.(*telegram.DocumentObj)
		dcId = int(stick.DcID)
		sty := &telegram.InputMediaDocument{
			ID: &telegram.InputDocumentObj{
				ID:            stick.ID,
				AccessHash:    stick.AccessHash,
				FileReference: stick.FileReference,
			},
		}
		if _, err := m.ReplyMedia(sty, telegram.MediaOptions{
			ReplyMarkup: keyb.Build(),
		}); err == nil {
			buisnessSent = true
		}
	}

	mediaOpt := telegram.MediaOptions{
		Caption: userString,
	}

	sendOpt := telegram.SendOptions{}

	if !buisnessSent {
		mediaOpt.ReplyMarkup = keyb.Build()
		sendOpt.ReplyMarkup = keyb.Build()
	}

	if uf.ProfilePhoto != nil {
		p := uf.ProfilePhoto.(*telegram.PhotoObj)
		if uf.PersonalPhoto != nil {
			p = uf.PersonalPhoto.(*telegram.PhotoObj)
		}
		dcId = int(p.DcID)
		var inp telegram.InputMedia
		inp = &telegram.InputMediaPhoto{
			ID: &telegram.InputPhotoObj{
				ID:            p.ID,
				AccessHash:    p.AccessHash,
				FileReference: p.FileReference,
			},
			Spoiler: true,
		}
		if len(p.VideoSizes) > 0 {
			dled, err := m.Client.DownloadMedia(p, &telegram.DownloadOptions{
				IsVideo: true,
			})
			if err == nil {
				ul, err := m.Client.UploadFile(dled)
				if err == nil {
					inp = &telegram.InputMediaUploadedDocument{
						File:         ul,
						NosoundVideo: true,
						Spoiler:      true,
						MimeType:     "video/mp4",
					}
				}
			}
		}

		dcFlag := getCountryFlag(dcId)
		mediaOpt.Caption = strings.ReplaceAll(userString, "{{dcId}}", fmt.Sprintf("DC%d - %s %s", dcId, dcLocationMap[dcId], dcFlag))
		_, err := m.ReplyMedia(inp, mediaOpt)
		if err != nil {
			m.Reply(userString, sendOpt)
		}
	} else {
		dcFlag := getCountryFlag(dcId)
		userString = strings.ReplaceAll(userString, "{{dcId}}", fmt.Sprintf("DC%d - %s %s", dcId, dcLocationMap[dcId], dcFlag))
		m.Reply(userString, sendOpt)
	}
	return nil
}

func getCountryFlag(dcId int) string {
	flags := map[int]string{
		1: "🇺🇸",
		2: "🇳🇱",
		3: "🇺🇸",
		4: "🇳🇱",
		5: "🇸🇬",
	}
	if flag, ok := flags[dcId]; ok {
		return flag
	}
	return ""
}

var st = time.Now()

func PingHandle(m *telegram.NewMessage) error {
	startTime := time.Now()
	sentMessage, _ := m.Reply("Pinging...")
	_, err := sentMessage.Edit(fmt.Sprintf("<code>Pong!</code> <code>%s</code>\n<code>Uptime ⚡ </code><b>%s</b>", time.Since(startTime).String(), time.Since(st).String()))
	return err
}

func init() {
	Mods.AddModule("Start", `<b>Here are the commands available in Start module:</b>

<code>/start</code> - check if the bot is alive
<code>/ping</code> - check the bot's response time
<code>/systeminfo</code> - get system information
<code>/info [user_id]</code> - get user information`)
}
