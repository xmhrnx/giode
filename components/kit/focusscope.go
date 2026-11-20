package kit

import (
	"gioui.org/io/key"
	"gioui.org/layout"
)

// Focus scopes: a modal dialog keeps keyboard focus inside it while it's
// open (a focus trap), moves it in when it opens and gives it back when
// it closes.

// FocusScope collects the focusable controls laid out inside it, in
// order. Focusable components report themselves (NoteFocusable;
// FocusVisible does it).
type FocusScope struct {
	tags []any
	// cur is the index of the control focused last frame, -1 for none.
	cur int
	// opened is set once focus has been moved in; restore is the tag
	// focused before.
	opened  bool
	restore any
	// id identifies the scope among the frame's traps.
	id uint64
}

var (
	scopeStack []*FocusScope
	// lastFocused is the tag last seen focused (FocusVisible), for a
	// dialog to give focus back to.
	lastFocused any
	// The topmost trap: the last to call Trap, in this frame and the
	// last. Only it moves focus.
	trapThis, trapLast uint64
	trapFrame          int64
	scopeSeq           uint64
)

// PushFocusScope collects the focusable controls laid out until pop into
// s.
func PushFocusScope(s *FocusScope) (pop func()) {
	s.tags = s.tags[:0]
	if s.id == 0 {
		scopeSeq++
		s.id = scopeSeq
	}
	scopeStack = append(scopeStack, s)
	n := len(scopeStack)
	return func() { scopeStack = scopeStack[:n-1] }
}

// NoteFocusable reports a focusable control (its focus tag) to the
// enclosing focus scopes, and to caches recording it.
func NoteFocusable(tag any) {
	for _, s := range scopeStack {
		s.tags = append(s.tags, tag)
	}
	for _, r := range focusRecords {
		*r = append(*r, tag)
	}
}

// focusRecords are the caches recording a subtree's focusable controls.
var focusRecords []*[]any

// PushFocusRecord collects into dst the focusable controls reported
// until pop. A cache (elements.Memo) records them so it can report them
// again when it replays the subtree without laying it out.
func PushFocusRecord(dst *[]any) (pop func()) {
	*dst = (*dst)[:0]
	focusRecords = append(focusRecords, dst)
	n := len(focusRecords)
	return func() { focusRecords = focusRecords[:n-1] }
}

// LastFocused returns the tag last seen with keyboard focus.
func LastFocused() any { return lastFocused }

// Trap keeps the focus inside s while it's the topmost trap: call it
// after laying out the scope's content, every frame the modal is open.
// The first time, it moves focus to the first control inside. When focus
// leaves - Tab past the last control, Shift+Tab before the first, or a
// click outside - it goes back to the first or the last.
func (s *FocusScope) Trap(gtx layout.Context) {
	if f := frameCount.Load(); f != trapFrame {
		trapFrame, trapLast, trapThis = f, trapThis, 0
	}
	trapThis = s.id
	if trapLast != 0 && trapLast != s.id {
		// Another modal is on top.
		return
	}
	if len(s.tags) == 0 {
		return
	}
	if !s.opened {
		s.opened = true
		s.restore = lastFocused
		s.cur = 0
		gtx.Execute(key.FocusCmd{Tag: s.tags[0]})
		return
	}
	for i, t := range s.tags {
		if gtx.Focused(t) {
			s.cur = i
			return
		}
	}
	// Focus left the scope: wrap around from the end it left by.
	next := 0
	if s.cur == 0 && len(s.tags) > 1 {
		next = len(s.tags) - 1
	}
	s.cur = next
	gtx.Execute(key.FocusCmd{Tag: s.tags[next]})
}

// Release gives focus back to what had it before the modal opened: call
// it when the modal closes.
func (s *FocusScope) Release(gtx layout.Context) {
	if !s.opened {
		return
	}
	s.opened = false
	if s.restore != nil {
		gtx.Execute(key.FocusCmd{Tag: s.restore})
	}
	s.restore = nil
}

// First returns the focus tag of the first control laid out in s, or nil
// when it has none.
func (s *FocusScope) First() any {
	if len(s.tags) == 0 {
		return nil
	}
	return s.tags[0]
}
