package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
)

var startTime = time.Now()

func SnapSaveHandler(m *telegram.NewMessage) error {
	return nil
}

func StartHandle(m *telegram.NewMessage) error {
	greeting := "✨ <b>Hello there!</b> ✨\n\n"
	greeting += "I'm <b>Julia</b>, your friendly bot companion! 🤖💙\n\n"

	m.Reply(greeting)
	m.React("❤")
	return nil
}

func GatherSystemInfo(m *telegram.NewMessage) error {
	m.ChatType()

	msg, _ := m.Reply("<code>...System Information...</code>")

	system, err := gatherSystemInfo()
	if err != nil {
		return err
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	hostInfo, _ := host.Info()
	loadAvg, _ := load.Avg()

	info := "<b>System Information</b>\n\n"

	info += fmt.Sprintf("⚡ <b>Goroutines:</b> <code>%d</code> | <b>Process Memory:</b> <code>%s</code>\n\n", runtime.NumGoroutine(), system.ProcessMemory)

	info += "➜ <b><i>Performance</i></b>\n"
	info += fmt.Sprintf("   ├ <b>CPU Usage:</b> <code>%.2f%%</code>\n", system.CPUPerc)
	if loadAvg != nil {
		info += fmt.Sprintf("   ├ <b>Load Average:</b> <code>%.2f, %.2f, %.2f</code>\n", loadAvg.Load1, loadAvg.Load5, loadAvg.Load15)
	}
	info += fmt.Sprintf("   ├ <b>Heap Allocated:</b> <code>%s</code>\n", HumanBytes(memStats.Alloc))
	info += fmt.Sprintf("   ├ <b>Heap System:</b> <code>%s</code>\n", HumanBytes(memStats.Sys))
	info += fmt.Sprintf("   └ <b>Uptime:</b> <i>%s</i>\n\n", system.Uptime)

	info += "➜ <b><i>Hardware</i></b>\n"
	info += fmt.Sprintf("   ├ <b>CPU:</b> <i>%s</i>\n", system.CPUName)
	info += fmt.Sprintf("   ├ <b>Cores:</b> <code>%d</code>\n", runtime.NumCPU())
	info += fmt.Sprintf("   ├ <b>Memory:</b> <code>%s</code> / <code>%s</code> <i>(%.1f%%)</i>\n", system.MemUsed, system.MemTotal, system.MemPerc)
	info += fmt.Sprintf("   └ <b>Disk:</b> <code>%s</code> / <code>%s</code> <i>(%.1f%%)</i>\n\n", system.DiskUsed, system.DiskTotal, system.DiskPerc)

	info += "➜ <b><i>Runtime</i></b>\n"
	info += fmt.Sprintf("   ├ <b>Go Version:</b> <code>%s</code>\n", runtime.Version())
	info += fmt.Sprintf("   ├ <b>Platform:</b> <code>%s/%s</code>\n", runtime.GOOS, runtime.GOARCH)
	if hostInfo != nil {
		info += fmt.Sprintf("   ├ <b>Hostname:</b> <code>%s</code>\n", hostInfo.Hostname)
		info += fmt.Sprintf("   ├ <b>Boot Time:</b> <i>%s</i>\n", time.Unix(int64(hostInfo.BootTime), 0).Format("2006-01-02 15:04:05"))
	}
	info += fmt.Sprintf("   ├ <b>GC Cycles:</b> <code>%d</code> | <b>Pauses:</b> <code>%s</code>\n", memStats.NumGC, time.Duration(memStats.PauseTotalNs).Round(time.Millisecond))
	info += fmt.Sprintf("   └ <b>PID:</b> <code>%d</code>\n\n", system.ProcessID)
	info += "<i>Have a great day! 🌟</i>"
	msg.Edit(info)
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

	name := un.FirstName
	if un.LastName != "" {
		name += " " + un.LastName
	}
	userString += "👤 <b>" + name + "</b>"

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

	if un.Username != "" {
		userString += "➜ 📧 <b>Username:</b> @" + un.Username + "\n"
	}

	userString += "➜ 🆔 <b>ID:</b> <code>" + strconv.FormatInt(un.ID, 10) + "</code>\n"

	userString += "➜ 🌐 <b>DC:</b> {{dcId}}\n"

	if un.Phone != "" {
		userString += "➜ 📱 <b>Phone:</b> +" + un.Phone + "\n"
	}

	if un.Restricted {
		userString += "➜ 🚫 <b>Restricted:</b> Yes\n"
	}
	if un.Scam {
		userString += "➜ ⚠️ <b>Scam:</b> Yes\n"
	}
	if un.Fake {
		userString += "➜ ⚠️ <b>Fake:</b> Yes\n"
	}

	if un.Support {
		userString += "➜ 🛟 <b>Support:</b> Yes\n"
	}

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

	if uf.CommonChatsCount > 0 {
		userString += "➜ 👥 <b>Common Groups:</b> " + strconv.Itoa(int(uf.CommonChatsCount)) + "\n"
	}

	if len(un.Usernames) > 0 {
		var usernames []string
		for _, v := range un.Usernames {
			usernames = append(usernames, "@"+v.Username)
		}
		userString += "\n📌 <b>Also known as:</b> " + strings.Join(usernames, ", ") + "\n"
	}

	if uf.Birthday != nil {
		userString += "\n➜ 🎂 <b>Birthday:</b> " + parseBirthday(uf.Birthday.Day, uf.Birthday.Month, uf.Birthday.Year) + "\n"
	}

	estimator := NewUserDateEstimator()
	estimatedTS := estimator.Estimate(un.ID)
	formattedDate, age := estimator.FormatTime(estimatedTS)
	userString += "\n<b>Account:</b> " + age + " | <code>" + formattedDate + "</code>\n"

	if uf.About != "" {
		userString += "\n💬 <b>Bio:</b> <i>" + uf.About + "</i>\n"
	}

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
		if _, err := m.ReplyMedia(sty, &telegram.MediaOptions{
			ReplyMarkup: keyb.Build(),
		}); err == nil {
			buisnessSent = true
		}
	}

	mediaOpt := &telegram.MediaOptions{
		Caption: userString,
	}

	sendOpt := &telegram.SendOptions{}

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

func generateCountdownGifFFmpeg(days, hours, minutes, seconds int) (string, error) {
	tmp := os.TempDir()
	framesDir := filepath.Join(tmp, "countdown_frames")
	gifPath := filepath.Join(tmp, "countdown_newyear.gif")

	os.MkdirAll(framesDir, 0755)
	defer os.RemoveAll(framesDir)

	fontPath, err := findFont()
	if err != nil {
		fontPath = "Arial"
	}
	fontPath = strings.ReplaceAll(fontPath, "\\", "/")
	fontPath = strings.ReplaceAll(fontPath, ":", "\\:")

	framesPerSecond := 10
	totalSeconds := 10
	frameCount := framesPerSecond * totalSeconds
	for i := 0; i < frameCount; i++ {
		// Calculate which second we're on
		secondOffset := i / framesPerSecond
		_ = i % framesPerSecond // subframe for animation variation

		s := seconds - secondOffset
		m := minutes
		h := hours
		d := days

		for s < 0 {
			s += 60
			m--
		}
		for m < 0 {
			m += 60
			h--
		}
		for h < 0 {
			h += 24
			d--
		}
		if d < 0 {
			d = 0
			h = 0
			m = 0
			s = 0
		}

		countdownText := fmt.Sprintf("%dd %02dh %02dm %02ds", d, h, m, s)
		framePath := filepath.Join(framesDir, fmt.Sprintf("frame_%03d.png", i))

		// Smooth animated sparkle offsets
		off := i * 2
		y1 := 30 + (i%20)*3
		y2 := 80 + ((i+5)%25)*2
		y3 := 50 + ((i+10)%15)*3
		y4 := 120 + ((i+15)%20)*2
		sz1 := 5 + (i % 5)
		sz2 := 6 + ((i + 2) % 4)
		sz3 := 4 + ((i + 4) % 4)

		filter := fmt.Sprintf(
			"[0:v]"+
				"geq=r='12+10*(Y/H)':g='5+18*(Y/H)':b='25+35*(Y/H)',"+
				// Thick top border
				"drawbox=x=0:y=0:w=iw:h=5:color=#ffd700@0.95:t=fill,"+
				"drawbox=x=0:y=5:w=iw:h=3:color=#ffaa00@0.6:t=fill,"+
				// Thick bottom border
				"drawbox=x=0:y=ih-5:w=iw:h=5:color=#ffd700@0.95:t=fill,"+
				"drawbox=x=0:y=ih-8:w=iw:h=3:color=#ffaa00@0.6:t=fill,"+
				// Left side accent line
				"drawbox=x=0:y=0:w=4:h=ih:color=#ffd700@0.4:t=fill,"+
				// Right side accent line
				"drawbox=x=iw-4:y=0:w=4:h=ih:color=#ffd700@0.4:t=fill,"+

				// TOP LEFT sparkle cluster
				"drawbox=x=%d:y=%d:w=%d:h=%d:color=#ffd700:t=fill,"+
				"drawbox=x=%d:y=%d:w=%d:h=%d:color=#ffffff@0.9:t=fill,"+
				"drawbox=x=%d:y=%d:w=%d:h=%d:color=#ffd700@0.8:t=fill,"+
				"drawbox=x=%d:y=%d:w=%d:h=%d:color=#ffffff@0.7:t=fill,"+

				// TOP RIGHT sparkle cluster
				"drawbox=x=iw-%d:y=%d:w=%d:h=%d:color=#ffd700:t=fill,"+
				"drawbox=x=iw-%d:y=%d:w=%d:h=%d:color=#ffffff@0.9:t=fill,"+
				"drawbox=x=iw-%d:y=%d:w=%d:h=%d:color=#ffd700@0.8:t=fill,"+
				"drawbox=x=iw-%d:y=%d:w=%d:h=%d:color=#ffffff@0.7:t=fill,"+

				// MIDDLE LEFT floating sparkles
				"drawbox=x=%d:y=%d:w=%d:h=%d:color=#ffd700@0.85:t=fill,"+
				"drawbox=x=%d:y=%d:w=4:h=4:color=#ffffff@0.6:t=fill,"+

				// MIDDLE RIGHT floating sparkles
				"drawbox=x=iw-%d:y=%d:w=%d:h=%d:color=#ffd700@0.85:t=fill,"+
				"drawbox=x=iw-%d:y=%d:w=4:h=4:color=#ffffff@0.6:t=fill,"+

				// BOTTOM sparkles
				"drawbox=x=%d:y=ih-%d:w=%d:h=%d:color=#ffd700@0.7:t=fill,"+
				"drawbox=x=iw-%d:y=ih-%d:w=%d:h=%d:color=#ffd700@0.7:t=fill,"+
				"drawbox=x=%d:y=ih-%d:w=5:h=5:color=#ffffff@0.5:t=fill,"+
				"drawbox=x=iw-%d:y=ih-%d:w=5:h=5:color=#ffffff@0.5:t=fill,"+

				// CENTER scattered sparkles
				"drawbox=x=%d:y=%d:w=4:h=4:color=#ffd700@0.6:t=fill,"+
				"drawbox=x=iw-%d:y=%d:w=4:h=4:color=#ffd700@0.6:t=fill,"+

				// Shadow text (big)
				"drawtext=fontfile='%s':text='%s':fontsize=72:fontcolor=#1a0a00@0.7:x=(w-text_w)/2+4:y=(h-text_h)/2-25+4,"+
				// Main countdown text (big golden)
				"drawtext=fontfile='%s':text='%s':fontsize=72:fontcolor=#ffd700:x=(w-text_w)/2:y=(h-text_h)/2-25,"+
				// Subtitle shadow
				"drawtext=fontfile='%s':text='NEW YEAR 2026':fontsize=28:fontcolor=#1a0a00@0.5:x=(w-text_w)/2+2:y=h-62,"+
				// Subtitle gold
				"drawtext=fontfile='%s':text='NEW YEAR 2026':fontsize=28:fontcolor=#ffe066:x=(w-text_w)/2:y=h-60"+
				"[out]",

			// Top left cluster
			25+off, y1, sz1, sz1,
			50+off, y2, sz2, sz2,
			90+off, y3, sz1+2, sz1+2,
			130+off, y1+30, sz3, sz3,

			// Top right cluster
			30+off, y1, sz1, sz1,
			60+off, y2, sz2, sz2,
			100+off, y3, sz1+2, sz1+2,
			140+off, y1+30, sz3, sz3,

			// Middle left
			40+off, y4, sz2, sz2,
			80+off, y4+40,

			// Middle right
			45+off, y4, sz2, sz2,
			90+off, y4+40,

			// Bottom
			60+off, 50+y1/2, sz1, sz1,
			70+off, 50+y1/2, sz1, sz1,
			200+off, 60,
			220+off, 60,

			// Center scattered
			350+off, y4,
			360+off, y4+20,

			// Text
			fontPath, countdownText,
			fontPath, countdownText,
			fontPath,
			fontPath,
		)

		cmd := exec.Command(
			"ffmpeg", "-y",
			"-f", "lavfi",
			"-i", "color=c=#0f0520:s=900x400:d=1",
			"-filter_complex", filter,
			"-map", "[out]",
			"-frames:v", "1",
			framePath,
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("frame %d failed: %v - %s", i, err, stderr.String())
		}
	}

	var stderr bytes.Buffer
	cmdGif := exec.Command(
		"ffmpeg", "-y",
		"-framerate", "24",
		"-i", filepath.Join(framesDir, "frame_%03d.png"),
		"-vf", "split[s0][s1];[s0]palettegen=max_colors=256[p];[s1][p]paletteuse=dither=bayer",
		"-loop", "0",
		gifPath,
	)
	cmdGif.Stderr = &stderr
	if err := cmdGif.Run(); err != nil {
		return "", fmt.Errorf("gif creation failed: %v - %s", err, stderr.String())
	}

	return gifPath, nil
}

func findFont() (string, error) {
	paths := []string{
		`C:/Windows/Fonts/arial.ttf`,
		`C:/Windows/Fonts/Arial.ttf`,
		`C:/Windows/Fonts/segoeui.ttf`,
		`C:/Windows/Fonts/consola.ttf`,
		`/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf`,
		`/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf`,
		`/usr/share/fonts/TTF/DejaVuSans.ttf`,
		`/System/Library/Fonts/Helvetica.ttc`,
		`/System/Library/Fonts/SFNSText.ttf`,
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no font found")
}

var st = time.Now()

func PingHandle(m *telegram.NewMessage) error {
	startTime := time.Now()
	sentMessage, _ := m.Reply("Pinging...")
	_, err := sentMessage.Edit(fmt.Sprintf("<code>Pong!</code> <code>%s</code>\n<code>Uptime ⚡ </code><b>%s</b>", time.Since(startTime).String(), time.Since(st).String()))
	return err
}

func NewYearHandle(m *telegram.NewMessage) error {
	ist, _ := time.LoadLocation("Asia/Kolkata")

	newYear := time.Date(2027, time.January, 1, 0, 0, 0, 0, ist)
	now := time.Now().In(ist)

	remaining := newYear.Sub(now)

	days := int(remaining.Hours()) / 24
	hours := int(remaining.Hours()) % 24
	minutes := int(remaining.Minutes()) % 60
	seconds := int(remaining.Seconds()) % 60
	milliseconds := remaining.Milliseconds() % 1000

	timeStr := fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
	msg := fmt.Sprintf("<b>New Year 2026 Countdown</b>\n<code>%d days, %s</code>", days, timeStr)

	// gifPath, err := generateCountdownGifFFmpeg(days, hours, minutes, seconds)
	// if err == nil {
	// 	defer os.Remove(gifPath)
	// 	fi, err := m.Client.UploadFile(gifPath)
	// 	if err != nil {
	// 		fmt.Printf("Error uploading GIF: %v\n", err)
	// 		m.Reply(msg)
	// 		return nil
	// 	}

	// 	gifMedia := &telegram.InputMediaUploadedDocument{
	// 		File:         fi,
	// 		Spoiler:      true,
	// 		NosoundVideo: true,
	// 		Attributes: []telegram.DocumentAttribute{
	// 			&telegram.DocumentAttributeAnimated{},
	// 		},
	// 		MimeType: "image/gif",
	// 	}
	// 	_, err = m.ReplyMedia(gifMedia, &telegram.MediaOptions{
	// 		Caption: msg,
	// 	})
	// } else {
	// 	fmt.Printf("Error generating GIF: %v\n", err)
	m.Reply(msg)
	// }
	return nil
}

type UDResponse struct {
	List []struct {
		Definition string `json:"definition"`
		Example    string `json:"example"`
		Word       string `json:"word"`
		Author     string `json:"author"`
	} `json:"list"`
}

func UDHandler(m *tg.NewMessage) error {
	term := m.Args()
	if term == "" {
		m.Reply("Usage: /ud <term>")
		return nil
	}

	resp, err := http.Get("http://api.urbandictionary.com/v0/define?term=" + url.QueryEscape(term))
	if err != nil {
		m.Reply("Failed to fetch Urban Dictionary")
		return nil
	}
	defer resp.Body.Close()

	var res UDResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil
	}

	if len(res.List) == 0 {
		m.Reply("No definition found")
		return nil
	}

	def := res.List[0]
	definition := strings.ReplaceAll(def.Definition, "[", "")
	definition = strings.ReplaceAll(definition, "]", "")
	example := strings.ReplaceAll(def.Example, "[", "")
	example = strings.ReplaceAll(example, "]", "")

	if len(definition) > 1000 {
		definition = definition[:1000] + "..."
	}

	msg := fmt.Sprintf("<b>%s</b>\n\n%s\n\n<i>%s</i>", def.Word, definition, example)
	m.Reply(msg)
	return nil
}

func init() {
	Mods.AddModule("Start", `<b>Here are the commands available in Start module:</b>

<code>/start</code> - check if the bot is alive
<code>/ping</code> - check the bot's response time
<code>/new</code> - time left until New Year's Eve 2026
<code>/systeminfo</code> - get system information
<code>/info [user_id]</code> - get user information
<code>/ud [term]</code> - Urban Dictionary lookup
<code>/translate [lang] [-r]: Translate reply. -r replaces original.</code>
<code>/new: count down to Next New Years.`)
}
