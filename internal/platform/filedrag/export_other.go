//go:build !windows && !(darwin && !ios) && !(linux && !android && giode_gio5)

package filedrag

import "github.com/xmhrnx/giode/internal/platform"

// Dragging out of the window isn't supported here: drags stay in it.
// InstallExporter lets Draggable.Export drag out of the window; done
// runs through post, on the UI goroutine.
func InstallExporter(h platform.Host) {}
