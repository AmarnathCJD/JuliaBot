package extras

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"go.etcd.io/bbolt"
	"html"
	"io"
	modules "main/modules"
	"main/modules/db"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
)

var popularLangCodes = []struct {
	Code string
	Name string
}{
	{"en", "English"},
	{"es", "Spanish"},
	{"fr", "French"},
	{"de", "German"},
	{"it", "Italian"},
	{"pt", "Portuguese"},
	{"ru", "Russian"},
	{"zh", "Chinese"},
	{"ja", "Japanese"},
	{"ko", "Korean"},
	{"ar", "Arabic"},
	{"hi", "Hindi"},
	{"bn", "Bengali"},
	{"ta", "Tamil"},
	{"te", "Telugu"},
	{"ml", "Malayalam"},
	{"kn", "Kannada"},
	{"mr", "Marathi"},
	{"gu", "Gujarati"},
	{"pa", "Punjabi"},
	{"ur", "Urdu"},
	{"fa", "Persian"},
	{"tr", "Turkish"},
	{"nl", "Dutch"},
	{"pl", "Polish"},
	{"uk", "Ukrainian"},
	{"sv", "Swedish"},
	{"no", "Norwegian"},
	{"da", "Danish"},
	{"fi", "Finnish"},
	{"cs", "Czech"},
	{"el", "Greek"},
	{"he", "Hebrew"},
	{"th", "Thai"},
	{"vi", "Vietnamese"},
	{"id", "Indonesian"},
	{"ms", "Malay"},
	{"ro", "Romanian"},
	{"hu", "Hungarian"},
	{"sw", "Swahili"},
}

func LangsHandler(m *tg.NewMessage) error {
	var sb strings.Builder
	sb.WriteString("<b>Popular Language Codes (ISO 639-1)</b>\n\n")
	for _, l := range popularLangCodes {
		sb.WriteString(fmt.Sprintf("<code>%s</code> - %s\n", l.Code, html.EscapeString(l.Name)))
	}
	sb.WriteString("\n<i>Use with /tr &lt;code&gt; replying to a message.</i>")
	m.Reply(sb.String())
	return nil
}

func DetectHandler(m *tg.NewMessage) error {
	text := m.Args()
	if text == "" && m.IsReply() {
		r, _ := m.GetReplyMessage()
		text = r.Text()
	}
	if strings.TrimSpace(text) == "" {
		m.Reply("Provide text or reply to a message: <code>/detect &lt;text&gt;</code>")
		return nil
	}

	code, confidence, translated, err := googleDetectLang(text)
	if err != nil {
		m.Reply("Detection failed")
		return nil
	}

	name := code
	for _, l := range popularLangCodes {
		if l.Code == code {
			name = l.Name + " (" + code + ")"
			break
		}
	}

	confPct := fmt.Sprintf("%.1f%%", confidence*100)
	out := fmt.Sprintf("<b>Detected Language:</b> %s\n<b>Confidence:</b> %s\n\n<b>English:</b>\n<code>%s</code>",
		html.EscapeString(name), confPct, html.EscapeString(translated))
	m.Reply(out)
	return nil
}

func googleDetectLang(text string) (string, float64, string, error) {
	api := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=en&dt=t&dt=ld&q=%s",
		url.QueryEscape(text))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(api)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, "", err
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", 0, "", err
	}

	var translated strings.Builder
	if len(result) > 0 {
		if chunks, ok := result[0].([]interface{}); ok {
			for _, c := range chunks {
				if line, ok := c.([]interface{}); ok && len(line) > 0 {
					if s, ok := line[0].(string); ok {
						translated.WriteString(s)
					}
				}
			}
		}
	}

	code := "unknown"
	if len(result) > 2 {
		if s, ok := result[2].(string); ok {
			code = s
		}
	}

	confidence := 0.0
	for i := len(result) - 1; i >= 0; i-- {
		if arr, ok := result[i].([]interface{}); ok && len(arr) >= 2 {
			codesArr, okA := arr[0].([]interface{})
			confsArr, okB := arr[len(arr)-1].([]interface{})
			if okA && okB && len(codesArr) > 0 && len(confsArr) > 0 {
				if _, ok := codesArr[0].(string); ok {
					if f, ok := confsArr[0].(float64); ok {
						confidence = f
						break
					}
				}
			}
		}
	}

	return code, confidence, translated.String(), nil
}

func registerTranslate2Handlers() {
	c := modules.Client
	c.On("cmd:langs", LangsHandler)
	c.On("cmd:detect", DetectHandler)
}

