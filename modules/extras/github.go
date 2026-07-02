package extras

import (
	"encoding/json"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"html"
	"io"
	modules "main/modules"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// === from github.go ===
type ghUser struct {
	Login       string `json:"login"`
	ID          int64  `json:"id"`
	AvatarURL   string `json:"avatar_url"`
	HTMLURL     string `json:"html_url"`
	Name        string `json:"name"`
	Company     string `json:"company"`
	Blog        string `json:"blog"`
	Location    string `json:"location"`
	Email       string `json:"email"`
	Bio         string `json:"bio"`
	TwitterUser string `json:"twitter_username"`
	PublicRepos int    `json:"public_repos"`
	PublicGists int    `json:"public_gists"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ghLicense struct {
	Name   string `json:"name"`
	SPDXID string `json:"spdx_id"`
}

type ghOwner struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type ghRepo struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	FullName        string     `json:"full_name"`
	Owner           ghOwner    `json:"owner"`
	HTMLURL         string     `json:"html_url"`
	Description     string     `json:"description"`
	Fork            bool       `json:"fork"`
	CreatedAt       string     `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
	PushedAt        string     `json:"pushed_at"`
	Homepage        string     `json:"homepage"`
	Size            int        `json:"size"`
	StargazersCount int        `json:"stargazers_count"`
	WatchersCount   int        `json:"watchers_count"`
	Language        string     `json:"language"`
	ForksCount      int        `json:"forks_count"`
	OpenIssuesCount int        `json:"open_issues_count"`
	DefaultBranch   string     `json:"default_branch"`
	License         *ghLicense `json:"license"`
	Topics          []string   `json:"topics"`
	Archived        bool       `json:"archived"`
	Disabled        bool       `json:"disabled"`
}

type ghCommitAuthor struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

type ghCommitInner struct {
	Author  ghCommitAuthor `json:"author"`
	Message string         `json:"message"`
}

type ghCommit struct {
	SHA    string        `json:"sha"`
	Commit ghCommitInner `json:"commit"`
	HTMLURL string        `json:"html_url"`
}

type ghSearchResult struct {
	TotalCount int `json:"total_count"`
}

type ghUserCache struct {
	user      *ghUser
	owned     []ghRepo
	starred   []ghRepo
	fetchedAt time.Time
}

type ghRepoCache struct {
	repo      *ghRepo
	commits   []ghCommit
	openPRs   int
	fetchedAt time.Time
}

var (
	ghUserCacheMap = make(map[string]*ghUserCache)
	ghUserCacheMu  sync.Mutex
	ghRepoCacheMap = make(map[string]*ghRepoCache)
	ghRepoCacheMu  sync.Mutex
	ghCacheTTL     = 10 * time.Minute
)

func ghDoRequest(url string, accept string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	if accept == "" {
		accept = "application/vnd.github+json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func ghFetchUser(name string) (*ghUser, error) {
	body, status, err := ghDoRequest("https://api.github.com/users/"+name, "")
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, fmt.Errorf("user not found")
	}
	if status != 200 {
		return nil, fmt.Errorf("api status %d", status)
	}
	var u ghUser
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func ghFetchOwnedRepos(name string) ([]ghRepo, error) {
	body, status, err := ghDoRequest("https://api.github.com/users/"+name+"/repos?per_page=100&type=owner&sort=updated", "application/vnd.github.mercy-preview+json")
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("api status %d", status)
	}
	var r []ghRepo
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func ghFetchStarredRepos(name string) ([]ghRepo, error) {
	body, status, err := ghDoRequest("https://api.github.com/users/"+name+"/starred?per_page=3", "application/vnd.github.mercy-preview+json")
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("api status %d", status)
	}
	var r []ghRepo
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func ghFetchRepo(full string) (*ghRepo, error) {
	body, status, err := ghDoRequest("https://api.github.com/repos/"+full, "application/vnd.github.mercy-preview+json")
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, fmt.Errorf("repo not found")
	}
	if status != 200 {
		return nil, fmt.Errorf("api status %d", status)
	}
	var r ghRepo
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func ghFetchCommits(full string) ([]ghCommit, error) {
	body, status, err := ghDoRequest("https://api.github.com/repos/"+full+"/commits?per_page=3", "")
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("api status %d", status)
	}
	var c []ghCommit
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, err
	}
	return c, nil
}

