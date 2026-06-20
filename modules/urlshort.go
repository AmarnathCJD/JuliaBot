package modules

import (
	"fmt"
	"html"
	"io"
	"main/modules/db"
	"net/http"
	"net/url"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
)

func urlshortHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func urlshortValidURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if !strings.HasPrefix(strings.ToLower(raw), "http://") && !strings.HasPrefix(strings.ToLower(raw), "https://") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}

func urlshortCacheBucket(uid int64) []byte {
	return []byte(fmt.Sprintf("shorts_user_%d", uid))
}

func urlshortSaveCache(uid int64, short, orig string) {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return
	}
	_ = database.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(urlshortCacheBucket(uid))
		if err != nil {
			return err
		}
		key := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
		val := []byte(short + "\n" + orig)
		if err := bucket.Put(key, val); err != nil {
			return err
		}
		keys := [][]byte{}
		_ = bucket.ForEach(func(k, _ []byte) error {
			kc := make([]byte, len(k))
			copy(kc, k)
			keys = append(keys, kc)
			return nil
		})
		if len(keys) > 10 {
			toDelete := len(keys) - 10
			for i := 0; i < toDelete; i++ {
				_ = bucket.Delete(keys[i])
			}
		}
		return nil
	})
}

func urlshortLoadCache(uid int64) []string {
	out := []string{}
	database, err := db.GetDB()
	if err != nil || database == nil {
		return out
	}
	_ = database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(urlshortCacheBucket(uid))
		if bucket == nil {
			return nil
		}
		_ = bucket.ForEach(func(_, v []byte) error {
			out = append(out, string(v))
			return nil
		})
		return nil
	})
	return out
}

func urlshortCallIsGd(target string) (string, error) {
	endpoint := "https://is.gd/create.php?format=simple&url=" + url.QueryEscape(target)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := urlshortHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(body))
	if resp.StatusCode >= 400 || text == "" {
		if text == "" {
			text = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return "", fmt.Errorf("%s", text)
	}
	if !strings.HasPrefix(strings.ToLower(text), "http://") && !strings.HasPrefix(strings.ToLower(text), "https://") {
		return "", fmt.Errorf("%s", text)
	}
	return text, nil
}

func urlshortExpand(short string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	req, err := http.NewRequest(http.MethodHead, short, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		req2, err2 := http.NewRequest(http.MethodGet, short, nil)
		if err2 != nil {
			return "", err
		}
		req2.Header.Set("User-Agent", "JuliaBot/1.0")
		resp2, err3 := client.Do(req2)
		if err3 != nil {
			return "", err3
		}
		defer resp2.Body.Close()
		if resp2.Request != nil && resp2.Request.URL != nil {
			return resp2.Request.URL.String(), nil
		}
		return "", fmt.Errorf("unable to resolve")
	}
	defer resp.Body.Close()
	if resp.Request != nil && resp.Request.URL != nil {
		final := resp.Request.URL.String()
		if final != "" && final != short {
			return final, nil
		}
	}
	if loc, err := resp.Location(); err == nil && loc != nil {
		return loc.String(), nil
	}
	return short, nil
}

func ShortHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/short &lt;url&gt;</code>")
		return nil
	}
	target := strings.Fields(args)[0]
	if !urlshortValidURL(target) {
		m.Reply("<b>Invalid URL.</b> Must start with <code>http://</code> or <code>https://</code>.")
		return nil
	}
	status, _ := m.Reply("<i>Shortening...</i>")
	short, err := urlshortCallIsGd(target)
	if err != nil {
		msg := "<b>Shorten failed:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	urlshortSaveCache(m.SenderID(), short, target)
	out := "<b>Short URL:</b> <code>" + html.EscapeString(short) + "</code>\n<b>Original:</b> <code>" + html.EscapeString(target) + "</code>"
	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func ExpandHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/expand &lt;shorturl&gt;</code>")
		return nil
	}
	target := strings.Fields(args)[0]
	if !urlshortValidURL(target) {
		m.Reply("<b>Invalid URL.</b> Must start with <code>http://</code> or <code>https://</code>.")
		return nil
	}
	status, _ := m.Reply("<i>Expanding...</i>")
	final, err := urlshortExpand(target)
	if err != nil {
		msg := "<b>Expand failed:</b> <code>" + html.EscapeString(err.Error()) + "</code>"
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	out := "<b>Short:</b> <code>" + html.EscapeString(target) + "</code>\n<b>Expanded:</b> <code>" + html.EscapeString(final) + "</code>"
	if status != nil {
		status.Edit(out)
	} else {
		m.Reply(out)
	}
	return nil
}

func ShortsHistoryHandler(m *tg.NewMessage) error {
	entries := urlshortLoadCache(m.SenderID())
	if len(entries) == 0 {
		m.Reply("<i>No shortened URLs yet. Use /short to create one.</i>")
		return nil
	}
	var sb strings.Builder
	sb.WriteString("<b>Recent shortened URLs:</b>\n")
	for i := len(entries) - 1; i >= 0; i-- {
		parts := strings.SplitN(entries[i], "\n", 2)
		if len(parts) != 2 {
			continue
		}
		sb.WriteString("\n<b>")
		sb.WriteString(fmt.Sprintf("%d", len(entries)-i))
		sb.WriteString(".</b> <code>")
		sb.WriteString(html.EscapeString(parts[0]))
		sb.WriteString("</code>\n   <i>")
		orig := parts[1]
		if len(orig) > 80 {
			orig = orig[:80] + "..."
		}
		sb.WriteString(html.EscapeString(orig))
		sb.WriteString("</i>")
	}
	m.Reply(sb.String())
	return nil
}

func init() { QueueHandlerRegistration(registerUrlshortHandlers) }

func registerUrlshortHandlers() {
	c := Client
	c.On("cmd:short", ShortHandler)
	c.On("cmd:expand", ExpandHandler)
	c.On("cmd:shorts", ShortsHistoryHandler)
}
