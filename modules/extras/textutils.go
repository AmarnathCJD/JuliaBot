package extras

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	bolt "go.etcd.io/bbolt"
	"hash"
	"html"
	"io"
	modules "main/modules"
	"main/modules/db"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// === from pick.go ===
var (
	pickRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	pickMu  sync.Mutex
)

func pickIntn(n int) int {
	if n <= 0 {
		return 0
	}
	pickMu.Lock()
	defer pickMu.Unlock()
	return pickRng.Intn(n)
}

func pickFormatName(u *tg.UserObj) string {
	if u == nil {
		return "user"
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		if u.Username != "" {
			name = "@" + u.Username
		} else {
			name = fmt.Sprintf("user %d", u.ID)
		}
	}
	return name
}

func pickMention(u *tg.UserObj) string {
	name := pickFormatName(u)
	return fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", u.ID, html.EscapeString(name))
}

func PickOneHandler(m *tg.NewMessage) error {
	arg := strings.TrimSpace(m.Args())
	if arg == "" {
		m.Reply("<b>Usage:</b> <code>/pickone a b c ...</code>")
		return nil
	}
	fields := strings.Fields(arg)
	if len(fields) < 2 {
		m.Reply("<b>Error:</b> provide at least 2 options.")
		return nil
	}
	if len(fields) > 200 {
		m.Reply("<b>Error:</b> too many options (max 200).")
		return nil
	}
	pick := fields[pickIntn(len(fields))]
	m.Reply(fmt.Sprintf("<b>Pick</b>\nOptions: <code>%d</code>\nPicked: <code>%s</code>",
		len(fields), html.EscapeString(pick)))
	return nil
}

func RandMemberHandler(m *tg.NewMessage) error {
	if !m.IsGroup() {
		m.Reply("<b>This command works in groups only.</b>")
		return nil
	}
	parts, _, err := m.Client.GetChatMembers(m.ChatID(), &tg.ParticipantOptions{
		Filter: &tg.ChannelParticipantsRecent{},
		Limit:  200,
	})
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch members.")
		return nil
	}
	candidates := make([]*tg.UserObj, 0, len(parts))
	for _, p := range parts {
		if p == nil || p.User == nil {
			continue
		}
		if p.User.Bot || p.User.Deleted {
			continue
		}
		candidates = append(candidates, p.User)
	}
	if len(candidates) == 0 {
		m.Reply("<b>Error:</b> no eligible members found.")
		return nil
	}
	pick := candidates[pickIntn(len(candidates))]
	m.Reply(fmt.Sprintf("<b>Random Member</b>\n%s", pickMention(pick)))
	return nil
}

func RandAdminHandler(m *tg.NewMessage) error {
	if !m.IsGroup() {
		m.Reply("<b>This command works in groups only.</b>")
		return nil
	}
	parts, _, err := m.Client.GetChatMembers(m.ChatID(), &tg.ParticipantOptions{
		Filter: &tg.ChannelParticipantsAdmins{},
		Limit:  200,
	})
	if err != nil {
		m.Reply("<b>Error:</b> failed to fetch admins.")
		return nil
	}
	candidates := make([]*tg.UserObj, 0, len(parts))
	for _, p := range parts {
		if p == nil || p.User == nil {
			continue
		}
		if p.User.Bot || p.User.Deleted {
			continue
		}
		if p.Status != tg.Admin && p.Status != tg.Creator {
			continue
		}
		candidates = append(candidates, p.User)
	}
	if len(candidates) == 0 {
		m.Reply("<b>Error:</b> no eligible admins found.")
		return nil
	}
	pick := candidates[pickIntn(len(candidates))]
	m.Reply(fmt.Sprintf("<b>Random Admin</b>\n%s", pickMention(pick)))
	return nil
}

func registerPickHandlers() {
	c := modules.Client
	c.On("cmd:pickone", PickOneHandler)
	c.On("cmd:randmember", RandMemberHandler)
	c.On("cmd:randadmin", RandAdminHandler)
}

func initFromSrc_pick_0_1() {
	modules.QueueHandlerRegistration(registerPickHandlers)

	modules.Mods.AddModule("Pick", `<b>Pick Module</b>

Random picker utilities.

<b>Commands:</b>
 • /pickone a b c ... - Pick one item randomly from args
 • /randmember - Pick a random group member (skips bots)
 • /randadmin - Pick a random group admin (skips bots)`)
}
// === from lorem_ipsum.go ===
var loremRng = rand.New(rand.NewSource(time.Now().UnixNano()))