func initFromSrc_translate2_0_1() {
	modules.QueueHandlerRegistration(registerTranslate2Handlers)
}
var jargonToSimple = map[string]string{
	"utilize":        "use",
	"utilization":    "use",
	"utilized":       "used",
	"utilizes":       "uses",
	"commence":       "start",
	"commenced":      "started",
	"commencement":   "start",
	"terminate":      "end",
	"terminated":     "ended",
	"termination":    "end",
	"facilitate":     "help",
	"facilitated":    "helped",
	"facilitates":    "helps",
	"facilitation":   "help",
	"endeavor":       "try",
	"endeavour":      "try",
	"endeavored":     "tried",
	"demonstrate":    "show",
	"demonstrated":   "shown",
	"demonstrates":   "shows",
	"demonstration":  "show",
	"approximately":  "about",
	"sufficient":     "enough",
	"sufficiently":   "enough",
	"insufficient":   "not enough",
	"subsequently":   "later",
	"subsequent":     "next",
	"previously":     "before",
	"previous":       "earlier",
	"currently":      "now",
	"current":        "now",
	"presently":      "now",
	"immediately":    "now",
	"immediate":      "quick",
	"frequently":     "often",
	"frequent":       "often",
	"occasionally":   "sometimes",
	"occasional":     "rare",
	"additionally":   "also",
	"additional":     "more",
	"furthermore":    "also",
	"moreover":       "also",
	"however":        "but",
	"nevertheless":   "but",
	"nonetheless":    "still",
	"therefore":      "so",
	"thus":           "so",
	"hence":          "so",
	"accordingly":    "so",
	"consequently":   "so",
	"regarding":      "about",
	"concerning":     "about",
	"pertaining":     "about",
	"abundant":       "plenty",
	"abundance":      "plenty",
	"accomplish":     "do",
	"accomplished":   "done",
	"accomplishment": "feat",
	"acquire":        "get",
	"acquired":       "got",
	"acquisition":    "buy",
	"adequate":       "enough",
	"adequately":     "enough",
	"advantageous":   "useful",
	"advantage":      "edge",
	"aggregate":      "total",
	"aggregation":    "total",
	"allocate":       "give",
	"allocated":      "given",
	"allocation":     "share",
	"alternative":    "other",
	"alternatively":  "or",
	"anticipate":     "expect",
	"anticipated":    "expected",
	"apparent":       "clear",
	"apparently":     "seems",
	"appropriate":    "right",
	"appropriately":  "right",
	"ascertain":      "find out",
	"ascertained":    "found",
	"assistance":     "help",
	"assist":         "help",
	"assisted":       "helped",
	"attempt":        "try",
	"attempted":      "tried",
	"attribute":      "trait",
	"attributes":     "traits",
	"beneficial":     "useful",
	"benefit":        "perk",
	"capability":     "skill",
	"capable":        "able",
	"clarification":  "clarity",
	"clarify":        "explain",
	"commensurate":   "equal",
	"comparable":     "similar",
	"compensation":   "pay",
	"comprehend":     "understand",
	"comprehension":  "grasp",
	"comprehensive":  "full",
	"comprise":       "include",
	"comprised":      "made of",
	"comprises":      "includes",
	"concurrent":     "joint",
	"concurrently":   "together",
	"considerable":   "big",
	"considerably":   "much",
	"construct":      "build",
	"constructed":    "built",
	"construction":   "build",
	"deliberate":     "planned",
	"deliberately":   "on purpose",
	"determine":      "decide",
	"determined":     "decided",
	"discontinue":    "stop",
	"discontinued":   "stopped",
	"disseminate":    "share",
	"dissemination":  "spread",
	"distribute":     "give out",
	"distributed":    "shared",
	"distribution":   "spread",
	"duplicate":      "copy",
	"duplicated":     "copied",
	"duplication":    "copy",
	"effectuate":     "cause",
	"elaborate":      "detail",
	"elaborated":     "explained",
	"eliminate":      "remove",
	"eliminated":     "removed",
	"elimination":    "removal",
	"encounter":      "meet",
	"encountered":    "met",
	"enhance":        "improve",
	"enhanced":       "improved",
	"enhancement":    "boost",
	"enumerate":      "list",
	"enumerated":     "listed",
	"equivalent":     "equal",
	"establish":      "set up",
	"established":    "set up",
	"evaluate":       "judge",
	"evaluated":      "judged",
	"evaluation":     "review",
	"exemplify":      "show",
	"expedite":       "speed up",
	"expedited":      "rushed",
	"expenditure":    "cost",
	"feasible":       "doable",
	"finalize":       "finish",
	"finalized":      "done",
	"fundamental":    "basic",
	"generate":       "make",
	"generated":      "made",
	"generation":     "making",
	"identify":       "spot",
	"identified":     "spotted",
	"identification": "id",
	"illustrate":     "show",
	"illustrated":    "shown",
	"illustration":   "image",
	"implement":      "do",
	"implemented":    "done",
	"implementation": "rollout",
	"indicate":       "show",
	"indicated":      "shown",
	"indication":     "sign",
	"individual":     "person",
	"individuals":    "people",
	"initiate":       "start",
	"initiated":      "started",
	"initiation":     "start",
	"inquire":        "ask",
	"inquiry":        "question",
	"institute":      "start",
	"instituted":     "set up",
	"investigate":    "look into",
	"investigated":   "checked",
	"investigation":  "probe",
	"locate":         "find",
	"located":        "found",
	"location":       "spot",
	"magnitude":      "size",
	"maintain":       "keep",
	"maintained":     "kept",
	"manufacture":    "make",
	"manufactured":   "made",
	"methodology":    "method",
	"minimize":       "lessen",
	"minimized":      "lessened",
	"modification":   "change",
	"modify":         "change",
	"modified":       "changed",
	"necessitate":    "require",
	"necessary":      "needed",
	"notwithstanding": "despite",
	"numerous":       "many",
	"objective":      "goal",
	"obligation":     "duty",
	"observation":    "view",
	"observe":        "see",
	"observed":       "seen",
	"obtain":         "get",
	"obtained":       "got",
	"operate":        "run",
	"operated":       "ran",
	"operation":      "run",
	"optimal":        "best",
	"optimize":       "improve",
	"optimized":      "tuned",
	"option":         "choice",
	"optional":       "extra",
	"originate":      "begin",
	"originated":     "began",
	"participate":    "join",
	"participated":   "joined",
	"participation":  "joining",
	"perceive":       "see",
	"perceived":      "seen",
	"perception":     "view",
	"perform":        "do",
	"performed":      "did",
	"performance":    "result",
	"permit":         "let",
	"permitted":      "allowed",
	"permission":     "ok",
	"persistent":     "lasting",
	"portion":        "part",
	"possess":        "own",
	"possessed":      "owned",
	"possession":     "owning",
	"potential":      "possible",
	"preliminary":    "early",
	"primary":        "main",
	"prioritize":     "rank",
	"procedure":      "step",
	"procure":        "get",
	"procured":       "got",
	"prohibit":       "ban",
	"prohibited":     "banned",
	"prohibition":    "ban",
	"purchase":       "buy",
	"purchased":      "bought",
	"quantify":       "count",
	"reside":         "live",
	"resided":        "lived",
	"residence":      "home",
	"resolve":        "fix",
	"resolved":       "fixed",
	"resolution":     "fix",
	"retain":         "keep",
	"retained":       "kept",
	"select":         "pick",
	"selected":       "picked",
	"selection":      "pick",
	"significant":    "big",
	"significantly":  "much",
	"specify":        "list",
	"specified":      "listed",
	"strategy":       "plan",
	"submit":         "send",
	"submitted":      "sent",
	"sustain":        "keep",
	"sustained":      "kept",
	"transmit":       "send",
	"transmitted":    "sent",
	"transmission":   "send",
	"transparent":    "clear",
	"undertake":      "do",
	"undertook":      "did",
	"validate":       "check",
	"validated":      "checked",
	"validation":     "check",
	"verify":         "check",
	"verified":       "checked",
	"verification":   "check",
	"vicinity":       "area",
	"warrant":        "need",
	"warranted":      "needed",
}

