//go:build giode_gio4

package giode

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/components/kit"
)

// The keyboard's return key reads Done for single-line fields, with the
// giode CLI's Gio patches (revision 14 on).
func init() {
	kit.SetReturnKeyDone = app.GiodeSetReturnKeyDone
}
