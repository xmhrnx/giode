//go:build darwin

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c

int giodeIncreaseContrast(void);
int giodeBoldText(void);
double giodeTextScale(void);
*/
import "C"

import "github.com/xmhrnx/giode/styles/properties"

// systemHighContrast reads Increase Contrast (macOS and iOS); -1 from
// the platform means it can't say.
func systemHighContrast() (bool, error) {
	v := C.giodeIncreaseContrast()
	if v < 0 {
		return false, ErrNotSupported
	}
	return v == 1, nil
}

func systemContrastPalettes() (light, dark *properties.Palette, ok bool) { return nil, nil, false }

// systemBoldText reads Bold Text (iOS).
func systemBoldText() (bool, error) {
	v := C.giodeBoldText()
	if v < 0 {
		return false, ErrNotSupported
	}
	return v == 1, nil
}

// systemTextScale reads Dynamic Type's text size (iOS).
func systemTextScale() (float32, error) {
	v := float32(C.giodeTextScale())
	if v <= 0 {
		return 1, ErrNotSupported
	}
	return v, nil
}

func gioScalesText(a *App) bool { return false }
