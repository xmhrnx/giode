package textrender

import (
	"hash/maphash"
	"image"
	"image/color"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/fonts"
)

// frame counts frames, to tell which cached entries are in use.
var frame atomic.Uint64

// Tick advances the frame counter and drops cache entries no frame has
// used for a while. The app calls it once per frame, before laying it
// out, and FrameDone after.
func Tick() {
	startIdleSweep.Do(func() { go idleSweeper() })
	between.Lock()
	inFrame.Store(true)
	between.Unlock()
	f := frame.Add(1)
	// Text laid out this frame uses a fresh shaper once Gio's cache of
	// laid out paragraphs has grown; see fonts.Refresh.
	fonts.Refresh(shapeBudget)
	if f%60 == 0 {
		sweepLayouts(f)
	}
}

// FrameDone marks the end of a frame's layout: until the next Tick, the
// idle sweeper may clear the caches.
func FrameDone() {
	inFrame.Store(false)
}

// shapeBudget is how much text, in bytes, is shaped with one shaper
// before it's replaced: Gio keeps about a hundred times that in laid
// out glyphs, some 25 MB.
const shapeBudget = 256 << 10

var (
	startIdleSweep sync.Once
	// inFrame is set from Tick to FrameDone; between is held while the
	// idle sweeper clears the caches, so a frame waits for it.
	inFrame atomic.Bool
	between sync.Mutex
)

// idleSweep is how long a window draws no frame before the caches let
// go of what the last frame didn't draw (a variable for tests).
var idleSweep atomic.Int64

func init() { idleSweep.Store(int64(10 * time.Second)) }

// idleSweeper clears the caches of a window that has stopped drawing:
// the caches are otherwise swept only as frames are drawn, so an idle
// window kept everything it had shown - a closed chat, a long-gone
// page - until the next input. Once no frame has been drawn for
// idleSweep, it drops what the last frame didn't use and lets Gio's
// shaper cache go; what's on screen stays.
func idleSweeper() {
	var last, swept uint64
	idle := 0
	for {
		time.Sleep(time.Duration(idleSweep.Load()) / 2)
		f := frame.Load()
		if f != last {
			last, idle = f, 0
			continue
		}
		if idle++; idle < 2 || swept == f {
			continue
		}
		between.Lock()
		if !inFrame.Load() {
			sweepUnused(f)
			fonts.Refresh(0)
			swept = f
		}
		between.Unlock()
	}
}

// runKey identifies a line image: its glyphs, their positions relative
// to the run's origin, and the color.
type runKey struct {
	hash uint64
	n    int
	col  color.NRGBA
}

// runImage is a line of glyphs drawn into an image, placed off from the
// run's origin. vector marks a run that has to be drawn as paths.
type runImage struct {
	op     paint.ImageOp
	off    image.Point
	empty  bool
	vector bool
	bytes  int
	used   uint64
}

var runCache struct {
	sync.Mutex
	m     map[runKey]*runImage
	bytes int
}

// maxRunBytes bounds the memory of cached line images; least recently
// used lines go first.
const maxRunBytes = 32 << 20

var seed = maphash.MakeSeed()

// DrawRun draws a line of glyphs in col. The glyphs are positioned as
// Gio's shaper placed them, in the coordinates of the current
// transform.
func DrawRun(ops *op.Ops, sh *text.Shaper, glyphs []text.Glyph, col color.NRGBA) {
	if len(glyphs) == 0 || col.A == 0 {
		return
	}
	drawRunKeyed(ops, sh, glyphs, col, runHash(glyphs))
}

// drawRunKeyed is DrawRun with the glyphs' runHash.
func drawRunKeyed(ops *op.Ops, sh *text.Shaper, glyphs []text.Glyph, col color.NRGBA, hash uint64) {
	if len(glyphs) == 0 || col.A == 0 {
		return
	}
	g0 := glyphs[0]
	ox := (g0.X - g0.Offset.X).Floor()
	oy := int(g0.Y)
	img := runImageFor(sh, glyphs, col, ox, oy, hash)
	if img.vector {
		drawVector(ops, sh, glyphs, col)
		return
	}
	if img.empty {
		return
	}
	defer op.Offset(image.Pt(ox+img.off.X, oy+img.off.Y)).Push(ops).Pop()
	img.op.Add(ops)
	paint.PaintOp{}.Add(ops)
}

// drawVector draws the glyphs as Gio's widgets do.
func drawVector(ops *op.Ops, sh *text.Shaper, glyphs []text.Glyph, col color.NRGBA) {
	g0 := glyphs[0]
	t := op.Affine(f32.AffineId().Offset(f32.Pt(float32(g0.X)/64, float32(g0.Y)))).Push(ops)
	outline := clip.Outline{Path: sh.Shape(glyphs)}.Op().Push(ops)
	paint.ColorOp{Color: col}.Add(ops)
	paint.PaintOp{}.Add(ops)
	outline.Pop()
	if call := sh.Bitmaps(glyphs); call != (op.CallOp{}) {
		call.Add(ops)
	}
	t.Pop()
}