func ghFetchOpenPRCount(full string) (int, error) {
	q := "repo:" + full + "+is:pr+is:open"
	body, status, err := ghDoRequest("https://api.github.com/search/issues?q="+q+"&per_page=1", "")
	if err != nil {
		return 0, err
	}
	if status != 200 {
		return 0, fmt.Errorf("api status %d", status)
	}
	var r ghSearchResult
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, err
	}
	return r.TotalCount, nil
}

func ghDownloadImage(url string, name string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty url")
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("gh_avatar_%s_%d.jpg", name, time.Now().UnixNano()))
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return tmp, nil
}

func ghFormatDate(s string) string {
	if s == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.UTC().Format("Jan 02, 2006")
}

func ghFormatRelative(s string) string {
	if s == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 30*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	if d < 365*24*time.Hour {
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	}
	return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
}

func ghTrim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func ghGetUserCached(name string) (*ghUserCache, error) {
	key := strings.ToLower(name)
	ghUserCacheMu.Lock()
	if c, ok := ghUserCacheMap[key]; ok && time.Since(c.fetchedAt) < ghCacheTTL {
		ghUserCacheMu.Unlock()
		return c, nil
	}
	ghUserCacheMu.Unlock()

	u, err := ghFetchUser(name)
	if err != nil {
		return nil, err
	}
	owned, _ := ghFetchOwnedRepos(name)
	starred, _ := ghFetchStarredRepos(name)

	sort.SliceStable(owned, func(i, j int) bool {
		return owned[i].StargazersCount > owned[j].StargazersCount
	})
	if len(owned) > 3 {
		owned = owned[:3]
	}
	if len(starred) > 3 {
		starred = starred[:3]
	}

	c := &ghUserCache{user: u, owned: owned, starred: starred, fetchedAt: time.Now()}
	ghUserCacheMu.Lock()
	ghUserCacheMap[key] = c
	ghUserCacheMu.Unlock()
	return c, nil
}

func ghGetRepoCached(full string) (*ghRepoCache, error) {
	key := strings.ToLower(full)
	ghRepoCacheMu.Lock()
	if c, ok := ghRepoCacheMap[key]; ok && time.Since(c.fetchedAt) < ghCacheTTL {
		ghRepoCacheMu.Unlock()
		return c, nil
	}
	ghRepoCacheMu.Unlock()

	r, err := ghFetchRepo(full)
	if err != nil {
		return nil, err
	}
	commits, _ := ghFetchCommits(full)
	prs, _ := ghFetchOpenPRCount(full)

	c := &ghRepoCache{repo: r, commits: commits, openPRs: prs, fetchedAt: time.Now()}
	ghRepoCacheMu.Lock()
	ghRepoCacheMap[key] = c
	ghRepoCacheMu.Unlock()
	return c, nil
}

