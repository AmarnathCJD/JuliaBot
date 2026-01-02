package modules

import (
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sort"
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

	days := timeBday.Sub(currTime).Hours() / 24

	return strconv.Itoa(int(days)) + " days"
}

func UpdateSourceCodeHandle(m *telegram.NewMessage) error {
	msg, _ := m.Reply("<code>Updating source code...</code>")
	defer msg.Edit("<code>Updated, restarting...</code>")

	exec.Command("git", "pull").Run()

	msg.Edit("<code>Synced with remote repo.</code>")
	exec.Command("bash", "-c", selfRestartCMD).Start()

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
		switch right {
		case "ban":
			return chat.AdminRights.BanUsers
		case "delete":
			return chat.AdminRights.DeleteMessages
		case "invite":
			return chat.AdminRights.InviteUsers
		case "promote":
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

type UserDateEstimator struct {
	sortedIDs [][2]int64
}

var telegramUserData = [][2]int64{
	{1000000, 1380326400}, {2768409, 1383264000}, {7679610, 1388448000}, {11538514, 1391212800},
	{15835244, 1392940000}, {23646077, 1393459200}, {38015510, 1393632000}, {44634663, 1399334400},
	{46145305, 1400198400}, {54845238, 1411257600}, {63263518, 1414454400}, {101260938, 1425600000},
	{101323197, 1426204800}, {103151531, 1433376000}, {103258382, 1432771200}, {109393468, 1439078400},
	{111220210, 1429574400}, {112594714, 1439683200}, {116812045, 1437696000}, {122600695, 1437782400},
	{124872445, 1439856000}, {125828524, 1444003200}, {130029930, 1441324800}, {133909606, 1444176000},
	{143445125, 1448928000}, {148670295, 1452211200}, {152079341, 1453420000}, {157242073, 1446768000},
	{171295414, 1457481600}, {181783990, 1460246400}, {222021233, 1465344000}, {225034354, 1466208000},
	{278941742, 1473465600}, {285253072, 1476835200}, {294851037, 1479600000}, {297621225, 1481846400},
	{328594461, 1482969600}, {337808429, 1487707200}, {341546272, 1487782800}, {352940995, 1487894400},
	{369669043, 1490918400}, {400169472, 1501459200}, {616816630, 1529625600}, {681896077, 1532821500},
	{727572658, 1543708800}, {796147074, 1541371800}, {925078064, 1563290400}, {928636984, 1581513420},
	{1054883348, 1585674420}, {1057704545, 1580393640}, {1145856008, 1586342040}, {1227964864, 1596127860},
	{1382531194, 1600188120}, {1658586909, 1613148540}, {1660971491, 1613329440}, {1692464211, 1615402500},
	{1719536397, 1619293500}, {1721844091, 1620224820}, {1772991138, 1617540360}, {1807942741, 1625520300},
	{1893429550, 1622040000}, {1972424006, 1631669400}, {1974255900, 1634000000}, {2030606431, 1631992680},
	{2041327411, 1631989620}, {2078711279, 1634321820}, {2104178931, 1638353220}, {2120496865, 1636714020},
	{2123596685, 1636503180}, {2138472342, 1637590800}, {3318845111, 1618028800}, {4317845111, 1620028800},
	{5162494923, 1652449800}, {5186883095, 1648764360}, {5304951856, 1656718440}, {5317829834, 1653152820},
	{5318092331, 1652024220}, {5336336790, 1646368100}, {5362593868, 1652024520}, {5387234031, 1662137700},
	{5396587273, 1648014800}, {5409444610, 1659025020}, {5416026704, 1660925460}, {5465223076, 1661710860},
	{5480654757, 1660926300}, {5499934702, 1662130740}, {5513192189, 1659626400}, {5522237606, 1654167240},
	{5537251684, 1664269800}, {5559167331, 1656718560}, {5568348673, 1654642200}, {5591759222, 1659025500},
	{5608562550, 1664012820}, {5614111200, 1661780160}, {5666819340, 1664112240}, {5684254605, 1662134040},
	{5684689868, 1661304720}, {5707112959, 1663803300}, {5756095415, 1660925940}, {5772670706, 1661539140},
	{5778063231, 1667477640}, {5802242180, 1671821040}, {5853442730, 1674866100}, {5859878513, 1673117760},
	{5885964106, 1671081840}, {5982648124, 1686941700}, {6020888206, 1675534800}, {6032606998, 1686998640},
	{6057123350, 1676198350}, {6058560984, 1686907980}, {6101607245, 1686830760}, {6108011341, 1681032060},
	{6132325730, 1692033840}, {6182056052, 1687870740}, {6279839148, 1688399160}, {6306077724, 1692442920},
	{6321562426, 1688486760}, {6364973680, 1696349340}, {6386727079, 1691696880}, {6429580803, 1692082680},
	{6527226055, 1690289160}, {6813121418, 1698489600}, {6865576492, 1699052400}, {6925870357, 1701192327},
	{6944368668, 1726144496}, {7682429075, 1725539696}, {8000499714, 1745152496}, {7798375391, 1745411696},
	{7321742406, 1726835696}, {7538420552, 1723379696}, {7745089650, 1745584496},
}

func NewUserDateEstimator() *UserDateEstimator {
	sorted := make([][2]int64, len(telegramUserData))
	copy(sorted, telegramUserData)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i][0] < sorted[j][0]
	})

	return &UserDateEstimator{
		sortedIDs: sorted,
	}
}

