//go:build !android && !ios && !js && !windows && !(darwin && !ios)

package shell

import "github.com/xmhrnx/giode/internal/platform"

// Linux has no system share sheet.
func shareNative(h platform.Host, title, text string, paths []string) error {
	return platform.ErrNotSupported
}
