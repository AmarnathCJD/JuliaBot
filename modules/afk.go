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
	"%s is AFK since %s.",
	"%s is not here since %s.",
	"%s is away since %s.",
	"%s is not available since %s.",
	"%s is not around since %s.",
	"%s is not present since %s.",
	"%s is not here since %s.",
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
			m.Reply(fmt.Sprintf("Welcome back %s! You were AFK for %s.", afk.Name, duration))
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
							m.ReplyMedia(media, tg.MediaOptions{
								Caption: msg,
							})
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
						switch entity.(type) {
						case *tg.MessageEntityMentionName:
							if afk, ok := afkList[m.Message.Entities[0].(*tg.MessageEntityMentionName).UserID]; ok {
								duration := time.Since(time.Unix(afk.Time, 0)).String()
								msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
								if afk.Media != "" {
									var msg = fmt.Sprintf(msg, afk.Name, duration)
									if afk.Message != "" {
										msg += "\nReason: " + afk.Message
									}
									media, _ := tg.ResolveBotFileID(afk.Media)
									m.ReplyMedia(media, tg.MediaOptions{
										Caption: msg,
									})
								} else {
									var msg = fmt.Sprintf(msg, afk.Name, duration)
									if afk.Message != "" {
										msg += "\nReason: " + afk.Message
									}

									m.Reply(msg)
								}
							}
						case *tg.MessageEntityMention:
							ent := m.Message.Entities[0].(*tg.MessageEntityMention)
							offset := ent.Offset
							length := ent.Length

							username := m.Text()[offset : offset+length]
							for _, afk := range afkList {
								if afk.Name == username {
									duration := time.Since(time.Unix(afk.Time, 0)).String()
									msg := randomAFKMessages[rand.Intn(len(randomAFKMessages))]
									if afk.Media != "" {
										var msg = fmt.Sprintf(msg, afk.Name, duration)
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}
										media, _ := tg.ResolveBotFileID(afk.Media)
										m.ReplyMedia(media, tg.MediaOptions{
											Caption: msg,
										})
									} else {
										var msg = fmt.Sprintf(msg, afk.Name, duration)
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
										var msg = fmt.Sprintf(msg, afk.Name, duration)
										if afk.Message != "" {
											msg += "\nReason: " + afk.Message
										}
										media, _ := tg.ResolveBotFileID(afk.Media)
										m.ReplyMedia(media, tg.MediaOptions{
											Caption: msg,
										})
									} else {
										var msg = fmt.Sprintf(msg, afk.Name, duration)
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
