package text

import "gioui.org/text"

// wrapPolicy is how text breaks into lines: between words only - a word
// too long for the line runs past its end rather than breaking between
// its letters. A single line cut off with "…" (TextOverflowEllipsis)
// still cuts a long word, to show as much of it as fits.
func wrapPolicy(maxLines int, truncator string) text.WrapPolicy {
	if maxLines == 1 && truncator != "" {
		return text.WrapHeuristically
	}
	return text.WrapWords
}