func ghBuildUserCaption(c *ghUserCache) string {
	u := c.user
	var sb strings.Builder
	displayName := u.Name
	if displayName == "" {
		displayName = u.Login
	}
	sb.WriteString("<b>")
	sb.WriteString(html.EscapeString(displayName))
	sb.WriteString("</b>")
	if u.Login != "" {
		sb.WriteString(" (<a href=\"")
		sb.WriteString(html.EscapeString(u.HTMLURL))
		sb.WriteString("\">@")
		sb.WriteString(html.EscapeString(u.Login))
		sb.WriteString("</a>)")
	}
	sb.WriteString("\n")
	if u.Bio != "" {
		sb.WriteString("<i>")
		sb.WriteString(html.EscapeString(ghTrim(u.Bio, 240)))
		sb.WriteString("</i>\n")
	}
	sb.WriteString("\n")
	if u.Location != "" {
		sb.WriteString("<b>Location:</b> ")
		sb.WriteString(html.EscapeString(u.Location))
		sb.WriteString("\n")
	}
	if u.Company != "" {
		sb.WriteString("<b>Company:</b> ")
		sb.WriteString(html.EscapeString(u.Company))
		sb.WriteString("\n")
	}
	if u.Blog != "" {
		blog := u.Blog
		if !strings.HasPrefix(blog, "http") {
			blog = "https://" + blog
		}
		sb.WriteString("<b>Blog:</b> <a href=\"")
		sb.WriteString(html.EscapeString(blog))
		sb.WriteString("\">")
		sb.WriteString(html.EscapeString(u.Blog))
		sb.WriteString("</a>\n")
	}
	if u.TwitterUser != "" {
		sb.WriteString("<b>Twitter:</b> @")
		sb.WriteString(html.EscapeString(u.TwitterUser))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("<b>Followers:</b> <code>%d</code>  <b>Following:</b> <code>%d</code>\n", u.Followers, u.Following))
	sb.WriteString(fmt.Sprintf("<b>Repos:</b> <code>%d</code>  <b>Gists:</b> <code>%d</code>\n", u.PublicRepos, u.PublicGists))
	sb.WriteString("\n")
	if u.CreatedAt != "" {
		sb.WriteString("<b>Joined:</b> ")
		sb.WriteString(html.EscapeString(ghFormatDate(u.CreatedAt)))
		sb.WriteString("\n")
	}
	if u.UpdatedAt != "" {
		sb.WriteString("<b>Updated:</b> ")
		sb.WriteString(html.EscapeString(ghFormatRelative(u.UpdatedAt)))
		sb.WriteString("\n")
	}

	if len(c.owned) > 0 {
		sb.WriteString("\n<b>Top Repos:</b>\n")
		for i, r := range c.owned {
			lang := ""
			if r.Language != "" {
				lang = " <i>(" + html.EscapeString(r.Language) + ")</i>"
			}
			sb.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a> [%d]%s\n", i+1, html.EscapeString(r.HTMLURL), html.EscapeString(r.Name), r.StargazersCount, lang))
		}
	}

	if len(c.starred) > 0 {
		sb.WriteString("\n<b>Recently Starred:</b>\n")
		for i, r := range c.starred {
			sb.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a> [%d]\n", i+1, html.EscapeString(r.HTMLURL), html.EscapeString(r.FullName), r.StargazersCount))
		}
	}

	return sb.String()
}

func ghBuildRepoCaption(c *ghRepoCache) string {
	r := c.repo
	var sb strings.Builder
	sb.WriteString("<b><a href=\"")
	sb.WriteString(html.EscapeString(r.HTMLURL))
	sb.WriteString("\">")
	sb.WriteString(html.EscapeString(r.FullName))
	sb.WriteString("</a></b>\n")
	if r.Description != "" {
		sb.WriteString("<i>")
		sb.WriteString(html.EscapeString(ghTrim(r.Description, 280)))
		sb.WriteString("</i>\n")
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("<b>Stars:</b> <code>%d</code>  <b>Forks:</b> <code>%d</code>  <b>Watchers:</b> <code>%d</code>\n", r.StargazersCount, r.ForksCount, r.WatchersCount))
	issues := r.OpenIssuesCount - c.openPRs
	if issues < 0 {
		issues = 0
	}
	sb.WriteString(fmt.Sprintf("<b>Open Issues:</b> <code>%d</code>  <b>Open PRs:</b> <code>%d</code>\n", issues, c.openPRs))
	sb.WriteString("\n")
	if r.Language != "" {
		sb.WriteString("<b>Language:</b> ")
		sb.WriteString(html.EscapeString(r.Language))
		sb.WriteString("\n")
	}
	if r.DefaultBranch != "" {
		sb.WriteString("<b>Default Branch:</b> <code>")
		sb.WriteString(html.EscapeString(r.DefaultBranch))
		sb.WriteString("</code>\n")
	}
	if r.License != nil && r.License.Name != "" {
		sb.WriteString("<b>License:</b> ")
		sb.WriteString(html.EscapeString(r.License.Name))
		sb.WriteString("\n")
	}
	if r.Homepage != "" {
		sb.WriteString("<b>Homepage:</b> <a href=\"")
		sb.WriteString(html.EscapeString(r.Homepage))
		sb.WriteString("\">")
		sb.WriteString(html.EscapeString(r.Homepage))
		sb.WriteString("</a>\n")
	}
	if r.UpdatedAt != "" {
		sb.WriteString("<b>Updated:</b> ")
		sb.WriteString(html.EscapeString(ghFormatRelative(r.UpdatedAt)))
		sb.WriteString("\n")
	}
	if r.PushedAt != "" {
		sb.WriteString("<b>Last Push:</b> ")
		sb.WriteString(html.EscapeString(ghFormatRelative(r.PushedAt)))
		sb.WriteString("\n")
	}

	if len(r.Topics) > 0 {
		tags := make([]string, 0, len(r.Topics))
		for _, t := range r.Topics {
			tags = append(tags, "<code>#"+html.EscapeString(t)+"</code>")
		}
		sb.WriteString("\n<b>Topics:</b> ")
		sb.WriteString(strings.Join(tags, " "))
		sb.WriteString("\n")
	}

	if len(c.commits) > 0 {
		sb.WriteString("\n<b>Recent Commits:</b>\n")
		for _, cm := range c.commits {
			sha := cm.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			msg := strings.SplitN(cm.Commit.Message, "\n", 2)[0]
			rel := ghFormatRelative(cm.Commit.Author.Date)
			sb.WriteString(fmt.Sprintf("- <a href=\"%s\"><code>%s</code></a> %s <i>(%s)</i>\n", html.EscapeString(cm.HTMLURL), html.EscapeString(sha), html.EscapeString(ghTrim(msg, 80)), html.EscapeString(rel)))
		}
	}

	return sb.String()
}

func GithubUserHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/gh &lt;username&gt;</code>")
		return err
	}
	name := strings.TrimPrefix(strings.Fields(arg)[0], "@")
	name = strings.TrimSpace(name)
	if name == "" {
		_, err := m.Reply("Invalid username.")
		return err
	}

	status, _ := m.Reply("Fetching <code>" + html.EscapeString(name) + "</code> from GitHub...")

	c, err := ghGetUserCached(name)
	if err != nil {
		msg := "Failed: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	caption := ghBuildUserCaption(c)

	b := tg.Button
	keyb := tg.NewKeyboard().AddRow(b.URL("View on GitHub", c.user.HTMLURL))

	avatarPath, derr := ghDownloadImage(c.user.AvatarURL, c.user.Login)
	if derr == nil && avatarPath != "" {
		defer os.Remove(avatarPath)
		_, merr := m.ReplyMedia(avatarPath, &tg.MediaOptions{
			Caption:     caption,
			FileName:    c.user.Login + ".jpg",
			MimeType:    "image/jpeg",
			ReplyMarkup: keyb.Build(),
			LinkPreview: false,
		})
		if merr == nil {
			if status != nil {
				status.Delete()
			}
			return nil
		}
	}

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

func GithubRepoHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/repo &lt;owner/repo&gt;</code>")
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

	status, _ := m.Reply("Fetching <code>" + html.EscapeString(full) + "</code> from GitHub...")

	c, err := ghGetRepoCached(full)
	if err != nil {
		msg := "Failed: <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
			return nil
		}
		_, e := m.Reply(msg)
		return e
	}

	caption := ghBuildRepoCaption(c)

	b := tg.Button
	keyb := tg.NewKeyboard().AddRow(b.URL("Open repo", c.repo.HTMLURL))

	avatarPath, derr := ghDownloadImage(c.repo.Owner.AvatarURL, c.repo.Owner.Login)
	if derr == nil && avatarPath != "" {
		defer os.Remove(avatarPath)
		_, merr := m.ReplyMedia(avatarPath, &tg.MediaOptions{
			Caption:     caption,
			FileName:    c.repo.Owner.Login + ".jpg",
			MimeType:    "image/jpeg",
			ReplyMarkup: keyb.Build(),
			LinkPreview: false,
		})
		if merr == nil {
			if status != nil {
				status.Delete()
			}
			return nil
		}
	}

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

func registerGithubHandlers() {
	c := modules.Client
	c.On("cmd:gh", GithubUserHandler)
	c.On("cmd:repo", GithubRepoHandler)
}

func initFromSrc_github_0_1() { modules.QueueHandlerRegistration(registerGithubHandlers) }
// === from github_contribs.go ===
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
	c := modules.Client
	c.On("cmd:ghcontrib", GhContribHandler)
}

func initFromSrc_github_contribs_1_1() { modules.QueueHandlerRegistration(registerGhContribHandlers) }
// === from github_emoji.go ===
var (
	ghEmojiMu      sync.RWMutex
	ghEmojiCache   map[string]string
	ghEmojiFetched time.Time
)

