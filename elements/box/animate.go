package box

import (
	"time"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Animate makes the box animate every change of its styles, under key:
// when the styles it's built with differ from the last frame's - a
// state changed, a row was selected, a panel expanded - it moves to the
// new ones over the styles' Transition (Duration and Easing), from
// wherever it was. A box is rebuilt every frame, so key (any
// comparable value, unique among animated boxes: a row's ID) keeps its
// animation between frames.
//
//	giode.Box(giode.Styles{
//		Height:     ifThen(open, 240, 48),
//		Background: ifThen(selected, "theme.primary/0.1", "theme.surface"),
//		Transition: giode.StyleTransition{Duration: 200 * time.Millisecond},
//	}, children...).Animate("panel-" + id)
//
// Colors, Opacity, Transform, BorderRadius, BorderWidth, the shadow,
// Padding, Gap, and fixed Width and Height interpolate; other
// properties change at once. With reduced motion every change is
// immediate. It returns b for chaining.
func (b *BoxElement) Animate(key any) *BoxElement {
	b.animKey = key
	return b
}

// Animation plays a keyframe animation on the box (see Keyframes): its
// frames' styles are merged over the box's. It returns b for chaining.
func (b *BoxElement) Animation(a *KeyframeAnimation) *BoxElement {
	b.anim = a
	return b
}

// animScope keeps Animate's states apart from other Keyed values.
var animScope byte

// boxAnim is an animated box's state.
type boxAnim struct {
	init    bool
	from    styles.Styles
	to      styles.Styles
	shown   styles.Styles
	start   time.Time
	running bool
}

// animated returns target as the box shows it this frame: on its way
// from the styles of an earlier frame.
func animated(gtx layout.Context, key any, target styles.Styles) styles.Styles {
	a := base.Keyed(&animScope, key, func() *boxAnim { return new(boxAnim) })
	tr := target.Transition
	if !a.init {
		a.init, a.to, a.shown = true, target, target
		return target
	}
	if target != a.to {
		if tr.Duration <= 0 || kit.ReducedMotion() {
			a.to, a.shown, a.running = target, target, false
			return target
		}
		a.from, a.to, a.start, a.running = a.shown, target, gtx.Now, true
	}
	if !a.running {
		return target
	}
	p := float32(gtx.Now.Sub(a.start)) / float32(tr.Duration)
	if p >= 1 || kit.ReducedMotion() {
		a.running, a.shown = false, target
		return target
	}
	a.shown = blendStyles(a.from, a.to, tr.Easing.Apply(max(p, 0)))
	kit.RequestFrame(gtx)
	return a.shown
}

// Keyframe is a step of a keyframe animation: Styles at At, a fraction
// of the animation from 0 to 1. Its styles are merged over the box's,
// so a keyframe sets only what it animates.
type Keyframe struct {
	At     float32
	Styles styles.Styles
}

// KeyframeAnimation plays styles through keyframes on a Box
// (BoxElement.Animation), like CSS @keyframes: a pulse, a shake, an
// entrance. It keeps its own time, so create it once. With reduced
// motion a repeating animation doesn't play, and a finite one jumps to
// its end.
type KeyframeAnimation struct {
	frames    []Keyframe
	dur       time.Duration
	delay     time.Duration
	easing    properties.Easing
	repeat    int
	alternate bool
	keepEnd   bool
	onDone    func()

	started  bool
	start    time.Time
	paused   bool
	pausedAt time.Duration
	done     bool
}

// Keyframes returns an animation through frames over d, played once.
// Frames are sorted by At; without one at 0 or 1 the box's own styles
// stand in at that end.
func Keyframes(d time.Duration, frames ...Keyframe) *KeyframeAnimation {
	fs := append([]Keyframe(nil), frames...)
	for i := 1; i < len(fs); i++ {
		for j := i; j > 0 && fs[j].At < fs[j-1].At; j-- {
			fs[j], fs[j-1] = fs[j-1], fs[j]
		}
	}
	if len(fs) == 0 || fs[0].At > 0 {
		fs = append([]Keyframe{{At: 0}}, fs...)
	}
	if fs[len(fs)-1].At < 1 {
		fs = append(fs, Keyframe{At: 1})
	}
	return &KeyframeAnimation{frames: fs, dur: d, easing: properties.EaseInOut, repeat: 1, keepEnd: true}
}

// Repeat plays it n times; 0 repeats it forever. It returns a for
// chaining.
func (a *KeyframeAnimation) Repeat(n int) *KeyframeAnimation {
	a.repeat = max(n, 0)
	return a
}

// Alternate plays every other repetition backwards. It returns a for
// chaining.
func (a *KeyframeAnimation) Alternate(on bool) *KeyframeAnimation {
	a.alternate = on
	return a
}

// Easing eases each step between two keyframes (EaseInOut by default).
// It returns a for chaining.
func (a *KeyframeAnimation) Easing(e properties.Easing) *KeyframeAnimation {
	a.easing = e
	return a
}

// Delay waits d before playing. It returns a for chaining.
func (a *KeyframeAnimation) Delay(d time.Duration) *KeyframeAnimation {
	a.delay = d
	return a
}

// KeepEnd keeps the last keyframe's styles once it has played (the
// default); off, the box goes back to its own styles. It returns a for
// chaining.
func (a *KeyframeAnimation) KeepEnd(on bool) *KeyframeAnimation {
	a.keepEnd = on
	return a
}

// OnDone sets a function run when it finishes playing. It returns a
// for chaining.
func (a *KeyframeAnimation) OnDone(fn func()) *KeyframeAnimation {
	a.onDone = fn
	return a
}

// Restart plays it again from the start, on the next frame.
func (a *KeyframeAnimation) Restart() {
	a.started, a.done, a.paused = false, false, false
}

// Pause holds it where it is; Resume goes on from there.
func (a *KeyframeAnimation) Pause() {
	a.paused = true
}

// Resume: see Pause.
func (a *KeyframeAnimation) Resume() {
	a.paused = false
}

// Done reports whether it has finished playing.
func (a *KeyframeAnimation) Done() bool { return a.done }

// apply returns base with the animation's styles at this frame.
func (a *KeyframeAnimation) apply(gtx layout.Context, base styles.Styles) styles.Styles {
	if !a.started {
		a.started, a.start, a.pausedAt = true, gtx.Now, 0
	}
	if a.paused {
		if a.pausedAt == 0 {
			a.pausedAt = max(gtx.Now.Sub(a.start), 1)
		}
		return a.at(base, a.pausedAt)
	}
	if a.pausedAt > 0 {
		// Resumed: go on from where it was held.
		a.start, a.pausedAt = gtx.Now.Add(-a.pausedAt), 0
	}
	if kit.ReducedMotion() {
		if a.repeat == 0 {
			return base
		}
		a.finish()
		return a.end(base)
	}
	if a.done {
		return a.end(base)
	}
	return a.at(base, gtx.Now.Sub(a.start), gtx)
}

// at returns base with the animation's styles elapsed after it started;
// a gtx asks for the next frame while it plays.
func (a *KeyframeAnimation) at(base styles.Styles, elapsed time.Duration, gtx ...layout.Context) styles.Styles {
	elapsed -= a.delay
	frame := func() {
		if len(gtx) > 0 {
			kit.RequestFrame(gtx[0])
		}
	}
	if elapsed < 0 {
		frame()
		return base
	}
	if a.dur <= 0 {
		a.finish()
		return a.end(base)
	}
	iter := int(elapsed / a.dur)
	if a.repeat > 0 && iter >= a.repeat {
		a.finish()
		return a.end(base)
	}
	p := float32(elapsed%a.dur) / float32(a.dur)
	if a.alternate && iter%2 == 1 {
		p = 1 - p
	}
	frame()
	return a.sample(base, p)
}

// sample returns base with the keyframes' styles at p.
func (a *KeyframeAnimation) sample(base styles.Styles, p float32) styles.Styles {
	fs := a.frames
	for i := 1; i < len(fs); i++ {
		if p <= fs[i].At || i == len(fs)-1 {
			from, to := styles.Merge(base, fs[i-1].Styles), styles.Merge(base, fs[i].Styles)
			span := fs[i].At - fs[i-1].At
			t := float32(1)
			if span > 0 {
				t = (p - fs[i-1].At) / span
			}
			return blendStyles(from, to, a.easing.Apply(t))
		}
	}
	return base
}

// end is base after the animation: with its last frame, or as it is.
func (a *KeyframeAnimation) end(base styles.Styles) styles.Styles {
	if !a.keepEnd {
		return base
	}
	last := 1
	if a.alternate && a.repeat > 0 && a.repeat%2 == 0 {
		last = 0
	}
	return a.sample(base, float32(last))
}

func (a *KeyframeAnimation) finish() {
	if a.done {
		return
	}
	a.done = true
	if a.onDone != nil {
		a.onDone()
	}
}
