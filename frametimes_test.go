package giode

import (
	"testing"
	"time"
)

func TestFrameStats(t *testing.T) {
	var r frameRecorder
	t0 := time.Unix(0, 0)
	for i := 0; i < 300; i++ {
		start := t0.Add(time.Duration(i) * 10 * time.Millisecond)
		if i >= 150 {
			start = start.Add(time.Second) // an idle gap at frame 150
		}
		build := 2 * time.Millisecond
		if i == 299 {
			build = 30 * time.Millisecond
		}
		r.add(start, start.Add(build), start.Add(build+5*time.Millisecond))
	}
	st := r.stats()
	if st.Frames != 240 {
		t.Errorf("frames %d, want the last 240", st.Frames)
	}
	if st.Build.Max != 30*time.Millisecond || st.Build.P95 != 2*time.Millisecond {
		t.Errorf("build %v, want max 30ms, p95 2ms", st.Build)
	}
	if st.Render.Avg != 5*time.Millisecond {
		t.Errorf("render avg %v, want 5ms", st.Render.Avg)
	}
	if st.Interval.Max != 10*time.Millisecond || int(st.FPS+0.5) != 100 {
		t.Errorf("interval %v, fps %v: want the idle gap left out, 100 fps", st.Interval, st.FPS)
	}
	r.reset()
	if st := r.stats(); st.Frames != 0 || st.FPS != 0 {
		t.Errorf("after reset: %+v", st)
	}
}
