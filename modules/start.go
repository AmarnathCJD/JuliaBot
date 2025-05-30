package modules

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var startTime = time.Now()

func StartHandle(m *telegram.NewMessage) error {
	m.Reply("Hellov! :)")
	m.React(getRandomEmoticon())
	return nil
}

func GatherSystemInfo(m *telegram.NewMessage) error {
	m.ChatType()

	msg, _ := m.Reply("<code>...System Information...</code>")

	if IsImageDepsInstalled() {
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

	info := "<b>💻 System Info:</b>\n\n"
	info += fmt.Sprintf("🖥️ <b>CPU:</b> %.2f%%\n", system.CPUPerc)
	info += fmt.Sprintf("📊 <b>Process Mem:</b> %s\n", system.ProcessMemory)
	info += fmt.Sprintf("⏱️ <b>Uptime:</b> %s\n", system.Uptime)
	info += fmt.Sprintf("🧑‍💻 <b>OS:</b> %s | <b>Arch:</b> %s\n", runtime.GOOS, runtime.GOARCH)
	info += fmt.Sprintf("🚀 <b>CPUs:</b> %d | <b>Goroutines:</b> %d\n", runtime.NumCPU(), runtime.NumGoroutine())
	info += fmt.Sprintf("🆔 <b>PID:</b> %d\n", system.ProcessID)
	info += fmt.Sprintf("💾 <b>Memory:</b> %s / %s (%.2f%%)\n", system.MemUsed, system.MemTotal, system.MemPerc)
	info += fmt.Sprintf("💽 <b>Disk:</b> %s / %s (%.2f%%)\n", system.DiskUsed, system.DiskTotal, system.DiskPerc)
	f, _ := telegram.ResolveBotFileID("AgAABZq_MRv8XKlV4gk2goxvC_A")

	_, err = msg.Edit(
		"",
		telegram.SendOptions{Caption: info, Media: f},
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
	userString += "<b>User Info:</b>\n"
	if un.FirstName != "" {
		userString += "<b>First Name:</b> " + un.FirstName + "\n"
	}
	if un.LastName != "" {
		userString += "<b>Last Name:</b> " + un.LastName + "\n"
	}
	userString += "<b>Is Bot:</b> " + fmt.Sprintf("%t", un.Bot) + "\n"
	if un.Verified {
		userString += "<b>Is Verified:</b> ✅\n"
	}
	userString += "<b>Data Center:</b> {{dcId}}\n"
	if un.Username != "" {
		userString += "<b>Username:</b> @" + un.Username + "\n"
	}
	if uf.About != "" {
		userString += "\n<i>" + uf.About + "</i>\n\n"
	}
	if un.Usernames != nil {
		userString += "<b>Res. Usernames:</b> [<b>" + func() string {
			var s string
			for _, v := range un.Usernames {
				s += "@" + v.Username + " "
			}
			return s
		}() + "</b>]\n"
	}

	userString += "<b>User Link:</b> <a href=\"tg://user?id=" + strconv.FormatInt(un.ID, 10) + "\">userLink</a>\n<b>User-ID:</b> <code>" + strconv.FormatInt(un.ID, 10) + "</code>\n"
	if uf.Birthday != nil {
		userString += "\n<b>Birthday:</b> " + parseBirthday(uf.Birthday.Day, uf.Birthday.Month, uf.Birthday.Year)
	}

	var keyb = telegram.NewKeyboard()
	sendableUser, err := m.Client.GetSendableUser(un)
	if err == nil {
		keyb.AddRow(
			telegram.Button.Mention("Go >> User Profile", sendableUser),
		)
	} else {
		keyb.AddRow(
			telegram.Button.URL("Go >> User Profile", "tg://user?id="+strconv.FormatInt(un.ID, 10)),
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

		mediaOpt.Caption = strings.ReplaceAll(userString, "{{dcId}}", fmt.Sprintf("<b>%d - %s</b>", dcId, dcLocationMap[dcId]))
		_, err := m.ReplyMedia(inp, mediaOpt)
		if err != nil {
			m.Reply(userString, sendOpt)
		}
	} else {
		userString = strings.ReplaceAll(userString, "{{dcId}}", fmt.Sprintf("<b>%d</b> - <b>%s</b>", dcId, dcLocationMap[dcId]))
		m.Reply(userString, sendOpt)
	}
	return nil
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
