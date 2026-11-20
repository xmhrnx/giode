package sharetarget

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

// Listen hands what other apps share with this one (through
// ShareActivity, which app.share_target declares) to deliver.
func Listen(deliver func(mobile.Shared)) error {
	return mobile.ListenShares(app.JavaVM(), app.AppContext(), deliver)
}

// IsShareScheme is for iOS's share extension.
func IsShareScheme(scheme string) bool { return false }
