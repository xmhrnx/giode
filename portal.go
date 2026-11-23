package giode

import (
	"image"
	"runtime"
	"sync"
	"sync/atomic"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/titlebar"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
	"github.com/xmhrnx/giode/ui"
)

// PortalFrame selects the frame around a portal window.
type PortalFrame uint8

const (
	// PortalFrameApp follows the main window: the platform's frame, or
	// Giode's title bar when Config.Frameless is set. The default.
	PortalFrameApp PortalFrame = iota
	// PortalFrameOS is the platform's own title bar and borders.
	PortalFrameOS
	// PortalFrameTitleBar is a frameless window with Giode's TitleBar
	// across its top: the portal's title, a drag area and the window
	// buttons, in the app's theme. PortalOptions.TitleBar adjusts it.
	PortalFrameTitleBar
	// PortalFrameNone has no frame at all: the view fills the window
	// and draws whatever chrome it wants (a TitleBar of its own, a
	// WindowDragArea) - a floating palette, a splash.
	PortalFrameNone
)

// PortalOptions configures a portal window; see App.OpenPortal.
type PortalOptions struct {
	// Title is the window's title.
	Title string
	// Size is the window's size in dp; zero is 480x360.
	Size image.Point
	// MinSize and MaxSize bound its size in dp when set.
	MinSize, MaxSize image.Point
	// FixedSize keeps it at Size.
	FixedSize bool
	// Frame selects its frame; PortalFrameApp by default.
	Frame PortalFrame
	// TitleBar, with PortalFrameTitleBar (or PortalFrameApp in a
	// frameless app), adjusts the title bar before it's first shown:
	// its variant, menus, icon, styles.
	TitleBar func(t *titlebar.TitleBar)
	// Background fills the window under the view; empty takes the
	// app's Config.Background.
	Background properties.Color
	// AlwaysOnTop keeps it above other windows.
	AlwaysOnTop bool
	// Center places it in the middle of the screen.
	Center bool
	// OnClose runs on the UI goroutine once it closed, whether by the
	// user, Close or the app's exit.
	OnClose func()
}

// PortalWindow is a window of its own showing part of the app: a tool
// palette, a detached tab, an inspector on a second screen. It shares
// the app - its State, its theme and language, Post - and lays its view
// out in turn with the main window's, so a State changed in one shows
// in both. Open it with App.OpenPortal.
//
// Phones and the web have one window per app: there a portal covers
// the main window instead, with a title bar holding the title and a
// close button, and Back closes it.
type PortalWindow struct {
	a    *App
	win  *app.Window
	opts PortalOptions
	view func() Element
	tb   *titlebar.TitleBar

	title     atomic.Pointer[string]
	focused   atomic.Bool
	maximized atomic.Bool
	closed    atomic.Bool
	closeOnce sync.Once
	// inline is set where it covers the main window instead.
	inline bool
	// block is the tag of the inline portal's input blocker.
	block int
	// calls run on its window's event goroutine, where Gio's Perform
	// and Option are safe.
	callMu sync.Mutex
	calls  []func()
}

// run has f run on its window's event goroutine.
func (w *PortalWindow) run(f func()) {
	w.callMu.Lock()
	w.calls = append(w.calls, f)
	w.callMu.Unlock()
	w.win.Invalidate()
}

// runCalls runs the queued calls.
func (w *PortalWindow) runCalls() {
	w.callMu.Lock()
	calls := w.calls
	w.calls = nil
	w.callMu.Unlock()
	for _, f := range calls {
		f()
	}
}

// singleWindow reports whether the platform shows one window per app;
// a variable for tests.
var singleWindow = func() bool {
	switch runtime.GOOS {
	case "android", "ios", "js":
		return true
	}
	return false
}

