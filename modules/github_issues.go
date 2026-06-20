package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type ghIssueLabel struct {
	Name string `json:"name"`
}

type ghIssueUser struct {
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
}

type ghIssuePR struct {
	HTMLURL string `json:"html_url"`
}

type ghIssue struct {
	Number      int            `json:"number"`
	Title       string         `json:"title"`
	HTMLURL     string         `json:"html_url"`
	State       string         `json:"state"`
	StateReason string         `json:"state_reason"`
	Comments    int            `json:"comments"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
	ClosedAt    string         `json:"closed_at"`
	Body        string         `json:"body"`
	User        ghIssueUser    `json:"user"`
	Assignees   []ghIssueUser  `json:"assignees"`
	Labels      []ghIssueLabel `json:"labels"`
	PullRequest *ghIssuePR     `json:"pull_request"`
}

func ghParseRepoSlug(s string) (string, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "https://github.com/")
	s = strings.TrimPrefix(s, "http://github.com/")
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	if !strings.Contains(s, "/") || strings.Count(s, "/") != 1 {
		return "", false
	}
	parts := strings.Split(s, "/")
	if parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return s, true
}

func ghFetchIssuesList(full string) ([]ghIssue, error) {
	body, status, err := ghDoRequest("https://api.github.com/repos/"+full+"/issues?state=open&per_page=15&sort=created&direction=desc", "")
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, fmt.Errorf("repo not found")
	}
	if status != 200 {
		return nil, fmt.Errorf("api status %d", status)
	}
	var out []ghIssue
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func ghFetchIssueOne(full string, num int) (*ghIssue, error) {
	body, status, err := ghDoRequest(fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", full, num), "")
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, fmt.Errorf("issue not found")
	}
	if status != 200 {
		return nil, fmt.Errorf("api status %d", status)
	}
	var out ghIssue
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func ghIssueStateBadge(i *ghIssue) string {
	if i.State == "closed" {
		if i.StateReason == "not_planned" {
			return "closed (not planned)"
		}
		if i.StateReason == "completed" {
			return "closed (completed)"
		}
		return "closed"
	}
	return "open"
}

func ghBuildIssueListCaption(full string, issues []ghIssue) string {
	var sb strings.Builder
	sb.WriteString("<b>Latest open issues in <a href=\"https://github.com/" + html.EscapeString(full) + "/issues\">" + html.EscapeString(full) + "</a></b>\n\n")
	shown := 0
	for _, it := range issues {
		if it.PullRequest != nil {
			continue
		}
		if shown >= 5 {
			break
		}
		shown++
		title := ghTrim(it.Title, 110)
		sb.WriteString(fmt.Sprintf("<b>#%d</b> <a href=\"%s\">%s</a>\n", it.Number, html.EscapeString(it.HTMLURL), html.EscapeString(title)))
		meta := fmt.Sprintf("by @%s &middot; %s &middot; %d comments", html.EscapeString(it.User.Login), html.EscapeString(ghFormatRelative(it.CreatedAt)), it.Comments)
		sb.WriteString("<i>" + meta + "</i>\n")
		if len(it.Labels) > 0 {
			tags := make([]string, 0, len(it.Labels))
			for _, l := range it.Labels {
				if l.Name == "" {
					continue
				}
				tags = append(tags, "<code>"+html.EscapeString(l.Name)+"</code>")
				if len(tags) >= 4 {
					break
				}
			}
			if len(tags) > 0 {
				sb.WriteString(strings.Join(tags, " ") + "\n")
			}
		}
		sb.WriteString("\n")
	}
	if shown == 0 {
		sb.WriteString("<i>No open issues found.</i>\n")
	}
	return sb.String()
}

func ghBuildIssueDetailCaption(full string, i *ghIssue) string {
	var sb strings.Builder
	kind := "Issue"
	if i.PullRequest != nil {
		kind = "Pull Request"
	}
	sb.WriteString(fmt.Sprintf("<b>%s #%d</b> &middot; <a href=\"https://github.com/%s\">%s</a>\n", kind, i.Number, html.EscapeString(full), html.EscapeString(full)))
	sb.WriteString("<b><a href=\"" + html.EscapeString(i.HTMLURL) + "\">" + html.EscapeString(ghTrim(i.Title, 220)) + "</a></b>\n\n")
	sb.WriteString("<b>State:</b> <code>" + html.EscapeString(ghIssueStateBadge(i)) + "</code>\n")
	if i.User.Login != "" {
		sb.WriteString("<b>Author:</b> <a href=\"" + html.EscapeString(i.User.HTMLURL) + "\">@" + html.EscapeString(i.User.Login) + "</a>\n")
	}
	sb.WriteString(fmt.Sprintf("<b>Comments:</b> <code>%d</code>\n", i.Comments))
	if i.CreatedAt != "" {
		sb.WriteString("<b>Created:</b> " + html.EscapeString(ghFormatDate(i.CreatedAt)) + " <i>(" + html.EscapeString(ghFormatRelative(i.CreatedAt)) + ")</i>\n")
	}
	if i.UpdatedAt != "" {
		sb.WriteString("<b>Updated:</b> " + html.EscapeString(ghFormatRelative(i.UpdatedAt)) + "\n")
	}
	if i.State == "closed" && i.ClosedAt != "" {
		sb.WriteString("<b>Closed:</b> " + html.EscapeString(ghFormatRelative(i.ClosedAt)) + "\n")
	}
	if len(i.Assignees) > 0 {
		names := make([]string, 0, len(i.Assignees))
		for _, a := range i.Assignees {
			if a.Login == "" {
				continue
			}
			names = append(names, "@"+html.EscapeString(a.Login))
			if len(names) >= 5 {
				break
			}
		}
		if len(names) > 0 {
			sb.WriteString("<b>Assignees:</b> " + strings.Join(names, ", ") + "\n")
		}
	}
	if len(i.Labels) > 0 {
		tags := make([]string, 0, len(i.Labels))
		for _, l := range i.Labels {
			if l.Name == "" {
				continue
			}
			tags = append(tags, "<code>"+html.EscapeString(l.Name)+"</code>")
			if len(tags) >= 8 {
				break
			}
		}
		if len(tags) > 0 {
			sb.WriteString("<b>Labels:</b> " + strings.Join(tags, " ") + "\n")
		}
	}
	body := strings.TrimSpace(i.Body)
	if body != "" {
		sb.WriteString("\n<b>Description:</b>\n")
		sb.WriteString("<blockquote>" + html.EscapeString(ghTrim(body, 600)) + "</blockquote>")
	}
	return sb.String()
}

func GithubIssuesListHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/ghissues &lt;owner/repo&gt;</code>")
		return err
	}
	full, ok := ghParseRepoSlug(strings.Fields(arg)[0])
	if !ok {
		_, err := m.Reply("Provide as <code>owner/repo</code>.")
		return err
	}

	status, _ := m.Reply("Fetching issues from <code>" + html.EscapeString(full) + "</code>...")

	issues, err := ghFetchIssuesList(full)
	if err != nil {
		msg := "Failed: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	caption := ghBuildIssueListCaption(full, issues)
	b := tg.Button
	keyb := tg.NewKeyboard().AddRow(b.URL("All issues", "https://github.com/"+full+"/issues"))

	if status != nil {
		status.Edit(caption, &tg.SendOptions{
			ReplyMarkup: keyb.Build(),
			LinkPreview: false,
		})
		return nil
	}
	_, e := m.Reply(caption, &tg.SendOptions{
		ReplyMarkup: keyb.Build(),
		LinkPreview: false,
	})
	return e
}

func GithubIssueDetailHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	fields := strings.Fields(arg)
	if len(fields) < 2 {
		_, err := m.Reply("<b>Usage:</b> <code>/ghissue &lt;owner/repo&gt; &lt;number&gt;</code>")
		return err
	}
	full, ok := ghParseRepoSlug(fields[0])
	if !ok {
		_, err := m.Reply("Provide as <code>owner/repo</code>.")
		return err
	}
	numStr := strings.TrimPrefix(fields[1], "#")
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		_, e := m.Reply("Issue number must be a positive integer.")
		return e
	}

	status, _ := m.Reply(fmt.Sprintf("Fetching <code>%s#%d</code>...", html.EscapeString(full), num))

	issue, err := ghFetchIssueOne(full, num)
	if err != nil {
		msg := "Failed: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	caption := ghBuildIssueDetailCaption(full, issue)
	label := "Open issue"
	if issue.PullRequest != nil {
		label = "Open PR"
	}
	b := tg.Button
	keyb := tg.NewKeyboard().AddRow(b.URL(label, issue.HTMLURL))

	if status != nil {
		status.Edit(caption, &tg.SendOptions{
			ReplyMarkup: keyb.Build(),
			LinkPreview: false,
		})
		return nil
	}
	_, e := m.Reply(caption, &tg.SendOptions{
		ReplyMarkup: keyb.Build(),
		LinkPreview: false,
	})
	return e
}

func registerGithubIssuesHandlers() {
	c := Client
	c.On("cmd:ghissues", GithubIssuesListHandler)
	c.On("cmd:ghissue", GithubIssueDetailHandler)
}

func init() { QueueHandlerRegistration(registerGithubIssuesHandlers) }
