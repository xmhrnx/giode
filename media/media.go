// Package media plays audio and video files and streams on each
// platform's own engine: AVPlayer on iOS and macOS, MediaPlayer on
// Android, Media Foundation's media engine on Windows and GStreamer on
// Linux (loaded when a player opens; nothing is needed to build).
//
//	p, err := media.Open(ctx, app, "talk.mp4", media.Options{Autoplay: true})
//	...
//	p.View(media.ViewOptions{}) // the picture, captions and controls
//
// Play, Pause, Seek, SetSpeed and SetVolume control it from any
// goroutine. Captions come from SRT or WebVTT files (AddCaptions).
package media

import (
	"context"
	"errors"
	"image"
	"sync"
	"time"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/video"
)

// State is where a player is.
type State int

const (
	// Loading is before the media's ready, and while it buffers.
	Loading State = iota
	Paused
	Playing
	// Ended is after the last frame, unless it loops.
	Ended
	// Failed is after an error: see Err.
	Failed
)

func (s State) String() string {
	return [...]string{"loading", "paused", "playing", "ended", "failed"}[s]
}

// Options configure Open.
type Options struct {
	// Autoplay starts playing as soon as the media's ready.
	Autoplay bool
	// Loop starts again from the beginning at the end.
	Loop bool
	// Muted starts with the sound off.
	Muted bool
	// Volume is 0 to 1; 1 when zero. Use Muted for silence.
	Volume float64
	// OnEnded is called at the end, unless it loops.
	OnEnded func()
	// OnError is called when playing fails.
	OnError func(error)
}

// Player plays one file or stream.
type Player struct {
	app     *giode.App
	opts    Options
	surface *video.Surface
	eng     engine

	mu       sync.Mutex
	state    State
	err      error
	speed    float64
	volume   float64
	muted    bool
	dur      time.Duration
	size     image.Point
	tracks   []Track
	track    int
	closed   bool
	stopTick chan struct{}
	// seekPos and seekAt cover the moment after a seek, before the
	// engine reports the new position.
	seekPos time.Duration
	seekAt  time.Time

	ui uiState
}

// ErrUnsupported is returned where the platform can't play media, or
// can't play this source.
var ErrUnsupported = errors.New("media: not supported")

// Open loads src, a file path or an http(s) or file URL. It returns
// once the engine has started loading; the player is Loading until the
// media's ready.
func Open(ctx context.Context, app *giode.App, src string, opts Options) (*Player, error) {
	if opts.Volume <= 0 {
		opts.Volume = 1
	}
	p := &Player{app: app, opts: opts, speed: 1, volume: opts.Volume, muted: opts.Muted, track: -1}
	p.surface = video.NewSurface(app)
	eng, err := newEngine(ctx, p, src)
	if err != nil {
		return nil, err
	}
	p.eng = eng
	eng.setVolume(p.effectiveVolume())
	return p, nil
}

// Surface is where the video's frames go; View shows it.
func (p *Player) Surface() *video.Surface { return p.surface }

// Play plays from the current position; from the start when it ended.
func (p *Player) Play() {
	p.mu.Lock()
	if p.closed || p.state == Failed {
		p.mu.Unlock()
		return
	}
	restart := p.state == Ended
	if p.state != Loading {
		p.state = Playing
	}
	p.mu.Unlock()
	if restart {
		p.Seek(0)
	}
	p.eng.play()
	p.startTicker()
	p.invalidate()
}

// Pause pauses.
func (p *Player) Pause() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	if p.state == Playing {
		p.state = Paused
	}
	p.mu.Unlock()
	p.eng.pause()
	p.invalidate()
}

// Toggle plays when paused and pauses when playing.
func (p *Player) Toggle() {
	if p.State() == Playing {
		p.Pause()
	} else {
		p.Play()
	}
}

// Seek moves to d, clamped to the media.
func (p *Player) Seek(d time.Duration) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	d = max(d, 0)
	if p.dur > 0 {
		d = min(d, p.dur)
	}
	p.seekPos, p.seekAt = d, time.Now()
	if p.state == Ended {
		p.state = Paused
	}
	p.mu.Unlock()
	p.eng.seek(d)
	p.invalidate()
}

