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
