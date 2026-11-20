//go:build giode_gio2 && linux && !android

package filedrag

import "gioui.org/app"

func installLinux() bool {
	app.GiodeSetFileDrag(func(kind int, x, y float32, uris string) {
		switch kind {
		case app.GiodeDragEnter:
			drag(Enter, x, y)
		case app.GiodeDragOver:
			drag(Over, x, y)
		case app.GiodeDragLeave:
			drag(Leave, x, y)
		case app.GiodeDragDrop:
			drop(ParseURIList(uris), x, y)
		}
	})
	return true
}
