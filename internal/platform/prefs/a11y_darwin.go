//go:build darwin

package prefs

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c

int giodeIncreaseContrast(void);
int giodeBoldText(void);
double giodeTextScale(void);
*/
import "C"

import (
	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/styles/properties"
)

// HighContrast reads Increase Contrast (macOS and iOS); -1 from
// the platform means it can't say.
func HighContrast() (bool, error) {
	v := C.giodeIncreaseContrast()
	if v < 0 {
		return false, platform.ErrNotSupported
	}
	return v == 1, nil
}

func ContrastPalettes() (light, dark *properties.Palette, ok bool) { return nil, nil, false }

// BoldText reads Bold Text (iOS).
func BoldText() (bool, error) {
	v := C.giodeBoldText()
	if v < 0 {
		return false, platform.ErrNotSupported
	}
	return v == 1, nil
}

// TextScale reads Dynamic Type's text size (iOS).
func TextScale() (float32, error) {
	v := float32(C.giodeTextScale())
	if v <= 0 {
		return 1, platform.ErrNotSupported
	}
	return v, nil
}

func GioScalesText(view app.ViewEvent) bool { return false }
