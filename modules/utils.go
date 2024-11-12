package modules

import (
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func parseBirthday(dat, month, year int32) string {
	months := []string{
		"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December",
	}
	result := strconv.Itoa(int(dat)) + ", " + months[month-1]
	if year != 0 {
		result += ", " + string(year)
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
	defer msg.Delete()

	processID := os.Getpid()
	exec.Command("git", "pull").Run()
	exec.Command("setsid", "go", "run", ".").Start()
	exec.Command("kill", "-9", strconv.Itoa(processID)).Run()

	return nil
}
