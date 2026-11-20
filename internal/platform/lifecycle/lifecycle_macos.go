//go:build darwin && !ios

package lifecycle

// lifecycle_macos.m adds universal links to Gio's app delegate.

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit
*/
import "C"
