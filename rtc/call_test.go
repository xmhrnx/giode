//go:build !js

package rtc

import (
	"context"
	"image"
	"image/color"
	"math"
	"sync"
	"testing"
	"time"
)

// rawVideo is a test "codec": frames go as raw RGBA under the VP8
// name, which only the packetization sees.
type rawVideo struct{}

func (rawVideo) MimeType() string   { return "video/VP8" }
func (rawVideo) Fmtp() string       { return "" }
func (rawVideo) PayloadType() uint8 { return 96 }
func (rawVideo) NewEncoder(w, h, fps int) (VideoEncoder, error) {
	return rawEnc{}, nil
}
func (rawVideo) NewDecoder() (VideoDecoder, error) { return rawDec{}, nil }

type rawEnc struct{}

func (rawEnc) Encode(f *image.RGBA, key bool) ([]byte, error) {
	w, h := f.Rect.Dx(), f.Rect.Dy()
	return append([]byte{byte(w), byte(h)}, f.Pix...), nil
}
func (rawEnc) Close() {}

type rawDec struct{}

func (rawDec) Decode(b []byte) (image.Image, error) {
	w, h := int(b[0]), int(b[1])
	if len(b) != 2+w*h*4 {
		return nil, nil
	}
	return &image.RGBA{Pix: b[2:], Stride: w * 4, Rect: image.Rect(0, 0, w, h)}, nil
}
func (rawDec) Close() {}

func connect(t *testing.T, a, b *Call, trickle bool, ac, bc chan string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	offer, err := a.Offer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := b.Answer(ctx, offer)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Accept(answer); err != nil {
		t.Fatal(err)
	}
	if trickle {
		go func() {
			for c := range ac {
				b.AddCandidate(c)
			}
		}()
		go func() {
			for c := range bc {
				a.AddCandidate(c)
			}
		}()
	}
	deadline := time.Now().Add(10 * time.Second)
	for a.State() != StateConnected || b.State() != StateConnected {
		if time.Now().After(deadline) {
			t.Fatalf("not connected: %v, %v", a.State(), b.State())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestAudioAndData(t *testing.T) {
	var (
		mu       sync.Mutex
		received []int16
		rate     int
	)
	ac, bc := make(chan string, 64), make(chan string, 64)
	a, err := New(context.Background(), nil, Config{SendAudio: true, loopback: true, OnCandidate: func(c string) { ac <- c }})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	gotChannel := make(chan *DataChannel, 1)
	b, err := New(context.Background(), nil, Config{
		loopback: true, OnCandidate: func(c string) { bc <- c },
		OnRemoteAudio: func(pcm []int16, r int) {
			mu.Lock()
			received = append(received, pcm...)
			rate = r
			mu.Unlock()
		},
		OnDataChannel: func(d *DataChannel) { gotChannel <- d },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	dc, err := a.OpenDataChannel("chat", true)
	if err != nil {
		t.Fatal(err)
	}
	opened := make(chan struct{})
	dc.OnOpen(func() { close(opened) })
	connect(t, a, b, true, ac, bc)

	if got := a.AudioCodec(); got != "audio/PCMU" {
		t.Errorf("codec %q", got)
	}
	// One second of 440 Hz, in 20 ms frames, in real time.
	sr := a.AudioSampleRate()
	for f := 0; f < 50; f++ {
		frame := make([]int16, sr/50)
		for i := range frame {
			n := f*len(frame) + i
			frame[i] = int16(8000 * math.Sin(2*math.Pi*440*float64(n)/float64(sr)))
		}
		if err := a.WriteAudio(frame); err != nil {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	got, r := append([]int16(nil), received...), rate
	mu.Unlock()
	if r != 8000 || len(got) < 6000 {
		t.Fatalf("received %d samples at %d Hz", len(got), r)
	}
	// Count zero crossings: 440 Hz crosses 880 times a second.
	cross := 0
	for i := 1; i < len(got); i++ {
		if (got[i-1] < 0) != (got[i] < 0) {
			cross++
		}
	}
	hz := float64(cross) / 2 / (float64(len(got)) / 8000)
	if hz < 420 || hz > 460 {
		t.Errorf("received %.0f Hz, want 440", hz)
	}
	if s := b.Stats(); s.AudioReceived < 40 || s.RemoteLevel < 0.05 {
		t.Errorf("stats %+v", s)
	}

	// Muted sends silence.
	a.SetMuted(true)
	mu.Lock()
	received = nil
	mu.Unlock()
	for f := 0; f < 10; f++ {
		frame := make([]int16, sr/50)
		for i := range frame {
			frame[i] = 8000
		}
		a.WriteAudio(frame)
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)
	mu.Lock()
	for _, s := range received {
		if s > 100 || s < -100 {
			t.Fatalf("muted audio got through: %d", s)
		}
	}
	mu.Unlock()

	// Data both ways.
	select {
	case <-opened:
	case <-time.After(5 * time.Second):
		t.Fatal("data channel never opened")
	}
	var bdc *DataChannel
	select {
	case bdc = <-gotChannel:
	case <-time.After(5 * time.Second):
		t.Fatal("no data channel on the other side")
	}
	msgs := make(chan string, 2)
	bdc.OnMessage(func(d []byte, text bool) {
		if text {
			msgs <- string(d)
			bdc.SendText("pong")
		}
	})
	dc.OnMessage(func(d []byte, text bool) { msgs <- string(d) })
	if bdc.Label() != "chat" {
		t.Errorf("label %q", bdc.Label())
	}
	dc.SendText("ping")
	for _, want := range []string{"ping", "pong"} {
		select {
		case m := <-msgs:
			if m != want {
				t.Errorf("got %q, want %q", m, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("no %q", want)
		}
	}
}

func TestVideo(t *testing.T) {
	a, err := New(context.Background(), nil, Config{loopback: true, VideoCodecs: []VideoCodec{rawVideo{}}, SendVideo: true, VideoFPS: 20})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := New(context.Background(), nil, Config{loopback: true, VideoCodecs: []VideoCodec{rawVideo{}}})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	connect(t, a, b, false, nil, nil)
	frame := image.NewRGBA(image.Rect(0, 0, 8, 6))
	for i := range frame.Pix {
		frame.Pix[i] = 255
	}
	frame.SetRGBA(1, 1, color.RGBA{R: 200, G: 10, B: 30, A: 255})
	deadline := time.Now().Add(5 * time.Second)
	for b.RemoteVideo().Size() != image.Pt(8, 6) {
		if time.Now().After(deadline) {
			t.Fatalf("no video: stats %+v / %+v", a.Stats(), b.Stats())
		}
		if err := a.SendVideo(frame); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got := b.RemoteVideo().Snapshot().RGBAAt(1, 1); got != (color.RGBA{R: 200, G: 10, B: 30, A: 255}) {
		t.Errorf("pixel %v", got)
	}
}
