// Package titlebar provides a custom window title bar - drag handle,
// title, optional icon and menus, and minimize/maximize/close buttons
// - for frameless windows that replace the operating system's own.

package titlebar

import (
	"github.com/xmhrnx/giode/components/kit"
)

// Window is what a title bar controls. *giode.App implements it.
type Window interface {
	Minimize()
	ToggleMaximize()
	Close()
	Maximized() bool
	Focused() bool
}

// Variant selects the look and button placement.
type Variant string

const (
	// VariantWindows places square minimize, maximize and close
	// buttons on the right; close turns red on hover. The default.
	VariantWindows Variant = "windows"
	// VariantMac places round close, minimize and maximize "traffic
	// light" buttons on the left and centers the title.
	VariantMac Variant = "mac"
)

// Button identifies a window button.
type Button int

const (
	ButtonMinimize Button = iota
	ButtonMaximize
	ButtonClose
)

var (
	defaultBackground = "theme.surface"
	defaultText       = "theme.text"
	defaultInactive   = "theme.muted"
	closeHover        = "#e81123"
	closePressed      = "#f1707a"
	// Traffic light colors, indexed by Button.
	macColors = [3]string{ButtonMinimize: "#febc2e", ButtonMaximize: "#28c840", ButtonClose: "#ff5f57"}
	// Traffic light glyphs, shown while the group is hovered.
	macGlyphs = [3]kit.Glyph{ButtonMinimize: kit.GlyphMinus, ButtonMaximize: kit.GlyphPlus, ButtonClose: kit.GlyphClose}
)
