package giode

import (
	"github.com/mehran9675/giode/ui"

	"context"
	"fmt"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"

	"github.com/mehran9675/giode/components/menu"
	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/styles/properties"
)

// SetContextMenu registers a menu to open with a right-click anywhere
// in the window. The app registers the right-click area and renders
// the menu on top automatically.
func (a *App) SetContextMenu(m *menu.Menu) {
	a.ctxMenu = m
}

// Run drives the event loop and renders view every frame until the
// window closes. view runs on every frame: build the UI from scratch,
// but reuse stateful components created before Run.
//
// Run blocks for the lifetime of the application: when the window
// closes it releases the single-instance lock, runs OnShutdown and
// terminates the process. Any window close error is printed to
// stderr.
func (a *App) Run(view func() Element) error {
	if a.rendererErr != nil {
		return a.rendererErr
	}
	if a.secondInstance {
		a.logf(LogLevelInfo, "giode: second instance exiting")
		// The running instance takes this launch's arguments (a context
		// action, a file) and comes to the front.
		if err := a.instance.Forward(os.Args[1:]); err != nil {
			a.logf(LogLevelWarning, "giode: couldn't reach the running instance: %v", err)
		}
		if a.cfg.SingleInstanceLock.OnSecondInstanceLaunch != nil {
			a.cfg.SingleInstanceLock.OnSecondInstanceLaunch()
		}
		return nil
	}
	if a.cfg.OnStartup != nil {
		a.cfg.OnStartup()
	}
	a.logf(LogLevelInfo, "giode: app started")
	a.warnTrayLimits()
	if a.instance != nil {
		a.instance.Serve(a.handleLaunch)
	}
	if !a.runContextAction(os.Args[1:]) {
		a.startupURL(os.Args[1:])
	}
	a.watchLifecycle()
	ctx, cancel := context.WithCancel(context.Background())
	a.watchSystemTheme(ctx.Done())
	if a.mcp != nil {
		a.mcp.start(ctx, a)
	}
	go func() {
		err := a.runLoop(view)
		cancel()
		if a.instance != nil {
			a.instance.Release()
		}
		if a.cfg.OnShutdown != nil {
			a.cfg.OnShutdown()
		}
		a.logf(LogLevelDebug, "giode: app stopped")
		if err != nil {
			fmt.Fprintln(os.Stderr, "giode:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	// app.Events blocks forever on most platforms, delivering the
	// events that aren't the window's - links opening the app; the
	// process exits from the window goroutine above once the window
	// closes.
	// It's called, not ranged over: on iOS it returns at once and links
	// arrive later, when a range loop's body may no longer run.
	app.Events(func(ev event.Event) bool {
		if u, ok := ev.(app.URLEvent); ok && u.URL != nil {
			a.openedURL(u.URL)
		}
		return true
	})
	return nil
}

// RunRaw is Run with a plain Gio layout function.
func (a *App) RunRaw(view func(gtx layout.Context) layout.Dimensions) error {
	return a.Run(func() Element { return elements.Raw(view) })
}

func (a *App) runLoop(view func() Element) error {
	var ops op.Ops
	ready := false
	for {
		ev := a.window.Event()
		a.explorer.ListenEvents(ev)
		if a.cfg.OnEvent != nil {
			a.cfg.OnEvent(ev)
		}
		switch e := ev.(type) {
		case app.DestroyEvent:
			if a.cfg.OnBeforeClose != nil {
				a.cfg.OnBeforeClose()
			}
			if a.mcp != nil {
				a.mcp.release()
			}
			a.logf(LogLevelDebug, "giode: window closed")
			return e.Err
		case app.ViewEvent:
			a.handleViewEvent(e)
		case app.ConfigEvent:
			a.handleConfig(e.Config)
		default:
			// A wakeup from Invalidate with no frame to follow, as while
			// the window is hidden: run the queued calls now, and ask
			// for a frame for what they changed (a visible window gets
			// one; a hidden one just wakes again with nothing queued).
			if a.runUICalls() {
				a.window.Invalidate()
			}
		case app.FrameEvent:
			start := time.Now()
			a.frameGeom.Store(&frameGeom{pxPerDp: e.Metric.PxPerDp, size: e.Size})
			a.runUICalls()
			gtx := a.frameContext(&ops, e)
			a.applyHideKeyboard(gtx)
			if a.mcp != nil {
				elements.BeginSemantics()
			}
			if a.cfg.Underlay == nil {
				paint.Fill(gtx.Ops, properties.CalcColor(a.cfg.Background))
			}
			// The same frame steps a custom host runs (ui.Frame), with the
			// window's own layers around the view.
			root := elements.Raw(func(gtx layout.Context) layout.Dimensions {
				layoutDrag(a, gtx)
				if a.cfg.OnKey != nil {
					elements.OnKey(a.cfg.OnKey)
				}
				dims := view().Layout(gtx)
				a.winPointer.layout(a, gtx)
				a.layoutClipboard(gtx)
				if a.ctxMenu != nil {
					a.ctxMenu.Layout(gtx)
				}
				return dims
			})
			if ui.Frame(gtx, root) {
				a.window.Invalidate()
			}
			var nodes []elements.SemanticNode
			if a.mcp != nil {
				nodes = elements.EndSemantics()
			}
			if a.cfg.ShowFrameTimes {
				a.layoutFrameTimes(gtx)
			}
			built := time.Now()
			e.Frame(gtx.Ops)
			a.frames.add(start, built, time.Now())
			a.updateSystemBars()
			if a.mcp != nil {
				a.mcp.frameDone(nodes, &ops, e.Size)
			}
			if !ready {
				ready = true
				a.logf(LogLevelDebug, "giode: first frame presented")
				if a.cfg.OnReady != nil {
					a.cfg.OnReady()
				}
			}
		}
	}
}

// logf forwards a message to the configured logger when the level is
// enabled.
func (a *App) logf(level LogLevel, format string, v ...any) {
	if a.cfg.Logger != nil && level >= a.cfg.LogLevel {
		a.cfg.Logger.Printf(format, v...)
	}
}
