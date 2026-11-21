//go:build !giode_gio2

package giopatch

import (
	"context"

	"gioui.org/app"
	"gioui.org/io/event"
)

// InjectsEvents reports whether Inject works.
const InjectsEvents = false

// Inject needs giode's Gio patches.
func Inject(ctx context.Context, w *app.Window, e event.Event) error {
	return ErrInjectUnpatched
}
