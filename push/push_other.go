//go:build !android && !darwin

package push

import (
	"context"

	"github.com/xmhrnx/giode"
)

func start(a *giode.App) error { return giode.ErrNotSupported }

func token(ctx context.Context) (string, error) { return "", giode.ErrNotSupported }

func requestPermission(ctx context.Context, a *giode.App) (bool, error) { return true, nil }

func unregister() error { return giode.ErrNotSupported }
