//go:build !android && !darwin

package store

import (
	"context"

	"github.com/xmhrnx/giode"
)

func products(ctx context.Context, ids []string) ([]Product, error) {
	return nil, giode.ErrNotSupported
}

func purchase(ctx context.Context, a *giode.App, p Product) (Purchase, error) {
	return Purchase{}, giode.ErrNotSupported
}

func finish(ctx context.Context, p Purchase, consume bool) error { return giode.ErrNotSupported }

func owned(ctx context.Context) ([]Purchase, error) { return nil, giode.ErrNotSupported }

func startUpdates() {}
