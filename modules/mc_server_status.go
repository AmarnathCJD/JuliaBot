package modules

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var mcServerHostRe = regexp.MustCompile(`^[A-Za-z0-9_.\-]{1,253}(?::\d{1,5})?$`)

type mcSrvMotd struct {
	Clean []string `json:"clean"`
	Raw   []string `json:"raw"`
}

type mcSrvPlayers struct {
	Online int `json:"online"`
	Max    int `json:"max"`
}

type mcSrvResponse struct {
	IP           string       `json:"ip"`
	Port         int          `json:"port"`
	Hostname     string       `json:"hostname"`
	Online       bool         `json:"online"`
	Version      string       `json:"version"`
	ProtocolName string       `json:"protocol_name"`
	Players      mcSrvPlayers `json:"players"`
	Motd         mcSrvMotd    `json:"motd"`
	Icon         string       `json:"icon"`
	EulaBlocked  bool         `json:"eula_blocked"`
	Software     string       `json:"software"`
}

func fetchMcServer(target string) (*mcSrvResponse, error) {
	url := "https://api.mcsrvstat.us/2/" + target
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "JuliaBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	var data mcSrvResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func saveFavicon(icon, target string) (string, error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(icon, prefix) {
		return "", fmt.Errorf("unsupported favicon format")
	}
	raw, err := base64.StdEncoding.DecodeString(icon[len(prefix):])
	if err != nil {
		return "", err
	}
	safe := regexp.MustCompile(`[^A-Za-z0-9_.\-]`).ReplaceAllString(target, "_")
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("mcsrv_%s_%d.png", safe, time.Now().UnixNano()))
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, strings.NewReader(string(raw))); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return tmp, nil
}

func McServerHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/mcserver &lt;ip[:port]&gt;</code>\n<b>Example:</b> <code>/mcserver mc.hypixel.net</code>")
		return err
	}
	arg = strings.Fields(arg)[0]
	arg = strings.TrimPrefix(arg, "http://")
	arg = strings.TrimPrefix(arg, "https://")
	arg = strings.TrimSuffix(arg, "/")
	if idx := strings.Index(arg, "/"); idx != -1 {
		arg = arg[:idx]
	}
	if !mcServerHostRe.MatchString(arg) {
		_, err := m.Reply("Invalid server address: <code>" + html.EscapeString(arg) + "</code>")
		return err
	}

	status, _ := m.Reply("Pinging Minecraft server <code>" + html.EscapeString(arg) + "</code>...")

	data, err := fetchMcServer(arg)
	if err != nil {
		msg := "Failed to query server: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	var sb strings.Builder
	sb.WriteString("<b>Minecraft Server Status</b>\n\n")

	displayHost := data.Hostname
	if displayHost == "" {
		displayHost = arg
	}
	sb.WriteString("<b>Host:</b> <code>" + html.EscapeString(displayHost) + "</code>\n")

	if data.IP != "" {
		addr := data.IP
		if data.Port != 0 {
			addr = fmt.Sprintf("%s:%d", data.IP, data.Port)
		}
		sb.WriteString("<b>Address:</b> <code>" + html.EscapeString(addr) + "</code>\n")
	}

	if data.Online {
		sb.WriteString("<b>Status:</b> Online\n")
	} else {
		sb.WriteString("<b>Status:</b> Offline\n")
		out := sb.String()
		if status != nil {
			status.Edit(out)
			return nil
		}
		_, e := m.Reply(out)
		return e
	}

	if data.Version != "" {
		sb.WriteString("<b>Version:</b> <code>" + html.EscapeString(data.Version) + "</code>\n")
	}
	if data.ProtocolName != "" {
		sb.WriteString("<b>Protocol:</b> <code>" + html.EscapeString(data.ProtocolName) + "</code>\n")
	}
	if data.Software != "" {
		sb.WriteString("<b>Software:</b> <code>" + html.EscapeString(data.Software) + "</code>\n")
	}

	sb.WriteString(fmt.Sprintf("<b>Players:</b> <code>%d / %d</code>\n", data.Players.Online, data.Players.Max))

	if len(data.Motd.Clean) > 0 {
		var motdLines []string
		for _, line := range data.Motd.Clean {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				motdLines = append(motdLines, html.EscapeString(trimmed))
			}
		}
		if len(motdLines) > 0 {
			sb.WriteString("<b>MOTD:</b>\n<blockquote>" + strings.Join(motdLines, "\n") + "</blockquote>\n")
		}
	}

	if data.EulaBlocked {
		sb.WriteString("<b>EULA:</b> Blocked\n")
	}

	caption := sb.String()

	if data.Icon != "" {
		iconPath, ierr := saveFavicon(data.Icon, arg)
		if ierr == nil {
			defer os.Remove(iconPath)
			if _, merr := m.ReplyMedia(iconPath, &tg.MediaOptions{
				Caption:  caption,
				FileName: "favicon.png",
				MimeType: "image/png",
			}); merr == nil {
				if status != nil {
					status.Delete()
				}
				return nil
			}
		}
	}

	if status != nil {
		status.Edit(caption)
		return nil
	}
	_, err = m.Reply(caption)
	return err
}

func registerMcServerHandlers() {
	c := Client
	c.On("cmd:mcserver", McServerHandler)
}

func init() {
	QueueHandlerRegistration(registerMcServerHandlers)
}
