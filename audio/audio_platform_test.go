//go:build windows || darwin

package audio

import (
	"context"
	"testing"
	"time"
)

// TestDevicesAndPlayback lists devices and, where there's an output
// (CI runners may have none), plays a moment of silence through it.
func TestDevicesAndPlayback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ins, err := Inputs(ctx)
	if err != nil {
		t.Logf("inputs: %v", err)
	}
	outs, err := Outputs(ctx)
	if err != nil {
		t.Logf("outputs: %v", err)
	}
	t.Logf("%d inputs, %d outputs", len(ins), len(outs))
	for _, d := range append(ins, outs...) {
		if d.ID == "" {
			t.Errorf("a device without an ID: %+v", d)
		}
	}
	if len(outs) == 0 {
		t.Skip("no output device")
	}
	out, err := OpenOutput(ctx, nil, Options{SampleRate: 48000})
	if err != nil {
		t.Skipf("opening the output: %v", err)
	}
	defer out.Close()
	if _, err := out.Write(ctx, make([]int16, 4800)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if b := out.Buffered(); b > 100*time.Millisecond {
		t.Errorf("%v still queued: nothing plays", b)
	}
}
