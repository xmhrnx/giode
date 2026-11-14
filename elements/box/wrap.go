package box

import (
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// wrapItem is one measured child: its recorded paint ops (so it is
// laid out exactly once) plus its size resolved into main/cross axis
// terms.
type wrapItem struct {
	ops         op.CallOp
	main, cross int
	// track holds what the item registered, shifted once its position
	// is known.
	track base.ChildTracker
	z     int
	// above is the distance from the item's top to its baseline, for
	// AlignBaseline.
	above int
}

// wrapLine is one run of items that fit within the main axis.
type wrapLine struct {
	// items are the line's items, a part of the box's.
	items     []wrapItem
	used      int // sum of item main sizes plus internal gaps
	thickness int // the line's extent on the cross axis
	// above is the largest distance from an item's top to its baseline,
	// where AlignBaseline puts the line's baseline.
	above int
}
