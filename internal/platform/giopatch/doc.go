// Package giopatch connects giode to the Gio patches the giode CLI
// builds with (the giode_gio* build tags): the window's first color,
// the keyboard's return key, AutoFill hints.
package giopatch

import "errors"

// ErrInjectUnpatched is Inject's error without giode's Gio patches.
var ErrInjectUnpatched = errors.New("ui_pointer and ui_key need giode's Gio patches: build and run the app with an up to date giode CLI (giode upgrade, giode run)")
