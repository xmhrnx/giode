package signin

import (
	"context"
	"net/url"

	"gioui.org/app"

	"github.com/xmhrnx/giode/auth"
	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
)

// OpenPage opens a sign-in page in a Custom Tab.
func OpenPage(ctx context.Context, h platform.Host, u string) error {
	return platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		return mobile.OpenCustomTab(app.JavaVM(), app.AppContext(), view, u)
	})
}

func Session(context.Context, platform.Host, string, string, bool) (*url.URL, error) {
	return nil, auth.ErrUnsupported
}
