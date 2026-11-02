//go:build !android && !darwin

package camera

import (
	"context"

	"gioui.org/f32"

	"github.com/xmhrnx/giode"
)

const supported = false

type platformCamera struct{}

func requestPermission(context.Context, *giode.App) (bool, error) {
	return false, giode.ErrNotSupported
}

func (c *Camera) open(context.Context) error { return giode.ErrNotSupported }

func (platformCamera) close()                                  {}
func (platformCamera) capture(context.Context) ([]byte, error) { return nil, giode.ErrNotSupported }
func (platformCamera) hasTorch() bool                          { return false }
func (platformCamera) setTorch(bool) error                     { return giode.ErrNotSupported }
func (platformCamera) focus(f32.Point) error                   { return giode.ErrNotSupported }
