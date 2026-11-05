package icon

import (
	"sync"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"github.com/xmhrnx/giode/internal/raster"
)

// segment is one command of a compiled path, in absolute coordinates.
// Arcs and the shorthand curves are resolved at compile time, leaving
// only the primitives clip.Path understands.
type segment struct {
	op  byte // 'M', 'L', 'Q', 'C' or 'Z'
	pts [3]f32.Point
}

// compiled is a parsed, ready-to-replay path.
type compiled []segment

// add replays the path into p.
func (c compiled) add(p *clip.Path) {
	for _, s := range c {
		switch s.op {
		case 'M':
			p.MoveTo(s.pts[0])
		case 'L':
			p.LineTo(s.pts[0])
		case 'Q':
			p.QuadTo(s.pts[0], s.pts[1])
		case 'C':
			p.CubeTo(s.pts[0], s.pts[1], s.pts[2])
		case 'Z':
			p.Close()
		}
	}
}

// segments returns c for internal/raster.
func (c compiled) segments() []raster.Segment {
	out := make([]raster.Segment, len(c))
	for i, s := range c {
		out[i] = raster.Segment{Op: s.op, Pts: s.pts}
	}
	return out
}

// transform returns c with every point mapped through t.
func (c compiled) transform(t f32.Affine2D) compiled {
	if t == (f32.Affine2D{}) || t == f32.AffineId() {
		return c
	}
	out := make(compiled, len(c))
	for i, s := range c {
		for j := range s.pts {
			s.pts[j] = t.Transform(s.pts[j])
		}
		out[i] = s
	}
	return out
}

// vertices returns the end points of every drawing segment, used to
// paint round stroke caps and joins.
func (c compiled) vertices() []f32.Point {
	var out []f32.Point
	for _, s := range c {
		switch s.op {
		case 'M', 'L':
			out = append(out, s.pts[0])
		case 'Q':
			out = append(out, s.pts[1])
		case 'C':
			out = append(out, s.pts[2])
		}
	}
	return out
}

// pathCache memoizes compiled path data, so path strings passed to SVG
// every frame are parsed once. It is bounded; past the limit it is
// dropped and refilled.
var (
	pathCacheMu sync.Mutex
	pathCache   = map[string]compiled{}
)

const maxCachedPaths = 2048

// compileCached compiles d, reusing an earlier compilation.
func compileCached(d string) compiled {
	pathCacheMu.Lock()
	c, ok := pathCache[d]
	pathCacheMu.Unlock()
	if ok {
		return c
	}
	c = compilePath(d)
	pathCacheMu.Lock()
	if len(pathCache) >= maxCachedPaths {
		pathCache = map[string]compiled{}
	}
	pathCache[d] = c
	pathCacheMu.Unlock()
	return c
}
