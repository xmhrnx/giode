// Package virtuallist provides a vertical list that only builds the
// rows in or near its viewport, so it handles any number of items at a
// constant cost per frame.

package virtuallist

func (l *List) maxOffset() float32 {
	return float32(max(0, l.n*l.itemPx-l.viewportH))
}

func (l *List) clamp(v float32) float32 {
	return max(0, min(l.maxOffset(), v))
}

// resolveScroll applies a pending ScrollToIndex.
func (l *List) resolveScroll() {
	if !l.hasPending || l.n == 0 || l.itemPx == 0 {
		return
	}
	l.hasPending = false
	i := max(0, min(l.n-1, l.pendingIdx))
	top := float32(i * l.itemPx)
	bottom := top + float32(l.itemPx)
	vh := float32(l.viewportH)
	switch l.pendingAlign {
	case AlignStart:
		l.offset = top
	case AlignCenter:
		l.offset = top - (vh-float32(l.itemPx))/2
	case AlignEnd:
		l.offset = bottom - vh
	default:
		if top < l.offset {
			l.offset = top
		} else if bottom > l.offset+vh {
			l.offset = bottom - vh
		}
	}
	l.offset = l.clamp(l.offset)
}
