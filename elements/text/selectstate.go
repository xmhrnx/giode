package text

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/widget"
)

// selectState is the persistent state of a selectable Text or
// RichText, kept in the keyed store under its key.
type selectState struct {
	// sel does mouse selection for Text: drag to select, double-click
	// for a word, Ctrl/Cmd+C to copy.
	sel widget.Selectable

	// tap observes presses, to detect a long-press or right-click.
	tap      int
	pressing bool
	pressAt  time.Time
	pressPos f32.Point

	// all is set while the whole text is selected and the Copy action
	// shows.
	all      bool
	copyText string
	copy     *copyAction
	// applied is set once a select-all reached sel (mouse).
	applied bool
	// origin is the text's position, exact from the last frame, to
	// place the Copy action.
	origin image.Point
	pass   func(gtx layout.Context) bool
}

// longPress is how long a finger must rest on selectable text to
// select it.
const longPress = 500 * time.Millisecond

func newSelectState() *selectState {
	return &selectState{copy: newCopyAction()}
}

func (s *selectState) selectAll() {
	s.all = true
}
