//go:build android || ios || js || (!linux && !darwin && !windows)

package elevate

import "github.com/xmhrnx/giode/internal/platform"

// IsElevated reports whether the process has administrator rights.
func IsElevated() bool { return false }

func Start(h platform.Host, exe string, args []string) error { return platform.ErrNotSupported }
