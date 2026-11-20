//go:build giode_gio7

package giopatch

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/components/kit"
)

// Text fields tell password managers and AutoFill what they hold, with
// the giode CLI's Gio patches (revision 22 on).
func init() {
	kit.SetAutofill = app.GiodeSetAutofill
}
