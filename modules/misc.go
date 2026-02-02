package modules

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func PasteBinHandler(m *telegram.NewMessage) error {
	if m.Args() == "" && !m.IsReply() {
		m.Reply("Please provide some text to paste")
		return nil
	}

	content := m.Args()

	if m.IsReply() {
		r, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("Error getting reply message")
			return nil
		}

		if r.IsMedia() {
			if r.Photo() != nil {
				m.Reply("<code>Photo</code> is not supported")
				return nil
			}

			if r.File.Size > 50*1024*200 { // 10MB
				m.Reply("File size too large, max 10MB")
				return nil
			}

			doc, err := r.Download()
			if err != nil {
				m.Reply("Error downloading file")
				return nil
			}

			f, err := os.ReadFile(doc)
			if err != nil {
				m.Reply("Error reading file")
				return nil
			}

			content = string(f)
		} else {
			content = r.Text()
		}
	}

	var (
		url      string
		provider string
		err      error
	)

	url, provider, err = postToPatbin(content)
	if err != nil {
		url, provider, err = postToSpaceBin(content)
		if err != nil {
			m.Reply("Error posting to paste services")
			return nil
		}
	}

	b := telegram.Button

	m.Reply(fmt.Sprintf("<b>Pasted to <a href='%s'>%s</a></b>", url, provider), &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().AddRow(
			b.URL("View Paste", url),
		).Build(),
	})

	return nil
}

// postToPatbin posts content to patbin.fun
func postToPatbin(content string) (string, string, error) {
	payload := fmt.Sprintf(`{"content":%q,"title":"","language":"text","is_public":true}`, content)

	req, err := http.NewRequest("POST", "https://patbin.fun/api/paste", bytes.NewBufferString(payload))
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error reading response: %w", err)
	}

	// Parse the ID from JSON response {"id":"xxx",...}
	// Simple extraction without full JSON parsing
	idStart := bytes.Index(body, []byte(`"id":"`))
	if idStart == -1 {
		return "", "", fmt.Errorf("id not found in response")
	}
	idStart += 6
	idEnd := bytes.Index(body[idStart:], []byte(`"`))
	if idEnd == -1 {
		return "", "", fmt.Errorf("id end not found in response")
	}

	pasteID := string(body[idStart : idStart+idEnd])
	return "https://patbin.fun/" + pasteID, "Patbin", nil
}

func postToSpaceBin(content string) (string, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("content", content); err != nil {
		return "", "", fmt.Errorf("error writing field: %w", err)
	}

	writer.Close()
	req, err := http.NewRequest("POST", "https://spaceb.in/", &body)
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	if location == "" {
		return "", "", fmt.Errorf("location header not found")
	}

	return "https://spaceb.in" + location, "SpaceBin", nil
}

func Gban(m *telegram.NewMessage) error {
	user, reason, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	message, _ := m.Reply("Enforcing global ban...")
	done := 0
	m.Client.Broadcast(context.Background(), nil, func(c telegram.Chat) error {
		_, err := m.Client.EditBanned(c, user, &telegram.BannedOptions{Ban: true})
		if err == nil {
			done++
		}
		return nil
	}, 600)

	message.Edit(fmt.Sprintf("Global ban enforced in %d groups.\nReason: %s", done, reason))
	return nil
}

func Ungban(m *telegram.NewMessage) error {
	user, _, err := GetUserFromContext(m)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}
	message, _ := m.Reply("Removing global ban...")
	done := 0
	m.Client.Broadcast(context.Background(), nil, func(c telegram.Chat) error {
		_, err := m.Client.EditBanned(c, user, &telegram.BannedOptions{Ban: false})
		if err == nil {
			done++
		}
		return nil
	}, 600)
	message.Edit(fmt.Sprintf("Global ban removed in %d groups.", done))
	return nil
}

func mathQuery(query string) (string, error) {
	c := &http.Client{}
	url := "https://evaluate-expression.p.rapidapi.com/?expression=" + url.QueryEscape(query)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-rapidapi-host", "evaluate-expression.p.rapidapi.com")
	req.Header.Add("x-rapidapi-key", "cf9e67ea99mshecc7e1ddb8e93d1p1b9e04jsn3f1bb9103c3f")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) == "" {
		return "", fmt.Errorf("invalid math expression")
	}

	return string(body), nil
}

func MathHandler(m *telegram.NewMessage) error {
	q := m.Args()
	if q == "" {
		m.Reply("please provide a mathematical expression")
		return nil
	}

	result, err := mathQuery(q)
	if err != nil {
		m.Reply("Error: " + err.Error())
		return nil
	}

	m.Reply(fmt.Sprintf("Evaluated: <code>%s</code>", result))
	return nil
}

