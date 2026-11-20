// Package ctxmenu installs the app's actions in file managers'
// context menus: Explorer, Finder's Services, Nautilus, Dolphin, Caja
// and Nemo.
package ctxmenu

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

// Flag is the argument the file manager starts the app with: the
// action's ID follows it, then the selected paths.
const Flag = "--giode-action="

// Action is a context menu action, as giode.ContextAction.
type Action struct {
	ID               string
	Label            string
	Files            bool
	Extensions       []string
	Folders          bool
	FolderBackground bool
}

// OnFiles reports whether the action applies to files.
func (c Action) OnFiles() bool {
	return c.Files || len(c.Extensions) > 0 || (!c.Folders && !c.FolderBackground)
}

// OnService receives a macOS Service's invocation: the action's ID and
// the selected files.
var OnService func(id string, paths []string)

// CLSID is the COM class ID of a context action's DropTarget verb on
// Windows: giode.ContextActionCLSID.
func CLSID(prefix, id string) string {
	h := sha1.Sum([]byte("giode-context-action:" + prefix + "." + id))
	b := h[:16]
	b[6] = b[6]&0x0f | 0x50
	b[8] = b[8]&0x3f | 0x80
	return strings.ToUpper(fmt.Sprintf("{%x-%x-%x-%x-%x}", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}
