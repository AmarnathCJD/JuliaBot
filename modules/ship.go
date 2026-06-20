package modules

import (
	"hash/crc32"
	"html"
	"sort"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type shipParty struct {
	ID   int64
	Name string
}

var shipCringeHigh = []string{
	"the universe just sighed in relief — soulmates detected, certified, stamped, sealed.",
	"a love so loud the stars filed a noise complaint.",
	"if true love had a face, it would be wearing both your usernames.",
	"this is the kind of pairing fanfic writers would call <i>too perfect to be canon</i>.",
	"cupid quit his job because his work here is done.",
	"angels are gossiping about you two in the group chat.",
	"this match is so strong, even WiFi routers boost their signal nearby.",
	"forget red strings of fate — this is a steel cable of destiny.",
	"the moon is blushing. the sun is jealous. you two won the cosmos.",
	"romance novels will be rewritten to use you two as the benchmark.",
}

var shipCringeMid = []string{
	"it's giving <i>maybe-if-mercury-stops-being-in-retrograde</i> energy.",
	"there's a spark — small, flickering, possibly just static from the carpet.",
	"could be cute. could be a cautionary tale. flip a coin.",
	"the vibes are mid but the chemistry might pop off after three coffees.",
	"this love story would be a slow burn… as in, abandoned by chapter 4.",
	"a situationship waiting to happen, but make it aesthetic.",
	"7-10 business days until one of you ghosts.",
	"could work in another timeline. probably one with better lighting.",
	"the math says no, but the heart says <i>eh, why not</i>.",
	"a tepid soup of feelings — edible, not memorable.",
}

var shipCringeLow = []string{
	"the universe just laughed and slammed the door.",
	"cupid took one look and unionized for hazard pay.",
	"this ship sank before it left the dock — the dock also sank.",
	"a pairing so cursed, even autocorrect refuses to suggest it.",
	"red flags so vibrant, NASA spotted them from orbit.",
	"this is less romance, more crime scene reenactment.",
	"please, for the love of WiFi, stay in separate group chats.",
	"the only thing connecting you two should be a restraining order.",
	"horoscope just said <i>no</i> and then said <i>NO</i> in all caps.",
	"if soulmates exist, yours both fled the country to avoid this.",
}

var shipPhases = []string{
	"locked eyes across a crowded room and immediately argued about pineapple on pizza",
	"slow-burn enemies-to-lovers arc spanning at least 47 chapters",
	"chaotic situationship featuring 3am voice notes and zero labels",
	"matching hoodies by week two, matching tattoos by month three",
	"the kind of couple that finishes each other's sandwiches",
	"a power couple running a joint TikTok account against their will",
	"long-distance pen pals for half a decade before anything happens",
	"high school sweethearts in a 90s coming-of-age montage",
	"office romance with awkward elevator small talk for 9 months",
	"meet-cute at a bookstore, fall in love over the same dog-eared paperback",
	"bickering rivals who eventually share a single umbrella in the rain",
	"app-match disaster turned best-friend roommates turned <i>oh</i>",
	"summer fling that accidentally becomes a 40-year marriage",
	"mutual pining disguised as aggressive playlist exchanges",
	"academic rivals who study together and never actually study",
	"road trip lovers who break down twice and never look back",
	"karaoke duet partners who realize halfway through they mean it",
	"cottagecore couple raising 12 chickens and a sourdough starter",
	"the messy will-they-won't-they arc that gets cancelled too soon",
	"friends-to-lovers, except it takes a decade and three weddings",
	"second-chance romance with a single dramatic airport monologue",
	"fake dating for a wedding that becomes very, very real",
	"co-conspirators in a heist movie that's actually about feelings",
	"midnight diner regulars who order the same pancakes every Sunday",
	"the couple that started as each other's <i>hate-follow</i>",
}

var shipFirstDates = []string{
	"shared a single milkshake with two straws like it's 1958",
	"got lost in an IKEA and pretended to live there for three hours",
	"stargazing on a rooftop, half a blanket, full constellation drama",
	"an aggressively competitive bowling night ending in a tie on purpose",
	"a mini-golf showdown that escalated into a windmill incident",
	"karaoke night, one shared microphone, zero shame",
	"thrift store challenge: $20, ugliest outfit wins",
	"midnight drive to nowhere with a playlist that says everything",
	"paint-and-sip turning into accidental abstract masterpieces",
	"farmer's market stroll ending in suspicious amounts of jam",
	"escape room where teamwork either saves or ends the relationship",
	"picnic in the park interrupted by a very confident duck",
	"arcade date with a stuffed-animal stakes tournament",
	"trying every flavor at the ice-cream shop and judging each one",
	"old-school drive-in movie with truly questionable popcorn",
	"cooking class disaster that doubles as a comedy special",
	"open-mic night where one of you accidentally signs up to perform",
	"bookstore date with a <i>buy each other a book blind</i> rule",
	"sunset beach walk with shells, sand, and unwanted seagull interest",
	"board game cafe marathon ending in a banned-from-Monopoly oath",
	"a haunted house tour where one of you screams the entire time",
	"baking bread together and inventing a new shape entirely",
	"a museum date narrated entirely in fake British accents",
	"late-night diner pancakes after a wholesome chaos of a day",
	"thrifted-record listening party in a tiny, tinier living room",
	"pottery class where the clay does not survive but the vibes do",
	"trampoline park giggle attack neither of you fully recovered from",
	"a chaotic Costco run with no list and infinite samples",
	"DIY pizza night with toppings that should not be legal",
	"botanical garden walk with one of you pretending to know every plant",
}

func shipPick(list []string, base uint32, salt uint32) string {
	if len(list) == 0 {
		return ""
	}
	idx := (base ^ salt) % uint32(len(list))
	return list[idx]
}

func shipLoveBar(pct int) string {
	filled := pct / 5
	if filled > 20 {
		filled = 20
	}
	if filled < 0 {
		filled = 0
	}
	var sb strings.Builder
	for i := 0; i < filled; i++ {
		sb.WriteString("❤️")
	}
	for i := filled; i < 20; i++ {
		sb.WriteString("🤍")
	}
	return sb.String()
}

func shipCringeLine(pct int, base uint32) string {
	switch {
	case pct >= 67:
		return shipPick(shipCringeHigh, base, 2654435761)
	case pct >= 34:
		return shipPick(shipCringeMid, base, 2246822519)
	default:
		return shipPick(shipCringeLow, base, 3266489917)
	}
}

func shipVerdictTitle(pct int) string {
	switch {
	case pct >= 90:
		return "WRITTEN IN THE STARS"
	case pct >= 75:
		return "DANGEROUSLY COMPATIBLE"
	case pct >= 60:
		return "VERY PROMISING"
	case pct >= 45:
		return "WORTH A SHOT"
	case pct >= 30:
		return "QUESTIONABLE BUT ENTERTAINING"
	case pct >= 15:
		return "BARELY HOLDING ON"
	default:
		return "CERTIFIED DISASTER"
	}
}

func shipResolveParty(m *tg.NewMessage, raw string) *shipParty {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	clean := strings.TrimPrefix(raw, "@")
	if id, err := strconv.ParseInt(clean, 10, 64); err == nil {
		user, err := m.Client.GetUser(id)
		if err == nil && user != nil {
			name := strings.TrimSpace(user.FirstName + " " + user.LastName)
			if name == "" {
				name = user.Username
			}
			if name == "" {
				name = strconv.FormatInt(id, 10)
			}
			return &shipParty{ID: id, Name: name}
		}
		return &shipParty{ID: id, Name: clean}
	}
	ent, err := m.Client.ResolveUsername(clean)
	if err != nil || ent == nil {
		return nil
	}
	user, ok := ent.(*tg.UserObj)
	if !ok {
		return nil
	}
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = user.Username
	}
	if name == "" {
		name = strconv.FormatInt(user.ID, 10)
	}
	return &shipParty{ID: user.ID, Name: name}
}

