//go:build (linux && !android) || windows || (darwin && !ios)

package media

import (
	"context"
	"image/color"
	"runtime"
	"testing"
	"time"
)

// testClip is a 2 s, 64×48 clip: red for a second, then blue, with a
// tone. Linux's GStreamer here may lack H.264; Windows lacks VP8.
func testClip() string {
	if runtime.GOOS != "linux" {
		return "testdata/clip.mp4"
	}
	return "testdata/clip.webm"
}

func TestPlayback(t *testing.T) {
	silentForTests = true
	ended := make(chan struct{}, 1)
	p, err := Open(context.Background(), nil, testClip(), Options{OnEnded: func() { ended <- struct{}{} }})
	if err != nil {
		t.Skipf("no media engine: %v", err)
	}
	defer p.Close()
	if !eventually(func() bool { return p.State() == Paused || p.State() == Failed }) {
		// AVFoundation reports to the main run loop, which a test
		// doesn't run.
		t.Skip("the engine never got ready")
	}
	if p.State() == Failed {
		t.Skipf("can't decode the clip here: %v", p.Err())
	}
	if d := p.Duration(); d < 1900*time.Millisecond || d > 2100*time.Millisecond {
		t.Errorf("duration %v", d)
	}
	waitFor(t, "first frame", func() bool { return p.Surface().Size().X == 64 })
	if c := centre(p); !isColor(c, 255, 0, 0) {
		t.Errorf("first frame %v, want red", c)
	}
	if p.VideoSize().X != 64 || p.VideoSize().Y != 48 {
		t.Errorf("video size %v", p.VideoSize())
	}

	p.Play()
	if p.State() != Playing {
		t.Fatalf("state %v", p.State())
	}
	waitFor(t, "the blue second", func() bool { return p.Position() > 1300*time.Millisecond })
	if c := centre(p); !isColor(c, 0, 0, 255) {
		t.Errorf("frame at %v is %v, want blue", p.Position(), c)
	}
	p.Pause()
	p.Seek(200 * time.Millisecond)
	waitFor(t, "the seek", func() bool { return isColor(centre(p), 255, 0, 0) })
	if pos := p.Position(); pos < 100*time.Millisecond || pos > 400*time.Millisecond {
		t.Errorf("position after seek %v", pos)
	}

	// Double speed: the remaining 1.8 s take about 0.9.
	p.SetSpeed(2)
	start := time.Now()
	p.Play()
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("never ended")
	}
	if el := time.Since(start); el > 1600*time.Millisecond {
		t.Errorf("double speed took %v", el)
	}
	if p.State() != Ended || p.Position() != p.Duration() {
		t.Errorf("after the end: %v at %v", p.State(), p.Position())
	}
	// Play again starts over.
	p.SetSpeed(1)
	p.Play()
	waitFor(t, "a restart", func() bool { return p.State() == Playing && p.Position() < time.Second })
}

func TestMissingFile(t *testing.T) {
	silentForTests = true
	failed := make(chan error, 1)
	p, err := Open(context.Background(), nil, "testdata/nope.webm", Options{OnError: func(err error) { failed <- err }})
	if err != nil {
		t.Skipf("no media engine: %v", err)
	}
	defer p.Close()
	select {
	case err := <-failed:
		if p.State() != Failed || err == nil {
			t.Errorf("state %v, %v", p.State(), err)
		}
	case <-time.After(5 * time.Second):
		if runtime.GOOS == "darwin" {
			t.Skip("AVFoundation reports to the main run loop, which a test doesn't run")
		}
		t.Fatal("no error for a missing file")
	}
}

func eventually(cond func() bool) bool {
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
	return true
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func centre(p *Player) color.RGBA {
	img := p.Surface().Snapshot()
	if img == nil {
		return color.RGBA{}
	}
	return img.RGBAAt(img.Rect.Dx()/2, img.Rect.Dy()/2)
}

func isColor(c color.RGBA, r, g, b int) bool {
	near := func(x uint8, y int) bool { return int(x)-y < 60 && y-int(x) < 60 }
	return near(c.R, r) && near(c.G, g) && near(c.B, b)
}
