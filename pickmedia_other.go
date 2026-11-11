//go:build !ios

package giode

import "context"

func (a *App) pickMedia(ctx context.Context, opts MediaOptions) ([]PickedFile, error) {
	return a.pickMediaFiles(ctx, opts)
}
