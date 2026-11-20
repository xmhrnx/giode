//go:build darwin && !ios

package shell

import "github.com/xmhrnx/giode/internal/platform/sandbox"

// Reveal shows path selected in a Finder window.
func Reveal(path string) error { return sandbox.Reveal(path) }
