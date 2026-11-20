package audio

import (
	"context"
	"os"
	"testing"
	"time"
)

func lookupEnv(k string) string { return os.Getenv(k) }

func TestRingAndFrames(t *testing.T) {
	o := &Output{opts: Options{SampleRate: 1000, Channels: 1}}
	o.ring = newRing(10)
	n, err := o.Write(context.Background(), []int16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if n != 10 || err != nil {
		t.Fatal(n, err)
	}
	if b := o.Buffered(); b != 10*time.Millisecond {
		t.Errorf("buffered %v", b)
	}
	dst := make([]int16, 4)
	o.pull(dst)
	if dst[0] != 1 || dst[3] != 4 {
		t.Errorf("pulled %v", dst)
	}
	// Writing past the space waits for room.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if n, err := o.Write(ctx, make([]int16, 6)); n != 4 || err == nil {
		t.Errorf("full: %d %v", n, err)
	}
	o.Flush()
	o.pull(dst)
	if dst[0] != 0 {
		t.Errorf("after flush %v", dst)
	}
	var frames [][]int16
	in := &Input{opts: Options{SampleRate: 1000, Channels: 1, Frame: 5 * time.Millisecond}, onData: func(s []int16) { frames = append(frames, s) }}
	in.deliver([]int16{1, 2, 3})
	in.deliver([]int16{4, 5, 6, 7, 8, 9, 10, 11})
	if len(frames) != 2 || frames[1][0] != 6 || len(in.pending) != 1 {
		t.Errorf("frames %v pending %v", frames, in.pending)
	}
	if Level([]int16{32767, -32768}) < 0.99 || Level(make([]int16, 10)) != 0 {
		t.Error("level")
	}
}
