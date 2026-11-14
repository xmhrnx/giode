package elements

import (
	"github.com/xmhrnx/giode/elements/canvas"
	"github.com/xmhrnx/giode/elements/divider"
	"github.com/xmhrnx/giode/elements/glyph"
	"github.com/xmhrnx/giode/elements/raw"
	"github.com/xmhrnx/giode/elements/responsive"
	"github.com/xmhrnx/giode/elements/table"
)

// Raw, Divider, Glyph, Canvas, Responsive and Table, re-exported from
// their own subpackages.
type (
	// RawFunc adapts a function to the Element interface.
	RawFunc = raw.RawFunc
	// DividerElement is a thin horizontal line.
	DividerElement = divider.DividerElement
	// Canvas's Painter draws on a Canvas, in dp.
	Painter = canvas.Painter
	// Path is a shape built from lines and curves, in dp.
	Path = canvas.Path
	// TextAnchor is the point of a text Painter.Text puts at (x, y).
	TextAnchor = canvas.TextAnchor
	// Breakpoint is a simplified media query.
	Breakpoint = responsive.Breakpoint
)

var (
	Raw         = raw.Raw
	Divider     = divider.Divider
	Glyph       = glyph.Glyph
	SizedPx     = glyph.SizedPx
	OffsetPx    = glyph.OffsetPx
	Canvas      = canvas.Canvas
	Responsive  = responsive.Responsive
	Table       = table.Table
	TableRow    = table.TableRow
	TableHeader = table.TableHeader
	TableCell   = table.TableCell
)

// The points of a text Painter.Text can put at (x, y).
const (
	TextAnchorTopLeft     = canvas.TextAnchorTopLeft
	TextAnchorTop         = canvas.TextAnchorTop
	TextAnchorTopRight    = canvas.TextAnchorTopRight
	TextAnchorLeft        = canvas.TextAnchorLeft
	TextAnchorCenter      = canvas.TextAnchorCenter
	TextAnchorRight       = canvas.TextAnchorRight
	TextAnchorBottomLeft  = canvas.TextAnchorBottomLeft
	TextAnchorBottom      = canvas.TextAnchorBottom
	TextAnchorBottomRight = canvas.TextAnchorBottomRight
)
