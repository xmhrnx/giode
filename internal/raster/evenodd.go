package raster

import "gioui.org/f32"

// The scanner fills with the nonzero rule only. evenOdd rewrites a path
// so nonzero filling gives evenodd's result: each subpath runs one way
// when an even number of the others enclose it and the other way when
// an odd number do, so nested subpaths alternate between filled and
// holes. That matches evenodd for icons' shapes - rings, targets,
// cut-outs - whose subpaths don't cross each other; a single subpath
// crossing itself (a pentagram) still fills by nonzero.
func evenOdd(path []Segment) []Segment {
	subs := splitSubpaths(path)
	if len(subs) < 2 {
		return path
	}
	polys := make([][]f32.Point, len(subs))
	for i, s := range subs {
		polys[i] = flatten(s)
	}
	out := make([]Segment, 0, len(path))
	for i, s := range subs {
		if len(polys[i]) < 3 {
			out = append(out, s...)
			continue
		}
		depth := 0
		probe := polys[i][0]
		for j, p := range polys {
			if j != i && len(p) >= 3 && inside(probe, p) {
				depth++
			}
		}
		positive := area(polys[i]) > 0
		if positive != (depth%2 == 0) {
			s = reverse(s)
		}
		out = append(out, s...)
	}
	return out
}

// splitSubpaths splits a path at each MoveTo.
func splitSubpaths(path []Segment) [][]Segment {
	var subs [][]Segment
	for _, s := range path {
		if s.Op == 'M' || len(subs) == 0 {
			subs = append(subs, nil)
		}
		subs[len(subs)-1] = append(subs[len(subs)-1], s)
	}
	return subs
}

// flatten approximates a subpath by its vertices and a few points on
// each curve: enough for areas and containment.
func flatten(sub []Segment) []f32.Point {
	var pts []f32.Point
	var cur f32.Point
	for _, s := range sub {
		switch s.Op {
		case 'M', 'L':
			cur = s.Pts[0]
			pts = append(pts, cur)
		case 'Q':
			for t := float32(0.25); t <= 1; t += 0.25 {
				u := 1 - t
				pts = append(pts, cur.Mul(u*u).Add(s.Pts[0].Mul(2*u*t)).Add(s.Pts[1].Mul(t*t)))
			}
			cur = s.Pts[1]
		case 'C':
			for t := float32(0.125); t <= 1; t += 0.125 {
				u := 1 - t
				pts = append(pts, cur.Mul(u*u*u).Add(s.Pts[0].Mul(3*u*u*t)).Add(s.Pts[1].Mul(3*u*t*t)).Add(s.Pts[2].Mul(t*t*t)))
			}
			cur = s.Pts[2]
		}
	}
	return pts
}

// area is a polygon's signed area (shoelace).
func area(p []f32.Point) float32 {
	var a float32
	for i := range p {
		q := p[(i+1)%len(p)]
		a += p[i].X*q.Y - q.X*p[i].Y
	}
	return a / 2
}

// inside reports whether pt is inside polygon p (even-odd crossing test).
func inside(pt f32.Point, p []f32.Point) bool {
	in := false
	for i, j := 0, len(p)-1; i < len(p); j, i = i, i+1 {
		a, b := p[i], p[j]
		if (a.Y > pt.Y) != (b.Y > pt.Y) && pt.X < (b.X-a.X)*(pt.Y-a.Y)/(b.Y-a.Y)+a.X {
			in = !in
		}
	}
	return in
}

// reverse returns a subpath drawn the other way round.
func reverse(sub []Segment) []Segment {
	// The point each segment starts from.
	starts := make([]f32.Point, len(sub))
	var cur f32.Point
	closed := false
	for i, s := range sub {
		starts[i] = cur
		switch s.Op {
		case 'M', 'L':
			cur = s.Pts[0]
		case 'Q':
			cur = s.Pts[1]
		case 'C':
			cur = s.Pts[2]
		case 'Z':
			closed = true
		}
	}
	out := []Segment{{Op: 'M', Pts: [3]f32.Point{cur}}}
	for i := len(sub) - 1; i >= 0; i-- {
		s, from := sub[i], starts[i]
		switch s.Op {
		case 'L':
			out = append(out, Segment{Op: 'L', Pts: [3]f32.Point{from}})
		case 'Q':
			out = append(out, Segment{Op: 'Q', Pts: [3]f32.Point{s.Pts[0], from}})
		case 'C':
			out = append(out, Segment{Op: 'C', Pts: [3]f32.Point{s.Pts[1], s.Pts[0], from}})
		}
	}
	if closed {
		out = append(out, Segment{Op: 'Z'})
	}
	return out
}