func loadGithubEmojis() (map[string]string, error) {
	ghEmojiMu.RLock()
	if ghEmojiCache != nil && time.Since(ghEmojiFetched) < 24*time.Hour {
		c := ghEmojiCache
		ghEmojiMu.RUnlock()
		return c, nil
	}
	ghEmojiMu.RUnlock()

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/emojis", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "JuliaBot")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	ghEmojiMu.Lock()
	ghEmojiCache = data
	ghEmojiFetched = time.Now()
	ghEmojiMu.Unlock()
	return data, nil
}

func downloadGithubEmojiImage(url string) ([]byte, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "JuliaBot")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "image/png"
	}
	return body, mime, nil
}

func GitHubEmojiHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/ghemoji &lt;name&gt;</code>")
		return nil
	}
	name := strings.ToLower(strings.TrimSpace(strings.Trim(arg, ":")))
	if name == "" {
		m.Reply("<b>Usage:</b> <code>/ghemoji &lt;name&gt;</code>")
		return nil
	}

	emojis, err := loadGithubEmojis()
	if err != nil {
		m.Reply("Failed to fetch GitHub emojis. Try again later.")
		return nil
	}

	url, ok := emojis[name]
	if !ok {
		var suggestions []string
		for k := range emojis {
			if strings.Contains(k, name) {
				suggestions = append(suggestions, k)
				if len(suggestions) >= 8 {
					break
				}
			}
		}
		if len(suggestions) > 0 {
			var b strings.Builder
			for i, s := range suggestions {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString("<code>")
				b.WriteString(html.EscapeString(s))
				b.WriteString("</code>")
			}
			m.Reply(fmt.Sprintf("<b>No emoji found for:</b> <code>%s</code>\n<b>Did you mean:</b> %s", html.EscapeString(name), b.String()))
			return nil
		}
		m.Reply(fmt.Sprintf("<b>No GitHub emoji found for:</b> <code>%s</code>", html.EscapeString(name)))
		return nil
	}

	caption := fmt.Sprintf("<b>:%s:</b>\n<a href=\"%s\">source</a>", html.EscapeString(name), html.EscapeString(url))

	imgBytes, mime, derr := downloadGithubEmojiImage(url)
	if derr != nil || len(imgBytes) == 0 {
		m.Reply(caption, &tg.SendOptions{LinkPreview: true})
		return nil
	}

	ext := strings.ToLower(path.Ext(strings.SplitN(url, "?", 2)[0]))
	if ext == "" {
		ext = ".png"
	}
	fileName := name + ext

	if _, err := m.ReplyMedia(imgBytes, &tg.MediaOptions{
		Caption:  caption,
		FileName: fileName,
		MimeType: mime,
	}); err != nil {
		m.Reply(caption, &tg.SendOptions{LinkPreview: true})
	}
	return nil
}

