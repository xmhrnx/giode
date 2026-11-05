package icon

import (
	"gioui.org/f32"
)

// rectPath compiles a rectangle with optional corner radii.
func rectPath(x, y, w, h, rx, ry float32) compiled {
	if w <= 0 || h <= 0 {
		return nil
	}
	if rx < 0 {
		rx = 0
	}
	if ry < 0 {
		ry = 0
	}
	if rx == 0 {
		rx = ry
	}
	if ry == 0 {
		ry = rx
	}
	rx, ry = min(rx, w/2), min(ry, h/2)
	pc := &pathCompiler{}
	if rx == 0 {
		pc.moveTo(f32.Pt(x, y))
		pc.lineTo(f32.Pt(x+w, y))
		pc.lineTo(f32.Pt(x+w, y+h))
		pc.lineTo(f32.Pt(x, y+h))
		pc.run('Z', nil)
		return pc.out
	}
	pc.moveTo(f32.Pt(x+rx, y))
	pc.lineTo(f32.Pt(x+w-rx, y))
	pc.arc(rx, ry, 0, false, true, f32.Pt(x+w, y+ry))
	pc.lineTo(f32.Pt(x+w, y+h-ry))
	pc.arc(rx, ry, 0, false, true, f32.Pt(x+w-rx, y+h))
	pc.lineTo(f32.Pt(x+rx, y+h))
	pc.arc(rx, ry, 0, false, true, f32.Pt(x, y+h-ry))
	pc.lineTo(f32.Pt(x, y+ry))
	pc.arc(rx, ry, 0, false, true, f32.Pt(x+rx, y))
	pc.run('Z', nil)
	return pc.out
}

// ellipsePath compiles an ellipse centered at (cx, cy).
func ellipsePath(cx, cy, rx, ry float32) compiled {
	if rx <= 0 || ry <= 0 {
		return nil
	}
	pc := &pathCompiler{}
	pc.moveTo(f32.Pt(cx+rx, cy))
	pc.arc(rx, ry, 0, false, true, f32.Pt(cx-rx, cy))
	pc.arc(rx, ry, 0, false, true, f32.Pt(cx+rx, cy))
	pc.run('Z', nil)
	return pc.out
}

// polyPath compiles a polyline, closed for polygons.
func polyPath(pts []float32, closed bool) compiled {
	if len(pts) < 4 {
		return nil
	}
	pc := &pathCompiler{}
	pc.moveTo(f32.Pt(pts[0], pts[1]))
	for i := 2; i+1 < len(pts); i += 2 {
		pc.lineTo(f32.Pt(pts[i], pts[i+1]))
	}
	if closed {
		pc.run('Z', nil)
	}
	return pc.out
}
