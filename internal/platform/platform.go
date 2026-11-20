// Package platform holds what giode's platform packages share: the
// errors the root package re-exports. Each feature's per-platform code
// lives in a package under it (internal/platform/sound, .../tray...),
// and the root package keeps the public API.
package platform

import (
	"errors"

	"github.com/xmhrnx/giode/host"
)

var (
	// ErrNotSupported is giode.ErrNotSupported.
	ErrNotSupported = host.ErrNotSupported
	// ErrCanceled is giode.ErrCanceled.
	ErrCanceled = errors.New("giode: canceled by the user")
	// ErrDeclined is giode.ErrDeclined.
	ErrDeclined = errors.New("giode: the user declined")
	// ErrNotificationsOff is giode.ErrNotificationsOff.
	ErrNotificationsOff = errors.New("giode: notifications are turned off for this app")
	// ErrNotFound is giode.ErrNotFound.
	ErrNotFound = errors.New("giode: not found")
)