// OpenPortal opens a portal window showing view, which runs every
// frame of it as the main view does. Call it from any goroutine once
// the app runs. It closes with Close, its own close button, or the
// app's exit.
func (a *App) OpenPortal(opts PortalOptions, view func() Element) *PortalWindow {
	if opts.Size == (image.Point{}) {
		opts.Size = image.Pt(480, 360)
	}
	if opts.Background == "" {
		opts.Background = a.cfg.Background
	}
	w := &PortalWindow{a: a, opts: opts, view: view, inline: singleWindow()}
	w.title.Store(&opts.Title)
	frame := opts.Frame
	if frame == PortalFrameApp {
		frame = PortalFrameOS
		if a.cfg.Frameless {
			frame = PortalFrameTitleBar
		}
	}
	if frame == PortalFrameTitleBar || w.inline {
		w.tb = titlebar.New(w, opts.Title)
		if runtime.GOOS == "darwin" {
			w.tb.Variant(titlebar.VariantMac)
		}
		if w.inline {
			w.tb.Hide(titlebar.ButtonMinimize, titlebar.ButtonMaximize)
		}
		if opts.TitleBar != nil {
			opts.TitleBar(w.tb)
		}
	}
	a.portalMu.Lock()
	a.portals = append(a.portals, w)
	a.portalMu.Unlock()
	if w.inline {
		a.invalidate()
		return w
	}

	o := []app.Option{
		app.Title(opts.Title),
		app.Size(unit.Dp(opts.Size.X), unit.Dp(opts.Size.Y)),
		app.Decorated(frame == PortalFrameOS),
	}
	switch {
	case opts.FixedSize:
		o = append(o, app.MinSize(unit.Dp(opts.Size.X), unit.Dp(opts.Size.Y)), app.MaxSize(unit.Dp(opts.Size.X), unit.Dp(opts.Size.Y)))
	default:
		if opts.MinSize != (image.Point{}) {
			o = append(o, app.MinSize(unit.Dp(opts.MinSize.X), unit.Dp(opts.MinSize.Y)))
		}
		if opts.MaxSize != (image.Point{}) {
			o = append(o, app.MaxSize(unit.Dp(opts.MaxSize.X), unit.Dp(opts.MaxSize.Y)))
		}
	}
	if opts.AlwaysOnTop {
		o = append(o, app.TopMost(true))
	}
	w.win = new(app.Window)
	w.win.Option(o...)
	if opts.Center {
		w.win.Perform(system.ActionCenter)
	}
	go w.loop()
	return w
}

// Portals returns the portal windows open now.
func (a *App) Portals() []*PortalWindow {
	a.portalMu.Lock()
	defer a.portalMu.Unlock()
	return append([]*PortalWindow(nil), a.portals...)
}

// Close closes it. Safe from any goroutine; closing a closed portal
// does nothing.
func (w *PortalWindow) Close() {
	if w.closed.Load() {
		return
	}
	if w.inline {
		w.finish()
		w.a.invalidate()
		return
	}
	w.run(func() { w.win.Perform(system.ActionClose) })
}

// Closed reports whether it closed.
func (w *PortalWindow) Closed() bool { return w.closed.Load() }

// SetTitle changes its title, in the platform's frame and its title
// bar.
func (w *PortalWindow) SetTitle(title string) {
	w.title.Store(&title)
	if w.win != nil {
		w.run(func() { w.win.Option(app.Title(title)) })
	}
	w.Invalidate()
}

// Title returns its title.
func (w *PortalWindow) Title() string { return *w.title.Load() }

// Raise brings it to the front and gives it focus.
func (w *PortalWindow) Raise() {
	if w.win != nil {
		w.run(func() { w.win.Perform(system.ActionRaise) })
	}
}

// Minimize minimizes it.
func (w *PortalWindow) Minimize() {
	if w.win != nil {
		w.run(func() { w.win.Option(app.Minimized.Option()) })
	}
}

// ToggleMaximize maximizes it, or restores it when it's maximized.
func (w *PortalWindow) ToggleMaximize() {
	if w.win == nil {
		return
	}
	mode := app.Maximized
	if w.maximized.Load() {
		mode = app.Windowed
	}
	w.run(func() { w.win.Option(mode.Option()) })
}

// Maximized reports whether it's maximized.
func (w *PortalWindow) Maximized() bool { return w.maximized.Load() }

