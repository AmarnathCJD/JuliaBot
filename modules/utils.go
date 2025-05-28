package modules

import (
	"fmt"
	"io"
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

func IsImageDepsInstalled() bool {
	// check of cairosvg and ffmpeg is installed
	_, err := exec.LookPath("cairosvg")
	if err != nil {
		return false
	}

	_, err = exec.LookPath("ffmpeg")
	return err == nil
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

type SystemInfo struct {
	NumGoroutines int
	ProcessID     int32
	ProcessMemory string
	OS            string
	Arch          string
	Uptime        string
	MemUsed       string
	MemTotal      string
	MemPerc       float64
	CPUPercent    string
	CPUPerc       float64
	CPUName       string
	DiskUsed      string
	DiskTotal     string
	DiskPerc      float64
}

func FillAndRenderSVG(webp bool) (string, error) {
	fi, err := os.Open("./assets/system.svg")
	if err != nil {
		return "", err
	}

	defer fi.Close()
	content, err := io.ReadAll(fi)
	if err != nil {
		return "", err
	}

	info, err := gatherSystemInfo()
	if err != nil {
		return "", err
	}

	svgContent := string(content)
	svgContent = strings.NewReplacer("{{cpu_perc}}", info.CPUPercent, "{{cpu_width}}", strconv.Itoa(findWidth(info.CPUPerc/100)),
		"{{mem_used}}", info.MemUsed, "{{mem_total}}", info.MemTotal, "{{mem_width}}", strconv.Itoa(findWidth(info.MemPerc/100)),
		"{{goroutines}}", strconv.Itoa(info.NumGoroutines), "{{pid}}", strconv.Itoa(int(info.ProcessID)),
		"{{mem_proc}}", info.ProcessMemory,
		"{{cpu_name}}", trimString(info.CPUName, 40), "{{disk_used}}", info.DiskUsed, "{{disk_total}}", info.DiskTotal, "{{disk_width}}", strconv.Itoa(findWidth(info.DiskPerc/100)),
		"{{operating_sys}}", info.OS, "{{arch}}", info.Arch, "{{uptime}}", info.Uptime).Replace(svgContent)

	fo, err := os.Create("./assets/system_rendered.svg")
	if err != nil {
		return "", err
	}

	defer fo.Close()
	_, err = fo.WriteString(svgContent)
	if err != nil {
		return "", err
	}

	// convert to png cairosvg a.svg -o o.png -s 4
	cmd := exec.Command("cairosvg", "./assets/system_rendered.svg", "-o", "./assets/system_rendered.png", "-s", "4")
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to convert svg to png: %w", err)
	}

	if webp {
		cmd = exec.Command("ffmpeg", "-i", "./assets/system_rendered.png", "-vcodec", "webp", "./assets/system_rendered.webp")
		err = cmd.Run()
		if err != nil {
			return "", fmt.Errorf("failed to convert png to webp: %w", err)
		}

		defer os.Remove("./assets/system_rendered.png")
		return "./assets/system_rendered.webp", nil
	}

	defer os.Remove("./assets/system_rendered.svg")
	return "./assets/system_rendered.png", nil
}

func findWidth(perc float64) int {
	totalW := 360
	return int(float64(totalW) * perc)
}

func trimString(s string, length int) string {
	if len(s) > length {
		return s[:length] + ""
	}

	return s
}

func mediaDownloadProgress(fname string, editMsg *telegram.NewMessage, pm *telegram.ProgressManager) func(atotalBytes, currentBytes int64) {
	return func(totalBytes int64, currentBytes int64) {
		text := ""
		text += "<b>📄 Name:</b> <code>%s</code>\n"
		text += "<b>💾 File Size:</b> <code>%.2f MiB</code>\n"
		text += "<b>⌛️ ETA:</b> <code>%s</code>\n"
		text += "<b>⏱ Speed:</b> <code>%s</code>\n"
		text += "<b>⚙️ Progress:</b> %s <code>%.2f%%</code>"

		size := float64(totalBytes) / 1024 / 1024
		eta := pm.GetETA(currentBytes)
		speed := pm.GetSpeed(currentBytes)
		percent := pm.GetProgress(currentBytes)

		progressbar := strings.Repeat("■", int(percent/10)) + strings.Repeat("□", 10-int(percent/10))

		message := fmt.Sprintf(text, fname, size, eta, speed, progressbar, percent)
		editMsg.Edit(message)
	}
}
