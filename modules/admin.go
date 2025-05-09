package modules

import (
	"os/exec"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func PromoteUserHandle(m *tg.NewMessage) error {
	m.Reply("Umbikko Myre")
	return nil
}

var (
	spotifyRestartCMD = "sudo systemctl restart spotdl.service"
	proxyRestartCMD   = "sudo systemctl restart wireproxy.service"
	selfRestartCMD    = "sudo systemctl restart rustybot.service"
)

func RestartSpotify(m *tg.NewMessage) error {
	m.Reply("Restarting Spotify...")
	if err := execCommand(spotifyRestartCMD); err != nil {
		return err
	}
	m.Reply("Spotify restarted successfully.")
	return nil
}

func RestartProxy(m *tg.NewMessage) error {
	m.Reply("Restarting WProxy...")
	if err := execCommand(proxyRestartCMD); err != nil {
		return err
	}
	m.Reply("Proxy restarted successfully.")
	return nil
}

func RestartHandle(m *tg.NewMessage) error {
	m.Reply("Restarting bot...")
	if err := execCommand(selfRestartCMD); err != nil {
		return err
	}
	m.Reply("Bot restarted successfully.")
	return nil
}

func execCommand(cmd string) error {
	command := exec.Command("bash", "-c", cmd)
	_, err := command.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}
