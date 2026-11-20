//go:build js

package giode

import (
	"syscall/js"

	"github.com/xmhrnx/giode/styles/properties"
)

// systemHighContrast reads the browser's prefers-contrast media query.
func systemHighContrast() (bool, error) {
	mm := js.Global().Get("matchMedia")
	if mm.Type() != js.TypeFunction {
		return false, ErrNotSupported
	}
	more := js.Global().Call("matchMedia", "(prefers-contrast: more)").Get("matches").Bool()
	forced := js.Global().Call("matchMedia", "(forced-colors: active)").Get("matches").Bool()
	return more || forced, nil
}

func systemContrastPalettes() (light, dark *properties.Palette, ok bool) { return nil, nil, false }

func systemBoldText() (bool, error) { return false, ErrNotSupported }

// systemTextScale: the browser's zoom already scales the page.
func systemTextScale() (float32, error) { return 1, ErrNotSupported }

func gioScalesText(a *App) bool { return true }
