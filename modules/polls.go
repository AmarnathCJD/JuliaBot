package modules

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"main/modules/db"
	"sort"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
)

var pollsBucket = []byte("polls")

type pollData struct {
	ID       string         `json:"id"`
	Author   int64          `json:"author"`
	Chat     int64          `json:"chat"`
	Question string         `json:"question"`
	Options  []string       `json:"options"`
	Votes    map[string]int `json:"votes"`
	Closed   bool           `json:"closed"`
}

func newPollID() string {
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		return "000000000000"
	}
	return hex.EncodeToString(b)
}

func loadPoll(id string) (*pollData, error) {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var p *pollData
	err = database.View(func(tx *bbolt.Tx) error {
		bk := tx.Bucket(pollsBucket)
		if bk == nil {
			return nil
		}
		raw := bk.Get([]byte(id))
		if raw == nil {
			return nil
		}
		var pd pollData
		if jerr := json.Unmarshal(raw, &pd); jerr != nil {
			return jerr
		}
		p = &pd
		return nil
	})
	return p, err
}

func savePoll(p *pollData) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db unavailable")
	}
	return database.Update(func(tx *bbolt.Tx) error {
		bk, berr := tx.CreateBucketIfNotExists(pollsBucket)
		if berr != nil {
			return berr
		}
		raw, jerr := json.Marshal(p)
		if jerr != nil {
			return jerr
		}
		return bk.Put([]byte(p.ID), raw)
	})
}

func pollTallyCounts(p *pollData) []int {
	counts := make([]int, len(p.Options))
	for _, idx := range p.Votes {
		if idx >= 0 && idx < len(counts) {
			counts[idx]++
		}
	}
	return counts
}

func pollBar(count, total int) string {
	if total <= 0 {
		return strings.Repeat("░", 10)
	}
	filled := (count * 10) / total
	if filled > 10 {
		filled = 10
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
}

func pollRender(p *pollData) string {
	var sb strings.Builder
	if p.Closed {
		sb.WriteString("<b>Poll (Closed)</b>\n")
	} else {
		sb.WriteString("<b>Poll</b>\n")
	}
	sb.WriteString("<b>Q:</b> ")
	sb.WriteString(html.EscapeString(p.Question))
	sb.WriteString("\n\n")

	counts := pollTallyCounts(p)
	total := 0
	for _, c := range counts {
		total += c
	}

	for i, opt := range p.Options {
		pct := 0
		if total > 0 {
			pct = (counts[i] * 100) / total
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, html.EscapeString(opt)))
		sb.WriteString(fmt.Sprintf("   <code>%s</code> %d votes (%d%%)\n", pollBar(counts[i], total), counts[i], pct))
	}

	sb.WriteString(fmt.Sprintf("\n<b>Total votes:</b> %d\n", total))
	sb.WriteString(fmt.Sprintf("<b>Poll ID:</b> <code>%s</code>\n", p.ID))
	if p.Closed {
		var winnerIdx int = -1
		var winnerCount int = -1
		tied := false
		for i, c := range counts {
			if c > winnerCount {
				winnerCount = c
				winnerIdx = i
				tied = false
			} else if c == winnerCount && c > 0 {
				tied = true
			}
		}
		if winnerCount > 0 {
			if tied {
				sb.WriteString("<i>Result: tied.</i>")
			} else if winnerIdx >= 0 {
				sb.WriteString(fmt.Sprintf("<i>Winner: %s</i>", html.EscapeString(p.Options[winnerIdx])))
			}
		} else {
			sb.WriteString("<i>No votes cast.</i>")
		}
	} else {
		sb.WriteString("<i>Tap an option to vote. You can change your vote.</i>")
	}
	return sb.String()
}

func pollBuildKeyboard(p *pollData) *tg.ReplyInlineMarkup {
	if p.Closed {
		return nil
	}
	b := tg.Button
	kb := tg.NewKeyboard()
	for i, opt := range p.Options {
		label := opt
		if len(label) > 30 {
			label = label[:27] + "..."
		}
		label = fmt.Sprintf("%d. %s", i+1, label)
		data := fmt.Sprintf("poll:%s:%d", p.ID, i)
		kb.AddRow(b.Data(label, data))
	}
	return kb.Build()
}

func PollHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/poll &lt;question&gt; | &lt;opt1&gt; | &lt;opt2&gt; | ...</code>\n<b>Example:</b> <code>/poll Best language? | Go | Rust | Python</code>")
		return nil
	}

	parts := strings.Split(args, "|")
	if len(parts) < 3 {
		m.Reply("<b>Need at least one question and two options.</b>\nFormat: <code>/poll question | opt1 | opt2</code>")
		return nil
	}

	question := strings.TrimSpace(parts[0])
	if question == "" {
		m.Reply("<b>Question cannot be empty.</b>")
		return nil
	}

	var options []string
	seen := make(map[string]bool)
	for _, p := range parts[1:] {
		opt := strings.TrimSpace(p)
		if opt == "" {
			continue
		}
		if len(opt) > 100 {
			opt = opt[:100]
		}
		key := strings.ToLower(opt)
		if seen[key] {
			continue
		}
		seen[key] = true
		options = append(options, opt)
	}

	if len(options) < 2 {
		m.Reply("<b>Need at least two distinct options.</b>")
		return nil
	}
	if len(options) > 10 {
		m.Reply("<b>Maximum 10 options allowed.</b>")
		return nil
	}
	if len(question) > 300 {
		m.Reply("<b>Question too long.</b> Max 300 characters.")
		return nil
	}

	p := &pollData{
		ID:       newPollID(),
		Author:   m.SenderID(),
		Chat:     m.ChatID(),
		Question: question,
		Options:  options,
		Votes:    map[string]int{},
		Closed:   false,
	}

	if err := savePoll(p); err != nil {
		m.Reply("<b>Failed to create poll.</b> Database error.")
		return nil
	}

	m.Reply(pollRender(p), &tg.SendOptions{ReplyMarkup: pollBuildKeyboard(p)})
	return nil
}

