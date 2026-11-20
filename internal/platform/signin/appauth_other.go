//go:build !android && !darwin

package signin

import (
	"context"
	"net/url"

	"github.com/xmhrnx/giode/auth"
	"github.com/xmhrnx/giode/internal/platform"
)

// OpenPage: the browser, which the caller opens.
func OpenPage(ctx context.Context, h platform.Host, u string) error { return platform.ErrNotSupported }

func Session(context.Context, platform.Host, string, string, bool) (*url.URL, error) {
	return nil, auth.ErrUnsupported
}