var simpleToFancy = map[string]string{
	"use":        "utilize",
	"used":       "utilized",
	"uses":       "utilizes",
	"start":      "commence",
	"started":    "commenced",
	"starts":     "commences",
	"end":        "terminate",
	"ended":      "terminated",
	"ends":       "terminates",
	"help":       "facilitate",
	"helped":     "facilitated",
	"helps":      "facilitates",
	"try":        "endeavor",
	"tried":      "endeavored",
	"tries":      "endeavors",
	"show":       "demonstrate",
	"showed":     "demonstrated",
	"shown":      "demonstrated",
	"shows":      "demonstrates",
	"about":      "regarding",
	"enough":     "sufficient",
	"later":      "subsequently",
	"next":       "subsequent",
	"before":     "previously",
	"now":        "presently",
	"often":      "frequently",
	"sometimes":  "occasionally",
	"also":       "additionally",
	"more":       "additional",
	"but":        "however",
	"so":         "therefore",
	"plenty":     "abundance",
	"do":         "perform",
	"did":        "performed",
	"done":       "accomplished",
	"get":        "acquire",
	"got":        "acquired",
	"gets":       "acquires",
	"give":       "allocate",
	"gave":       "allocated",
	"gives":      "allocates",
	"other":      "alternative",
	"expect":     "anticipate",
	"expected":   "anticipated",
	"clear":      "transparent",
	"right":      "appropriate",
	"big":        "substantial",
	"small":      "diminutive",
	"fast":       "expeditious",
	"slow":       "lethargic",
	"build":      "construct",
	"built":      "constructed",
	"builds":     "constructs",
	"decide":     "determine",
	"decided":    "determined",
	"stop":       "discontinue",
	"stopped":    "discontinued",
	"share":      "disseminate",
	"shared":     "disseminated",
	"copy":       "duplicate",
	"copied":     "duplicated",
	"copies":     "duplicates",
	"remove":     "eliminate",
	"removed":    "eliminated",
	"meet":       "encounter",
	"met":        "encountered",
	"improve":    "enhance",
	"improved":   "enhanced",
	"list":       "enumerate",
	"listed":     "enumerated",
	"equal":      "equivalent",
	"judge":      "evaluate",
	"judged":     "evaluated",
	"finish":     "finalize",
	"finished":   "finalized",
	"basic":      "fundamental",
	"make":       "generate",
	"made":       "generated",
	"makes":      "generates",
	"spot":       "identify",
	"spotted":    "identified",
	"join":       "participate",
	"joined":     "participated",
	"see":        "observe",
	"saw":        "observed",
	"seen":       "observed",
	"sees":       "observes",
	"run":        "operate",
	"ran":        "operated",
	"runs":       "operates",
	"best":       "optimal",
	"choice":     "option",
	"begin":      "originate",
	"began":      "originated",
	"begins":     "originates",
	"let":        "permit",
	"allowed":    "permitted",
	"own":        "possess",
	"owned":      "possessed",
	"owns":       "possesses",
	"possible":   "potential",
	"main":       "primary",
	"step":       "procedure",
	"ban":        "prohibit",
	"banned":     "prohibited",
	"buy":        "procure",
	"bought":     "procured",
	"buys":       "procures",
	"live":       "reside",
	"lived":      "resided",
	"lives":      "resides",
	"home":       "residence",
	"fix":        "resolve",
	"fixed":      "resolved",
	"fixes":      "resolves",
	"keep":       "retain",
	"kept":       "retained",
	"keeps":      "retains",
	"pick":       "select",
	"picked":     "selected",
	"picks":      "selects",
	"plan":       "strategy",
	"send":       "transmit",
	"sent":       "transmitted",
	"sends":      "transmits",
	"check":      "verify",
	"checked":    "verified",
	"checks":     "verifies",
	"area":       "vicinity",
	"need":       "require",
	"needed":     "required",
	"needs":      "requires",
	"happy":      "elated",
	"sad":        "melancholy",
	"angry":      "irate",
	"tired":      "fatigued",
	"smart":      "perspicacious",
	"smell":      "olfactory sensation",
	"think":      "contemplate",
	"thought":    "contemplated",
	"thinks":     "contemplates",
	"talk":       "converse",
	"talked":     "conversed",
	"talks":      "converses",
	"walk":       "perambulate",
	"walked":     "perambulated",
	"walks":      "perambulates",
	"eat":        "consume",
	"ate":        "consumed",
	"eats":       "consumes",
	"drink":      "imbibe",
	"drank":      "imbibed",
	"drinks":     "imbibes",
	"sleep":      "slumber",
	"slept":      "slumbered",
	"sleeps":     "slumbers",
}

