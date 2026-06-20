package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type ghContribDay struct {
	Date              string `json:"date"`
	ContributionCount int    `json:"contributionCount"`
	ContributionLevel string `json:"contributionLevel"`
	Color             string `json:"color"`
}

type ghContribResponse struct {
	Contributions      [][]ghContribDay `json:"contributions"`
	TotalContributions int              `json:"totalContributions"`
}

var ghContribClient = &http.Client{Timeout: 30 * time.Second}

func ghContribValidUser(u string) bool {
	if u == "" || len(u) > 39 {
		return false
	}
	for _, r := range u {
		if !(r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	if strings.HasPrefix(u, "-") || strings.HasSuffix(u, "-") {
		return false
	}
	return true
}

func ghContribFlatten(weeks [][]ghContribDay) []ghContribDay {
	out := make([]ghContribDay, 0, len(weeks)*7)
	for _, w := range weeks {
		out = append(out, w...)
	}
	return out
}

func ghContribStreaks(days []ghContribDay) (current, longest int, lastActive string, activeDays int) {
	run := 0
	for _, d := range days {
		if d.ContributionCount > 0 {
			run++
			activeDays++
			lastActive = d.Date
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	cur := 0
	for i := len(days) - 1; i >= 0; i-- {
		d := days[i]
		if d.ContributionCount > 0 {
			cur++
			continue
		}
		if cur == 0 && (d.Date == today || d.Date == yesterday) {
			continue
		}
		break
	}
	current = cur
	return
}

func ghContribFetch(user string) (*ghContribResponse, int, error) {
	endpoint := "https://github-contributions-api.deno.dev/" + user + ".json"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := ghContribClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out ghContribResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, resp.StatusCode, err
	}
	return &out, resp.StatusCode, nil
}

func GhContribHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/ghcontrib &lt;user&gt;</code>")
		return nil
	}
	user := strings.TrimPrefix(strings.Fields(arg)[0], "@")
	user = strings.TrimSpace(user)
	if !ghContribValidUser(user) {
		m.Reply("<b>Invalid GitHub username.</b>")
		return nil
	}
	status, _ := m.Reply("<i>Fetching contributions for <code>" + html.EscapeString(user) + "</code>...</i>")
	data, code, err := ghContribFetch(user)
	if err != nil {
		msg := ""
		switch code {
		case http.StatusNotFound, http.StatusBadRequest:
			msg = "<b>User not found:</b> <code>" + html.EscapeString(user) + "</code>"
		default:
			msg = "<b>Error:</b> failed to fetch contributions (<code>" + html.EscapeString(err.Error()) + "</code>)"
		}
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	days := ghContribFlatten(data.Contributions)
	if len(days) == 0 {
		msg := "<b>No contribution data available for</b> <code>" + html.EscapeString(user) + "</code>."
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	current, longest, lastActive, activeDays := ghContribStreaks(days)
	first := days[0].Date
	last := days[len(days)-1].Date

	var sb strings.Builder
	sb.WriteString("<b>GitHub Contributions:</b> <a href=\"https://github.com/")
	sb.WriteString(html.EscapeString(user))
	sb.WriteString("\">")
	sb.WriteString(html.EscapeString(user))
	sb.WriteString("</a>\n")
	sb.WriteString("\n<b>Total (past year):</b> <code>")
	sb.WriteString(fmt.Sprintf("%d", data.TotalContributions))
	sb.WriteString("</code>")
	sb.WriteString("\n<b>Active Days:</b> <code>")
	sb.WriteString(fmt.Sprintf("%d / %d", activeDays, len(days)))
	sb.WriteString("</code>")
	sb.WriteString("\n<b>Current Streak:</b> <code>")
	sb.WriteString(fmt.Sprintf("%d days", current))
	sb.WriteString("</code>")
	sb.WriteString("\n<b>Longest Streak:</b> <code>")
	sb.WriteString(fmt.Sprintf("%d days", longest))
	sb.WriteString("</code>")
	if lastActive != "" {
		sb.WriteString("\n<b>Last Active:</b> <code>")
		sb.WriteString(html.EscapeString(lastActive))
		sb.WriteString("</code>")
	}
	sb.WriteString("\n<b>Range:</b> <code>")
	sb.WriteString(html.EscapeString(first))
	sb.WriteString(" → ")
	sb.WriteString(html.EscapeString(last))
	sb.WriteString("</code>")

	out := sb.String()
	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func registerGhContribHandlers() {
	c := Client
	c.On("cmd:ghcontrib", GhContribHandler)
}

func init() { QueueHandlerRegistration(registerGhContribHandlers) }
