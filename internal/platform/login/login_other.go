//go:build android || ios || js || (!linux && !darwin && !windows)

package login

import "github.com/xmhrnx/giode/internal/platform"

func Set(Entry, bool) error { return platform.ErrNotSupported }

func Enabled(name string) bool { return false }

func ByStartupTask() bool { return false }
