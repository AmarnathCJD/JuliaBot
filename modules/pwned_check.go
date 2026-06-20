package modules

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func pwnedCheckLookup(input string) (int, string, error) {
	sum := sha1.Sum([]byte(input))
	full := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix := full[:5]
	suffix := full[5:]

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", "https://api.pwnedpasswords.com/range/"+prefix, nil)
	if err != nil {
		return 0, full, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0 pwned-check")
	req.Header.Set("Add-Padding", "true")

	resp, err := client.Do(req)
	if err != nil {
		return 0, full, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, full, fmt.Errorf("api status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(parts[0], suffix) {
			n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return 0, full, nil
			}
			if n <= 0 {
				return 0, full, nil
			}
			return n, full, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, full, err
	}
	return 0, full, nil
}

func PwnedCheckHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/pwned &lt;password_or_string&gt;</code>\n\nChecks against the Have I Been Pwned database using k-anonymity. Only the first 5 chars of the SHA1 hash are sent.")
		return err
	}

	status, _ := m.Reply("<code>checking breach database...</code>")

	count, full, err := pwnedCheckLookup(arg)
	if err != nil {
		msg := "lookup failed: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	var sb strings.Builder
	sb.WriteString("<b>Have I Been Pwned</b>\n\n")
	sb.WriteString("<b>SHA1:</b> <code>" + html.EscapeString(full) + "</code>\n")
	sb.WriteString("<b>Prefix sent:</b> <code>" + html.EscapeString(full[:5]) + "</code>\n\n")

	if count > 0 {
		sb.WriteString("<b>Status:</b> PWNED\n")
		sb.WriteString("<b>Times seen in breaches:</b> <code>" + strconv.Itoa(count) + "</code>\n\n")
		sb.WriteString("This string has appeared in known data breaches. Do not use it as a password.")
	} else {
		sb.WriteString("<b>Status:</b> SAFE\n")
		sb.WriteString("<b>Times seen in breaches:</b> <code>0</code>\n\n")
		sb.WriteString("This string was not found in the Have I Been Pwned breach corpus.")
	}

	out := sb.String()
	if status != nil {
		status.Edit(out)
		return nil
	}
	_, err = m.Reply(out)
	return err
}

func registerPwnedCheckHandlers() {
	c := Client
	c.On("cmd:pwned", PwnedCheckHandler)
}

func init() {
	QueueHandlerRegistration(registerPwnedCheckHandlers)
}
