package picker

import "testing"

func TestSplitPaths(t *testing.T) {
	got := splitPaths([]byte("/Users/me/a b.txt\x00/Users/me/line\nbreak.pdf\x00"))
	if len(got) != 2 || got[0] != "/Users/me/a b.txt" || got[1] != "/Users/me/line\nbreak.pdf" {
		t.Errorf("splitPaths: %q", got)
	}
	if got := splitPaths(nil); got != nil {
		t.Errorf("splitPaths(nil) = %q", got)
	}
}
