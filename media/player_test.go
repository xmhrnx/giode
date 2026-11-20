package media

import (
	"image"
	"strings"
	"sync"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/video"
)

// fakeEngine records calls and keeps a position.
type fakeEngine struct {
	mu    sync.Mutex
	calls []string
	pos   time.Duration
}

func (f *fakeEngine) log(s string) {
	f.mu.Lock()
	f.calls = append(f.calls, s)
	f.mu.Unlock()
}
func (f *fakeEngine) play()                { f.log("play") }
func (f *fakeEngine) pause()               { f.log("pause") }
func (f *fakeEngine) seek(d time.Duration) { f.log("seek " + d.String()); f.pos = d }
func (f *fakeEngine) setRate(r float64)    { f.log("rate") }
func (f *fakeEngine) setVolume(v float64)  { f.log("volume") }
func (f *fakeEngine) position() time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pos
}
func (f *fakeEngine) close() { f.log("close") }

func fakePlayer(opts Options) (*Player, *fakeEngine) {
	f := &fakeEngine{}
	p := &Player{opts: opts, speed: 1, volume: 1, track: -1, surface: video.NewSurface(nil), eng: f}
	return p, f
}

func TestPlayerStates(t *testing.T) {
	ended := 0
	p, f := fakePlayer(Options{Autoplay: true, OnEnded: func() { ended++ }})
	if p.State() != Loading {
		t.Fatal(p.State())
	}
	p.ready(10*time.Second, image.Pt(640, 360))
	if p.State() != Playing || p.Duration() != 10*time.Second || p.VideoSize() != image.Pt(640, 360) {
		t.Fatalf("after ready: %v %v %v", p.State(), p.Duration(), p.VideoSize())
	}
	p.Seek(20 * time.Second) // clamped
	if p.Position() != 10*time.Second {
		t.Errorf("seek past the end: %v", p.Position())
	}
	p.ended()
	if p.State() != Ended || ended != 1 {
		t.Errorf("ended: %v %d", p.State(), ended)
	}
	p.Play() // from the start
	if p.State() != Playing || !strings.Contains(strings.Join(f.calls, ","), "seek 0s,play") {
		t.Errorf("replay: %v %v", p.State(), f.calls)
	}
	p.Toggle()
	if p.State() != Paused {
		t.Error("toggle didn't pause")
	}
	p.SetMuted(true)
	if p.effectiveVolume() != 0 {
		t.Error("muted volume")
	}
	p.failed(errTest)
	if p.State() != Failed || p.Err() != errTest {
		t.Error("failed")
	}
	p.Play() // ignored once failed
	if p.State() != Failed {
		t.Error("played after failing")
	}
	p.Close()
	p.Close()
	if n := strings.Count(strings.Join(f.calls, ","), "close"); n != 1 {
		t.Errorf("closed %d times", n)
	}
}

type testErr struct{}

func (testErr) Error() string { return "boom" }

var errTest error = testErr{}

func TestLoop(t *testing.T) {
	p, f := fakePlayer(Options{Loop: true})
	p.ready(time.Second, image.Point{})
	p.Play()
	p.ended()
	if p.State() != Playing || !strings.HasSuffix(strings.Join(f.calls, ","), "seek 0s,play") {
		t.Errorf("loop: %v %v", p.State(), f.calls)
	}
}

func TestViewLaysOut(t *testing.T) {
	p, f := fakePlayer(Options{})
	p.ready(90*time.Second, image.Pt(64, 48))
	p.AddCaptions(Track{Label: "English", Cues: []Cue{{0, time.Minute, "Hello"}}})
	f.pos = 30 * time.Second
	if p.Caption() != "Hello" || p.CaptionTrack() != 0 {
		t.Fatalf("caption %q", p.Caption())
	}
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(400, 300)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	dims := p.View(ViewOptions{}).Layout(gtx)
	if dims.Size != image.Pt(400, 300) {
		t.Errorf("view size %v", dims.Size)
	}
	if v := p.ui.seek.Value(); v < 0.32 || v > 0.34 {
		t.Errorf("seek bar at %v", v)
	}
	p.SetCaptionTrack(-1)
	if p.Caption() != "" {
		t.Error("captions off still shows")
	}
}
