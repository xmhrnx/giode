// Package audio streams the microphone and the speaker: capture for
// recording, dictation and calls, playback for calls and generated
// sound, with device selection. Voice streams turn on the platform's
// echo cancellation, noise suppression and gain control, as calls need.
//
//	in, err := audio.OpenInput(ctx, app, audio.Options{Voice: true}, func(s []int16) {
//		encoder.Write(s) // 20 ms of 48 kHz mono
//	})
//	out, err := audio.OpenOutput(ctx, app, audio.Options{Voice: true})
//	out.Write(decoded)
//
// Samples are 16-bit, interleaved by channel. Where it runs:
//
//   - Android: AudioRecord and AudioTrack; voice streams use the
//     communication mode and the platform's echo canceler, noise
//     suppressor and gain control.
//   - iOS and macOS: an audio unit, the voice-processing one for voice
//     streams (echo cancellation, noise suppression and gain control).
//   - Windows: WASAPI; voice streams are communications streams, which
//     the system's voice processing (Voice Clarity on Windows 11, or the
//     driver's) applies to.
//   - Linux: PulseAudio or PipeWire; voice streams ask for the sound
//     server's echo-cancel filter.
//
// The web isn't supported. Apps need the microphone permission: on
// Android the package adds RECORD_AUDIO to the manifest; on iOS and
// macOS set ios.usage.microphone (NSMicrophoneUsageDescription).
package audio

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/xmhrnx/giode"
)

// ErrPermissionDenied is returned when the user didn't allow the
// microphone.
var ErrPermissionDenied = errors.New("audio: microphone permission denied")

// ErrClosed is returned by a closed stream's methods.
var ErrClosed = errors.New("audio: closed")

// Device is an audio input or output device.
type Device struct {
	// ID selects the device in Options.
	ID string
	// Name is shown to the user: "MacBook Pro Microphone", "Jabra
	// Evolve2".
	Name string
	// Default is the system's choice.
	Default bool
}

// Options configure a stream.
type Options struct {
	// Device is the ID of the device to use; empty for the default.
	Device string
	// SampleRate is the sample rate: 48000 by default.
	SampleRate int
	// Channels is 1 (the default) or 2.
	Channels int
	// Frame is how much audio OnData receives at a time: 20 ms by
	// default.
	Frame time.Duration
	// Voice processes the stream for speech in calls: echo
	// cancellation, noise suppression and gain control where the
	// platform has them, and the call audio route on phones (the
	// earpiece or headset; SpeakerPhone for the loudspeaker).
	Voice bool
	// SpeakerPhone plays a voice stream through the loudspeaker on
	// phones.
	SpeakerPhone bool
}

func (o *Options) defaults() {
	if o.SampleRate <= 0 {
		o.SampleRate = 48000
	}
	if o.Channels != 2 {
		o.Channels = 1
	}
	if o.Frame <= 0 {
		o.Frame = 20 * time.Millisecond
	}
}

// frameSamples is the number of samples (all channels) in a frame.
func (o *Options) frameSamples() int {
	return int(int64(o.SampleRate)*int64(o.Frame)/int64(time.Second)) * o.Channels
}

// Inputs lists the microphones and other input devices.
func Inputs(ctx context.Context) ([]Device, error) { return devices(ctx, true) }

// Outputs lists the speakers, headphones and other output devices.
func Outputs(ctx context.Context) ([]Device, error) { return devices(ctx, false) }

// RequestPermission asks for the microphone, once; it reports whether
// the app may record.
func RequestPermission(ctx context.Context, a *giode.App) (bool, error) {
	return requestPermission(ctx, a)
}

// Input is an open input stream.
type Input struct {
	opts   Options
	onData func([]int16)
	plat   platformInput

	mu     sync.Mutex
	closed bool
	// pending collects samples until a frame's worth.
	pending []int16
}

// OpenInput starts capturing, calling onData with each frame of samples
// on its own goroutine. onData must return quickly: keep up, or copy and
// hand the work off. The samples are onData's to keep.
func OpenInput(ctx context.Context, a *giode.App, opts Options, onData func(samples []int16)) (*Input, error) {
	opts.defaults()
	if ok, err := requestPermission(ctx, a); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrPermissionDenied
	}
	in := &Input{opts: opts, onData: onData}
	if err := in.open(ctx, a); err != nil {
		return nil, err
	}
	return in, nil
}

