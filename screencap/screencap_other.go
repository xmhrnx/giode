//go:build !linux && !windows && !darwin

package screencap

import (
	"context"

	"github.com/xmhrnx/giode"
)

func sources(context.Context) ([]Source, error) { return nil, giode.ErrNotSupported }

type platformCapture struct{}

func (platformCapture) stop() {}

func (c *Capture) start(context.Context, *giode.App) error { return giode.ErrNotSupported }
