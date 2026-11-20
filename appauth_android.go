package giode

import (
	"context"
	"net/url"

	"gioui.org/app"

	"github.com/xmhrnx/giode/auth"
	"github.com/xmhrnx/giode/internal/mobile"
)

// openSignIn opens a sign-in page in a Custom Tab.
func (a *App) openSignIn(ctx context.Context, u string) error {
	return a.onAndroidMain(ctx, func(view uintptr) error {
		return mobile.OpenCustomTab(app.JavaVM(), app.AppContext(), view, u)
	})
}

func (a *App) authSession(context.Context, string, string, bool) (*url.URL, error) {
	return nil, auth.ErrUnsupported
}
