// Package spellcheck checks spelling with the platform's spell checker:
// NSSpellChecker on macOS, UITextChecker on iOS, the Windows spell
// checker, Android's spell checker service, and Hunspell on Linux
// (loaded when first used, with the dictionaries the system has
// installed). Input uses it for its red underlines and suggestions
// (Input.SpellCheck).
//
//	for _, m := range spellcheck.Check("Teh quick brwn fox", "") {
//		fmt.Println(m.Word, spellcheck.Suggest(m.Word, ""))
//	}
package spellcheck

import (
	"strings"
	"sync"
	"unicode"
	"unicode/utf16"
)

// Misspelling is a misspelled word: runes Start to End of the text.
type Misspelling struct {
	Start, End int
	Word       string
}

var ignored sync.Map // words ignored this session

// Available reports whether lang (a BCP 47 tag like "en-US" or "fa";
// "" for the user's) can be checked here.
func Available(lang string) bool { return available(lang) }

// Check returns the misspelled words of text in lang ("" for the
// user's language), in order. Text that can't be checked returns none.
func Check(text, lang string) []Misspelling {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	ms := check(text, lang)
	out := ms[:0]
	for _, m := range ms {
		if _, ok := ignored.Load(strings.ToLower(m.Word)); ok || skip(m.Word) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// Suggest returns corrections for word, best first.
func Suggest(word, lang string) []string {
	if word == "" {
		return nil
	}
	s := suggest(word, lang)
	if len(s) > 6 {
		s = s[:6]
	}
	return s
}

// Learn adds word to the user's dictionary, where the platform keeps
// one, and to this session's.
func Learn(word string) {
	Ignore(word)
	learn(word)
}

// Ignore stops reporting word for the rest of the session.
func Ignore(word string) { ignored.Store(strings.ToLower(word), true) }

// skip leaves out what isn't a word to check: numbers, addresses,
// paths, single letters.
func skip(w string) bool {
	n := 0
	for _, r := range w {
		n++
		if unicode.IsDigit(r) || r == '@' || r == '/' || r == '.' || r == '_' {
			return true
		}
	}
	return n < 2
}

// words splits text into words, as rune ranges: letters and marks,
// with apostrophes and joiners inside a word.
func words(text string) [][2]int {
	rs := []rune(text)
	var out [][2]int
	in := func(i int) bool {
		r := rs[i]
		if unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r) || r == '‌' || r == '‍' {
			return true
		}
		// An apostrophe between letters: don't, l'homme.
		return (r == '\'' || r == '’') && i > 0 && i+1 < len(rs) && unicode.IsLetter(rs[i-1]) && unicode.IsLetter(rs[i+1])
	}
	for i := 0; i < len(rs); {
		if !in(i) {
			i++
			continue
		}
		j := i
		for j < len(rs) && in(j) {
			j++
		}
		out = append(out, [2]int{i, j})
		i = j
	}
	return out
}

// byWord checks text a word at a time with ok, for checkers of single
// words.
func byWord(text string, ok func(word string) bool) []Misspelling {
	rs := []rune(text)
	var out []Misspelling
	for _, w := range words(text) {
		word := string(rs[w[0]:w[1]])
		if skip(word) {
			continue
		}
		if !ok(word) {
			out = append(out, Misspelling{w[0], w[1], word})
		}
	}
	return out
}

// runeIndex converts UTF-16 offsets in text to rune offsets.
type runeIndex []int

func newRuneIndex(text string) runeIndex {
	var idx runeIndex
	for i, r := range []rune(text) {
		idx = append(idx, i)
		if utf16.RuneLen(r) == 2 {
			idx = append(idx, i)
		}
	}
	return append(idx, len([]rune(text)))
}

func (x runeIndex) at(u16 int) int {
	if u16 < 0 {
		return 0
	}
	if u16 >= len(x) {
		return x[len(x)-1]
	}
	return x[u16]
}

// fromUTF16 makes a Misspelling from UTF-16 offsets.
func fromUTF16(text string, x runeIndex, start, length int) Misspelling {
	a, b := x.at(start), x.at(start+length)
	return Misspelling{a, b, string([]rune(text)[a:b])}
}
