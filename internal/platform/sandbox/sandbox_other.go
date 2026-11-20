//go:build !darwin || ios

package sandbox

import "github.com/xmhrnx/giode/internal/platform"

// Only macOS has the App Sandbox.

func Enabled() bool                         { return false }
func SetLoginItem(on bool) error            { return platform.ErrNotSupported }
func LoginItemEnabled() bool                { return false }
func Open(target string, isPath bool) error { return platform.ErrNotSupported }
func Reveal(path string) error              { return platform.ErrNotSupported }
func RememberFolder(path string)            {}
