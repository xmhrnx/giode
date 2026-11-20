//go:build android || ios || js || (!linux && !darwin && !windows && !freebsd && !openbsd && !netbsd && !dragonfly)

package shell

import "github.com/xmhrnx/giode/internal/platform"

// Reveal has no file manager to show path in.
func Reveal(path string) error { return platform.ErrNotSupported }
