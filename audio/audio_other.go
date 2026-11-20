//go:build !linux && !windows && !darwin

package audio

import (
	"context"

	"github.com/xmhrnx/giode"
)

func devices(context.Context, bool) ([]Device, error) { return nil, giode.ErrNotSupported }

func requestPermission(context.Context, *giode.App) (bool, error) {
	return false, giode.ErrNotSupported
}

type platformInput struct{}

func (in *Input) open(context.Context, *giode.App) error { return giode.ErrNotSupported }
func (platformInput) close() error                       { return nil }

type platformOutput struct{}

func (o *Output) open(context.Context, *giode.App) error { return giode.ErrNotSupported }
func (platformOutput) close() error                      { return nil }
