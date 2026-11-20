//go:build linux && !android && !giode_gio8

package window

import "github.com/xmhrnx/giode/internal/platform"

const CloseNeedsPatch = true

func Init(h platform.Host, transparent bool, closeRequest func() bool) {}

func waylandSetHidden(h platform.Host, hide bool) bool { return false }
func waylandActivate(h platform.Host) bool             { return false }
