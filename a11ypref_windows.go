//go:build windows

package giode

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows/registry"

	"github.com/xmhrnx/giode/styles/properties"
)

var (
	procSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
	procGetSysColor           = user32.NewProc("GetSysColor")
)

const (
	spiGetHighContrast = 0x0042
	hcfHighContrastOn  = 0x1
)

// highContrastW is HIGHCONTRASTW.
type highContrastW struct {
	size   uint32
	flags  uint32
	scheme *uint16
}

func systemHighContrast() (bool, error) {
	hc := highContrastW{size: uint32(unsafe.Sizeof(highContrastW{}))}
	r, _, err := procSystemParametersInfoW.Call(spiGetHighContrast, uintptr(hc.size), uintptr(unsafe.Pointer(&hc)), 0)
	if r == 0 {
		return false, fmt.Errorf("%w: %v", ErrNotSupported, err)
	}
	return hc.flags&hcfHighContrastOn != 0, nil
}

// sysColor returns system color index i as a hex color.
func sysColor(i uintptr) properties.Color {
	c, _, _ := procGetSysColor.Call(i)
	return properties.Color(fmt.Sprintf("#%02x%02x%02x", byte(c), byte(c>>8), byte(c>>16)))
}

// systemContrastPalettes are the user's high-contrast theme colors
// (Aquatic, Desert, Dusk, Night sky, or their own), the same in both
// modes: the theme decides light or dark.
func systemContrastPalettes() (light, dark *properties.Palette, ok bool) {
	const (
		colorWindow        = 5
		colorWindowText    = 8
		colorHighlight     = 13
		colorHighlightText = 14
		colorBtnFace       = 15
		colorBtnText       = 18
		colorHotLight      = 26
	)
	p := &properties.Palette{
		Primary:    sysColor(colorHighlight),
		OnPrimary:  sysColor(colorHighlightText),
		Secondary:  sysColor(colorBtnText),
		Background: sysColor(colorWindow),
		Surface:    sysColor(colorBtnFace),
		Text:       sysColor(colorWindowText),
		Border:     sysColor(colorWindowText),
		Muted:      sysColor(colorWindowText),
		Accent:     sysColor(colorHotLight),
		Info:       sysColor(colorHotLight),
	}
	return p, p, true
}

// systemBoldText: Windows has no bold text setting.
func systemBoldText() (bool, error) { return false, ErrNotSupported }

// systemTextScale reads Settings > Accessibility > Text size, stored as
// a percentage.
func systemTextScale() (float32, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Accessibility`, registry.QUERY_VALUE)
	if err != nil {
		return 1, nil
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("TextScaleFactor")
	if err != nil || v < 100 || v > 225 {
		return 1, nil
	}
	return float32(v) / 100, nil
}

func gioScalesText(a *App) bool { return false }
