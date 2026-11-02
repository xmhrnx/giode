package giode

import (
	"sync/atomic"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles/properties"
)

// currentApp is the most recently created App, for the package-level
// functions that need its window (SafeArea, Haptic).
var currentApp atomic.Pointer[App]

// safeArea holds the last frame's insets, in dp.
var safeArea atomic.Pointer[properties.Inset]

// SafeArea returns the parts of the window covered by system UI - the
// status bar, the navigation bar or home indicator, a notch or camera
// cutout, and on iOS the on-screen keyboard - as insets in dp, as of the
// last frame. Read it during layout.
//
// By default giode lays the view out inside the safe area, so content
// never sits under system UI and this is only informative. With
// Config.EdgeToEdge the view covers the whole window and SafeArea is
// what to pad content with: a header's top, a composer's bottom.
//
// On Android the window shrinks to make room for the on-screen
// keyboard, so content laid out above the bottom edge stays above the
// keyboard either way.
func SafeArea() properties.Inset {
	if p := safeArea.Load(); p != nil {
		return *p
	}
	return properties.Inset{}
}

// frameContext returns the layout context for a frame: inside the safe
// area, or over the whole window with EdgeToEdge. It records the
// insets for SafeArea.
func (a *App) frameContext(ops *op.Ops, e app.FrameEvent) layout.Context {
	in := properties.Inset{
		Top: int(e.Insets.Top), Right: int(e.Insets.Right),
		Bottom: int(e.Insets.Bottom), Left: int(e.Insets.Left),
	}
	safeArea.Store(&in)
	// The app's locale: language and reading direction (see i18n).
	loc := system.Locale{Language: i18n.Locale(), Direction: i18n.Direction()}
	if !a.cfg.EdgeToEdge {
		gtx := app.NewContext(ops, e)
		gtx.Locale = loc
		return gtx
	}
	ops.Reset()
	return layout.Context{
		Ops:         ops,
		Now:         e.Now,
		Source:      e.Source,
		Metric:      e.Metric,
		Locale:      loc,
		Constraints: layout.Exact(e.Size),
	}
}

// HideKeyboard hides the on-screen keyboard and takes keyboard focus
// from the focused field, on the next frame. Safe to call from any
// goroutine.
func (a *App) HideKeyboard() {
	a.hideKeyboard.Store(true)
	a.window.Invalidate()
}

// applyHideKeyboard carries out a pending HideKeyboard.
func (a *App) applyHideKeyboard(gtx layout.Context) {
	if a.hideKeyboard.Swap(false) {
		gtx.Execute(key.FocusCmd{})
		gtx.Execute(key.SoftKeyboardCmd{Show: false})
	}
}
