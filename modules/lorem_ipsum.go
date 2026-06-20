package modules

import (
	"fmt"
	"html"
	"math/rand"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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
	c := Client
	c.On("cmd:lorem", LoremHandler)
}

func init() {
	QueueHandlerRegistration(registerLoremHandlers)
}