func NightModeHandler(m *telegram.NewMessage) error {
	if !IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "change_info") {
		m.Reply("You need Change Info rights to use this command")
		return nil
	}

	args := m.Args()
	if args == "" {
		m.Reply("Usage: /nightmode on/off")
		return nil
	}

	var enable bool
	switch strings.ToLower(args) {
	case "on":
		enable = true
	case "off":
		enable = false
	default:
		m.Reply("Usage: /nightmode on/off")
		return nil
	}

	chat, err := m.Client.GetChat(m.ChatID())
	if err != nil {
		m.Reply("Error fetching chat info")
		return nil
	}

	current := chat.DefaultBannedRights
	if current == nil {
		current = &telegram.ChatBannedRights{}
	}

	current.SendMessages = enable

	_, err = m.Client.MessagesEditChatDefaultBannedRights(m.Peer, current)
	if err != nil {
		m.Reply("Failed to toggle night mode: " + err.Error())
		return nil
	}

	if enable {
		m.Reply("Night mode enabled. Messages are restricted.")
	} else {
		m.Reply("Night mode disabled. Messages allowed.")
	}
	return nil
}

// Stats tracking structures
type UserStats struct {
	MessageCount int
	MediaCount   int
	FirstSeen    time.Time
	LastSeen     time.Time
	Username     string
	FirstName    string
}

type ChatStats struct {
	TotalMessages      int
	TotalUsers         map[int64]*UserStats
	DailyMessages      map[string]int           // date -> count
	DailyUserMessages  map[string]map[int64]int // date -> userID -> count
	WeeklyUserMessages map[int64]int            // userID -> count (last 7 days)
	HourlyActivity     map[int]int              // hour (0-23) -> count
	MediaStats         map[string]int           // media type -> count
	LastReset          time.Time
}

var chatStats = make(map[int64]*ChatStats)
var statsLock sync.RWMutex

// TrackMessageStats tracks all message statistics
func TrackMessageStats(m *telegram.NewMessage) error {
	if m.IsPrivate() || m.IsChannel() {
		return nil
	}

	statsLock.Lock()
	defer statsLock.Unlock()

	chatID := m.ChatID()
	userID := m.SenderID()

	// Initialize chat stats if needed
	if chatStats[chatID] == nil {
		chatStats[chatID] = &ChatStats{
			TotalUsers:         make(map[int64]*UserStats),
			DailyMessages:      make(map[string]int),
			DailyUserMessages:  make(map[string]map[int64]int),
			WeeklyUserMessages: make(map[int64]int),
			HourlyActivity:     make(map[int]int),
			MediaStats:         make(map[string]int),
			LastReset:          time.Now(),
		}
	}

	stats := chatStats[chatID]
	now := time.Now()
	today := now.Format("2006-01-02")
	hour := now.Hour()

	// Initialize user stats if needed
	if stats.TotalUsers[userID] == nil {
		user, _ := m.Client.GetUser(userID)
		username := ""
		firstName := "User"
		if user != nil {
			username = user.Username
			firstName = user.FirstName
		}
		stats.TotalUsers[userID] = &UserStats{
			FirstSeen: now,
			Username:  username,
			FirstName: firstName,
		}
	}

	userStats := stats.TotalUsers[userID]
	userStats.MessageCount++
	userStats.LastSeen = now

	// Track media
	if m.IsMedia() {
		userStats.MediaCount++
		if m.Photo() != nil {
			stats.MediaStats["photo"]++
		} else if m.Video() != nil {
			stats.MediaStats["video"]++
		} else if m.Document() != nil {
			stats.MediaStats["document"]++
		} else if m.Sticker() != nil {
			stats.MediaStats["sticker"]++
		} else if m.Voice() != nil {
			stats.MediaStats["voice"]++
		} else {
			stats.MediaStats["other"]++
		}
	}

	// Update counters
	stats.TotalMessages++
	stats.DailyMessages[today]++
	stats.HourlyActivity[hour]++

	// Daily user messages
	if stats.DailyUserMessages[today] == nil {
		stats.DailyUserMessages[today] = make(map[int64]int)
	}
	stats.DailyUserMessages[today][userID]++

	// Weekly user messages (recalculate from daily data)
	weekAgo := now.AddDate(0, 0, -7)
	stats.WeeklyUserMessages[userID] = 0
	for dateStr, userMsgs := range stats.DailyUserMessages {
		date, _ := time.Parse("2006-01-02", dateStr)
		if date.After(weekAgo) {
			stats.WeeklyUserMessages[userID] += userMsgs[userID]
		}
	}

	return nil
}