var pirateMap = map[string]string{
	"hello":     "ahoy",
	"hi":        "ahoy",
	"hey":       "arrr",
	"my":        "me",
	"friend":    "matey",
	"friends":   "mateys",
	"yes":       "aye",
	"no":        "nay",
	"is":        "be",
	"are":       "be",
	"am":        "be",
	"the":       "thar",
	"you":       "ye",
	"your":      "yer",
	"yours":     "yers",
	"you're":    "ye be",
	"there":     "thar",
	"over":      "o'er",
	"and":       "an'",
	"of":        "o'",
	"to":        "t'",
	"with":      "wit'",
	"old":       "ol'",
	"good":      "fine",
	"man":       "scallywag",
	"woman":     "lass",
	"men":       "scallywags",
	"women":     "lasses",
	"boy":       "lad",
	"girl":      "lass",
	"boys":      "lads",
	"girls":     "lasses",
	"money":     "doubloons",
	"gold":      "booty",
	"treasure":  "booty",
	"drink":     "grog",
	"drinks":    "grog",
	"beer":      "grog",
	"rum":       "grog",
	"food":      "grub",
	"ship":      "vessel",
	"boat":      "vessel",
	"ocean":     "sea",
	"sea":       "briny deep",
	"flag":      "jolly roger",
	"police":    "navy",
	"officer":   "scallywag",
	"sir":       "cap'n",
	"madam":     "lass",
	"crazy":     "barmy",
	"angry":     "scurvy",
	"stupid":    "scurvy",
	"awesome":   "ship-shape",
	"great":     "grand",
	"amazing":   "splendid",
	"foolish":   "lily-livered",
	"coward":    "lily-livered",
	"strong":    "hearty",
	"brave":     "hearty",
	"weak":      "lily-livered",
	"happy":     "merry",
	"sad":       "downhearted",
	"laugh":     "guffaw",
	"laughed":   "guffawed",
	"yelling":   "bellowin'",
	"shouting":  "bellowin'",
	"shout":     "bellow",
	"shouted":   "bellowed",
	"home":      "port",
	"family":    "crew",
	"team":      "crew",
	"group":     "crew",
	"work":      "swab",
	"working":   "swabbin'",
	"worked":    "swabbed",
	"fight":     "duel",
	"fighting":  "battlin'",
	"fought":    "battled",
	"steal":     "plunder",
	"stole":     "plundered",
	"stealing":  "plunderin'",
	"running":   "runnin'",
	"jumping":   "jumpin'",
	"talking":   "yammerin'",
	"singing":   "singin'",
	"sailing":   "sailin'",
	"fishing":   "fishin'",
	"swimming":  "swimmin'",
	"drinking":  "guzzlin'",
	"eating":    "feastin'",
	"sleeping":  "snorin'",
	"morning":   "mornin'",
	"evening":   "evenin'",
	"nothing":   "nothin'",
	"something": "somethin'",
	"anything":  "anythin'",
	"hello!":    "ahoy!",
	"thanks":    "much obliged",
	"thank":     "thank ye",
	"sorry":     "beggin' yer pardon",
	"please":    "pray",
	"goodbye":   "farewell",
	"bye":       "fare thee well",
	"city":      "port",
	"town":      "port",
	"king":      "cap'n",
	"queen":     "cap'n's wife",
	"boss":      "cap'n",
	"leader":    "cap'n",
}

