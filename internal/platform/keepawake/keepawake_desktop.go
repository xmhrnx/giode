//go:build (darwin && !ios) || windows || (linux && !android) || freebsd || openbsd || netbsd || dragonfly

package keepawake

import (
	"context"

	"github.com/xmhrnx/giode/host"
	"github.com/xmhrnx/giode/internal/platform"
)

// Desktops have no haptics.
func Haptic(h platform.Host, kind int) {}

// KeepScreenOn holds off the screen saver and display sleep.
func KeepScreenOn(ctx context.Context, h platform.Host, on bool) error {
	return host.KeepScreenOn(on)
}

// KeepAlive keeps the computer from sleeping while idle.
func KeepAlive(h platform.Host, reason string, onStart func()) (func(), error) {
	return host.KeepAwake(reason)
}