// SetSpeed sets the playback rate: 1 is normal, 0.5 half speed, 2
// double. Speech keeps its pitch where the platform can.
func (p *Player) SetSpeed(r float64) {
	if r <= 0 {
		return
	}
	p.mu.Lock()
	p.speed = r
	p.mu.Unlock()
	p.eng.setRate(r)
	p.invalidate()
}

// Speed is the playback rate.
func (p *Player) Speed() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.speed
}

// SetVolume sets the volume, 0 to 1.
func (p *Player) SetVolume(v float64) {
	p.mu.Lock()
	p.volume = min(max(v, 0), 1)
	p.mu.Unlock()
	p.eng.setVolume(p.effectiveVolume())
	p.invalidate()
}

// Volume is the volume, 0 to 1.
func (p *Player) Volume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

// SetMuted turns the sound off or back on.
func (p *Player) SetMuted(m bool) {
	p.mu.Lock()
	p.muted = m
	p.mu.Unlock()
	p.eng.setVolume(p.effectiveVolume())
	p.invalidate()
}

// Muted reports whether the sound is off.
func (p *Player) Muted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.muted
}

func (p *Player) effectiveVolume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.muted {
		return 0
	}
	return p.volume
}

// Position is how far it has played.
func (p *Player) Position() time.Duration {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0
	}
	if !p.seekAt.IsZero() && time.Since(p.seekAt) < 500*time.Millisecond {
		d := p.seekPos
		p.mu.Unlock()
		return d
	}
	ended, dur := p.state == Ended, p.dur
	p.mu.Unlock()
	if ended && dur > 0 {
		return dur
	}
	return max(p.eng.position(), 0)
}

// Duration is the media's length; zero until known, and for live
// streams.
func (p *Player) Duration() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.dur
}

// State is where the player is.
func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// Err is why it Failed.
func (p *Player) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

// VideoSize is the picture's size; zero for audio, and until known.
func (p *Player) VideoSize() image.Point {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.size
}

// Close stops playing and releases the engine.
func (p *Player) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.stopTickerLocked()
	p.mu.Unlock()
	p.eng.close()
	p.surface.Clear()
}

// The engine reports through these, from any goroutine.

// ready is called once the media's loaded.
func (p *Player) ready(dur time.Duration, size image.Point) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.dur, p.size = dur, size
	play := p.opts.Autoplay || p.state == Playing
	if p.state == Loading {
		p.state = Paused
	}
	p.mu.Unlock()
	if play {
		p.Play()
	}
	p.invalidate()
}

// setDuration updates the duration when the engine learns it late.
func (p *Player) setDuration(d time.Duration) {
	p.mu.Lock()
	p.dur = d
	p.mu.Unlock()
	p.invalidate()
}

func (p *Player) setVideoSize(s image.Point) {
	p.mu.Lock()
	p.size = s
	p.mu.Unlock()
}

// ended is called at the end of the media.
func (p *Player) ended() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	if p.opts.Loop {
		p.mu.Unlock()
		p.eng.seek(0)
		p.eng.play()
		return
	}
	p.state = Ended
	p.stopTickerLocked()
	fn := p.opts.OnEnded
	p.mu.Unlock()
	p.eng.pause()
	p.invalidate()
	if fn != nil {
		fn()
	}
}

// failed is called when playing fails.
func (p *Player) failed(err error) {
	p.mu.Lock()
	if p.closed || p.state == Failed {
		p.mu.Unlock()
		return
	}
	p.state, p.err = Failed, err
	p.stopTickerLocked()
	fn := p.opts.OnError
	p.mu.Unlock()
	p.invalidate()
	if fn != nil {
		fn(err)
	}
}

func (p *Player) invalidate() {
	if p.app != nil {
		p.app.Invalidate()
	}
}

// startTicker redraws while playing, so the position moves even
// without video frames.
func (p *Player) startTicker() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopTick != nil || p.app == nil {
		return
	}
	stop := make(chan struct{})
	p.stopTick = stop
	go func() {
		t := time.NewTicker(250 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				p.invalidate()
			}
		}
	}()
}

func (p *Player) stopTickerLocked() {
	if p.stopTick != nil {
		close(p.stopTick)
		p.stopTick = nil
	}
}

// engine is a platform's player. Its methods may be called from any
// goroutine; it reports to its Player's ready, ended, failed and pushes
// frames to its surface.
type engine interface {
	play()
	pause()
	seek(time.Duration)
	setRate(float64)
	setVolume(float64)
	position() time.Duration
	close()
}
