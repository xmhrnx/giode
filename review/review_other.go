//go:build !android && !darwin

package review

import (
	"context"

	"github.com/xmhrnx/giode"
)

func request(ctx context.Context, a *giode.App) error { return giode.ErrNotSupported }
