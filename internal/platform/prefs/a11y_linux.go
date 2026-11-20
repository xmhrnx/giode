//go:build linux && !android

package prefs

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/styles/properties"
)

// HighContrast reads the desktop portal's contrast setting
// (org.freedesktop.appearance, 1 for more contrast), then GNOME's.
func HighContrast() (bool, error) {
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
	return false, platform.ErrNotSupported
}

func ContrastPalettes() (light, dark *properties.Palette, ok bool) { return nil, nil, false }

// BoldText: Linux desktops have no bold text setting.
func BoldText() (bool, error) { return false, platform.ErrNotSupported }

// TextScale reads GNOME's text scaling factor.
func TextScale() (float32, error) {
	v, err := readPortalSetting("org.gnome.desktop.interface", "text-scaling-factor")
	if err != nil {
		return 1, err
	}
	if f, ok := v.(float64); ok && f > 0 {
		return float32(f), nil
	}
	return 1, platform.ErrNotSupported
}

// GioScalesText: on Wayland, Gio applies GNOME's text scaling factor to
// the whole UI itself.
func GioScalesText(view app.ViewEvent) bool {
	_, wayland := view.(app.WaylandViewEvent)
	return wayland
}