// deliver gathers captured samples into frames for onData.
func (in *Input) deliver(s []int16) {
	in.mu.Lock()
	if in.closed {
		in.mu.Unlock()
		return
	}
	n := in.opts.frameSamples()
	in.pending = append(in.pending, s...)
	var frames [][]int16
	for len(in.pending) >= n {
		f := make([]int16, n)
		copy(f, in.pending)
		frames = append(frames, f)
		in.pending = in.pending[n:]
	}
	if len(in.pending) == 0 {
		in.pending = in.pending[:0:0]
	}
	in.mu.Unlock()
	for _, f := range frames {
		in.onData(f)
	}
}

// Options returns the stream's options, its defaults filled in.
func (in *Input) Options() Options { return in.opts }

// Close stops capturing.
func (in *Input) Close() error {
	in.mu.Lock()
	if in.closed {
		in.mu.Unlock()
		return nil
	}
	in.closed = true
	in.mu.Unlock()
	return in.plat.close()
}

// Output is an open output stream.
type Output struct {
	opts Options
	plat platformOutput
	ring *ring

	mu     sync.Mutex
	closed bool
}

// maxQueued is how much audio Write queues before it blocks.
const maxQueued = 300 * time.Millisecond

// OpenOutput opens an output stream; Write queues samples to play.
func OpenOutput(ctx context.Context, a *giode.App, opts Options) (*Output, error) {
	opts.defaults()
	o := &Output{opts: opts}
	o.ring = newRing(int(int64(opts.SampleRate)*int64(maxQueued)/int64(time.Second)) * opts.Channels)
	if err := o.open(ctx, a); err != nil {
		return nil, err
	}
	return o, nil
}

// Write queues samples to play, blocking while more than 300 ms are
// queued, until ctx ends or the stream closes. It returns how many it
// queued.
func (o *Output) Write(ctx context.Context, samples []int16) (int, error) {
	n := 0
	for len(samples) > 0 {
		if o.isClosed() {
			return n, ErrClosed
		}
		k := o.ring.write(samples)
		n += k
		samples = samples[k:]
		if len(samples) == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return n, ctx.Err()
		case <-o.ring.space:
		case <-time.After(10 * time.Millisecond):
		}
	}
	return n, nil
}

// Buffered is how much audio is queued.
func (o *Output) Buffered() time.Duration {
	return time.Duration(o.ring.len()/o.opts.Channels) * time.Second / time.Duration(o.opts.SampleRate)
}

// Flush drops what's queued, as interrupting a sound does.
func (o *Output) Flush() { o.ring.reset() }

// Options returns the stream's options, its defaults filled in.
func (o *Output) Options() Options { return o.opts }

func (o *Output) isClosed() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.closed
}

// Close stops playing.
func (o *Output) Close() error {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return nil
	}
	o.closed = true
	o.mu.Unlock()
	return o.plat.close()
}

// pull fills dst from the queue, silence where it's empty, for the
// platform's playback.
func (o *Output) pull(dst []int16) {
	n := o.ring.read(dst)
	clear(dst[n:])
}

// ring is a queue of samples.
type ring struct {
	mu    sync.Mutex
	buf   []int16
	r, n  int
	space chan struct{}
}

func newRing(size int) *ring {
	return &ring{buf: make([]int16, max(size, 1)), space: make(chan struct{}, 1)}
}

func (q *ring) write(s []int16) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	k := min(len(s), len(q.buf)-q.n)
	for i := 0; i < k; i++ {
		q.buf[(q.r+q.n+i)%len(q.buf)] = s[i]
	}
	q.n += k
	return k
}

func (q *ring) read(dst []int16) int {
	q.mu.Lock()
	k := min(len(dst), q.n)
	for i := 0; i < k; i++ {
		dst[i] = q.buf[(q.r+i)%len(q.buf)]
	}
	q.r = (q.r + k) % len(q.buf)
	q.n -= k
	q.mu.Unlock()
	if k > 0 {
		select {
		case q.space <- struct{}{}:
		default:
		}
	}
	return k
}

func (q *ring) len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.n
}

func (q *ring) reset() {
	q.mu.Lock()
	q.r, q.n = 0, 0
	q.mu.Unlock()
}

// Level returns the loudness of samples from 0 to 1, for meters: their
// root mean square on a 60 dB scale.
func Level(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		f := float64(s) / 32768
		sum += f * f
	}
	rms := sum / float64(len(samples))
	if rms <= 1e-12 {
		return 0
	}
	db := 10 * math.Log10(rms) // -inf..0
	return max(0, min(1, (db+60)/60))
}