var loremWords = []string{
	"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit",
	"sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore",
	"magna", "aliqua", "enim", "ad", "minim", "veniam", "quis", "nostrud",
	"exercitation", "ullamco", "laboris", "nisi", "aliquip", "ex", "ea", "commodo",
	"consequat", "duis", "aute", "irure", "in", "reprehenderit", "voluptate",
	"velit", "esse", "cillum", "eu", "fugiat", "nulla", "pariatur", "excepteur",
	"sint", "occaecat", "cupidatat", "non", "proident", "sunt", "culpa", "qui",
	"officia", "deserunt", "mollit", "anim", "id", "est", "laborum", "at", "vero",
	"eos", "accusamus", "iusto", "odio", "dignissimos", "ducimus", "blanditiis",
	"praesentium", "voluptatum", "deleniti", "atque", "corrupti", "quos", "dolores",
	"quas", "molestias", "excepturi", "sint", "obcaecati", "cupiditate", "provident",
	"similique", "mollitia", "animi", "laborum", "dolorum", "fuga", "harum",
	"quidem", "rerum", "facilis", "expedita", "distinctio", "nam", "libero",
	"tempore", "cum", "soluta", "nobis", "eligendi", "optio", "cumque", "nihil",
	"impedit", "quo", "minus", "maxime", "placeat", "facere", "possimus", "omnis",
	"assumenda", "repellendus", "temporibus", "autem", "quibusdam", "officiis",
	"debitis", "necessitatibus", "saepe", "eveniet", "voluptates", "repudiandae",
	"recusandae", "itaque", "earum", "hic", "tenetur", "sapiente", "delectus",
	"reiciendis", "voluptatibus", "maiores", "alias", "perferendis", "doloribus",
	"asperiores", "repellat", "neque", "porro", "quisquam", "dolorem", "ipsam",
	"quia", "voluptas", "aspernatur", "aut", "odit", "fugit", "consequuntur",
	"magni", "ratione", "sequi", "nesciunt", "neque", "porro", "quisquam", "est",
	"qui", "dolorem", "ipsum", "quia", "dolor", "sit", "amet", "consectetur",
	"adipisci", "velit", "numquam", "eius", "modi", "tempora", "incidunt", "magnam",
	"aliquam", "quaerat", "ullam", "corporis", "suscipit", "laboriosam", "nisi",
	"aliquid", "ex", "ea", "commodi", "consequatur", "autem", "vel", "eum", "iure",
	"reprehenderit", "qui", "in", "ea", "voluptate", "velit", "esse", "quam",
	"nihil", "molestiae", "consequatur", "vel", "illum", "qui", "dolorem", "fugiat",
	"quo", "voluptas", "nulla",
}

func loremCapitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 32
	}
	return string(r)
}

func loremSentence() string {
	wordCount := 6 + loremRng.Intn(10)
	words := make([]string, wordCount)
	for i := 0; i < wordCount; i++ {
		words[i] = loremWords[loremRng.Intn(len(loremWords))]
	}
	words[0] = loremCapitalize(words[0])
	if wordCount > 4 {
		commaPos := 2 + loremRng.Intn(wordCount-3)
		words[commaPos] = words[commaPos] + ","
	}
	return strings.Join(words, " ") + "."
}

func loremParagraph(idx int) string {
	if idx == 0 {
		intro := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."
		sentenceCount := 3 + loremRng.Intn(4)
		sentences := make([]string, 0, sentenceCount+1)
		sentences = append(sentences, intro)
		for i := 0; i < sentenceCount; i++ {
			sentences = append(sentences, loremSentence())
		}
		return strings.Join(sentences, " ")
	}
	sentenceCount := 4 + loremRng.Intn(4)
	sentences := make([]string, sentenceCount)
	for i := 0; i < sentenceCount; i++ {
		sentences[i] = loremSentence()
	}
	return strings.Join(sentences, " ")
}

func LoremHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	n := 1
	if args != "" {
		parsed, err := strconv.Atoi(args)
		if err != nil || parsed < 1 {
			m.Reply("<b>Usage:</b> <code>/lorem [N]</code>\nN must be a positive integer.")
			return nil
		}
		n = parsed
	}
	if n > 20 {
		n = 20
	}
	paragraphs := make([]string, n)
	for i := 0; i < n; i++ {
		paragraphs[i] = loremParagraph(i)
	}
	body := strings.Join(paragraphs, "\n\n")
	out := fmt.Sprintf("<b>Lorem Ipsum</b> <i>(%d paragraph(s))</i>\n\n%s", n, html.EscapeString(body))
	if len(out) > 4000 {
		out = out[:4000] + "\n... (truncated)"
	}
	m.Reply(out)
	return nil
}

func registerLoremHandlers() {
	c := modules.Client
	c.On("cmd:lorem", LoremHandler)
}

func initFromSrc_lorem_ipsum_1_1() {
	modules.QueueHandlerRegistration(registerLoremHandlers)
}
// === from echo.go ===
func echoReplyCode(m *tg.NewMessage, s string) {
	escaped := html.EscapeString(s)
	if len(escaped) > 4000 {
		escaped = escaped[:4000] + "\n... (truncated)"
	}
	m.Reply("<code>" + escaped + "</code>")
}

func EchoHandler(m *tg.NewMessage) error {
	text := modules.ExtractText(m)
	if text == "" {
		m.Reply("usage: /echo &lt;text&gt;")
		return nil
	}
	m.Client.SendMessage(m.ChatID(), html.EscapeString(text))
	return nil
}

func ReverseItHandler(m *tg.NewMessage) error {
	text := modules.ExtractText(m)
	if text == "" {
		m.Reply("usage: /reverseit &lt;text&gt;")
		return nil
	}
	runes := []rune(text)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	echoReplyCode(m, string(runes))
	return nil
}

func ClapHandler(m *tg.NewMessage) error {
	text := modules.ExtractText(m)
	if text == "" {
		m.Reply("usage: /clap &lt;text&gt;")
		return nil
	}
	parts := strings.Fields(text)
	if len(parts) == 0 {
		m.Reply("usage: /clap &lt;text&gt;")
		return nil
	}
	echoReplyCode(m, strings.Join(parts, " 👏 "))
	return nil
}

func UpperHandler(m *tg.NewMessage) error {
	text := modules.ExtractText(m)
	if text == "" {
		m.Reply("usage: /upper &lt;text&gt;")
		return nil
	}
	echoReplyCode(m, strings.ToUpper(text))
	return nil
}

func LowerHandler(m *tg.NewMessage) error {
	text := modules.ExtractText(m)
	if text == "" {
		m.Reply("usage: /lower &lt;text&gt;")
		return nil
	}
	echoReplyCode(m, strings.ToLower(text))
	return nil
}

func TitleHandler(m *tg.NewMessage) error {
	text := modules.ExtractText(m)
	if text == "" {
		m.Reply("usage: /title &lt;text&gt;")
		return nil
	}
	echoReplyCode(m, strings.Title(strings.ToLower(text)))
	return nil
}

func LenHandler(m *tg.NewMessage) error {
	text := modules.ExtractText(m)
	if text == "" {
		m.Reply("usage: /len &lt;text&gt;")
		return nil
	}
	chars := utf8.RuneCountInString(text)
	bytes := len(text)
	words := len(strings.Fields(text))
	out := fmt.Sprintf("chars: %d\nbytes: %d\nwords: %d", chars, bytes, words)
	m.Reply("<code>" + html.EscapeString(out) + "</code>")
	return nil
}

func CountHandler(m *tg.NewMessage) error {
	text := modules.ExtractText(m)
	if text == "" {
		m.Reply("reply to a message or supply text: /count &lt;text&gt;")
		return nil
	}
	chars := utf8.RuneCountInString(text)
	bytes := len(text)
	out := fmt.Sprintf("chars: %d\nbytes: %d", chars, bytes)
	m.Reply("<code>" + html.EscapeString(out) + "</code>")
	return nil
}