// runHash identifies a run's glyphs and their positions relative to its
// origin: the same line anywhere hashes the same.
func runHash(glyphs []text.Glyph) uint64 {
	if len(glyphs) == 0 {
		return 0
	}
	g0 := glyphs[0]
	base := fixed.I((g0.X - g0.Offset.X).Floor())
	oy := int(g0.Y)
	var h maphash.Hash
	h.SetSeed(seed)
	var buf [24]byte
	for _, g := range glyphs {
		putU64(buf[0:], uint64(g.ID))
		putU64(buf[8:], uint64(int64(g.X-g.Offset.X-base)))
		putU64(buf[16:], uint64(int64(int(g.Y)-oy)<<32|int64(uint32(g.Offset.Y))))
		h.Write(buf[:])
	}
	return h.Sum64()
}

func runImageFor(sh *text.Shaper, glyphs []text.Glyph, col color.NRGBA, ox, oy int, hash uint64) *runImage {
	key := runKey{hash: hash, n: len(glyphs), col: col}
	now := frame.Load()
	runCache.Lock()
	if img, ok := runCache.m[key]; ok {
		img.used = now
		runCache.Unlock()
		return img
	}
	runCache.Unlock()

	img := composeRun(sh, glyphs, col, ox, oy)
	img.used = now

	runCache.Lock()
	defer runCache.Unlock()
	if runCache.m == nil {
		runCache.m = make(map[runKey]*runImage)
	}
	runCache.m[key] = img
	runCache.bytes += img.bytes
	if runCache.bytes > maxRunBytes {
		evictRuns()
	}
	return img
}

// runIdleFrames is how many frames a line image may go unused before
// it's dropped: text scrolled away, a message deleted, a streamed line
// that grew. Text that comes back is drawn again from cached glyphs.
const runIdleFrames = 300

// sweepRuns drops line images no frame has drawn for runIdleFrames, so
// the cache holds what's on screen rather than filling its budget.
func sweepRuns(now uint64) {
	runCache.Lock()
	defer runCache.Unlock()
	for k, v := range runCache.m {
		if now-v.used > runIdleFrames {
			runCache.bytes -= v.bytes
			delete(runCache.m, k)
		}
	}
}

// evictRuns drops the least recently used line images until the cache
// is down to three quarters of its budget.
func evictRuns() {
	type entry struct {
		k    runKey
		used uint64
	}
	entries := make([]entry, 0, len(runCache.m))
	for k, v := range runCache.m {
		entries = append(entries, entry{k, v.used})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].used < entries[j].used })
	for _, e := range entries {
		if runCache.bytes <= maxRunBytes*3/4 {
			break
		}
		runCache.bytes -= runCache.m[e.k].bytes
		delete(runCache.m, e.k)
	}
}

// composeRun rasterizes a run's glyphs into one image.
func composeRun(sh *text.Shaper, glyphs []text.Glyph, col color.NRGBA, ox, oy int) *runImage {
	if vectorOnly.Load() {
		return &runImage{vector: true}
	}
	fs := faces(sh)
	type placed struct {
		b  *glyphBitmap
		at image.Point
	}
	ps := make([]placed, 0, len(glyphs))
	var bounds image.Rectangle
	for _, g := range glyphs {
		ax := g.X - g.Offset.X
		ix := ax.Floor()
		sub := uint8((ax - fixed.I(ix) + 64/subpixels/2) / (64 / subpixels))
		if sub == subpixels {
			ix, sub = ix+1, 0
		}
		b, ok := rasterGlyph(fs, g.ID, sub)
		if !ok {
			return &runImage{vector: true}
		}
		if b.img == nil {
			continue
		}
		at := image.Pt(ix-ox+b.off.X, int(g.Y)-oy-g.Offset.Y.Round()+b.off.Y)
		ps = append(ps, placed{b, at})
		bounds = bounds.Union(image.Rectangle{Min: at, Max: at.Add(b.img.Rect.Size())})
	}
	if bounds.Empty() {
		return &runImage{empty: true}
	}
	cov := make([]uint16, bounds.Dx()*bounds.Dy())
	w := bounds.Dx()
	for _, p := range ps {
		src := p.b.img
		sw := src.Rect.Dx()
		for y := 0; y < src.Rect.Dy(); y++ {
			row := (p.at.Y-bounds.Min.Y+y)*w + p.at.X - bounds.Min.X
			for x, a := range src.Pix[y*src.Stride : y*src.Stride+sw] {
				if a != 0 {
					cov[row+x] += uint16(a)
				}
			}
		}
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, bounds.Dy()))
	ca := uint32(col.A)
	lut := coverageLUT(col)
	for i, c := range cov {
		if c == 0 {
			continue
		}
		if c > 255 {
			c = 255
		}
		px := out.Pix[i*4 : i*4+4 : i*4+4]
		px[0], px[1], px[2] = col.R, col.G, col.B
		px[3] = uint8((uint32(lut[c])*ca + 127) / 255)
	}
	imgOp := paint.NewImageOp(out)
	imgOp.Filter = paint.FilterNearest
	return &runImage{op: imgOp, off: bounds.Min, bytes: len(out.Pix)}
}

func putU64(b []byte, v uint64) {
	for i := 0; i < 8; i++ {
		b[i] = byte(v >> (8 * i))
	}
}

// CacheStats reports the size of the text caches: line images (count
// and bytes) and laid out plain and rich texts.
func CacheStats() (runs, runBytes, plain, rich int) {
	runCache.Lock()
	runs, runBytes = len(runCache.m), runCache.bytes
	runCache.Unlock()
	layouts.Lock()
	plain = len(layouts.m)
	layouts.Unlock()
	richLayouts.Lock()
	rich = len(richLayouts.m)
	richLayouts.Unlock()
	return
}
