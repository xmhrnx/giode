//go:build !darwin && !android

package notify

import (
	"context"

	"github.com/xmhrnx/giode/internal/platform"
)

func Show(h platform.Host, title, body string) error {
	return xnotify(title, body)
}

// RequestPermission reports true: desktops other than
// macOS don't ask.
func RequestPermission(ctx context.Context, h platform.Host) (bool, error) {
	return true, nil
}

// AskOnce: only Android asks.
func AskOnce(h platform.Host) {}
