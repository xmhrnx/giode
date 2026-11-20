//go:build js

package keepawake

import (
	"context"

	"github.com/xmhrnx/giode/internal/platform"
)

func Haptic(h platform.Host, kind int) {}

func KeepScreenOn(ctx context.Context, h platform.Host, on bool) error {
	return platform.ErrNotSupported
}

func KeepAlive(h platform.Host, reason string, onStart func()) (func(), error) {
	return func() {}, platform.ErrNotSupported
}
