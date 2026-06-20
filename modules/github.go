package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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
	sb.WriteString("<b>" + html.EscapeString(displayName) + "</b>")
	if u.Login != "" {
		sb.WriteString(" (<a href=\"" + html.EscapeString(u.HTMLURL) + "\">@" + html.EscapeString(u.Login) + "</a>)")
	}
	sb.WriteString("\n")
	if u.Bio != "" {
		sb.WriteString("<i>" + html.EscapeString(ghTrim(u.Bio, 240)) + "</i>\n")
	}
	sb.WriteString("\n")
	if u.Location != "" {
		sb.WriteString("<b>Location:</b> " + html.EscapeString(u.Location) + "\n")
	}
	if u.Company != "" {
		sb.WriteString("<b>Company:</b> " + html.EscapeString(u.Company) + "\n")
	}
	if u.Blog != "" {
		blog := u.Blog
		if !strings.HasPrefix(blog, "http") {
			blog = "https://" + blog
		}
		sb.WriteString("<b>Blog:</b> <a href=\"" + html.EscapeString(blog) + "\">" + html.EscapeString(u.Blog) + "</a>\n")
	}
	if u.TwitterUser != "" {
		sb.WriteString("<b>Twitter:</b> @" + html.EscapeString(u.TwitterUser) + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("<b>Followers:</b> <code>%d</code>  <b>Following:</b> <code>%d</code>\n", u.Followers, u.Following))
	sb.WriteString(fmt.Sprintf("<b>Repos:</b> <code>%d</code>  <b>Gists:</b> <code>%d</code>\n", u.PublicRepos, u.PublicGists))
	sb.WriteString("\n")
	if u.CreatedAt != "" {
		sb.WriteString("<b>Joined:</b> " + html.EscapeString(ghFormatDate(u.CreatedAt)) + "\n")
	}
	if u.UpdatedAt != "" {
		sb.WriteString("<b>Updated:</b> " + html.EscapeString(ghFormatRelative(u.UpdatedAt)) + "\n")
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
	sb.WriteString("<b><a href=\"" + html.EscapeString(r.HTMLURL) + "\">" + html.EscapeString(r.FullName) + "</a></b>\n")
	if r.Description != "" {
		sb.WriteString("<i>" + html.EscapeString(ghTrim(r.Description, 280)) + "</i>\n")
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
		sb.WriteString("<b>Language:</b> " + html.EscapeString(r.Language) + "\n")
	}
	if r.DefaultBranch != "" {
		sb.WriteString("<b>Default Branch:</b> <code>" + html.EscapeString(r.DefaultBranch) + "</code>\n")
	}
	if r.License != nil && r.License.Name != "" {
		sb.WriteString("<b>License:</b> " + html.EscapeString(r.License.Name) + "\n")
	}
	if r.Homepage != "" {
		sb.WriteString("<b>Homepage:</b> <a href=\"" + html.EscapeString(r.Homepage) + "\">" + html.EscapeString(r.Homepage) + "</a>\n")
	}
	if r.UpdatedAt != "" {
		sb.WriteString("<b>Updated:</b> " + html.EscapeString(ghFormatRelative(r.UpdatedAt)) + "\n")
	}
	if r.PushedAt != "" {
		sb.WriteString("<b>Last Push:</b> " + html.EscapeString(ghFormatRelative(r.PushedAt)) + "\n")
	}

	if len(r.Topics) > 0 {
		tags := make([]string, 0, len(r.Topics))
		for _, t := range r.Topics {
			tags = append(tags, "<code>#"+html.EscapeString(t)+"</code>")
		}
		sb.WriteString("\n<b>Topics:</b> " + strings.Join(tags, " ") + "\n")
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
	c := Client
	c.On("cmd:gh", GithubUserHandler)
	c.On("cmd:repo", GithubRepoHandler)
}

func init() { QueueHandlerRegistration(registerGithubHandlers) }
