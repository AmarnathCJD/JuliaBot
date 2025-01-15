package modules

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var startTime = time.Now()

func StartHandle(m *telegram.NewMessage) error {
	m.Reply("Hellow! :)")
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
	if un.Username != "" {
		userString += "<b>Username:</b> @" + un.Username + "\n"
	}
	if uf.About != "" {
		userString += "<b>About:</b> <code>" + uf.About + "</code>\n"
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
	userString += "<b>User Link:</b> <a href=\"tg://user?id=" + strconv.FormatInt(un.ID, 10) + "\">userLink</a>\n<b>ID:</b> <code>" + strconv.FormatInt(un.ID, 10) + "</code>\n"
	if uf.Birthday != nil {
		userString += "\n<b>Birthday:</b> " + parseBirthday(uf.Birthday.Day, uf.Birthday.Month, uf.Birthday.Year)
	}

	var keyb = telegram.NewKeyboard().AddRow(
		telegram.Button.URL("User Profile", "tg://user?id="+strconv.Itoa(int(uf.ID))),
	).Build()

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
		sty := &telegram.InputMediaDocument{
			ID: &telegram.InputDocumentObj{
				ID:            stick.ID,
				AccessHash:    stick.AccessHash,
				FileReference: stick.FileReference,
			},
		}
		if _, err := m.ReplyMedia(sty, telegram.MediaOptions{
			ReplyMarkup: keyb,
		}); err == nil {
			buisnessSent = true
		}
	}

	mediaOpt := telegram.MediaOptions{
		Caption: userString,
	}

	sendOpt := telegram.SendOptions{}

	if !buisnessSent {
		mediaOpt.ReplyMarkup = keyb
		sendOpt.ReplyMarkup = keyb
	}

	if uf.ProfilePhoto != nil {
		p := uf.ProfilePhoto.(*telegram.PhotoObj)
		if uf.PersonalPhoto != nil {
			p = uf.PersonalPhoto.(*telegram.PhotoObj)
		}
		inp := &telegram.InputMediaPhoto{
			ID: &telegram.InputPhotoObj{
				ID:            p.ID,
				AccessHash:    p.AccessHash,
				FileReference: p.FileReference,
			},
			Spoiler: true,
		}
		_, err := m.ReplyMedia(inp, mediaOpt)
		if err != nil {
			m.Reply(userString, sendOpt)
		}
	} else {
		m.Reply(userString, sendOpt)
	}
	return nil
}

func PingHandle(m *telegram.NewMessage) error {
	startTime := time.Now()
	sentMessage, _ := m.Reply("Pinging...")
	fmt.Println("Pong!")
	_, err := sentMessage.Edit(fmt.Sprintf("<code>Pong!</code> <code>%s</code>", time.Since(startTime).String()))
	return err
}

func init() {
	Mods.AddModule("Start", `<b>Here are the commands available in Start module:</b>

<code>/start</code> - check if the bot is alive
<code>/ping</code> - check the bot's response time
<code>/systeminfo</code> - get system information
<code>/info [user_id]</code> - get user information`)
}
