//go:build !linux && !windows && !darwin

package media

import (
	"context"
	"fmt"
)

func newEngine(context.Context, *Player, string) (engine, error) {
	return nil, fmt.Errorf("%w on this platform", ErrUnsupported)
}

var silentForTests bool
