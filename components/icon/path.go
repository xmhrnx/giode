package icon

import (
	"math"
	"strconv"
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

// argCount is the number of arguments each path command takes.
func argCount(cmd byte) int {
	switch cmd {
	case 'M', 'L', 'T':
		return 2
	case 'H', 'V':
		return 1
	case 'C':
		return 6
	case 'S', 'Q':
		return 4
	case 'A':
		return 7
	}
	return 0
}

// pathCompiler turns SVG path data into segments. State (current
// point, last control point) persists across commands.
type pathCompiler struct {
	out   compiled
	cur   f32.Point
	start f32.Point
	ctrl  f32.Point // reflected control point for S/T
	open  bool      // a subpath has been started
	prev  byte
}

// compilePath parses SVG path data: commands M, L, H, V, C, S, Q, T, A
// and Z in absolute and relative form, with the compact number syntax
// ("1.5.5", "10-5") SVG allows. Malformed trailing data is ignored, as
// browsers do.
func compilePath(d string) compiled {
	pc := &pathCompiler{}
	var (
		cmd  byte
		args []float32
		arg  int // index of the next argument (arc flags are special)
	)
	flush := func() {
		if cmd != 0 {
			pc.run(cmd, args)
		}
		args = args[:0]
		arg = 0
	}
	for i := 0; i < len(d); {
		c := d[i]
		switch {
		case c == ' ' || c == ',' || c == '\t' || c == '\n' || c == '\r' || c == '\f':
			i++
		case (c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') && c != 'e' && c != 'E':
			flush()
			cmd = c
			i++
			if c == 'z' || c == 'Z' {
				pc.run(c, nil)
				cmd = 0
			}
		default:
			// Arc flags (the 4th and 5th arc arguments) are single
			// digits that may be packed without separators: "a1 1 0 00 1 1".
			if (cmd == 'a' || cmd == 'A') && (arg%7 == 3 || arg%7 == 4) && (c == '0' || c == '1') {
				args = append(args, float32(c-'0'))
				arg++
				i++
				continue
			}
			j := numberEnd(d, i)
			if j == i {
				i++ // skip an unexpected byte
				continue
			}
			v, err := strconv.ParseFloat(d[i:j], 32)
			if err == nil {
				args = append(args, float32(v))
				arg++
			}
			i = j
		}
	}
	flush()
	return pc.out
}

// numberEnd returns the end of the number token starting at i.
func numberEnd(d string, i int) int {
	j := i
	if j < len(d) && (d[j] == '-' || d[j] == '+') {
		j++
	}
	seenDot, seenDigit := false, false
	for j < len(d) {
		switch c := d[j]; {
		case c >= '0' && c <= '9':
			seenDigit = true
			j++
		case c == '.' && !seenDot:
			seenDot = true
			j++
		case (c == 'e' || c == 'E') && seenDigit && j+1 < len(d):
			k := j + 1
			if d[k] == '-' || d[k] == '+' {
				k++
			}
			if k < len(d) && d[k] >= '0' && d[k] <= '9' {
				j = k
				for j < len(d) && d[j] >= '0' && d[j] <= '9' {
					j++
				}
			}
			return j
		default:
			return j
		}
	}
	return j
}

func (pc *pathCompiler) emit(op byte, pts ...f32.Point) {
	s := segment{op: op}
	copy(s.pts[:], pts)
	pc.out = append(pc.out, s)
}

func (pc *pathCompiler) moveTo(p f32.Point) {
	pc.emit('M', p)
	pc.cur, pc.start, pc.ctrl = p, p, p
	pc.open = true
}

func (pc *pathCompiler) lineTo(p f32.Point) {
	if !pc.open {
		pc.moveTo(pc.cur)
	}
	pc.emit('L', p)
	pc.cur, pc.ctrl = p, p
}

// run executes one command with its (possibly repeated) arguments.
func (pc *pathCompiler) run(cmd byte, args []float32) {
	rel := cmd >= 'a'
	cc := cmd
	if rel {
		cc = cmd - 'a' + 'A'
	}
	if cc == 'Z' {
		if pc.open {
			pc.emit('Z')
			pc.cur, pc.ctrl = pc.start, pc.start
			pc.open = false
		}
		pc.prev = 'Z'
		return
	}
	n := argCount(cc)
	if n == 0 {
		return
	}
	pt := func(a []float32) f32.Point {
		if rel {
			return f32.Pt(pc.cur.X+a[0], pc.cur.Y+a[1])
		}
		return f32.Pt(a[0], a[1])
	}
	for i := 0; i+n <= len(args); i += n {
		a := args[i : i+n]
		switch cc {
		case 'M':
			if i == 0 {
				pc.moveTo(pt(a))
			} else {
				// Extra coordinate pairs after a moveto are implicit
				// linetos.
				pc.lineTo(pt(a))
			}
		case 'L':
			pc.lineTo(pt(a))
		case 'H':
			x := a[0]
			if rel {
				x += pc.cur.X
			}
			pc.lineTo(f32.Pt(x, pc.cur.Y))
		case 'V':
			y := a[0]
			if rel {
				y += pc.cur.Y
			}
			pc.lineTo(f32.Pt(pc.cur.X, y))
		case 'C':
			pc.cubic(pt(a[0:2]), pt(a[2:4]), pt(a[4:6]))
		case 'S':
			c1 := pc.cur
			if pc.prev == 'C' || pc.prev == 'S' {
				c1 = reflect(pc.ctrl, pc.cur)
			}
			pc.cubic(c1, pt(a[0:2]), pt(a[2:4]))
		case 'Q':
			pc.quad(pt(a[0:2]), pt(a[2:4]))
		case 'T':
			c1 := pc.cur
			if pc.prev == 'Q' || pc.prev == 'T' {
				c1 = reflect(pc.ctrl, pc.cur)
			}
			pc.quad(c1, pt(a))
		case 'A':
			pc.arc(a[0], a[1], a[2], a[3] != 0, a[4] != 0, pt(a[5:7]))
		}
		pc.prev = cc
	}
}

func (pc *pathCompiler) cubic(c1, c2, to f32.Point) {
	if !pc.open {
		pc.moveTo(pc.cur)
	}
	pc.emit('C', c1, c2, to)
	pc.cur, pc.ctrl = to, c2
}

func (pc *pathCompiler) quad(c, to f32.Point) {
	if !pc.open {
		pc.moveTo(pc.cur)
	}
	pc.emit('Q', c, to)
	pc.cur, pc.ctrl = to, c
}

// reflect mirrors p through center.
func reflect(p, center f32.Point) f32.Point {
	return f32.Pt(2*center.X-p.X, 2*center.Y-p.Y)
}

// arc appends an SVG elliptical arc as cubic Béziers, following the
// endpoint-to-center conversion of the SVG specification (appendix
// F.6) and splitting the sweep into pieces of at most 90 degrees.
func (pc *pathCompiler) arc(rx, ry, rotDeg float32, large, sweep bool, to f32.Point) {
	from := pc.cur
	if from == to {
		return
	}
	rx, ry = float32(math.Abs(float64(rx))), float32(math.Abs(float64(ry)))
	if rx == 0 || ry == 0 {
		pc.lineTo(to)
		return
	}
	phi := float64(rotDeg) * math.Pi / 180
	sinPhi, cosPhi := math.Sincos(phi)

	// Step 1: compute (x1', y1').
	dx, dy := float64(from.X-to.X)/2, float64(from.Y-to.Y)/2
	x1 := cosPhi*dx + sinPhi*dy
	y1 := -sinPhi*dx + cosPhi*dy

	// Scale up radii that are too small.
	r1, r2 := float64(rx), float64(ry)
	if lambda := x1*x1/(r1*r1) + y1*y1/(r2*r2); lambda > 1 {
		s := math.Sqrt(lambda)
		r1, r2 = r1*s, r2*s
	}

	// Step 2: compute (cx', cy').
	num := r1*r1*r2*r2 - r1*r1*y1*y1 - r2*r2*x1*x1
	den := r1*r1*y1*y1 + r2*r2*x1*x1
	coef := 0.0
	if den != 0 && num > 0 {
		coef = math.Sqrt(num / den)
	}
	if large == sweep {
		coef = -coef
	}
	cxp := coef * r1 * y1 / r2
	cyp := -coef * r2 * x1 / r1

	// Step 3: compute the center.
	cx := cosPhi*cxp - sinPhi*cyp + float64(from.X+to.X)/2
	cy := sinPhi*cxp + cosPhi*cyp + float64(from.Y+to.Y)/2

	// Step 4: start angle and sweep.
	angle := func(ux, uy, vx, vy float64) float64 {
		a := math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
		return a
	}
	theta1 := angle(1, 0, (x1-cxp)/r1, (y1-cyp)/r2)
	delta := angle((x1-cxp)/r1, (y1-cyp)/r2, (-x1-cxp)/r1, (-y1-cyp)/r2)
	if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	} else if sweep && delta < 0 {
		delta += 2 * math.Pi
	}

	n := int(math.Ceil(math.Abs(delta) / (math.Pi / 2)))
	if n < 1 {
		n = 1
	}
	step := delta / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	point := func(t float64) (x, y, dxdt, dydt float64) {
		st, ct := math.Sincos(t)
		ex, ey := r1*ct, r2*st
		x = cosPhi*ex - sinPhi*ey + cx
		y = sinPhi*ex + cosPhi*ey + cy
		tx, ty := -r1*st, r2*ct
		dxdt = cosPhi*tx - sinPhi*ty
		dydt = sinPhi*tx + cosPhi*ty
		return
	}
	t := theta1
	for i := 0; i < n; i++ {
		x0, y0, dx0, dy0 := point(t)
		x3, y3, dx3, dy3 := point(t + step)
		c1 := f32.Pt(float32(x0+k*dx0), float32(y0+k*dy0))
		c2 := f32.Pt(float32(x3-k*dx3), float32(y3-k*dy3))
		end := f32.Pt(float32(x3), float32(y3))
		if i == n-1 {
			end = to
		}
		pc.cubic(c1, c2, end)
		t += step
	}
}

// --- Basic shapes ------------------------------------------------------------

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