var shakespeareMap = map[string]string{
	"you":       "thou",
	"your":      "thy",
	"yours":     "thine",
	"yourself":  "thyself",
	"you're":    "thou art",
	"you've":    "thou hast",
	"you'll":    "thou shalt",
	"you'd":     "thou wouldst",
	"are":       "art",
	"is":        "doth be",
	"am":        "be",
	"have":      "hast",
	"has":       "hath",
	"had":       "hadst",
	"do":        "dost",
	"does":      "doth",
	"did":       "didst",
	"will":      "shalt",
	"would":     "wouldst",
	"could":     "couldst",
	"should":    "shouldst",
	"can":       "canst",
	"may":       "mayst",
	"might":     "mightst",
	"hello":     "hail",
	"hi":        "hail",
	"hey":       "hark",
	"goodbye":   "fare thee well",
	"bye":       "farewell",
	"yes":       "aye",
	"no":        "nay",
	"my":        "mine",
	"sir":       "good sir",
	"madam":     "good lady",
	"man":       "gentleman",
	"woman":     "lady",
	"friend":    "good fellow",
	"hello!":    "hail!",
	"please":    "prithee",
	"truly":     "verily",
	"really":    "verily",
	"indeed":    "forsooth",
	"perhaps":   "mayhap",
	"maybe":     "mayhap",
	"before":    "ere",
	"earlier":   "ere",
	"between":   "betwixt",
	"often":     "oft",
	"over":      "o'er",
	"ever":      "e'er",
	"never":     "ne'er",
	"thanks":    "much gratitude",
	"thank":     "thank thee",
	"sorry":     "I beg thy pardon",
	"problem":   "quandary",
	"trouble":   "vexation",
	"crazy":     "mad",
	"angry":     "wroth",
	"happy":     "merry",
	"sad":       "forlorn",
	"good":      "fair",
	"bad":       "foul",
	"great":     "wondrous",
	"awesome":   "wondrous",
	"amazing":   "marvelous",
	"beautiful": "fair",
	"ugly":      "loathsome",
	"smart":     "wise",
	"stupid":    "foolish",
	"speak":     "speaketh",
	"speaks":    "speaketh",
	"think":     "methinks",
	"thinks":    "methinks",
	"go":        "goest",
	"goes":      "goeth",
	"come":      "comest",
	"comes":     "cometh",
	"see":       "seest",
	"sees":      "seeth",
	"know":      "knowest",
	"knows":     "knoweth",
	"say":       "sayest",
	"says":      "sayeth",
	"said":      "didst say",
	"tell":      "tellest",
	"tells":     "telleth",
	"told":      "didst tell",
	"give":      "givest",
	"gives":     "giveth",
	"gave":      "didst give",
	"take":      "takest",
	"takes":     "taketh",
	"took":      "didst take",
	"want":      "desire",
	"wants":     "desireth",
	"wanted":    "didst desire",
	"love":      "adore",
	"loves":     "adoreth",
	"loved":     "didst adore",
	"hate":      "loathe",
	"hates":     "loatheth",
	"need":      "require",
	"needs":     "requireth",
	"work":      "labor",
	"works":     "laboreth",
	"fight":     "duel",
	"fights":    "dueleth",
	"king":      "sovereign",
	"queen":     "majesty",
	"prince":    "noble",
	"princess":  "fair maiden",
	"girl":      "maiden",
	"boy":       "lad",
	"home":      "abode",
	"house":     "manor",
	"food":      "feast",
	"drink":     "draught",
	"money":     "coin",
	"sword":     "blade",
	"horse":     "steed",
	"city":      "kingdom",
	"town":      "village",
}

func emTransformWord(w string, mapping map[string]string) string {
	if w == "" {
		return w
	}
	stripped := w
	leading := ""
	trailing := ""
	for len(stripped) > 0 {
		r := rune(stripped[0])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' {
			break
		}
		leading += string(stripped[0])
		stripped = stripped[1:]
	}
	for len(stripped) > 0 {
		r := rune(stripped[len(stripped)-1])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' {
			break
		}
		trailing = string(stripped[len(stripped)-1]) + trailing
		stripped = stripped[:len(stripped)-1]
	}
	if stripped == "" {
		return w
	}
	lower := strings.ToLower(stripped)
	replacement, ok := mapping[lower]
	if !ok {
		return w
	}
	final := emMatchCase(stripped, replacement)
	return leading + final + trailing
}

func emMatchCase(orig, repl string) string {
	if orig == "" {
		return repl
	}
	allUpper := true
	for _, r := range orig {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			allUpper = false
			break
		}
	}
	if allUpper && len(orig) > 1 {
		return strings.ToUpper(repl)
	}
	firstRune := rune(orig[0])
	if unicode.IsUpper(firstRune) {
		runes := []rune(repl)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			return string(runes)
		}
	}
	return repl
}

