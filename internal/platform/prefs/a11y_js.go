//go:build js

package prefs

import (
	"syscall/js"

	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/styles/properties"
)

// HighContrast reads the browser's prefers-contrast media query.
func HighContrast() (bool, error) {
	mm := js.Global().Get("matchMedia")
	if mm.Type() != js.TypeFunction {
		return false, platform.ErrNotSupported
	}
	more := js.Global().Call("matchMedia", "(prefers-contrast: more)").Get("matches").Bool()
	forced := js.Global().Call("matchMedia", "(forced-colors: active)").Get("matches").Bool()
	return more || forced, nil
}

func ContrastPalettes() (light, dark *properties.Palette, ok bool) { return nil, nil, false }

func BoldText() (bool, error) { return false, platform.ErrNotSupported }

// TextScale: the browser's zoom already scales the page.
func TextScale() (float32, error) { return 1, platform.ErrNotSupported }

func GioScalesText(view app.ViewEvent) bool { return true }
