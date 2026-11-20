//go:build !ios && !android

package telecom

import (
	"context"

	"github.com/xmhrnx/giode"
)

const supported = false

func setup(*giode.App) error                     { return ErrNotSupported }
func reportIncoming(context.Context, Call) error { return ErrNotSupported }
func startOutgoing(Call) error                   { return ErrNotSupported }
func reportConnected(string) error               { return ErrNotSupported }
func end(string) error                           { return ErrNotSupported }
func reportEnded(string, Reason) error           { return ErrNotSupported }
func setMuted(string, bool) error                { return ErrNotSupported }