func emTransform(text string, mapping map[string]string) string {
	words := strings.Fields(text)
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = emTransformWord(w, mapping)
	}
	return strings.Join(out, " ")
}

func emGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func DejargonHandler(m *tg.NewMessage) error {
	text := emGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/dejargon &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := emTransform(text, jargonToSimple)
	m.Reply("<b>Plain:</b> " + html.EscapeString(out))
	return nil
}

func UpgradeHandler(m *tg.NewMessage) error {
	text := emGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/upgrade &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := emTransform(text, simpleToFancy)
	m.Reply("<b>Fancy:</b> " + html.EscapeString(out))
	return nil
}

func PirateHandler(m *tg.NewMessage) error {
	text := emGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/pirate &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := emTransform(text, pirateMap)
	suffixes := []string{" Arrr!", " Yo ho ho!", " Shiver me timbers!", " Avast!", " Yarrr!"}
	idx := len(text) % len(suffixes)
	out = out + suffixes[idx]
	m.Reply("<b>Pirate:</b> " + html.EscapeString(out))
	return nil
}

func BardSpeakHandler(m *tg.NewMessage) error {
	text := emGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/bardspeak &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := emTransform(text, shakespeareMap)
	suffixes := []string{" Forsooth!", " Verily!", " Anon!", " Hark!", " 'Tis so!"}
	idx := len(text) % len(suffixes)
	out = out + suffixes[idx]
	m.Reply("<b>Bard:</b> " + html.EscapeString(out))
	return nil
}

func registerTranslateEmojiHandlers() {
	c := modules.Client
	c.On("cmd:dejargon", DejargonHandler)
	c.On("cmd:upgrade", UpgradeHandler)
	c.On("cmd:pirate", PirateHandler)
	c.On("cmd:bardspeak", BardSpeakHandler)
}

func initFromSrc_translate_emoji_1_1() {
	modules.QueueHandlerRegistration(registerTranslateEmojiHandlers)
}
var owoFaces = []string{
	"OwO", "UwU", "owo", "uwu", ">w<", "^w^", ":3", "x3", ">_<", "nya~",
}

var uwuFaces = []string{
	"UwU", "uwu", "OwO", ">w<", "^w^", "rawr x3", "nyaa~~", ":3", "x3",
	"(◕ᴗ◕✿)", "(*≧ω≦*)", "(´｡• ω •｡`)", ">.<", "ʕ•ᴥ•ʕ", "( ˘ ³˘)♥",
}

var catSuffixes = []string{
	"nya", "nyaa", "meow", "mrrp", "purr", "nya~", "meow~", "mrow",
	":3", "purrr", "nyaaa", "mreow",
}

func tsGetInput(m *tg.NewMessage) string {
	text := strings.TrimSpace(m.Args())
	if text == "" && m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil {
			text = strings.TrimSpace(r.Text())
		}
	}
	return text
}

func tsOwofyTransform(text string, strong bool) string {
	var b strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		switch r {
		case 'r', 'l':
			b.WriteRune('w')
		case 'R', 'L':
			b.WriteRune('W')
		case 'n', 'N':
			if i+1 < len(runes) {
				next := runes[i+1]
				if next == 'a' || next == 'o' || next == 'u' || next == 'e' || next == 'i' ||
					next == 'A' || next == 'O' || next == 'U' || next == 'E' || next == 'I' {
					b.WriteRune(r)
					b.WriteRune('y')
					continue
				}
			}
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	result := b.String()
	if strong {
		result = tsStutter(result)
	}
	return result
}

func tsStutter(text string) string {
	words := strings.Fields(text)
	out := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) > 2 && unicode.IsLetter(rune(w[0])) && rand.Intn(100) < 35 {
			first := string(w[0])
			out = append(out, first+"-"+w)
		} else {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

func tsSprinkleFaces(text string, faces []string, every int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text + " " + faces[rand.Intn(len(faces))]
	}
	out := make([]string, 0, len(words)+4)
	for i, w := range words {
		out = append(out, w)
		if (i+1)%every == 0 && i != len(words)-1 {
			out = append(out, faces[rand.Intn(len(faces))])
		}
	}
	out = append(out, faces[rand.Intn(len(faces))])
	return strings.Join(out, " ")
}

func tsCatspeak(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text + " " + catSuffixes[rand.Intn(len(catSuffixes))]
	}
	out := make([]string, 0, len(words)+4)
	for i, w := range words {
		ww := w
		runes := []rune(ww)
		if len(runes) > 0 {
			last := runes[len(runes)-1]
			if last == '.' || last == '!' || last == '?' {
				core := string(runes[:len(runes)-1])
				ww = core + " " + catSuffixes[rand.Intn(len(catSuffixes))] + string(last)
			} else if rand.Intn(100) < 25 {
				ww = ww + "~"
			}
		}
		out = append(out, ww)
		if (i+1)%4 == 0 && i != len(words)-1 {
			out = append(out, catSuffixes[rand.Intn(len(catSuffixes))])
		}
	}
	out = append(out, catSuffixes[rand.Intn(len(catSuffixes))]+"!")
	return strings.Join(out, " ")
}

