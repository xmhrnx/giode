// Package titlebar provides a custom window title bar - drag handle,
// title, optional icon and menus, and minimize/maximize/close buttons
// - for frameless windows that replace the operating system's own.

package titlebar

import (
	"gioui.org/gesture"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// TitleBar is a stateful custom title bar. Create it once with New and
// lay it out every frame at the top of the window:
//
//	bar := giode.TitleBar(app, "My app")
//	app.Run(func() giode.Element {
//		return giode.Stack(giode.Styles{Width: -1, Height: -1}, bar, content())
//	})
//
// Use it with Config{Frameless: true} (and without Draggable, so only
// the bar drags the window). The bar's empty space drags the window;
// on Windows it is also a real caption, so double-clicking maximizes
// and dragging to a screen edge snaps. Everything placed in it -
// buttons, menus - stays clickable.
type TitleBar struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	win     Window
	title   string
	icon    elements.Element
	st      styles.Styles
	variant Variant
	height  unit.Dp

	leading, trailing []elements.Element

	hidden  [3]bool
	buttons [3]widget.Clickable
	onClose func() bool

	dbl gesture.Click
}

// New returns a title bar for win. The styles argument is optional:
// Background (default theme.surface), Color (title and glyphs, default
// theme.text), Height (default 36dp, 28dp for VariantMac), FontSize,
// FontWeight, Padding, BorderColor/BorderWidth (a line under the bar).
func New(win Window, title string, st ...styles.Styles) *TitleBar {
	t := &TitleBar{win: win, title: title, variant: VariantWindows}
	if len(st) > 0 {
		t.st = st[0]
	}
	return t
}

// Styles replaces the bar's styles (see New) - solid in a window, a
// transparent fade over fullscreen video. It returns t for chaining.
func (t *TitleBar) Styles(st styles.Styles) *TitleBar {
	t.st = st
	return t
}

// Title replaces the title. It returns t for chaining.
func (t *TitleBar) Title(title string) *TitleBar {
	t.title = title
	return t
}

// Icon sets an element (typically an icon) shown before the title. It
// returns t for chaining.
func (t *TitleBar) Icon(icon elements.Element) *TitleBar {
	t.icon = icon
	return t
}

// Leading adds elements after the title - menus (File, Edit, ...),
// tabs, a search field. They stay clickable inside the drag area. It
// returns t for chaining.
func (t *TitleBar) Leading(els ...elements.Element) *TitleBar {
	t.leading = append(t.leading, els...)
	return t
}

// Trailing adds elements just before the window buttons - an account
// button, a theme switch. It returns t for chaining.
func (t *TitleBar) Trailing(els ...elements.Element) *TitleBar {
	t.trailing = append(t.trailing, els...)
	return t
}

// Variant sets the look and button placement. It returns t for
// chaining.
func (t *TitleBar) Variant(v Variant) *TitleBar {
	t.variant = v
	return t
}

// Hide removes window buttons, e.g. Hide(ButtonMaximize) for a window
// that shouldn't be maximized. It returns t for chaining.
func (t *TitleBar) Hide(buttons ...Button) *TitleBar {
	for _, b := range buttons {
		if b >= 0 && int(b) < len(t.hidden) {
			t.hidden[b] = true
		}
	}
	return t
}

// OnClose sets a handler run when the close button is clicked; return
// false to keep the window open (to ask about unsaved changes, say).
// Otherwise the button then closes the window as the system's would:
// to the tray with Config.CloseToTray, else ending the app.
// It returns t for chaining.
func (t *TitleBar) OnClose(fn func() bool) *TitleBar {
	t.onClose = fn
	return t
}

// Press performs a window button's action, as a click would.
func (t *TitleBar) Press(b Button) {
	switch b {
	case ButtonMinimize:
		t.win.Minimize()
	case ButtonMaximize:
		t.win.ToggleMaximize()
	case ButtonClose:
		if t.onClose != nil && !t.onClose() {
			return
		}
		// Close as the system's close button would: a window that closes
		// to the tray hides there (*giode.App's RequestClose).
		if r, ok := t.win.(interface{ RequestClose() }); ok {
			r.RequestClose()
		} else {
			t.win.Close()
		}
	}
}

func (t *TitleBar) barHeight() unit.Dp {
	if t.st.Height > 0 {
		return unit.Dp(t.st.Height)
	}
	if t.variant == VariantMac {
		return 28
	}
	return 36
}

func (t *TitleBar) visible() []Button {
	var out []Button
	for _, b := range []Button{ButtonMinimize, ButtonMaximize, ButtonClose} {
		if !t.hidden[b] {
			out = append(out, b)
		}
	}
	return out
}
