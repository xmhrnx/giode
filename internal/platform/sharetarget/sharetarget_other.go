//go:build !android && !ios

package sharetarget

import "github.com/xmhrnx/giode/internal/mobile"

// Desktops and the web aren't share targets.
func Listen(deliver func(mobile.Shared)) error { return nil }

// IsShareScheme is for iOS's share extension.
func IsShareScheme(scheme string) bool { return false }
