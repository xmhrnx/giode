//go:build linux && !android

package giode

import (
	"errors"

	"github.com/godbus/dbus/v5"
)

// systemDark reads the XDG desktop portal's appearance setting:
// color-scheme 1 means dark.
func systemDark() (bool, error) {
	v, err := readPortalSetting("org.freedesktop.appearance", "color-scheme")
	if err != nil {
		return false, err
	}
	scheme, ok := v.(uint32)
	if !ok {
		return false, ErrNotSupported
	}
	return scheme == 1, nil
}

// systemReducedMotion reads the desktop's animation setting through the
// portal: GNOME's enable-animations, or KDE's animation speed at 0.
func systemReducedMotion() (bool, error) {
	if v, err := readPortalSetting("org.gnome.desktop.interface", "enable-animations"); err == nil {
		if on, ok := v.(bool); ok {
			return !on, nil
		}
	}
	if v, err := readPortalSetting("org.kde.kdeglobals.KDE", "AnimationDurationFactor"); err == nil {
		if f, ok := v.(float64); ok {
			return f == 0, nil
		}
	}
	return false, ErrNotSupported
}

// readPortalSetting reads a setting through the XDG desktop portal.
func readPortalSetting(namespace, key string) (any, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, errors.Join(ErrNotSupported, err)
	}
	var v dbus.Variant
	err = conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop").
		Call("org.freedesktop.portal.Settings.Read", 0, namespace, key).Store(&v)
	if err != nil {
		return nil, errors.Join(ErrNotSupported, err)
	}
	// Read wraps the value in a second variant.
	if inner, ok := v.Value().(dbus.Variant); ok {
		v = inner
	}
	return v.Value(), nil
}
