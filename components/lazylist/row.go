// Package lazylist provides a vertical list of rows of any height that
// only builds and lays out the rows in or near its viewport - giode's
// FlatList, ListView.builder or LazyColumn.

package lazylist

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"github.com/xmhrnx/giode/elements"
)

// row is what the list remembers of a row.
type row struct {
	height  int
	width   int
	version any
	cache   *elements.MemoCache
	used    uint64
}

// placed is a row laid out this frame.
type placed struct {
	idx   int
	y, h  int
	call  op.CallOp
	track elements.ChildTracker
}

// layoutRow lays out row i at the list's width and returns its drawing
// and height.
func (l *List) layoutRow(gtx layout.Context, i int) placed {
	k := l.key(i)
	r := l.rows[k]
	if r == nil {
		r = &row{}
		l.rows[k] = r
	}
	r.used = l.frame
	rgtx := gtx
	rgtx.Constraints = layout.Constraints{Min: image.Pt(gtx.Constraints.Max.X, 0), Max: image.Pt(gtx.Constraints.Max.X, 1<<24)}
	m := op.Record(gtx.Ops)
	track := elements.TrackChild()
	var dims layout.Dimensions
	if l.version != nil {
		v := l.version(i)
		if r.cache == nil {
			r.cache = elements.NewMemoCache()
		}
		dims = r.cache.Layout(rgtx, func() elements.Element { return l.item(i) }, v)
		r.version = v
	} else if el := l.item(i); el != nil {
		dims = el.Layout(rgtx)
	}
	track.End()
	call := m.Stop()
	if fresh := r.width == 0; fresh || r.height != dims.Size.Y || r.width != gtx.Constraints.Max.X {
		// A row new or changed: it moves the estimate and the average
		// height. A row drawn as before doesn't, so rows of different
		// heights don't keep the estimate changing frame after frame.
		l.est.valid = false
		r.height, r.width = dims.Size.Y, gtx.Constraints.Max.X
		if l.measure == 0 {
			l.measure = max(1, r.height)
		} else {
			l.measure = (l.measure*7 + r.height) / 8
		}
	}
	return placed{idx: i, h: r.height, call: call, track: track}
}

// heightOf is row i's height as last measured, or the estimate.
func (l *List) heightOf(i int) int {
	if r := l.rows[l.key(i)]; r != nil && r.width == l.width {
		if l.version == nil || r.version == l.version(i) {
			return r.height
		}
	}
	return max(1, l.measure)
}