func initFromSrc_github_emoji_2_1() { modules.QueueHandlerRegistration(registerGitHubEmojiHandlers) }
func registerGitHubEmojiHandlers() {
	c := modules.Client
	c.On("cmd:ghemoji", GitHubEmojiHandler)
}
// === from github_issues.go ===
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
	sb.WriteString("<b>Latest open issues in <a href=\"https://github.com/")
	sb.WriteString(html.EscapeString(full))
	sb.WriteString("/issues\">")
	sb.WriteString(html.EscapeString(full))
	sb.WriteString("</a></b>\n\n")
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
		sb.WriteString("<i>")
		sb.WriteString(meta)
		sb.WriteString("</i>\n")
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
				sb.WriteString(strings.Join(tags, " "))
				sb.WriteString("\n")
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
	sb.WriteString("<b><a href=\"")
	sb.WriteString(html.EscapeString(i.HTMLURL))
	sb.WriteString("\">")
	sb.WriteString(html.EscapeString(ghTrim(i.Title, 220)))
	sb.WriteString("</a></b>\n\n")
	sb.WriteString("<b>State:</b> <code>")
	sb.WriteString(html.EscapeString(ghIssueStateBadge(i)))
	sb.WriteString("</code>\n")
	if i.User.Login != "" {
		sb.WriteString("<b>Author:</b> <a href=\"")
		sb.WriteString(html.EscapeString(i.User.HTMLURL))
		sb.WriteString("\">@")
		sb.WriteString(html.EscapeString(i.User.Login))
		sb.WriteString("</a>\n")
	}
	sb.WriteString(fmt.Sprintf("<b>Comments:</b> <code>%d</code>\n", i.Comments))
	if i.CreatedAt != "" {
		sb.WriteString("<b>Created:</b> ")
		sb.WriteString(html.EscapeString(ghFormatDate(i.CreatedAt)))
		sb.WriteString(" <i>(")
		sb.WriteString(html.EscapeString(ghFormatRelative(i.CreatedAt)))
		sb.WriteString(")</i>\n")
	}
	if i.UpdatedAt != "" {
		sb.WriteString("<b>Updated:</b> ")
		sb.WriteString(html.EscapeString(ghFormatRelative(i.UpdatedAt)))
		sb.WriteString("\n")
	}
	if i.State == "closed" && i.ClosedAt != "" {
		sb.WriteString("<b>Closed:</b> ")
		sb.WriteString(html.EscapeString(ghFormatRelative(i.ClosedAt)))
		sb.WriteString("\n")
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
			sb.WriteString("<b>Assignees:</b> ")
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString("\n")
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
			sb.WriteString("<b>Labels:</b> ")
			sb.WriteString(strings.Join(tags, " "))
			sb.WriteString("\n")
		}
	}
	body := strings.TrimSpace(i.Body)
	if body != "" {
		sb.WriteString("\n<b>Description:</b>\n")
		sb.WriteString("<blockquote>")
		sb.WriteString(html.EscapeString(ghTrim(body, 600)))
		sb.WriteString("</blockquote>")
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
	c := modules.Client
	c.On("cmd:ghissues", GithubIssuesListHandler)
	c.On("cmd:ghissue", GithubIssueDetailHandler)
}

func initFromSrc_github_issues_3_1() { modules.QueueHandlerRegistration(registerGithubIssuesHandlers) }
// === from github_releases.go ===
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
	sb.WriteString("<b>Latest Release - <a href=\"https://github.com/")
	sb.WriteString(html.EscapeString(full))
	sb.WriteString("\">")
	sb.WriteString(html.EscapeString(full))
	sb.WriteString("</a></b>\n\n")
	sb.WriteString("<b>Release:</b> <a href=\"")
	sb.WriteString(html.EscapeString(r.HTMLURL))
	sb.WriteString("\">")
	sb.WriteString(html.EscapeString(title))
	sb.WriteString("</a>\n")
	if r.TagName != "" {
		sb.WriteString("<b>Tag:</b> <code>")
		sb.WriteString(html.EscapeString(r.TagName))
		sb.WriteString("</code>\n")
	}
	flags := []string{}
	if r.Prerelease {
		flags = append(flags, "prerelease")
	}
	if r.Draft {
		flags = append(flags, "draft")
	}
	if len(flags) > 0 {
		sb.WriteString("<b>Type:</b> <code>")
		sb.WriteString(html.EscapeString(strings.Join(flags, ", ")))
		sb.WriteString("</code>\n")
	}
	if r.Author.Login != "" {
		sb.WriteString("<b>Author:</b> <a href=\"")
		sb.WriteString(html.EscapeString(r.Author.HTMLURL))
		sb.WriteString("\">@")
		sb.WriteString(html.EscapeString(r.Author.Login))
		sb.WriteString("</a>\n")
	}
	sb.WriteString("<b>Published:</b> ")
	sb.WriteString(html.EscapeString(ghrelFormatDate(r.PublishedAt)))
	sb.WriteString("\n")

	if len(r.Assets) > 0 {
		sb.WriteString("\n<b>Assets (")
		sb.WriteString(fmt.Sprintf("%d", len(r.Assets)))
		sb.WriteString("):</b>\n")
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
		sb.WriteString("\n<b>Notes:</b>\n<blockquote>")
		sb.WriteString(html.EscapeString(ghrelTrim(body, 500)))
		sb.WriteString("</blockquote>")
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
	c := modules.Client
	c.On("cmd:ghrelease", GithubReleaseHandler)
}

func initFromSrc_github_releases_4_1() {
	modules.QueueHandlerRegistration(registerGithubReleaseHandlers)
}

func init() {
	initFromSrc_github_0_1()
	initFromSrc_github_contribs_1_1()
	initFromSrc_github_emoji_2_1()
	initFromSrc_github_issues_3_1()
	initFromSrc_github_releases_4_1()
}
