// Package appshell provides a full-window application layout: an
// optional fixed header, navbar, aside and footer around a main
// content area that's automatically offset so it never sits under
// them.
package appshell

import (
	"github.com/xmhrnx/giode/components/kit"
	"image"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/scroll"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var defaultChrome = "#1e293b"

// Slot is a sized AppShell region: Header/Footer are height tall,
// Navbar/Aside are width wide. Build one with Header, Navbar, Aside
// or Footer.
type Slot struct {
	content elements.Element
	size    int
	st      styles.Styles
}

// Header returns a Slot for Config.Header: a header height tall,
// spanning the full width, fixed to the top.
func Header(height int, content elements.Element, st ...styles.Styles) *Slot {
	return &Slot{content: content, size: height, st: first(st)}
}

// Navbar returns a Slot for Config.Navbar: a navbar width wide,
// spanning the height between Header and Footer (if present), fixed
// to the left (the right in a right-to-left locale).
func Navbar(width int, content elements.Element, st ...styles.Styles) *Slot {
	return &Slot{content: content, size: width, st: first(st)}
}

// Aside returns a Slot for Config.Aside: an aside width wide,
// spanning the height between Header and Footer (if present), fixed
// to the right (the left in a right-to-left locale).
func Aside(width int, content elements.Element, st ...styles.Styles) *Slot {
	return &Slot{content: content, size: width, st: first(st)}
}

// Footer returns a Slot for Config.Footer: a footer height tall,
// spanning the full width, fixed to the bottom.
func Footer(height int, content elements.Element, st ...styles.Styles) *Slot {
	return &Slot{content: content, size: height, st: first(st)}
}

// Main wraps content as an AppShell's main content area: it fills
// whatever space Header/Navbar/Aside/Footer leave once Config is
// resolved.
func Main(content elements.Element, st ...styles.Styles) elements.Element {
	s := styles.Merge(styles.Styles{Width: -1, Height: -1}, first(st))
	return elements.Box(s, content)
}

// Section groups content inside a Navbar/Aside, e.g. a scrollable
// list of nav links - a thin wrapper over Scroll, since a fixed-size
// sidebar is the common place content needs to scroll independently
// of the page.
func Section(content func() elements.Element, st ...styles.Styles) elements.Element {
	return scroll.New(content, st...)
}

// Config configures an AppShell's optional slots. Header, Navbar,
// Aside and Footer are optional - leave nil to omit that slot - Main
// is the required content area, typically built with Main.
type Config struct {
	Header *Slot
	Navbar *Slot
	Aside  *Slot
	Footer *Slot
	Main   elements.Element
}

// New returns a full-window layout: Header/Navbar/Aside/Footer (each
// optional) are fixed to their respective edges, and Main fills the
// remaining space, offset by whichever of them are present. The
// styles argument styles the root container; Width/Height default to
// -1 (fill the window).
//
// Sizing Aside (flush right) and Footer (flush bottom) needs the
// window's own size, only known at layout time, so New defers
// building the actual child styles to a Raw element's Layout rather
// than computing them upfront - the usual approach elsewhere in the
// library (e.g. Right/Bottom-anchored Box children) only supports a
// nonzero offset from the far edge, since Go's int zero value can't
// distinguish "flush" from "not set" the way it can for Left/Top.
func New(cfg Config, st ...styles.Styles) elements.Element {
	root := styles.Merge(styles.Styles{Width: -1, Height: -1}, first(st))

	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		// Slot sizes are in dp like every other Styles length, so the
		// window size is converted to dp to place the far-edge slots.
		px := gtx.Constraints.Max
		scale := gtx.Metric.PxPerDp
		if scale <= 0 {
			scale = 1
		}
		size := image.Pt(int(float32(px.X)/scale+0.5), int(float32(px.Y)/scale+0.5))

		var headerH, footerH, navbarW, asideW int
		if cfg.Header != nil {
			headerH = cfg.Header.size
		}
		if cfg.Footer != nil {
			footerH = cfg.Footer.size
		}
		if cfg.Navbar != nil {
			navbarW = cfg.Navbar.size
		}
		if cfg.Aside != nil {
			asideW = cfg.Aside.size
		}

		// Right to left, the navbar is on the right and the aside on
		// the left.
		rtl := kit.RTL(gtx)
		startW, endW := navbarW, asideW
		if rtl {
			startW, endW = asideW, navbarW
		}
		children := []elements.Element{
			elements.Box(styles.Styles{
				Position: properties.PositionAbsolute,
				Top:      properties.Top(headerH),
				Left:     properties.Left(startW),
				Right:    properties.Right(endW),
				Bottom:   properties.Bottom(footerH),
			}, cfg.Main),
		}
		if cfg.Header != nil {
			hs := styles.Merge(styles.Styles{Background: defaultChrome}, cfg.Header.st)
			hs.Position = properties.PositionAbsolute
			hs.Top, hs.Left = 0, 0
			hs.Width, hs.Height = -1, properties.Height(cfg.Header.size)
			children = append(children, elements.Box(hs, cfg.Header.content))
		}
		if cfg.Navbar != nil {
			ns := styles.Merge(styles.Styles{Background: defaultChrome}, cfg.Navbar.st)
			ns.Position = properties.PositionAbsolute
			ns.Top, ns.Left = properties.Top(headerH), 0
			if rtl {
				ns.Left = properties.Left(size.X - cfg.Navbar.size)
			}
			ns.Width, ns.Height = properties.Width(cfg.Navbar.size), -1
			ns.Bottom = properties.Bottom(footerH)
			children = append(children, elements.Box(ns, cfg.Navbar.content))
		}
		if cfg.Footer != nil {
			fs := styles.Merge(styles.Styles{Background: defaultChrome}, cfg.Footer.st)
			fs.Position = properties.PositionAbsolute
			// Flush to the bottom edge: computed here (see the doc
			// comment above) rather than via Bottom, whose zero value
			// can't be told apart from "unset".
			fs.Top, fs.Left = properties.Top(size.Y-cfg.Footer.size), 0
			fs.Width, fs.Height = -1, properties.Height(cfg.Footer.size)
			children = append(children, elements.Box(fs, cfg.Footer.content))
		}
		if cfg.Aside != nil {
			as := styles.Merge(styles.Styles{Background: defaultChrome}, cfg.Aside.st)
			as.Position = properties.PositionAbsolute
			// Flush to the right edge: computed here for the same
			// reason Footer's Top is.
			as.Top, as.Left = properties.Top(headerH), properties.Left(size.X-cfg.Aside.size)
			if rtl {
				as.Left = 0
			}
			as.Width, as.Height = properties.Width(cfg.Aside.size), -1
			as.Bottom = properties.Bottom(footerH)
			children = append(children, elements.Box(as, cfg.Aside.content))
		}

		return elements.Box(root, children...).Layout(gtx)
	})
}

func first(st []styles.Styles) styles.Styles {
	if len(st) > 0 {
		return st[0]
	}
	return styles.Styles{}
}