// Focused reports whether it has the keyboard focus: the window in
// front, or for one covering the main window, the main window's.
func (w *PortalWindow) Focused() bool {
	if w.inline {
		return w.a.Focused()
	}
	return w.focused.Load()
}

// Invalidate asks for a new frame of it.
func (w *PortalWindow) Invalidate() {
	if w.inline {
		w.a.invalidate()
		return
	}
	w.win.Invalidate()
}

// Window exposes its *app.Window for low-level Gio access; nil where it
// covers the main window instead.
func (w *PortalWindow) Window() *app.Window { return w.win }

// finish marks it closed, takes it off the app's list and runs OnClose
// on the UI goroutine.
func (w *PortalWindow) finish() {
	w.closeOnce.Do(func() {
		w.closed.Store(true)
		a := w.a
		a.portalMu.Lock()
		for i, p := range a.portals {
			if p == w {
				a.portals = append(a.portals[:i], a.portals[i+1:]...)
				break
			}
		}
		a.portalMu.Unlock()
		if w.opts.OnClose != nil {
			a.Post(w.opts.OnClose)
		}
	})
}

// loop runs its window's events. Its frames are laid out holding the
// app's frame lock, one window at a time, as elements keep per-frame
// state in package variables.
func (w *PortalWindow) loop() {
	a := w.a
	var ops op.Ops
	for {
		ev := w.win.Event()
		w.runCalls()
		switch e := ev.(type) {
		case app.DestroyEvent:
			w.finish()
			if e.Err != nil {
				a.logf(LogLevelError, "giode: portal window: %v", e.Err)
			}
			return
		case app.ConfigEvent:
			w.focused.Store(e.Config.Focused)
			w.maximized.Store(e.Config.Mode == app.Maximized)
			w.win.Invalidate()
		case app.FrameEvent:
			a.frameMu.Lock()
			if s := a.appliedTextScale(); s != 1 {
				e.Metric.PxPerSp *= s
			}
			gtx := app.NewContext(&ops, e)
			gtx.Locale = system.Locale{Language: i18n.Locale(), Direction: i18n.Direction()}
			paint.Fill(gtx.Ops, properties.CalcColor(w.opts.Background))
			if ui.Frame(gtx, w.root()) {
				w.win.Invalidate()
			}
			// A focused field here isn't the main window's.
			elements.TakeKeyboardOwner()
			a.frameMu.Unlock()
			e.Frame(gtx.Ops)
		}
	}
}

// root is its view, under its title bar when it has one.
func (w *PortalWindow) root() Element {
	view := w.view()
	if w.tb == nil {
		return view
	}
	w.tb.Title(w.Title())
	return elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1, Height: -1},
		w.tb,
		elements.Box(styles.Styles{FlexGrow: 1, Width: -1}, view),
	)
}

// layoutInline draws the portals that cover the main window, the last
// opened on top, and has Back close the topmost. It runs in the main
// window's frame, after its view.
func (a *App) layoutInlinePortals(gtx layout.Context) {
	a.portalMu.Lock()
	var open []*PortalWindow
	for _, p := range a.portals {
		if p.inline {
			open = append(open, p)
		}
	}
	a.portalMu.Unlock()
	if len(open) == 0 {
		return
	}
	top := open[len(open)-1]
	elements.HandleBack(top.Close)
	for _, p := range open {
		// It takes every press: the main window's view beneath doesn't
		// get them.
		area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
		event.Op(gtx.Ops, &p.block)
		for {
			if _, ok := gtx.Event(pointer.Filter{Target: &p.block, Kinds: pointer.Press | pointer.Release | pointer.Scroll | pointer.Drag, ScrollX: pointer.ScrollRange{Min: -1e6, Max: 1e6}, ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6}}); !ok {
				break
			}
		}
		paint.Fill(gtx.Ops, properties.CalcColor(p.opts.Background))
		area.Pop()
		pgtx := gtx
		pgtx.Constraints = layout.Exact(gtx.Constraints.Max)
		p.root().Layout(pgtx)
	}
}
