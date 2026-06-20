package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var ghrelHTTPClient = &http.Client{Timeout: 30 * time.Second}

type ghrelAuthor struct {
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
}

type ghrelAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	DownloadCount      int    `json:"download_count"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

type ghrelRelease struct {
	HTMLURL     string       `json:"html_url"`
	TagName     string       `json:"tag_name"`
	Name        string       `json:"name"`
	Body        string       `json:"body"`
	Draft       bool         `json:"draft"`
	Prerelease  bool         `json:"prerelease"`
	CreatedAt   string       `json:"created_at"`
	PublishedAt string       `json:"published_at"`
	Author      ghrelAuthor  `json:"author"`
	Assets      []ghrelAsset `json:"assets"`
}

type ghrelError struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

func ghrelFetchLatest(full string) (*ghrelRelease, int, error) {
	url := "https://api.github.com/repos/" + full + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := ghrelHTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode != http.StatusOK {
		var ge ghrelError
		if jerr := json.Unmarshal(body, &ge); jerr == nil && ge.Message != "" {
			return nil, resp.StatusCode, fmt.Errorf("%s", ge.Message)
		}
		return nil, resp.StatusCode, fmt.Errorf("status %d", resp.StatusCode)
	}
	var r ghrelRelease
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, resp.StatusCode, err
	}
	return &r, resp.StatusCode, nil
}

func ghrelHumanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(n)/float64(div), units[exp])
}

func ghrelFormatDate(s string) string {
	if s == "" {
		return "unknown"
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.UTC().Format("Jan 02, 2006 15:04 UTC")
}

func ghrelTrim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func ghrelBuildCaption(full string, r *ghrelRelease) string {
	var sb strings.Builder
	title := r.Name
	if title == "" {
		title = r.TagName
	}
	sb.WriteString("<b>Latest Release - <a href=\"https://github.com/" + html.EscapeString(full) + "\">" + html.EscapeString(full) + "</a></b>\n\n")
	sb.WriteString("<b>Release:</b> <a href=\"" + html.EscapeString(r.HTMLURL) + "\">" + html.EscapeString(title) + "</a>\n")
	if r.TagName != "" {
		sb.WriteString("<b>Tag:</b> <code>" + html.EscapeString(r.TagName) + "</code>\n")
	}
	flags := []string{}
	if r.Prerelease {
		flags = append(flags, "prerelease")
	}
	if r.Draft {
		flags = append(flags, "draft")
	}
	if len(flags) > 0 {
		sb.WriteString("<b>Type:</b> <code>" + html.EscapeString(strings.Join(flags, ", ")) + "</code>\n")
	}
	if r.Author.Login != "" {
		sb.WriteString("<b>Author:</b> <a href=\"" + html.EscapeString(r.Author.HTMLURL) + "\">@" + html.EscapeString(r.Author.Login) + "</a>\n")
	}
	sb.WriteString("<b>Published:</b> " + html.EscapeString(ghrelFormatDate(r.PublishedAt)) + "\n")

	if len(r.Assets) > 0 {
		sb.WriteString("\n<b>Assets (" + fmt.Sprintf("%d", len(r.Assets)) + "):</b>\n")
		shown := r.Assets
		if len(shown) > 8 {
			shown = shown[:8]
		}
		for i, a := range shown {
			sb.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a> <i>(%s, %d downloads)</i>\n",
				i+1,
				html.EscapeString(a.BrowserDownloadURL),
				html.EscapeString(a.Name),
				html.EscapeString(ghrelHumanSize(a.Size)),
				a.DownloadCount,
			))
		}
		if len(r.Assets) > 8 {
			sb.WriteString(fmt.Sprintf("<i>...and %d more</i>\n", len(r.Assets)-8))
		}
	} else {
		sb.WriteString("\n<b>Assets:</b> <i>none</i>\n")
	}

	body := strings.TrimSpace(r.Body)
	if body != "" {
		sb.WriteString("\n<b>Notes:</b>\n<blockquote>" + html.EscapeString(ghrelTrim(body, 500)) + "</blockquote>")
	}

	return sb.String()
}

func GithubReleaseHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/ghrelease &lt;owner/repo&gt;</code>")
		return err
	}
	full := strings.Fields(arg)[0]
	full = strings.TrimPrefix(full, "https://github.com/")
	full = strings.TrimPrefix(full, "http://github.com/")
	full = strings.TrimSuffix(full, "/")
	full = strings.TrimSuffix(full, ".git")
	if !strings.Contains(full, "/") || strings.Count(full, "/") != 1 {
		_, err := m.Reply("Provide as <code>owner/repo</code>.")
		return err
	}

	status, _ := m.Reply("Fetching latest release of <code>" + html.EscapeString(full) + "</code>...")

	r, code, err := ghrelFetchLatest(full)
	if err != nil {
		msg := "<b>Failed:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if code == http.StatusNotFound {
			msg = "<b>No release found</b> for <code>" + html.EscapeString(full) + "</code>."
		}
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	caption := ghrelBuildCaption(full, r)

	b := tg.Button
	keyb := tg.NewKeyboard().AddRow(b.URL("View Release", r.HTMLURL))

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

func registerGithubReleaseHandlers() {
	c := Client
	c.On("cmd:ghrelease", GithubReleaseHandler)
}

func init() {
	QueueHandlerRegistration(registerGithubReleaseHandlers)
}
