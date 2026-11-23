package terminal

import (
	"image"
	"image/color"

	"gioui.org/op"
)

// drawBlock draws a box drawing or block element character as shapes
// filling cell [x0, x1) of height h, so lines join across cells, which
// glyphs from a font don't. It reports whether r is one.
func drawBlock(ops *op.Ops, r rune, x0, x1, h, px int, c color.NRGBA) bool {
	w := x1 - x0
	switch {
	case r >= 0x2500 && r <= 0x257f:
		lines, ok := boxLines[r]
		if !ok {
			return false
		}
		drawLines(ops, lines, x0, w, h, px, c)
		return true
	case r == 0x2580: // upper half
		fill(ops, image.Rect(x0, 0, x1, h/2), c)
	case r >= 0x2581 && r <= 0x2588: // lower eighths, to full
		n := int(r - 0x2580)
		fill(ops, image.Rect(x0, h-h*n/8, x1, h), c)
	case r >= 0x2589 && r <= 0x258f: // left eighths
		n := int(0x2590 - r)
		fill(ops, image.Rect(x0, 0, x0+w*n/8, h), c)
	case r == 0x2590: // right half
		fill(ops, image.Rect(x0+w/2, 0, x1, h), c)
	case r >= 0x2591 && r <= 0x2593: // shades
		s := c
		s.A = uint8(int(c.A) * int(r-0x2590) / 4)
		fill(ops, image.Rect(x0, 0, x1, h), s)
	case r == 0x2594: // upper eighth
		fill(ops, image.Rect(x0, 0, x1, max(1, h/8)), c)
	case r == 0x2595: // right eighth
		fill(ops, image.Rect(x1-max(1, w/8), 0, x1, h), c)
	case r >= 0x2596 && r <= 0x259f: // quadrants
		// Upper left, upper right, lower left, lower right.
		q := [...]uint8{0b0010, 0b0001, 0b1000, 0b1011, 0b1001, 0b1110, 0b1101, 0b0100, 0b0110, 0b0111}[r-0x2596]
		mx, my := x0+w/2, h/2
		quads := [4]image.Rectangle{
			image.Rect(x0, 0, mx, my), image.Rect(mx, 0, x1, my),
			image.Rect(x0, my, mx, h), image.Rect(mx, my, x1, h),
		}
		for i, rect := range quads {
			if q&(0b1000>>i) != 0 {
				fill(ops, rect, c)
			}
		}
	default:
		return false
	}
	return true
}

// drawLines draws lines from the cell's center to its edges: up, right,
// down, left, light (1), heavy (2) or double (3).
func drawLines(ops *op.Ops, l [4]uint8, x0, w, h, px int, c color.NRGBA) {
	light := max(1, (px+6)/12)
	heavy := light * 2
	cx, cy := x0+(w-light)/2, (h-light)/2
	thick := func(k uint8) int {
		if k == 2 {
			return heavy
		}
		return light
	}
	// The center square the lines meet in, as wide as the thickest
	// line across.
	vw, hw := 0, 0
	for i, k := range l {
		if k == 0 {
			continue
		}
		t := thick(k)
		if k == 3 {
			t = 3 * light
		}
		if i%2 == 0 {
			vw = max(vw, t)
		} else {
			hw = max(hw, t)
		}
	}
	// Each half line reaches across the band the lines at right angles
	// to it take at the center, so corners and junctions close.
	hw, vw = max(hw, light), max(vw, light)
	bandTop, bandLeft := cy-(hw-light)/2, cx-(vw-light)/2
	bandBottom, bandRight := bandTop+hw, bandLeft+vw
	for i, k := range l {
		if k == 0 {
			continue
		}
		t, offs := thick(k), []int{0}
		if k == 3 {
			// Double: two light lines a light width apart.
			t, offs = light, []int{-light, light}
		}
		for _, o := range offs {
			x, y := cx+o-(t-light)/2, cy+o-(t-light)/2
			switch i {
			case 0:
				fill(ops, image.Rect(x, 0, x+t, bandBottom), c)
			case 2:
				fill(ops, image.Rect(x, bandTop, x+t, h), c)
			case 1:
				fill(ops, image.Rect(bandLeft, y, x0+w, y+t), c)
			case 3:
				fill(ops, image.Rect(x0, y, bandRight, y+t), c)
			}
		}
	}
}