func OwofyHandler(m *tg.NewMessage) error {
	text := tsGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/owofy &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := tsOwofyTransform(text, false)
	out = tsSprinkleFaces(out, owoFaces, 5)
	m.Reply("<b>OwO:</b> " + html.EscapeString(out))
	return nil
}

func UwuHandler(m *tg.NewMessage) error {
	text := tsGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/uwu &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := tsOwofyTransform(text, true)
	out = tsSprinkleFaces(out, uwuFaces, 3)
	m.Reply("<b>UwU:</b> " + html.EscapeString(out))
	return nil
}

func CatspeakHandler(m *tg.NewMessage) error {
	text := tsGetInput(m)
	if text == "" {
		m.Reply("<b>Usage:</b> <code>/catspeak &lt;text&gt;</code> or reply to a message.")
		return nil
	}
	out := tsCatspeak(text)
	m.Reply("<b>Catspeak:</b> " + html.EscapeString(out))
	return nil
}

func registerTranslateSpecialHandlers() {
	c := modules.Client
	c.On("cmd:owofy", OwofyHandler)
	c.On("cmd:uwu", UwuHandler)
	c.On("cmd:catspeak", CatspeakHandler)
}

func initFromSrc_translate_special_2_1() {
	modules.QueueHandlerRegistration(registerTranslateSpecialHandlers)
}
func TranslateHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("Reply to a message to translate it")
		return nil
	}

	args := m.Args()
	targetLang := "en"
	replaceMode := false

	if args != "" {
		parts := strings.Fields(args)
		for _, p := range parts {
			if p == "-r" {
				replaceMode = true
			} else {
				targetLang = p
			}
		}
	}

	r, _ := m.GetReplyMessage()
	text := r.Text()
	if text == "" {
		m.Reply("No text to translate")
		return nil
	}

	translated, src, err := googleTranslate(text, targetLang)
	if err != nil {
		m.Reply("Translation failed")
		return nil
	}

	if replaceMode && modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "delete") {
		r.Delete()
		m.Delete()
		m.Respond(fmt.Sprintf("<b>Translated from %s:</b>\n%s", src, translated))
	} else {
		m.Reply(fmt.Sprintf("<b>Translated (%s -> %s):</b>\n<code>%s</code>", src, targetLang, translated))
	}

	return nil
}

func googleTranslate(text, target string) (string, string, error) {
	api := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=%s&dt=t&q=%s",
		target, url.QueryEscape(text))

	resp, err := http.Get(api)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Simplify JSON parsing for [ [ ["trans", "orig",..] ], .. , "src" ]
	// Just strict parsing is hard with random mixed types, use interface{}
	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	if len(result) > 0 {
		chunks := result[0].([]interface{})
		var sb strings.Builder
		for _, c := range chunks {
			line := c.([]interface{})
			if len(line) > 0 {
				sb.WriteString(line[0].(string))
			}
		}

		src := "unknown"
		if len(result) > 2 {
			src = result[2].(string)
		}
		return sb.String(), src, nil
	}

	return "", "", fmt.Errorf("no result")
}

func registerTranslatorHandlers() {
	c := modules.Client
	c.On("cmd:tr", TranslateHandler)
}

func initFromSrc_translator_3_1() {
	modules.QueueHandlerRegistration(registerTranslatorHandlers)

	modules.Mods.AddModule("Translator", `<b>Translator Module</b>
	
Commands:
- /tr <lang> [-r]: Translate reply. -r replaces original.`)
}
type autotrConfig struct {
	Enabled bool   `json:"e"`
	Lang    string `json:"l"`
	Min     int    `json:"m"`
}

var (
	autotrBucket = []byte("autotr")
	autotrCache  = make(map[int64]*autotrConfig)
	autotrMu     sync.RWMutex
)

func autotrChatKey(chatID int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(chatID))
	return b
}

func autotrLoad(chatID int64) *autotrConfig {
	autotrMu.RLock()
	if c, ok := autotrCache[chatID]; ok {
		autotrMu.RUnlock()
		return c
	}
	autotrMu.RUnlock()

	cfg := &autotrConfig{Enabled: false, Lang: "en", Min: 4}
	database, err := db.GetDB()
	if err != nil || database == nil {
		return cfg
	}
	_ = database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(autotrBucket)
		if b == nil {
			return nil
		}
		raw := b.Get(autotrChatKey(chatID))
		if raw == nil {
			return nil
		}
		var c autotrConfig
		if err := json.Unmarshal(raw, &c); err == nil {
			if c.Lang == "" {
				c.Lang = "en"
			}
			if c.Min <= 0 {
				c.Min = 4
			}
			cfg = &c
		}
		return nil
	})
	autotrMu.Lock()
	autotrCache[chatID] = cfg
	autotrMu.Unlock()
	return cfg
}

