//go:build !windows && !(darwin && !ios) && !(linux && !android)

package window

import "github.com/xmhrnx/giode/internal/platform"

// Phones and the web have one window, which apps don't hide.

const CloseNeedsPatch = false

func Init(h platform.Host, transparent bool, closeRequest func() bool) {}
func MenuBarApp()                                                      {}
func Hide(h platform.Host) bool                                        { return false }
func Show(h platform.Host)                                             {}
func Raise(h platform.Host) bool                                       { return false }
func Shown(h platform.Host, interceptClose bool)                       {}
func InterceptClose(h platform.Host)                                   {}
