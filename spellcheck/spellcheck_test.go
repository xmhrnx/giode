package spellcheck

import (
	"slices"
	"testing"
)

func TestWords(t *testing.T) {
	text := "Don't stop — سلام‌ها, l'homme 42x"
	var got []string
	rs := []rune(text)
	for _, w := range words(text) {
		got = append(got, string(rs[w[0]:w[1]]))
	}
	want := []string{"Don't", "stop", "سلام‌ها", "l'homme", "42x"}
	if !slices.Equal(got, want) {
		t.Errorf("words %q, want %q", got, want)
	}
	if !skip("42x") || !skip("a") || !skip("me@x.io") || skip("hello") {
		t.Error("skip")
	}
	x := newRuneIndex("a😀b")
	if x.at(0) != 0 || x.at(1) != 1 || x.at(3) != 2 || x.at(4) != 3 {
		t.Errorf("utf16 index %v", x)
	}
}

func TestCheck(t *testing.T) {
	if !Available("en-US") {
		t.Skip("no English spell checker here")
	}
	ms := Check("Teh quick brwn fox jumps", "en-US")
	var bad []string
	for _, m := range ms {
		bad = append(bad, m.Word)
	}
	if !slices.Equal(bad, []string{"Teh", "brwn"}) {
		t.Fatalf("misspelled %q", bad)
	}
	if ms[1].Start != 10 || ms[1].End != 14 {
		t.Errorf("brwn at %d-%d", ms[1].Start, ms[1].End)
	}
	s := Suggest("brwn", "en-US")
	if !slices.Contains(s, "brown") {
		t.Errorf("suggestions %q", s)
	}
	Ignore("teh")
	if len(Check("Teh fox", "en-US")) != 0 {
		t.Error("ignored word still reported")
	}
	Learn("giodeword")
	if len(Check("a giodeword here", "en-US")) != 0 {
		t.Error("learned word reported")
	}
}
