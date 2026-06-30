package modules

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type AFK struct {
	Name    string
	Message string
	Media   string
	Time    int64
}

var afkList = make(map[int64]AFK)
var afkMu sync.RWMutex

var randomAFKMessages = []string{
	"<b>%s</b> is AFK since <b>%s</b>.",
	"<b>%s</b> is AFK for <b>%s</b>.",
	"Mr. <b>%s</b> is AFK for <b>%s</b>.",
	"<b>%s</b> has been AFK since <b>%s</b>.",
	"<b>%s</b> stepped away and is AFK for <b>%s</b>.",
	"<b>%s</b> is currently AFK for <b>%s</b>.",
}

func AFKHandler(m *tg.NewMessage) error {
	if m.Sender == nil {
		return nil
	}
	text := m.Text()
	cmd := ""
	if fields := strings.Fields(text); len(fields) > 0 {
		cmd = fields[0]
	}
	isAfkCmd := cmd == "/afk" || cmd == "!afk" || cmd == ".afk" ||
		strings.HasPrefix(cmd, "/afk@") || strings.HasPrefix(cmd, "!afk@") || strings.HasPrefix(cmd, ".afk@")
	if isAfkCmd {
		media := ""
		if m.IsReply() {
			r, err := m.GetReplyMessage()
			if err == nil {
				if r.IsMedia() && r.File != nil {
					media = r.File.FileID
				}
			}
		}
		afkMu.Lock()
		afkList[m.Sender.ID] = AFK{
			Name:    m.Sender.Username,
			Message: m.Args(),
			Media:   media,
			Time:    time.Now().Unix(),
		}
		afkMu.Unlock()

		m.Reply("You are now AFK.")
		return nil
	} else {
		afkMu.RLock()
		afk, ok := afkList[m.SenderID()]
		afkMu.RUnlock()
		if ok {
			afkMu.Lock()
			delete(afkList, m.SenderID())
			afkMu.Unlock()
			duration := time.Since(time.Unix(afk.Time, 0)).String()
			m.Reply(fmt.Sprintf("Welcome back <b>%s</b>! You were AFK for %s.", afk.Name, duration))
		} else {
			if m.IsReply() {
				r, err := m.GetReplyMessage()
				if err == nil {
					afkMu.RLock()
					afk, ok := afkList[r.SenderID()]
					afkMu.RUnlock()
					if ok {
						duration := time.Since(time.Unix(afk.Time, 0)).String()
						msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
						if afk.Media != "" {
							var msg = fmt.Sprintf(msg, afk.Name, duration)
							if afk.Message != "" {
								msg += "\nReason: " + afk.Message
							}
							media, _ := tg.ResolveBotFileID(afk.Media)
							if IsSticker(media) {
								m.ReplyMedia(media)
								m.Respond(msg)
							} else {
								m.ReplyMedia(media, &tg.MediaOptions{
									Caption: msg,
								})
							}
						} else {
							var msg = fmt.Sprintf(msg, afk.Name, duration)
							if afk.Message != "" {
								msg += "\nReason: " + afk.Message
							}

							m.Reply(msg)
						}
					}
				}
			} else {
				if len(m.Message.Entities) > 0 {
					for _, entity := range m.Message.Entities {
						switch e := entity.(type) {
						case *tg.MessageEntityMentionName:
							afkMu.RLock()
							afk, ok := afkList[e.UserID]
							afkMu.RUnlock()
							if ok {
								duration := time.Since(time.Unix(afk.Time, 0)).String()
								msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
								if afk.Media != "" {
									var msg = fmt.Sprintf(msg, afk.Name, duration)
									if afk.Message != "" {
										msg += "\nReason: " + afk.Message
									}
									media, _ := tg.ResolveBotFileID(afk.Media)
									if IsSticker(media) {
										m.ReplyMedia(media)
										m.Respond(msg)
									} else {
										m.ReplyMedia(media, &tg.MediaOptions{
											Caption: msg,
										})
									}
								} else {
									var msg = fmt.Sprintf(msg, afk.Name, duration)
									if afk.Message != "" {
										msg += "\nReason: " + afk.Message
									}

									m.Reply(msg)
								}
							}
						case *tg.MessageEntityMention:
							offset := e.Offset
							length := e.Length

							username := m.Text()[offset : offset+length]
							afkMu.RLock()
							afkSnap := make(map[int64]AFK, len(afkList))
							for k, v := range afkList {
								afkSnap[k] = v
							}
							afkMu.RUnlock()
							for _, afk := range afkSnap {
								if afk.Name == username {
									duration := time.Since(time.Unix(afk.Time, 0)).String()
									msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
									if afk.Media != "" {
										var msg = fmt.Sprintf(msg, afk.Name, trimDecimal(duration))
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}
										media, _ := tg.ResolveBotFileID(afk.Media)
										if IsSticker(media) {
											m.ReplyMedia(media)
											m.Respond(msg)
										} else {
											m.ReplyMedia(media, &tg.MediaOptions{
												Caption: msg,
											})
										}
									} else {
										var msg = fmt.Sprintf(msg, afk.Name, trimDecimal(duration))
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}

										m.Reply(msg)
									}
								}
							}

							user, err := m.Client.ResolvePeer(username)
							if err == nil {
								peerId := m.Client.GetPeerID(user)
								afkMu.RLock()
								afk, ok := afkList[peerId]
								afkMu.RUnlock()
								if ok {
									duration := time.Since(time.Unix(afk.Time, 0)).String()
									msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
									if afk.Media != "" {
										var msg = fmt.Sprintf(msg, afk.Name, trimDecimal(duration))
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}
										media, _ := tg.ResolveBotFileID(afk.Media)
										if IsSticker(media) {
											m.ReplyMedia(media)
											m.Respond(msg)
										} else {
											m.ReplyMedia(media, &tg.MediaOptions{
												Caption: msg,
											})
										}
									} else {
										var msg = fmt.Sprintf(msg, afk.Name, trimDecimal(duration))
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}

										m.Reply(msg)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func trimDecimal(s string) string {
	if strings.Contains(s, ".") {
		return strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

func IsSticker(m tg.MessageMedia) bool {
	switch m := m.(type) {
	case *tg.MessageMediaDocument:
		attrs := m.Document.(*tg.DocumentObj).Attributes
		for _, attr := range attrs {
			if _, ok := attr.(*tg.DocumentAttributeSticker); ok {
				return true
			}
		}
	}

	return false
}

func SedHandler(m *tg.NewMessage) error {
	text := m.Text()
	if len(text) < 4 {
		return nil
	}

	if text[0] != 's' {
		return nil
	}

	delimiter := text[1]
	if delimiter != '/' && delimiter != '\\' {
		return nil
	}

	parts := strings.Split(text[2:], string(delimiter))
	if len(parts) < 2 {
		return nil
	}

	find := parts[0]
	replace := parts[1]

	if find == "" {
		return nil
	}

	if !m.IsReply() {
		return nil
	}

	replyMsg, err := m.GetReplyMessage()
	if err != nil {
		return nil
	}

	originalText := replyMsg.Text()
	if originalText == "" {
		return nil
	}

	if !strings.Contains(originalText, find) {
		return nil
	}

	newText := strings.Replace(originalText, find, replace, -1)
	m.Reply(newText)

	return nil
}

func registerAFKHandlers() {
	c := Client
	c.On(tg.OnNewMessage, AFKHandler)
	c.On(tg.OnNewMessage, SedHandler)
}

func init() {
	QueueHandlerRegistration(registerAFKHandlers)
}
