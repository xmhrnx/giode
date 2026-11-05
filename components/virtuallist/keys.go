// Package virtuallist provides a vertical list that only builds the
// rows in or near its viewport, so it handles any number of items at a
// constant cost per frame.

package virtuallist

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
)

// handleKeys processes focus and navigation keys.
func (l *List) handleKeys(gtx layout.Context) {
	page := max(1, l.viewportH/max(1, l.itemPx)-1)
	if l.wantFocus {
		l.wantFocus = false
		gtx.Execute(key.FocusCmd{Tag: l})
	}
	filters := []event.Filter{
		key.FocusFilter{Target: l},
		key.Filter{Focus: l, Name: key.NameUpArrow},
		key.Filter{Focus: l, Name: key.NameDownArrow},
		key.Filter{Focus: l, Name: key.NamePageUp},
		key.Filter{Focus: l, Name: key.NamePageDown},
		key.Filter{Focus: l, Name: key.NameHome},
		key.Filter{Focus: l, Name: key.NameEnd},
		key.Filter{Focus: l, Name: key.NameReturn},
		key.Filter{Focus: l, Name: key.NameEnter},
	}
	for _, n := range l.extraKeys {
		filters = append(filters, key.Filter{Focus: l, Name: n})
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case key.FocusEvent:
			l.focused = e.Focus
			if e.Focus && l.focusIdx < 0 && l.n > 0 && !l.pointerFocus {
				l.SetFocusedIndex(max(0, l.first))
			}
			l.pointerFocus = false
		case key.Event:
			if e.State != key.Press || l.n == 0 {
				continue
			}
			cur := max(0, l.focusIdx)
			// Keyboard moves stay within the current rows.
			clampN := func(i int) int { return max(0, min(l.n-1, i)) }
			switch e.Name {
			case key.NameUpArrow:
				l.SetFocusedIndex(clampN(cur - 1))
			case key.NameDownArrow:
				if l.focusIdx < 0 {
					l.SetFocusedIndex(0)
				} else {
					l.SetFocusedIndex(clampN(cur + 1))
				}
			case key.NamePageUp:
				l.SetFocusedIndex(clampN(cur - page))
			case key.NamePageDown:
				l.SetFocusedIndex(clampN(cur + page))
			case key.NameHome:
				l.SetFocusedIndex(0)
			case key.NameEnd:
				l.SetFocusedIndex(l.n - 1)
			case key.NameReturn, key.NameEnter:
				if l.focusIdx >= 0 && l.onActivate != nil {
					l.onActivate(l.focusIdx)
				}
			default:
				if l.onKey != nil {
					l.onKey(e, l.focusIdx)
				}
			}
			kit.RequestFrame(gtx)
		}
	}
	// A press anywhere in the list focuses it.
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &l.focusIdx, Kinds: pointer.Press})
		if !ok {
			break
		}
		if pe, ok := ev.(pointer.Event); ok && pe.Kind == pointer.Press {
			if !l.focused {
				l.pointerFocus = true
			}
			gtx.Execute(key.FocusCmd{Tag: l})
			if l.itemPx > 0 && !l.noClickRow {
				if i := int((pe.Position.Y + l.offset) / float32(l.itemPx)); i >= 0 && i < l.n {
					l.SetFocusedIndex(i)
				}
			}
		}
	}
}
