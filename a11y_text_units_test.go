package giode

import "testing"

func TestTextUnits(t *testing.T) {
	r := []rune("Ada  Lovelace wrote")
	if s, e := enclosingUnit(r, 6, unitWord); s != 5 || e != 14 {
		t.Errorf("word at 6 = [%d, %d)", s, e)
	}
	if s, e := enclosingUnit(r, 0, unitWord); s != 0 || e != 5 {
		t.Errorf("word at 0 = [%d, %d)", s, e)
	}
	if s, e := enclosingUnit(r, 3, unitLine); s != 0 || e != len(r) {
		t.Errorf("line = [%d, %d)", s, e)
	}
	if s, e := enclosingUnit(r, 4, unitChar); s != 4 || e != 5 {
		t.Errorf("char at 4 = [%d, %d)", s, e)
	}
	if p, m := moveBy(r, 0, unitWord, 2); p != 14 || m != 2 {
		t.Errorf("two words on = %d (%d)", p, m)
	}
	if p, m := moveBy(r, 14, unitWord, -1); p != 5 || m != -1 {
		t.Errorf("a word back = %d (%d)", p, m)
	}
	if p, m := moveBy(r, 17, unitChar, 5); p != len(r) || m != 2 {
		t.Errorf("past the end = %d (%d)", p, m)
	}
}
