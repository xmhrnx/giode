//go:build linux && !android

package giode

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/styles/properties"
)

// systemHighContrast reads the desktop portal's contrast setting
// (org.freedesktop.appearance, 1 for more contrast), then GNOME's.
func systemHighContrast() (bool, error) {
	if v, err := readPortalSetting("org.freedesktop.appearance", "contrast"); err == nil {
		if c, ok := v.(uint32); ok {
			return c == 1, nil
		}
	}
	if v, err := readPortalSetting("org.gnome.desktop.a11y.interface", "high-contrast"); err == nil {
		if on, ok := v.(bool); ok {
			return on, nil
		}
	}
	return false, ErrNotSupported
}

func systemContrastPalettes() (light, dark *properties.Palette, ok bool) { return nil, nil, false }

// systemBoldText: Linux desktops have no bold text setting.
func systemBoldText() (bool, error) { return false, ErrNotSupported }

// systemTextScale reads GNOME's text scaling factor.
func systemTextScale() (float32, error) {
	v, err := readPortalSetting("org.gnome.desktop.interface", "text-scaling-factor")
	if err != nil {
		return 1, err
	}
	if f, ok := v.(float64); ok && f > 0 {
		return float32(f), nil
	}
	return 1, ErrNotSupported
}

// gioScalesText: on Wayland, Gio applies GNOME's text scaling factor to
// the whole UI itself.
func gioScalesText(a *App) bool {
	_, wayland := a.NativeView().(app.WaylandViewEvent)
	return wayland
}
