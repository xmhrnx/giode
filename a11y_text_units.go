package giode

import "unicode"

// Text units screen readers move by, over a node's text as runes: what
// UI Automation's text ranges (and AT-SPI's) need. A Giode text field is
// one line, so a line, a paragraph, a page and the document are all of
// it.

type textUnit int

const (
	unitChar textUnit = iota
	unitFormat
	unitWord
	unitLine
	unitParagraph
	unitPage
	unitDocument
)

// isBoundary reports whether i is a boundary of unit u in t.
func isBoundary(t []rune, i int, u textUnit) bool {
	switch {
	case i <= 0 || i >= len(t):
		return true
	case u == unitChar:
		return true
	case u == unitWord:
		return unicode.IsSpace(t[i-1]) && !unicode.IsSpace(t[i])
	}
	return false
}

// nextBoundary returns the first boundary of u after pos, len(t) at most.
func nextBoundary(t []rune, pos int, u textUnit) int {
	for i := pos + 1; i < len(t); i++ {
		if isBoundary(t, i, u) {
			return i
		}
	}
	return len(t)
}

// prevBoundary returns the last boundary of u before pos, 0 at least.
func prevBoundary(t []rune, pos int, u textUnit) int {
	for i := min(pos, len(t)) - 1; i > 0; i-- {
		if isBoundary(t, i, u) {
			return i
		}
	}
	return 0
}

// enclosingUnit returns the unit u containing pos: [start, end).
func enclosingUnit(t []rune, pos int, u textUnit) (int, int) {
	pos = max(0, min(pos, len(t)))
	if pos == len(t) && pos > 0 {
		// At the end: the last unit.
		pos--
	}
	start := pos
	if !isBoundary(t, pos, u) {
		start = prevBoundary(t, pos, u)
	}
	return start, nextBoundary(t, start, u)
}

// moveBy moves pos by count boundaries of u, forward or back, and
// returns where it lands and how many it moved.
func moveBy(t []rune, pos int, u textUnit, count int) (int, int) {
	moved := 0
	for moved < count && pos < len(t) {
		pos = nextBoundary(t, pos, u)
		moved++
	}
	for moved > count && pos > 0 {
		pos = prevBoundary(t, pos, u)
		moved--
	}
	return pos, moved
}
