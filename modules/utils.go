package modules

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

func parseBirthday(dat, month, year int32) string {
	months := []string{
		"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December",
	}
	result := strconv.Itoa(int(dat)) + ", " + months[month-1]
	if year != 0 {
		result += ", " + strconv.Itoa(int(year))
	}

	return result + "; is in " + tillDate(dat, month)
}

func tillDate(dat, month int32) string {
	currYear := time.Now().Year()

	timeBday := time.Date(currYear, time.Month(month), int(dat), 0, 0, 0, 0, time.UTC)
	currTime := time.Now()

	if timeBday.Before(currTime) {
		timeBday = time.Date(currYear+1, time.Month(month), int(dat), 0, 0, 0, 0, time.UTC)
	}

	// convert to days only
	days := timeBday.Sub(currTime).Hours() / 24

	return strconv.Itoa(int(days)) + " days"
}

func UpdateSourceCodeHandle(m *telegram.NewMessage) error {
	msg, _ := m.Reply("<code>Updating source code...</code>")
	defer msg.Edit("<code>Updated, restarting...</code>")

	exec.Command("git", "pull").Run()

	msg.Edit("<code>Synced with remote repo.</code>")
	exec.Command("bash", "-c", selfRestartCMD).Start()
	// get current pid and kill it
	pid := os.Getpid()
	process, err := os.FindProcess(pid)
	if err != nil {
		msg.Edit("<code>Error while restarting: " + err.Error() + "</code>")
		return err
	}
	process.Kill()

	return nil
}

func gatherSystemInfo() (*SystemInfo, error) {
	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}

	memr, err := proc.MemoryInfo()
	if err != nil {
		return nil, err
	}

	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return nil, err
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, err
	}

	processMemory := HumanBytes(memr.RSS)
	cpuPercentStr := strconv.FormatFloat(cpuPercent[0], 'f', 2, 64) + "%"

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	memUsed := HumanBytes(memInfo.Used)
	memTotal := HumanBytes(memInfo.Total)

	diskInfo, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}

	return &SystemInfo{
		NumGoroutines: runtime.NumGoroutine(),
		ProcessID:     pid,
		ProcessMemory: processMemory,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH + " (" + strconv.Itoa(runtime.NumCPU()) + " CPUs)",
		Uptime:        time.Since(startTime).Round(time.Second).String(),
		MemUsed:       memUsed,
		MemTotal:      memTotal,
		CPUPercent:    cpuPercentStr,
		CPUPerc:       cpuPercent[0],
		MemPerc:       memInfo.UsedPercent,
		CPUName:       cpuInfo[0].ModelName,
		DiskUsed:      HumanBytes(diskInfo.Used),
		DiskTotal:     HumanBytes(diskInfo.Total),
		DiskPerc:      diskInfo.UsedPercent,
	}, nil
}

type SystemInfo struct {
	NumGoroutines int
	ProcessID     int32
	ProcessMemory string
	OS            string
	Arch          string
	Uptime        string
	MemUsed       string
	MemTotal      string
	CPUPercent    string
	CPUPerc       float64
	MemPerc       float64
	CPUName       string
	DiskUsed      string
	DiskTotal     string
	DiskPerc      float64
}

func HumanBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatUint(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func trimString(s string, length int) string {
	if len(s) > length {
		return s[:length] + ""
	}

	return s
}

func IsUserAdmin(bot *telegram.Client, userID int, chatID int, right string) bool {
	member, err := bot.GetChatMember(chatID, userID)
	if err != nil {
		return false
	}

	if member.Status == telegram.Admin || member.Status == telegram.Creator {
		if right == "promote" {
			if member.Rights.AddAdmins {
				return true
			}
		} else if right == "ban" {
			if member.Rights.BanUsers {
				return true
			}
		} else if right == "delete" {
			if member.Rights.DeleteMessages {
				return true
			}
		}
	}
	return false
}

func CanBot(bot *telegram.Client, chat *telegram.Channel, right string) bool {
	if chat.AdminRights != nil {
		if right == "ban" {
			return chat.AdminRights.BanUsers
		} else if right == "delete" {
			return chat.AdminRights.DeleteMessages
		} else if right == "invite" {
			return chat.AdminRights.InviteUsers
		} else if right == "promote" {
			return chat.AdminRights.AddAdmins
		}
	}
	return false
}

func GetUserFromContext(m *telegram.NewMessage) (telegram.InputPeer, string, error) {
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			return nil, "", err
		}
		peer, err := m.Client.ResolvePeer(reply.Sender)
		if err != nil {
			return nil, "", err
		}

		return peer, m.Args(), nil
	} else if m.Args() != "" {
		arg := m.Args()
		args := strings.Split(arg, " ")
		arg = args[0]

		argInt, err := strconv.Atoi(arg)
		if err == nil {
			user, err := m.Client.ResolvePeer(argInt)
			if err != nil {
				return nil, "", err
			}
			if len(args) > 1 {
				return user, strings.Join(args[1:], " "), nil
			}
			return user, "", nil
		} else {
			user, err := m.Client.ResolvePeer(arg)
			if err != nil {
				return nil, "", err
			}
			if len(args) > 1 {
				return user, strings.Join(args[1:], " "), nil
			}
			return user, "", nil
		}
	}
	return nil, "", errors.New("no user found in context")
}
