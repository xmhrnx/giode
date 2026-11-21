//go:build !windows

package window

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/platform"
)

// SubclassOptions says which drops the window takes (Windows).
type SubclassOptions struct {
	AcceptFiles, DragEvents bool
}

// Subclass: only Windows' windows need a procedure of giode's.
func Subclass(h platform.Host, view app.ViewEvent, opts SubclassOptions) {}
