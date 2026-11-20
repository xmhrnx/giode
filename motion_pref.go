//go:build !linux || android

package giode

import (
	"errors"

	"gioui.org/x/pref/theme"
)

func systemReducedMotion() (bool, error) {
	v, err := theme.IsReducedMotion()
	if err != nil {
		return false, errors.Join(ErrNotSupported, err)
	}
	return v, nil
}