func shipExtractMentionParties(m *tg.NewMessage) []*shipParty {
	var out []*shipParty
	if m.Message == nil {
		return out
	}
	text := m.Message.Message
	for _, entity := range m.Message.Entities {
		switch e := entity.(type) {
		case *tg.MessageEntityMentionName:
			user, err := m.Client.GetUser(e.UserID)
			if err == nil && user != nil {
				name := strings.TrimSpace(user.FirstName + " " + user.LastName)
				if name == "" {
					name = user.Username
				}
				if name == "" {
					name = strconv.FormatInt(user.ID, 10)
				}
				out = append(out, &shipParty{ID: user.ID, Name: name})
			}
		case *tg.MessageEntityMention:
			start := int(e.Offset)
			end := start + int(e.Length)
			if start < 0 || end > len(text) || start >= end {
				continue
			}
			uname := strings.TrimPrefix(text[start:end], "@")
			p := shipResolveParty(m, uname)
			if p != nil {
				out = append(out, p)
			}
		}
	}
	return out
}

func ShipHandler(m *tg.NewMessage) error {
	usage := "<b>/ship</b> &lt;user1&gt; &lt;user2&gt;\n<i>Discover your cosmic compatibility — or your romantic apocalypse.</i>\n\nExample: <code>/ship @alice @bob</code>"
	args := strings.TrimSpace(m.Args())

	mentionParties := shipExtractMentionParties(m)
	var parties []*shipParty
	seen := map[int64]bool{}

	for _, p := range mentionParties {
		if p == nil || seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		parties = append(parties, p)
	}

	if len(parties) < 2 && args != "" {
		for _, tok := range strings.Fields(args) {
			if strings.HasPrefix(tok, "@") || isAllDigits(strings.TrimPrefix(tok, "@")) {
				p := shipResolveParty(m, tok)
				if p == nil || seen[p.ID] {
					continue
				}
				seen[p.ID] = true
				parties = append(parties, p)
				if len(parties) >= 2 {
					break
				}
			}
		}
	}

	if m.IsReply() && len(parties) < 2 {
		if r, err := m.GetReplyMessage(); err == nil && r != nil {
			rid := r.SenderID()
			if rid != 0 && !seen[rid] {
				user, err := m.Client.GetUser(rid)
				name := strconv.FormatInt(rid, 10)
				if err == nil && user != nil {
					n := strings.TrimSpace(user.FirstName + " " + user.LastName)
					if n != "" {
						name = n
					} else if user.Username != "" {
						name = user.Username
					}
				}
				seen[rid] = true
				parties = append(parties, &shipParty{ID: rid, Name: name})
			}
		}
	}

	if len(parties) < 2 {
		m.Reply(usage)
		return nil
	}

	a := parties[0]
	b := parties[1]

	if a.ID == b.ID {
		m.Reply("<b>" + html.EscapeString(a.Name) + "</b> ❤️ <b>" + html.EscapeString(a.Name) + "</b>\n\n<i>...lol no, go log off.</i>")
		return nil
	}

	ids := []int64{a.ID, b.ID}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	key := strconv.FormatInt(ids[0], 10) + ":" + strconv.FormatInt(ids[1], 10)
	hash := crc32.ChecksumIEEE([]byte(key))
	pct := int(hash % 101)

	bar := shipLoveBar(pct)
	cringe := shipCringeLine(pct, hash)
	phase := shipPick(shipPhases, hash, 16777619)
	date := shipPick(shipFirstDates, hash, 2166136261)
	title := shipVerdictTitle(pct)

	nameA := html.EscapeString(a.Name)
	nameB := html.EscapeString(b.Name)

	var sb strings.Builder
	sb.WriteString("<b>")
	sb.WriteString(nameA)
	sb.WriteString("</b>  ❤️  <b>")
	sb.WriteString(nameB)
	sb.WriteString("</b>\n\n")
	sb.WriteString("<b>Compatibility:</b> <code>")
	sb.WriteString(strconv.Itoa(pct))
	sb.WriteString("%</code>\n")
	sb.WriteString(bar)
	sb.WriteString("\n\n")
	sb.WriteString("<b>Verdict:</b> <i>")
	sb.WriteString(title)
	sb.WriteString("</i>\n")
	sb.WriteString(cringe)
	sb.WriteString("\n\n<b>Predicted phase:</b>\n")
	sb.WriteString("<i>")
	sb.WriteString(phase)
	sb.WriteString("</i>\n\n<b>First date energy:</b>\n")
	sb.WriteString("<i>")
	sb.WriteString(date)
	sb.WriteString("</i>\n\n")
	sb.WriteString("<i>Powered by sacred CRC32 math and unsolicited romantic opinions.</i>")

	m.Reply(sb.String())
	return nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func registerShipHandlers() {
	c := Client
	c.On("cmd:ship", ShipHandler)
}

func init() { QueueHandlerRegistration(registerShipHandlers) }
