//go:build !android && !darwin

package giode

import (
	"context"
	"net/url"

	"github.com/xmhrnx/giode/auth"
)

func (a *App) openSignIn(ctx context.Context, u string) error { return a.OpenURL(ctx, u) }

func (a *App) authSession(context.Context, string, string, bool) (*url.URL, error) {
	return nil, auth.ErrUnsupported
}
