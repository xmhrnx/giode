package combobox

import (
	"context"
	"image"
	"sync"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Combobox is a text field that picks one option: the options matching
// what's typed show below it, best first, and Up, Down and Enter pick
// one. Options come from a list (Options) or, as the user types, from
// a Source, debounced, with loading and empty states. Create it once
// with New and lay it out every frame.
type Combobox struct {
	elements.Contextual

	field    *input.Input
	list     List
	options  []Option
	src      fetcher
	open     bool
	onSelect func(Option)
	value    Option
	hasValue bool
	lastText string
	outside  int
	keepText bool // the field shows the picked label: don't reopen
	now      time.Time
	panel    image.Rectangle // where the list was drawn, in the window
}

// New returns an empty Combobox; give it options with Options or
// Source.
func New(placeholder string, st ...styles.Styles) *Combobox {
	c := &Combobox{}
	c.field = input.New(placeholder, st...)
	c.field.OnKeys(c.key, key.NameUpArrow, key.NameDownArrow, key.NameReturn, key.NameEnter, key.NameEscape, key.NamePageUp, key.NamePageDown)
	c.field.OnFocus(func() { c.show() })
	c.list.OnPick = c.pick
	c.src.debounce = 250 * time.Millisecond
	return c
}

// Options sets a fixed list of options, filtered as the user types.
func (c *Combobox) Options(opts ...Option) *Combobox {
	c.options = opts
	c.refilter()
	return c
}

// Strings sets options from plain strings.
func (c *Combobox) Strings(values ...string) *Combobox {
	opts := make([]Option, len(values))
	for i, v := range values {
		opts[i] = Option{Value: v}
	}
	return c.Options(opts...)
}

// Source fetches options as the user types instead, after they pause
// for the debounce time (250 ms by default).
func (c *Combobox) Source(fn Source) *Combobox {
	c.src.fn = fn
	return c
}

// Debounce sets how long typing must pause before Source is asked.
func (c *Combobox) Debounce(d time.Duration) *Combobox {
	c.src.debounce = d
	return c
}

// MinChars is how many characters must be typed before Source is
// asked: 0 asks with an empty query as soon as the field is focused.
func (c *Combobox) MinChars(n int) *Combobox {
	c.src.minChars = n
	return c
}

// Messages sets the texts shown while loading and when nothing
// matches.
func (c *Combobox) Messages(loading, empty string) *Combobox {
	c.list.LoadingText, c.list.EmptyText = loading, empty
	return c
}

// OnSelect sets the handler called when an option is picked.
func (c *Combobox) OnSelect(fn func(Option)) *Combobox {
	c.onSelect = fn
	return c
}

// Value is the picked option.
func (c *Combobox) Value() (Option, bool) { return c.value, c.hasValue }

// SetValue picks o without calling OnSelect.
func (c *Combobox) SetValue(o Option) {
	c.value, c.hasValue = o, true
	c.setText(o.text())
}

// Clear empties the field and the value.
func (c *Combobox) Clear() {
	c.value, c.hasValue = Option{}, false
	c.setText("")
}

// Text is what's typed.
func (c *Combobox) Text() string { return c.field.Text() }

// Input is the underlying field, for its other settings.
func (c *Combobox) Input() *input.Input { return c.field }

// Opened reports whether the option list shows.
func (c *Combobox) Opened() bool { return c.open }

func (c *Combobox) setText(s string) {
	c.field.SetText(s)
	c.lastText = s
	c.keepText = true
	c.open = false
}

func (c *Combobox) show() {
	c.open = true
	if c.src.fn == nil {
		c.refilter()
	} else {
		c.src.request(c.query(), c.src.minChars == 0, c.now)
	}
}

// query is the text to search: none while the field shows the value.
func (c *Combobox) query() string {
	if c.keepText {
		return ""
	}
	return c.field.Text()
}

func (c *Combobox) refilter() {
	c.list.SetItems(Filter(c.options, c.query()))
}

func (c *Combobox) key(e key.Event) {
	switch e.Name {
	case key.NameUpArrow, key.NameDownArrow:
		if !c.open {
			c.show()
			return
		}
		if e.Name == key.NameUpArrow {
			c.list.Move(-1)
		} else {
			c.list.Move(1)
		}
	case key.NamePageUp:
		c.list.First()
	case key.NamePageDown:
		c.list.Last()
	case key.NameReturn, key.NameEnter:
		if o, ok := c.list.Selected(); ok && c.open {
			c.pick(o)
		}
	case key.NameEscape:
		c.open = false
	}
}

func (c *Combobox) pick(o Option) {
	c.value, c.hasValue = o, true
	c.setText(o.text())
	if c.onSelect != nil {
		c.onSelect(o)
	}
}

// Layout lays out the field and, while open, its option list.
func (c *Combobox) Layout(gtx layout.Context) layout.Dimensions {
	kit.OpaqueInput()
	c.now = gtx.Now
	if c.now.IsZero() {
		c.now = time.Now()
	}
	dims := c.field.Layout(gtx)
	kit.BlockDrag(gtx, dims.Size)
	c.AnchorContext(gtx, dims.Size)

	if text := c.field.Text(); text != c.lastText {
		c.lastText = text
		c.keepText = false
		c.open = c.field.Focused()
		if c.src.fn == nil {
			c.refilter()
		} else {
			c.src.request(text, len([]rune(text)) >= c.src.minChars, c.now)
		}
	}
	if c.src.fn != nil {
		c.src.poll(gtx, &c.list)
	}
	if c.open {
		elements.HandleDismiss(func() { c.open = false })
		elements.RegisterOverlay(func() elements.Element { return elements.Raw(c.layoutOutside) })
		trigger, width := anchorBelow(dims.Size), dims.Size.X
		elements.RegisterPortalAround(gtx, trigger, 4, elements.PlaceAuto, func() elements.Element {
			return elements.Raw(func(gtx layout.Context) layout.Dimensions {
				d := c.list.Layout(gtx, width)
				at := elements.AbsolutePosition()
				c.panel = image.Rectangle{Min: at, Max: at.Add(d.Size)}
				return d
			})
		})
	}
	return dims
}

// layoutOutside closes the list on a press elsewhere, letting the
// press through.
func (c *Combobox) layoutOutside(gtx layout.Context) layout.Dimensions {
	closeOnPress(gtx, &c.outside, &c.open, c.panel)
	return layout.Dimensions{}
}

// closeOnPress closes on a press outside keep.
func closeOnPress(gtx layout.Context, tag event.Tag, open *bool, keep image.Rectangle) {
	for {
		ev, ok := gtx.Source.Event(pointer.Filter{Target: tag, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && *open && !image.Pt(int(e.Position.X), int(e.Position.Y)).In(keep) {
			*open = false
			kit.RequestFrame(gtx)
		}
	}
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, tag)
}

// fetcher runs a Source: debounced, one request at a time, results of
// stale queries dropped.
type fetcher struct {
	fn       Source
	debounce time.Duration
	minChars int

	query   string
	due     time.Time // when to ask, after the debounce
	waiting bool

	mu       sync.Mutex
	gen      int
	cancel   context.CancelFunc
	inflight bool
	done     bool
	result   []Option
	err      error
}

// request asks for query after the debounce; ok false clears instead.
func (f *fetcher) request(query string, ok bool, now time.Time) {
	f.query = query
	f.mu.Lock()
	f.gen++
	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}
	f.inflight = false
	if !ok {
		f.done, f.result, f.err = true, nil, nil
		f.waiting = false
		f.mu.Unlock()
		return
	}
	f.mu.Unlock()
	f.waiting = true
	f.due = now.Add(f.debounce)
}

// poll starts a due request and takes finished results into l.
func (f *fetcher) poll(gtx layout.Context, l *List) {
	now := gtx.Now
	if now.IsZero() {
		now = time.Now()
	}
	if f.waiting {
		l.Loading = true
		if now.Before(f.due) {
			kit.RequestFrameAt(gtx, f.due)
		} else {
			f.waiting = false
			f.start()
		}
	}
	f.mu.Lock()
	inflight, done := f.inflight, f.done
	res, err := f.result, f.err
	f.done = false
	f.mu.Unlock()
	if inflight {
		l.Loading = true
		kit.RequestFrameAt(gtx, now.Add(30*time.Millisecond))
	}
	if done {
		l.Loading = false
		l.Err = err
		// Highlight the matched letters, in the server's order.
		for i := range res {
			_, res[i].matched, _ = Match(f.query, res[i].text())
		}
		l.SetItems(res)
	}
}

func (f *fetcher) start() {
	f.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.inflight = true
	gen, query, fn := f.gen, f.query, f.fn
	f.mu.Unlock()
	go func() {
		res, err := fn(ctx, query)
		f.mu.Lock()
		defer f.mu.Unlock()
		if gen != f.gen {
			return // a newer query took over
		}
		f.inflight, f.done, f.result, f.err = false, true, res, err
		cancel()
	}()
}
