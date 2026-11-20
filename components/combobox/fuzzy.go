package combobox

import (
	"sort"
	"strings"
	"unicode"
)

// Match scores how well query matches s, letters in order but not
// necessarily together ("nwtb" finds "New tab"), ignoring case. A
// higher score is a better match: consecutive letters, letters starting
// words and an early start count for more. positions are the matched
// runes' indexes in s, for highlighting. An empty query matches
// everything with score 0.
func Match(query, s string) (score int, positions []int, ok bool) {
	q := []rune(strings.ToLower(query))
	if len(q) == 0 {
		return 0, nil, true
	}
	rs := []rune(s)
	// Greedy from each start where the first letter matches; keep the
	// best.
	best := -1 << 30
	for start := 0; start < len(rs); start++ {
		if unicode.ToLower(rs[start]) != q[0] {
			continue
		}
		sc, pos, ok := matchFrom(q, rs, start)
		if ok && sc > best {
			best, positions = sc, pos
		}
	}
	if positions == nil {
		return 0, nil, false
	}
	return best, positions, true
}

func matchFrom(q, rs []rune, start int) (int, []int, bool) {
	pos := make([]int, 0, len(q))
	score := 0
	qi := 0
	prev := -2
	for i := start; i < len(rs) && qi < len(q); i++ {
		if unicode.ToLower(rs[i]) != q[qi] {
			continue
		}
		sc := 1
		if i == prev+1 {
			sc += 5 // consecutive
		}
		if i == 0 || isBoundary(rs[i-1], rs[i]) {
			sc += 8 // starts a word
		}
		score += sc
		pos = append(pos, i)
		prev = i
		qi++
	}
	if qi < len(q) {
		return 0, nil, false
	}
	// Earlier and tighter matches are better.
	score -= start / 2
	score -= (pos[len(pos)-1] - pos[0] - len(pos) + 1) / 3
	return score, pos, true
}

func isBoundary(prev, r rune) bool {
	return !unicode.IsLetter(prev) && !unicode.IsDigit(prev) ||
		unicode.IsLower(prev) && unicode.IsUpper(r)
}

// Filter returns the options matching query, best first, with their
// matched positions set; options with equal scores keep their order.
func Filter(options []Option, query string) []Option {
	type scored struct {
		o     Option
		score int
	}
	var out []scored
	for _, o := range options {
		text := o.text()
		sc, pos, ok := Match(query, text)
		if !ok {
			// Also search the detail and keywords, without highlighting.
			for _, k := range append([]string{o.Detail}, o.Keywords...) {
				if s, _, ok2 := Match(query, k); ok2 && k != "" {
					sc, ok = s-10, true
					break
				}
			}
			if !ok {
				continue
			}
		}
		o.matched = pos
		out = append(out, scored{o, sc})
	}
	if query != "" {
		sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	}
	res := make([]Option, len(out))
	for i, s := range out {
		res[i] = s.o
	}
	return res
}
