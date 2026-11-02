package giode

import (
	"fmt"
	"image"
	"slices"
	"sync"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// FrameStats summarizes how long recent frames took, from
// App.FrameStats.
type FrameStats struct {
	// Frames is the number of frames measured: up to the last 240,
	// since the app started or ResetFrameStats.
	Frames int
	// Build is the time spent building a frame: running the view
	// function and laying it out, on the UI goroutine.
	Build FrameTiming
	// Render is the time spent handing the frame to the GPU: Gio
	// encodes it, submits it and presents it. Presenting waits when the
	// GPU is behind, so a GPU-bound UI shows up here; it may also wait
	// for the display's refresh, so on a vsynced display Render alone
	// can reach the refresh interval while the app is otherwise idle.
	Render FrameTiming
	// Interval is the time from one frame's start to the next, while
	// frames follow each other (an animation, a stream, a scroll):
	// gaps over a quarter second, when the app had nothing to draw, are
	// left out.
	Interval FrameTiming
	// FPS is the frame rate over those intervals; 0 without any.
	FPS float64
}

// FrameTiming is the average, 95th percentile and worst of a frame
// phase.
type FrameTiming struct {
	Avg, P95, Max time.Duration
}

// String formats t in milliseconds.
func (t FrameTiming) String() string {
	return fmt.Sprintf("avg %s, p95 %s, max %s", ms(t.Avg), ms(t.P95), ms(t.Max))
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
}

// idleGap is the longest interval counted as consecutive frames.
const idleGap = 250 * time.Millisecond

type frameSample struct {
	build, render, interval time.Duration
}

// frameRecorder keeps the timings of the last frames. The UI goroutine
// writes it; FrameStats may read it from anywhere.
type frameRecorder struct {
	mu        sync.Mutex
	buf       [240]frameSample
	n, next   int
	lastStart time.Time
}

func (r *frameRecorder) add(start, built, rendered time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := frameSample{build: built.Sub(start), render: rendered.Sub(built)}
	if !r.lastStart.IsZero() {
		s.interval = start.Sub(r.lastStart)
	}
	r.lastStart = start
	r.buf[r.next] = s
	r.next = (r.next + 1) % len(r.buf)
	r.n = min(r.n+1, len(r.buf))
}

func (r *frameRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.n, r.next = 0, 0
	r.lastStart = time.Time{}
}

func (r *frameRecorder) stats() FrameStats {
	r.mu.Lock()
	samples := make([]frameSample, r.n)
	for i := range samples {
		samples[i] = r.buf[(r.next-r.n+i+len(r.buf))%len(r.buf)]
	}
	r.mu.Unlock()
	st := FrameStats{Frames: len(samples)}
	var build, render, interval []time.Duration
	for _, s := range samples {
		build = append(build, s.build)
		render = append(render, s.render)
		if s.interval > 0 && s.interval <= idleGap {
			interval = append(interval, s.interval)
		}
	}
	st.Build, st.Render, st.Interval = timing(build), timing(render), timing(interval)
	if st.Interval.Avg > 0 {
		st.FPS = float64(time.Second) / float64(st.Interval.Avg)
	}
	return st
}

func timing(ds []time.Duration) FrameTiming {
	if len(ds) == 0 {
		return FrameTiming{}
	}
	slices.Sort(ds)
	var sum time.Duration
	for _, d := range ds {
		sum += d
	}
	return FrameTiming{
		Avg: sum / time.Duration(len(ds)),
		P95: ds[min(len(ds)-1, len(ds)*95/100)],
		Max: ds[len(ds)-1],
	}
}

// FrameStats returns how long the last frames (up to 240) took to build
// and to render, and the frame rate while frames followed each other.
// The built-in MCP server reports it as the frame_stats tool;
// Config.ShowFrameTimes shows it on screen. Safe to call from any
// goroutine.
func (a *App) FrameStats() FrameStats {
	return a.frames.stats()
}

// ResetFrameStats discards the frames measured so far, to measure one
// interaction (a streamed reply, a scroll) on its own.
func (a *App) ResetFrameStats() {
	a.frames.reset()
}

// layoutFrameTimes draws the frame time counter in the top right
// corner, for Config.ShowFrameTimes. It shows the stats as of the
// previous frame.
func (a *App) layoutFrameTimes(gtx layout.Context) {
	st := a.frames.stats()
	if st.Frames == 0 {
		return
	}
	fps := "idle"
	if st.FPS > 0 {
		fps = fmt.Sprintf("%.0f fps", st.FPS)
	}
	label := fmt.Sprintf("build %s  render %s  %s", ms(st.Build.Avg), ms(st.Render.Avg), fps)
	box := elements.Box(styles.Styles{
		Background:   "#000000b3",
		BorderRadius: 6,
		Padding:      properties.SymmetricInset(8, 4),
	}, elements.Text(label, styles.Styles{Color: "#ffffff", FontSize: 11}))
	macro := op.Record(gtx.Ops)
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	dims := box.Layout(cgtx)
	call := macro.Stop()
	top := gtx.Dp(8)
	if a.cfg.EdgeToEdge {
		top += gtx.Dp(unit.Dp(SafeArea().Top))
	}
	pos := image.Pt(gtx.Constraints.Max.X-dims.Size.X-gtx.Dp(8), top)
	defer op.Offset(pos).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}
