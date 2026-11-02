package appshell

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
)

type capture struct {
	pos  image.Point
	size image.Point
}

func (c *capture) element() elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		c.pos = elements.AbsolutePosition()
		c.size = gtx.Constraints.Max
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

// TestAppShell_PositionsAndSizesSlots lays out every slot at once and
// checks each one's absolute screen position and measured size,
// guarding the Right/Bottom-anchoring workaround New's own doc
// comment describes (Aside and Footer are flush against the right/
// bottom edge, which the shared Box absolute-positioning primitive
// can't express directly - see boxabsolute.go).
func TestAppShell_PositionsAndSizesSlots(t *testing.T) {
	const (
		winW, winH = 800, 600
		headerH    = 50
		footerH    = 40
		navbarW    = 80
		asideW     = 60
	)

	var header, navbar, aside, footer, main capture
	cfg := Config{
		Header: Header(headerH, header.element()),
		Navbar: Navbar(navbarW, navbar.element()),
		Aside:  Aside(asideW, aside.element()),
		Footer: Footer(footerH, footer.element()),
		Main:   main.element(),
	}
	shell := New(cfg)

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(winW, winH))}
	shell.Layout(gtx)

	cases := []struct {
		name     string
		c        capture
		wantPos  image.Point
		wantSize image.Point
	}{
		{"Header", header, image.Pt(0, 0), image.Pt(winW, headerH)},
		{"Footer", footer, image.Pt(0, winH-footerH), image.Pt(winW, footerH)},
		{"Navbar", navbar, image.Pt(0, headerH), image.Pt(navbarW, winH-headerH-footerH)},
		{"Aside", aside, image.Pt(winW-asideW, headerH), image.Pt(asideW, winH-headerH-footerH)},
		{"Main", main, image.Pt(navbarW, headerH), image.Pt(winW-navbarW-asideW, winH-headerH-footerH)},
	}
	for _, c := range cases {
		if c.c.pos != c.wantPos {
			t.Errorf("%s pos = %v, want %v", c.name, c.c.pos, c.wantPos)
		}
		if c.c.size != c.wantSize {
			t.Errorf("%s size = %v, want %v", c.name, c.c.size, c.wantSize)
		}
	}
}

// TestAppShell_OmittedSlotsLeaveMainFullSize confirms Main falls back
// to the full window when Header/Navbar/Aside/Footer are all omitted
// (nil Config fields) - the case most likely to trip the Right/
// Bottom-anchoring ambiguity, since every offset is 0.
func TestAppShell_OmittedSlotsLeaveMainFullSize(t *testing.T) {
	var main capture
	shell := New(Config{Main: main.element()})

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(400, 300))}
	shell.Layout(gtx)

	if main.pos != (image.Point{}) {
		t.Errorf("Main pos = %v, want (0,0)", main.pos)
	}
	if main.size != (image.Point{X: 400, Y: 300}) {
		t.Errorf("Main size = %v, want (400,300)", main.size)
	}
}

// TestAppShell_OnlyFooterKeepsHeaderlessNavbarAtTop guards the
// specific ambiguity case worked through in New's doc comment: with
// no Header (headerH=0) but a Footer present, Navbar/Aside must stay
// top-anchored rather than the shared absolute-positioning
// primitive's Top==0-with-Bottom>0 branch pulling them down to sit
// just above the footer.
func TestAppShell_OnlyFooterKeepsHeaderlessNavbarAtTop(t *testing.T) {
	const footerH = 40
	var navbar capture
	shell := New(Config{
		Footer: Footer(footerH, elements.Raw(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		})),
		Navbar: Navbar(80, navbar.element()),
		Main:   elements.Raw(func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
	})

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(400, 300))}
	shell.Layout(gtx)

	if navbar.pos != (image.Point{X: 0, Y: 0}) {
		t.Errorf("Navbar pos = %v, want (0,0) (top-anchored, not pulled down above the footer)", navbar.pos)
	}
	if navbar.size != (image.Point{X: 80, Y: 300 - footerH}) {
		t.Errorf("Navbar size = %v, want (80,%d)", navbar.size, 300-footerH)
	}
}