func StatsHandler(m *telegram.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("Statistics can only be viewed in groups")
		return nil
	}

	statsLock.RLock()
	stats := chatStats[m.ChatID()]
	statsLock.RUnlock()

	if stats == nil || stats.TotalMessages == 0 {
		m.Reply("No statistics available yet. Start chatting to generate data!")
		return nil
	}

	var resp strings.Builder
	resp.WriteString("<b>📊 Group Statistics</b>\n\n")

	// Overall stats
	resp.WriteString(fmt.Sprintf("<b>📈 Overall:</b>\n"))
	resp.WriteString(fmt.Sprintf("  Total Messages: <code>%d</code>\n", stats.TotalMessages))
	resp.WriteString(fmt.Sprintf("  Total Users: <code>%d</code>\n", len(stats.TotalUsers)))
	resp.WriteString(fmt.Sprintf("  Tracking Since: <code>%s</code>\n\n", stats.LastReset.Format("Jan 02, 2006")))

	// Today's stats
	today := time.Now().Format("2006-01-02")
	todayMsgs := stats.DailyMessages[today]
	resp.WriteString(fmt.Sprintf("<b>📅 Today:</b>\n"))
	resp.WriteString(fmt.Sprintf("  Messages: <code>%d</code>\n\n", todayMsgs))

	// Top users today
	type userRank struct {
		UserID   int64
		Name     string
		Count    int
		MediaCnt int
	}

	var todayTop []userRank
	if stats.DailyUserMessages[today] != nil {
		for userID, count := range stats.DailyUserMessages[today] {
			if userStats, ok := stats.TotalUsers[userID]; ok {
				name := userStats.FirstName
				if userStats.Username != "" {
					name = "@" + userStats.Username
				}
				todayTop = append(todayTop, userRank{userID, name, count, userStats.MediaCount})
			}
		}
	}

	sort.Slice(todayTop, func(i, j int) bool {
		return todayTop[i].Count > todayTop[j].Count
	})

	resp.WriteString("<b>🏆 Top Users Today:</b>\n")
	if len(todayTop) == 0 {
		resp.WriteString("  No messages today yet\n\n")
	} else {
		for i := 0; i < len(todayTop) && i < 5; i++ {
			medal := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣"}[i]
			resp.WriteString(fmt.Sprintf("  %s %s - <code>%d</code> msgs\n",
				medal, todayTop[i].Name, todayTop[i].Count))
		}
		resp.WriteString("\n")
	}

	// Top users this week
	var weekTop []userRank
	for userID, count := range stats.WeeklyUserMessages {
		if count > 0 {
			if userStats, ok := stats.TotalUsers[userID]; ok {
				name := userStats.FirstName
				if userStats.Username != "" {
					name = "@" + userStats.Username
				}
				weekTop = append(weekTop, userRank{userID, name, count, userStats.MediaCount})
			}
		}
	}

	sort.Slice(weekTop, func(i, j int) bool {
		return weekTop[i].Count > weekTop[j].Count
	})

	resp.WriteString("<b>📅 Top Users This Week:</b>\n")
	if len(weekTop) == 0 {
		resp.WriteString("  No data for this week\n\n")
	} else {
		for i := 0; i < len(weekTop) && i < 5; i++ {
			medal := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣"}[i]
			resp.WriteString(fmt.Sprintf("  %s %s - <code>%d</code> msgs\n",
				medal, weekTop[i].Name, weekTop[i].Count))
		}
		resp.WriteString("\n")
	}

	// All-time top users
	var allTimeTop []userRank
	for userID, userStats := range stats.TotalUsers {
		name := userStats.FirstName
		if userStats.Username != "" {
			name = "@" + userStats.Username
		}
		allTimeTop = append(allTimeTop, userRank{userID, name, userStats.MessageCount, userStats.MediaCount})
	}

	sort.Slice(allTimeTop, func(i, j int) bool {
		return allTimeTop[i].Count > allTimeTop[j].Count
	})

	resp.WriteString("<b>🌟 All-Time Top Users:</b>\n")
	for i := 0; i < len(allTimeTop) && i < 5; i++ {
		medal := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣"}[i]
		resp.WriteString(fmt.Sprintf("  %s %s - <code>%d</code> msgs (<code>%d</code> media)\n",
			medal, allTimeTop[i].Name, allTimeTop[i].Count, allTimeTop[i].MediaCnt))
	}
	resp.WriteString("\n")

	// Most active hours
	type hourRank struct {
		Hour  int
		Count int
	}
	var hours []hourRank
	for hour, count := range stats.HourlyActivity {
		hours = append(hours, hourRank{hour, count})
	}
	sort.Slice(hours, func(i, j int) bool {
		return hours[i].Count > hours[j].Count
	})

	resp.WriteString("<b>⏰ Most Active Hours (UTC):</b>\n")
	for i := 0; i < len(hours) && i < 3; i++ {
		resp.WriteString(fmt.Sprintf("  %02d:00 - <code>%d</code> msgs\n",
			hours[i].Hour, hours[i].Count))
	}
	resp.WriteString("\n")

	// Media statistics
	if len(stats.MediaStats) > 0 {
		totalMedia := 0
		for _, count := range stats.MediaStats {
			totalMedia += count
		}
		resp.WriteString(fmt.Sprintf("<b>📎 Media Shared:</b> <code>%d</code> total\n", totalMedia))

		type mediaRank struct {
			Type  string
			Count int
		}
		var media []mediaRank
		for mediaType, count := range stats.MediaStats {
			media = append(media, mediaRank{mediaType, count})
		}
		sort.Slice(media, func(i, j int) bool {
			return media[i].Count > media[j].Count
		})

		for _, m := range media {
			resp.WriteString(fmt.Sprintf("  %s: <code>%d</code>\n", m.Type, m.Count))
		}
	}

	m.Reply(resp.String())
	return nil
}
