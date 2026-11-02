//go:build darwin && !ios

package giode

import (
	"context"

	"github.com/xmhrnx/giode/host"
)

func (a *App) haptic(HapticKind) {}

func (a *App) keepScreenOn(ctx context.Context, on bool) error { return host.KeepScreenOn(on) }

func (a *App) keepAlive(reason string) (func(), error) { return host.KeepAwake(reason) }