func autotrSave(chatID int64, cfg *autotrConfig) error {
	database, err := db.GetDB()
	if err != nil || database == nil {
		return fmt.Errorf("db unavailable")
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	err = database.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(autotrBucket)
		if err != nil {
			return err
		}
		return b.Put(autotrChatKey(chatID), data)
	})
	if err == nil {
		autotrMu.Lock()
		autotrCache[chatID] = cfg
		autotrMu.Unlock()
	}
	return err
}

func AutoTrHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		m.Reply("<b>Auto-translate works in groups only.</b>")
		return nil
	}
	if !modules.IsUserAdmin(m.Client, m.SenderID(), m.ChatID(), "") {
		return nil
	}

	args := strings.TrimSpace(m.Args())
	cfg := autotrLoad(m.ChatID())

	if args == "" || args == "status" {
		state := "off"
		if cfg.Enabled {
			state = "on"
		}
		m.Reply(fmt.Sprintf("<b>Auto-Translate</b>\n • State: <code>%s</code>\n • Lang: <code>%s</code>\n • Min chars: <code>%d</code>\n\n<i>Usage:</i>\n <code>/autotr on|off</code>\n <code>/autotr lang &lt;iso&gt;</code>\n <code>/autotr min &lt;chars&gt;</code>",
			state, html.EscapeString(cfg.Lang), cfg.Min))
		return nil
	}

	parts := strings.Fields(args)
	sub := strings.ToLower(parts[0])

	switch sub {
	case "on", "enable":
		cfg.Enabled = true
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Auto-translate enabled.</b> Target: <code>%s</code>", html.EscapeString(cfg.Lang)))
	case "off", "disable":
		cfg.Enabled = false
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply("<b>Auto-translate disabled.</b>")
	case "lang", "language":
		if len(parts) < 2 {
			m.Reply("<b>Usage:</b> <code>/autotr lang &lt;iso&gt;</code>")
			return nil
		}
		lang := strings.ToLower(strings.TrimSpace(parts[1]))
		if len(lang) < 2 || len(lang) > 8 {
			m.Reply("<b>Invalid language code.</b>")
			return nil
		}
		cfg.Lang = lang
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Target language set to</b> <code>%s</code>", html.EscapeString(lang)))
	case "min":
		if len(parts) < 2 {
			m.Reply("<b>Usage:</b> <code>/autotr min &lt;chars&gt;</code>")
			return nil
		}
		var n int
		_, err := fmt.Sscanf(parts[1], "%d", &n)
		if err != nil || n < 1 || n > 4096 {
			m.Reply("<b>Invalid number.</b> Must be between 1 and 4096.")
			return nil
		}
		cfg.Min = n
		if err := autotrSave(m.ChatID(), cfg); err != nil {
			m.Reply("<b>Failed to save settings.</b>")
			return nil
		}
		m.Reply(fmt.Sprintf("<b>Minimum length set to</b> <code>%d</code>", n))
	default:
		m.Reply("<b>Unknown subcommand.</b> Use <code>on</code>, <code>off</code>, <code>lang</code>, <code>min</code>, or <code>status</code>.")
	}
	return nil
}

func AutoTrWatcher(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	if m.Sender != nil && m.Sender.Bot {
		return nil
	}
	if m.Message != nil && m.Message.ViaBotID != 0 {
		return nil
	}
	text := strings.TrimSpace(m.Text())
	if text == "" {
		return nil
	}
	if t := text; len(t) > 0 && (t[0] == '/' || t[0] == '!' || t[0] == '.') {
		return nil
	}

	cfg := autotrLoad(m.ChatID())
	if !cfg.Enabled {
		return nil
	}
	if len([]rune(text)) < cfg.Min {
		return nil
	}

	translated, src, err := googleTranslate(text, cfg.Lang)
	if err != nil || translated == "" {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(src), strings.TrimSpace(cfg.Lang)) {
		return nil
	}
	if strings.TrimSpace(translated) == strings.TrimSpace(text) {
		return nil
	}

	m.Reply(fmt.Sprintf("<blockquote><i>%s→%s</i> %s</blockquote>",
		html.EscapeString(src), html.EscapeString(cfg.Lang), html.EscapeString(translated)))
	return nil
}

func registerAutoTranslateHandlers() {
	c := modules.Client
	c.On("cmd:autotr", AutoTrHandler)
	c.On(tg.OnNewMessage, AutoTrWatcher)

	modules.Mods.AddModule("AutoTranslate", `<b>Auto-Translate Module</b>

<b>Commands:</b>
 • /autotr on|off - Toggle auto-translation for this chat
 • /autotr lang &lt;iso&gt; - Set target language (default: en)
 • /autotr min &lt;chars&gt; - Set minimum message length (default: 4)
 • /autotr status - Show current settings

<i>Admin only. Skips bots, commands, and short messages.</i>`)
}

func initFromSrc_autotranslate_4_1() {
	modules.QueueHandlerRegistration(registerAutoTranslateHandlers)
}

func init() {
	initFromSrc_translate2_0_1()
	initFromSrc_translate_emoji_1_1()
	initFromSrc_translate_special_2_1()
	initFromSrc_translator_3_1()
	initFromSrc_autotranslate_4_1()
}
