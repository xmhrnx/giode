//go:build !android && !ios

package speech

import (
	"context"

	"github.com/xmhrnx/giode/internal/platform"
)

func Listen(ctx context.Context, h platform.Host, lang string) (<-chan Result, error) {
	return nil, platform.ErrNotSupported
}
