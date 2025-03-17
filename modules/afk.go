package modules

import (
	"fmt"
	"math/rand"
	"strings"
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

var randomAFKMessages = []string{
	"<b>%s</b> is AFK since <b>%s</b>.",
	"<b>%s</b> is AFK for <b>%s</b>.",
	"Mr. <b>%s</b> is AFK for <b>%s</b>.",
	"<b>%s</b> has been AFK since <b>%s</b>.",
	"<b>%s</b> stepped away and is AFK for <b>%s</b>.",
	"<b>%s</b> is currently AFK for <b>%s</b>.",
}

func AFKHandler(m *tg.NewMessage) error {
	if strings.HasPrefix(m.Text(), "/afk") || strings.HasPrefix(m.Text(), "!afk") || strings.HasPrefix(m.Text(), ".afk") {
		media := ""
		if m.IsReply() {
			r, err := m.GetReplyMessage()
			if err == nil {
				if r.IsMedia() {
					media = r.File.FileID
				}
			}
		}
		afkList[m.Sender.ID] = AFK{
			Name:    m.Sender.Username,
			Message: m.Args(),
			Media:   media,
			Time:    time.Now().Unix(),
		}

		m.Reply("You are now AFK.")
		return nil
	} else {
		if afk, ok := afkList[m.SenderID()]; ok {
			delete(afkList, m.SenderID())
			duration := time.Since(time.Unix(afk.Time, 0)).String()
			m.Reply(fmt.Sprintf("Welcome back <b>%s</b>! You were AFK for %s.", afk.Name, duration))
		} else {
			if m.IsReply() {
				r, err := m.GetReplyMessage()
				if err == nil {
					if afk, ok := afkList[r.SenderID()]; ok {
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
								m.ReplyMedia(media, tg.MediaOptions{
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
							if afk, ok := afkList[e.UserID]; ok {
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
										m.ReplyMedia(media, tg.MediaOptions{
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
							for _, afk := range afkList {
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
											m.ReplyMedia(media, tg.MediaOptions{
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
								if afk, ok := afkList[peerId]; ok {
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
											m.ReplyMedia(media, tg.MediaOptions{
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
