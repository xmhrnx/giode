//go:build !windows && !(darwin && !ios) && !(linux && !android && !nox11)

package window

import "github.com/xmhrnx/giode/internal/platform"

// Buttons removes the title bar buttons asked for.
func Buttons(h platform.Host, noMin, noMax, noClose bool) {}
