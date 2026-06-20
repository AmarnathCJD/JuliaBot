package modules

import (
	"fmt"
	"html"
	"math/rand"
	"strings"
	"time"
	"unicode"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var palindromeRng = rand.New(rand.NewSource(time.Now().UnixNano()))

var palindromePhrases = []string{
	"A man, a plan, a canal: Panama",
	"Madam, in Eden, I'm Adam",
	"Was it a car or a cat I saw?",
	"No lemon, no melon",
	"Step on no pets",
	"Never odd or even",
	"Eva, can I see bees in a cave?",
	"Mr. Owl ate my metal worm",
	"Do geese see God?",
	"Rats live on no evil star",
	"Borrow or rob?",
	"Sit on a potato pan, Otis",
	"Top spot",
	"Race fast, safe car",
	"Don't nod",
	"Pull up if I pull up",
	"Was it a rat I saw?",
	"Murder for a jar of red rum",
	"A Toyota's a Toyota",
	"Yo, banana boy!",
	"Ah, Satan sees Natasha",
	"Cigar? Toss it in a can, it is so tragic",
	"Dammit, I'm mad!",
	"Go hang a salami, I'm a lasagna hog",
	"Now I won",
	"Red rum, sir, is murder",
	"Some men interpret nine memos",
	"Tarzan raised Desi Arnaz' rat",
	"Was it Eliot's toilet I saw?",
	"Eve, mad Adam, Eve!",
}

func normalizePalindrome(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func isPalindromeString(s string) bool {
	runes := []rune(s)
	n := len(runes)
	for i := 0; i < n/2; i++ {
		if runes[i] != runes[n-1-i] {
			return false
		}
	}
	return true
}

func IsPalindromeHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/ispalindrome &lt;text&gt;</code>")
		return nil
	}
	normalized := normalizePalindrome(args)
	if normalized == "" {
		m.Reply("<b>Error:</b> no alphanumeric characters to check.")
		return nil
	}
	verdict := "No"
	explanation := "The cleaned text does not read the same forwards and backwards."
	if isPalindromeString(normalized) {
		verdict = "Yes"
		explanation = "Ignoring case, spaces, and punctuation, the text reads the same forwards and backwards."
	}
	reversed := []rune(normalized)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	out := fmt.Sprintf("<b>Palindrome Check</b>\n\n<b>Input:</b> <code>%s</code>\n<b>Cleaned:</b> <code>%s</code>\n<b>Reversed:</b> <code>%s</code>\n<b>Result:</b> <i>%s</i>\n\n%s",
		html.EscapeString(args),
		html.EscapeString(normalized),
		html.EscapeString(string(reversed)),
		verdict,
		html.EscapeString(explanation),
	)
	m.Reply(out)
	return nil
}

func AnagramHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("<b>Usage:</b> <code>/anagram &lt;a&gt; &lt;b&gt;</code>")
		return nil
	}
	parts := strings.Fields(args)
	if len(parts) < 2 {
		m.Reply("<b>Usage:</b> <code>/anagram &lt;a&gt; &lt;b&gt;</code>\nProvide two words separated by a space.")
		return nil
	}
	a := parts[0]
	b := strings.Join(parts[1:], " ")
	na := normalizePalindrome(a)
	nb := normalizePalindrome(b)
	if na == "" || nb == "" {
		m.Reply("<b>Error:</b> both inputs must contain alphanumeric characters.")
		return nil
	}
	countA := map[rune]int{}
	countB := map[rune]int{}
	for _, r := range na {
		countA[r]++
	}
	for _, r := range nb {
		countB[r]++
	}
	match := len(countA) == len(countB)
	if match {
		for k, v := range countA {
			if countB[k] != v {
				match = false
				break
			}
		}
	}
	verdict := "No"
	explanation := "The two strings do not share the same letters in the same counts."
	if match {
		verdict = "Yes"
		explanation = "Both strings contain exactly the same letters with the same frequency."
	}
	out := fmt.Sprintf("<b>Anagram Check</b>\n\n<b>A:</b> <code>%s</code>\n<b>B:</b> <code>%s</code>\n<b>Result:</b> <i>%s</i>\n\n%s",
		html.EscapeString(a),
		html.EscapeString(b),
		verdict,
		html.EscapeString(explanation),
	)
	m.Reply(out)
	return nil
}

func MakePaliHandler(m *tg.NewMessage) error {
	pick := palindromePhrases[palindromeRng.Intn(len(palindromePhrases))]
	out := fmt.Sprintf("<b>Random Palindrome</b>\n\n<i>%s</i>", html.EscapeString(pick))
	m.Reply(out)
	return nil
}

func registerPalindromeHandlers() {
	c := Client
	c.On("cmd:ispalindrome", IsPalindromeHandler)
	c.On("cmd:anagram", AnagramHandler)
	c.On("cmd:makepali", MakePaliHandler)
}

func init() {
	QueueHandlerRegistration(registerPalindromeHandlers)
}