func initFromSrc_echo_2_1() { modules.QueueHandlerRegistration(registerEchoHandlers) }
func registerEchoHandlers() {
	c := modules.Client
	c.On("cmd:echo", EchoHandler, tg.CustomFilter(modules.FilterOwner))
	c.On("cmd:reverseit", ReverseItHandler)
	c.On("cmd:clap", ClapHandler)
	c.On("cmd:upper", UpperHandler)
	c.On("cmd:lower", LowerHandler)
	c.On("cmd:title", TitleHandler)
	c.On("cmd:len", LenHandler)
	c.On("cmd:count", CountHandler)
}
// === from hash.go ===
func hashComputeText(algo, text string) (string, error) {
	var h hash.Hash
	switch strings.ToLower(algo) {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return "", fmt.Errorf("unsupported algorithm")
	}
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashComputeFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hMD5 := md5.New()
	hSHA1 := sha1.New()
	hSHA256 := sha256.New()
	hSHA512 := sha512.New()

	mw := io.MultiWriter(hMD5, hSHA1, hSHA256, hSHA512)
	if _, err := io.Copy(mw, f); err != nil {
		return nil, err
	}

	return map[string]string{
		"md5":    hex.EncodeToString(hMD5.Sum(nil)),
		"sha1":   hex.EncodeToString(hSHA1.Sum(nil)),
		"sha256": hex.EncodeToString(hSHA256.Sum(nil)),
		"sha512": hex.EncodeToString(hSHA512.Sum(nil)),
	}, nil
}

func HashHandler(m *tg.NewMessage) error {
	if m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil && r.IsMedia() {
			status, _ := m.Reply("<code>downloading file...</code>")

			path, err := m.Client.DownloadMedia(r.Media())
			if err != nil {
				msg := "error downloading: " + html.EscapeString(err.Error())
				if status != nil {
					status.Edit(msg)
				} else {
					m.Reply(msg)
				}
				return nil
			}
			defer os.Remove(path)

			if status != nil {
				status.Edit("<code>computing hashes...</code>")
			}

			hashes, err := hashComputeFile(path)
			if err != nil {
				msg := "error computing: " + html.EscapeString(err.Error())
				if status != nil {
					status.Edit(msg)
				} else {
					m.Reply(msg)
				}
				return nil
			}

			var sb strings.Builder
			sb.WriteString("<b>File Hashes</b>\n\n")
			sb.WriteString("<b>MD5:</b>\n<code>" + html.EscapeString(hashes["md5"]) + "</code>\n\n")
			sb.WriteString("<b>SHA1:</b>\n<code>" + html.EscapeString(hashes["sha1"]) + "</code>\n\n")
			sb.WriteString("<b>SHA256:</b>\n<code>" + html.EscapeString(hashes["sha256"]) + "</code>\n\n")
			sb.WriteString("<b>SHA512:</b>\n<code>" + html.EscapeString(hashes["sha512"]) + "</code>")

			out := sb.String()
			if status != nil {
				status.Edit(out)
			} else {
				m.Reply(out)
			}
			return nil
		}
	}

	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("usage: <code>/hash &lt;md5|sha1|sha256|sha512&gt; &lt;text&gt;</code>\nor reply to a file with <code>/hash</code>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		m.Reply("usage: <code>/hash &lt;md5|sha1|sha256|sha512&gt; &lt;text&gt;</code>")
		return nil
	}

	algo := strings.ToLower(strings.TrimSpace(parts[0]))
	text := parts[1]

	digest, err := hashComputeText(algo, text)
	if err != nil {
		m.Reply("error: unsupported algorithm. use md5, sha1, sha256, or sha512")
		return nil
	}

	out := "<b>" + strings.ToUpper(algo) + ":</b>\n<code>" + html.EscapeString(digest) + "</code>"
	m.Reply(out)
	return nil
}

func initFromSrc_hash_3_1() { modules.QueueHandlerRegistration(registerHashHandlers) }

func registerHashHandlers() {
	c := modules.Client
	c.On("cmd:hash", HashHandler)
}
// === from urlshort.go ===
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

func initFromSrc_urlshort_4_1() { modules.QueueHandlerRegistration(registerUrlshortHandlers) }

func registerUrlshortHandlers() {
	c := modules.Client
	c.On("cmd:short", ShortHandler)
	c.On("cmd:expand", ExpandHandler)
	c.On("cmd:shorts", ShortsHistoryHandler)
}

func init() {
	initFromSrc_pick_0_1()
	initFromSrc_lorem_ipsum_1_1()
	initFromSrc_echo_2_1()
	initFromSrc_hash_3_1()
	initFromSrc_urlshort_4_1()
}
