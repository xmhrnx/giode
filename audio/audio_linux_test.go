//go:build linux && !android

package audio

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"
)

// TestLoopback plays a tone to a sink and records it back from the
// sink's monitor. It runs where a PulseAudio or PipeWire server is (CI
// has none) and GIODE_AUDIO_SINK names a sink to use.
func TestLoopback(t *testing.T) {
	sink := getenv("GIODE_AUDIO_SINK")
	if sink == "" {
		t.Skip("set GIODE_AUDIO_SINK to a sink to play into")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	outs, err := Outputs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range outs {
		found = found || d.ID == sink
	}
	if !found {
		t.Fatalf("sink %q not among %v", sink, outs)
	}
	var mu sync.Mutex
	var got []int16
	in, err := OpenInput(ctx, nil, Options{Device: sink + ".monitor", SampleRate: 16000}, func(s []int16) {
		mu.Lock()
		got = append(got, s...)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := OpenOutput(ctx, nil, Options{Device: sink, SampleRate: 16000})
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	tone := make([]int16, 16000) // a second of 440 Hz
	for i := range tone {
		tone[i] = int16(8000 * math.Sin(2*math.Pi*440*float64(i)/16000))
	}
	if _, err := out.Write(ctx, tone); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond)
	mu.Lock()
	rec := append([]int16(nil), got...)
	mu.Unlock()
	if len(rec) < 8000 {
		t.Fatalf("recorded %d samples", len(rec))
	}
	if l := Level(rec); l < 0.5 {
		t.Errorf("level %.2f: the tone wasn't recorded", l)
	}
	// Zero crossings: about 880 a second.
	cross := 0
	for i := 1; i < len(rec); i++ {
		if (rec[i-1] < 0) != (rec[i] < 0) {
			cross++
		}
	}
	secs := float64(len(rec)) / 16000
	t.Logf("recorded %.2fs, level %.2f, %.0f crossings a second", secs, Level(rec), float64(cross)/secs)
}

func getenv(k string) string { return lookupEnv(k) }