func ClosePollHandler(m *tg.NewMessage) error {
	id := strings.TrimSpace(m.Args())
	if id == "" {
		m.Reply("<b>Usage:</b> <code>/closepoll &lt;poll_id&gt;</code>")
		return nil
	}

	p, err := loadPoll(id)
	if err != nil || p == nil {
		m.Reply(fmt.Sprintf("<b>Poll not found:</b> <code>%s</code>", html.EscapeString(id)))
		return nil
	}

	if p.Author != m.SenderID() {
		m.Reply("<b>Only the poll author can close this poll.</b>")
		return nil
	}

	if p.Closed {
		m.Reply("<b>Poll already closed.</b>")
		return nil
	}

	p.Closed = true
	if err := savePoll(p); err != nil {
		m.Reply("<b>Failed to close poll.</b>")
		return nil
	}

	counts := pollTallyCounts(p)
	type kv struct {
		Idx   int
		Label string
		Count int
	}
	results := make([]kv, 0, len(p.Options))
	for i, opt := range p.Options {
		results = append(results, kv{Idx: i, Label: opt, Count: counts[i]})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Count > results[j].Count
	})

	var sb strings.Builder
	sb.WriteString("<b>Poll Closed</b>\n")
	sb.WriteString("<b>Q:</b> " + html.EscapeString(p.Question) + "\n\n")
	sb.WriteString("<b>Final Results:</b>\n")
	total := 0
	for _, r := range results {
		total += r.Count
	}
	for rank, r := range results {
		pct := 0
		if total > 0 {
			pct = (r.Count * 100) / total
		}
		sb.WriteString(fmt.Sprintf("#%d %s — %d votes (%d%%)\n", rank+1, html.EscapeString(r.Label), r.Count, pct))
	}
	sb.WriteString(fmt.Sprintf("\n<b>Total votes:</b> %d\n", total))
	sb.WriteString(fmt.Sprintf("<b>Poll ID:</b> <code>%s</code>", p.ID))

	m.Reply(sb.String())
	return nil
}

func PollCallbackHandler(c *tg.CallbackQuery) error {
	data := c.DataString()
	if !strings.HasPrefix(data, "poll:") {
		return nil
	}
	body := strings.TrimPrefix(data, "poll:")
	parts := strings.SplitN(body, ":", 2)
	if len(parts) != 2 {
		c.Answer("Invalid poll data.", &tg.CallbackOptions{Alert: true})
		return nil
	}
	pollID := parts[0]
	optIdx, err := strconv.Atoi(parts[1])
	if err != nil {
		c.Answer("Invalid option.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	p, lerr := loadPoll(pollID)
	if lerr != nil || p == nil {
		c.Answer("Poll not found.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	if p.Closed {
		c.Answer("Poll is closed.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	if optIdx < 0 || optIdx >= len(p.Options) {
		c.Answer("Invalid option index.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	userKey := strconv.FormatInt(c.SenderID, 10)
	prev, had := p.Votes[userKey]
	if had && prev == optIdx {
		c.Answer("Already voted: "+p.Options[optIdx], &tg.CallbackOptions{Alert: false})
		return nil
	}

	if p.Votes == nil {
		p.Votes = map[string]int{}
	}
	p.Votes[userKey] = optIdx

	if err := savePoll(p); err != nil {
		c.Answer("Failed to save vote.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	c.Edit(pollRender(p), &tg.SendOptions{ReplyMarkup: pollBuildKeyboard(p)})
	if had {
		c.Answer("Vote changed to: "+p.Options[optIdx], &tg.CallbackOptions{Alert: false})
	} else {
		c.Answer("Voted: "+p.Options[optIdx], &tg.CallbackOptions{Alert: false})
	}
	return nil
}

func registerPollsHandlers() {
	c := Client
	c.On("cmd:poll", PollHandler)
	c.On("cmd:closepoll", ClosePollHandler)
	c.On("callback:poll:", PollCallbackHandler)
}

func init() {
	QueueHandlerRegistration(registerPollsHandlers)

	Mods.AddModule("Polls", `<b>Polls Module</b>

Create inline-button polls with live tally and one-vote-per-user (changeable).

<b>Commands:</b>
 - /poll &lt;question&gt; | &lt;opt1&gt; | &lt;opt2&gt; | ... — create a poll (2-10 options)
 - /closepoll &lt;poll_id&gt; — close a poll (author only) and show final results

<b>Notes:</b>
 - Tap any option to vote; tap a different option to change your vote.
 - The poll message updates live with a bar tally and percentages.
 - Each poll has a short hex ID shown in the message; use it with /closepoll.`)
}
