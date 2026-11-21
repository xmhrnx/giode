//go:build giode_gio2

package giopatch

import (
	"context"

	"gioui.org/app"
	"gioui.org/io/event"
)

// InjectsEvents reports whether Inject works.
const InjectsEvents = true

// Inject delivers e to the window's input through giode's patched Gio.
// It must not run on the UI goroutine.
func Inject(ctx context.Context, w *app.Window, e event.Event) error {
	done := make(chan struct{})
	go func() {
		w.GiodeInject(e)
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
