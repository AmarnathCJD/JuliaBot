package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var mockAPIClient = &http.Client{Timeout: 30 * time.Second}

type mockPost struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type mockTodo struct {
	UserID    int    `json:"userId"`
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func fetchMockAPI(endpoint string, out any) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := mockAPIClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jsonplaceholder returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func RandomPostHandler(m *tg.NewMessage) error {
	id := rand.Intn(100) + 1
	endpoint := fmt.Sprintf("https://jsonplaceholder.typicode.com/posts/%d", id)
	var post mockPost
	if err := fetchMockAPI(endpoint, &post); err != nil {
		m.Reply("<b>Error:</b> failed to fetch random post.")
		return nil
	}
	if post.ID == 0 {
		m.Reply("<b>Error:</b> no post received.")
		return nil
	}
	title := strings.TrimSpace(post.Title)
	body := strings.TrimSpace(post.Body)
	reply := fmt.Sprintf(
		"<b>Random Post #%d</b>\n<b>Author (userId):</b> %d\n\n<b>%s</b>\n<blockquote>%s</blockquote>",
		post.ID,
		post.UserID,
		html.EscapeString(title),
		html.EscapeString(body),
	)
	m.Reply(reply)
	return nil
}

func RandomTodoHandler(m *tg.NewMessage) error {
	id := rand.Intn(200) + 1
	endpoint := fmt.Sprintf("https://jsonplaceholder.typicode.com/todos/%d", id)
	var todo mockTodo
	if err := fetchMockAPI(endpoint, &todo); err != nil {
		m.Reply("<b>Error:</b> failed to fetch random todo.")
		return nil
	}
	if todo.ID == 0 {
		m.Reply("<b>Error:</b> no todo received.")
		return nil
	}
	status := "Pending"
	if todo.Completed {
		status = "Completed"
	}
	title := strings.TrimSpace(todo.Title)
	reply := fmt.Sprintf(
		"<b>Random Todo #%d</b>\n<b>Owner (userId):</b> %d\n<b>Status:</b> %s\n\n<blockquote>%s</blockquote>",
		todo.ID,
		todo.UserID,
		status,
		html.EscapeString(title),
	)
	m.Reply(reply)
	return nil
}

func registerMockAPIHandlers() {
	c := Client
	c.On("cmd:randompost", RandomPostHandler)
	c.On("cmd:randomtodo", RandomTodoHandler)
}

func init() {
	QueueHandlerRegistration(registerMockAPIHandlers)
}