func (u *UserDateEstimator) Estimate(userID int64) int64 {
	left, right := 0, len(u.sortedIDs)-1

	if userID <= u.sortedIDs[0][0] {
		return u.sortedIDs[0][1]
	}

	if userID >= u.sortedIDs[right][0] {
		return u.sortedIDs[right][1]
	}

	for left < right-1 {
		mid := (left + right) / 2
		if u.sortedIDs[mid][0] < userID {
			left = mid
		} else {
			right = mid
		}
	}

	prev := u.sortedIDs[left]
	curr := u.sortedIDs[right]

	ratio := float64(userID-prev[0]) / float64(curr[0]-prev[0])
	estimatedTS := int64(float64(prev[1]) + ratio*float64(curr[1]-prev[1]))

	now := time.Now().Unix()
	if estimatedTS > now {
		return now
	}

	return estimatedTS
}

func (u *UserDateEstimator) FormatTime(unixTime int64) (string, string) {
	createdTime := time.Unix(unixTime, 0).UTC()
	now := time.Now().UTC()

	formattedDate := createdTime.Format("2006-01-02 15:04:05")

	diff := now.Sub(createdTime)
	diffMs := diff.Milliseconds()

	if diffMs < 60000 {
		return formattedDate, "just now"
	}

	if diffMs < 3600000 {
		mins := diffMs / 60000
		if mins == 1 {
			return formattedDate, "a minute ago"
		}
		return formattedDate, fmt.Sprintf("%d minutes ago", mins)
	}

	if diffMs < 86400000 {
		hrs := diffMs / 3600000
		if hrs == 1 {
			return formattedDate, "an hour ago"
		}
		return formattedDate, fmt.Sprintf("%d hours ago", hrs)
	}

	if diffMs < 604800000 {
		days := diffMs / 86400000
		if days == 1 {
			return formattedDate, "yesterday"
		}
		return formattedDate, fmt.Sprintf("%d days ago", days)
	}

	if diffMs < 2592000000 {
		weeks := diffMs / 604800000
		if weeks == 1 {
			return formattedDate, "last week"
		}
		return formattedDate, fmt.Sprintf("%d weeks ago", weeks)
	}

	if diffMs < 31536000000 {
		months := diffMs / 2592000000
		if months == 1 {
			return formattedDate, "last month"
		}
		return formattedDate, fmt.Sprintf("%d months ago", months)
	}

	years := diffMs / 31536000000
	if years == 1 {
		return formattedDate, "last year"
	}
	return formattedDate, fmt.Sprintf("%d years ago", years)
}

func polyfit(x, y []float64, degree int) []float64 {
	n := len(x)
	coeffs := make([]float64, degree+1)

	for i := 0; i <= degree; i++ {
		for j := 0; j < n; j++ {
			pow := math.Pow(x[j], float64(i))
			for k := 0; k <= degree; k++ {
				coeffs[k] += pow * math.Pow(x[j], float64(k)) * y[j]
			}
		}
	}

	return coeffs
}

func polyeval(coeffs []float64, x float64) float64 {
	result := 0.0
	for i, c := range coeffs {
		result += c * math.Pow(x, float64(i))
	}
	return result
}
