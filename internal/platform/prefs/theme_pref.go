//go:build !linux || android

package prefs

import (
	"errors"

	"gioui.org/x/pref/theme"
	"github.com/xmhrnx/giode/internal/platform"
)

func Dark() (bool, error) {
	dark, err := theme.IsDarkMode()
	if err != nil {
		return false, errors.Join(platform.ErrNotSupported, err)
	}
	return dark, nil
}

func ReducedMotion() (bool, error) {
	v, err := theme.IsReducedMotion()
	if err != nil {
		return false, errors.Join(platform.ErrNotSupported, err)
	}
	return v, nil
}
