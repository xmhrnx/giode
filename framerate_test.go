package giode

import (
	"testing"
	"time"
)

func TestFrameWait(t *testing.T) {
	t0 := time.Now()
	for _, c := range []struct {
		fps   int
		since time.Duration
		want  time.Duration
	}{
		{0, time.Millisecond, 0},              // no cap
		{30, 10 * time.Millisecond, 23333333}, // 33.3 ms apart
		{30, 40 * time.Millisecond, 0},        // already late
		{60, 0, 16666666},
	} {
		got := frameWait(t0, t0.Add(c.since), c.fps)
		if got < 0 {
			got = 0
		}
		if got != c.want {
			t.Errorf("fps %d after %v: wait %v, want %v", c.fps, c.since, got, c.want)
		}
	}
	if w := frameWait(time.Time{}, t0, 30); w != 0 {
		t.Errorf("first frame waits %v", w)
	}
}

func TestSetMaxFPS(t *testing.T) {
	a := &App{}
	a.SetMaxFPS(24)
	if a.MaxFPS() != 24 {
		t.Errorf("MaxFPS %d", a.MaxFPS())
	}
	a.SetMaxFPS(-5)
	if a.MaxFPS() != 0 {
		t.Errorf("negative cap: %d, want 0", a.MaxFPS())
	}
}
