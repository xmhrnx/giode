//go:build giode_gio && !giode_gio2

package giopatch

import (
	"errors"

	"gioui.org/app"
)

// Target is a frame's GPU target, as Gio's patches report it.
type Target = app.GiodeTarget

// SetUnderlay hands the underlay to giode's patched Gio, from a giode
// CLI without the giode_gio2 patches: no release or after-frame hooks.
func SetUnderlay(draw, after func(Target), release func(), forceGL bool) error {
	if release != nil {
		return errors.New("giode: Config.UnderlayRelease needs newer Gio patches: update the giode CLI (giode upgrade)")
	}
	app.GiodeSetUnderlay(draw, forceGL)
	return nil
}
