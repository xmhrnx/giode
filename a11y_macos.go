//go:build darwin && !ios

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

#include <stdint.h>
#include <stdlib.h>

void giodeAXInstall(uintptr_t view);
void giodeAXChanged(uintptr_t view, int32_t focus, int32_t *changed, int n, char *announce, int assertive, int layout);
*/
import "C"

import (
	"image"
	"sync"
	"sync/atomic"
	"unsafe"

	"gioui.org/app"

	"github.com/xmhrnx/giode/elements"
)

// VoiceOver on macOS: Giode adds accessibility methods to Gio's view
// (a11y_macos.m) whose elements read the tree (a11y_tree.go) through the
// exported functions here, on the main thread. The first time VoiceOver
// or another assistive app asks the view for its elements, frames start
// recording the tree.

var (
	axMu    sync.Mutex
	axApps  = map[uintptr]*App{}
	axViews = map[*App]uintptr{}
	// axGen counts tree updates, for elements to know their cache is old.
	axGen atomic.Int64
)

func installA11y(a *App) {
	t := newA11yTree()
	a.a11y = t
	var focus int32
	t.notify = func() {
		axGen.Add(1)
		changed, announce := t.takeChanges()
		axMu.Lock()
		view := axViews[a]
		axMu.Unlock()
		if view == 0 {
			return
		}
		f := t.focusedNode()
		moved := f != focus
		focus = f
		layout := false
		var ids []int32
		for _, k := range changed {
			if k == 0 {
				layout = true
				continue
			}
			ids = append(ids, k)
		}
		if !moved && len(ids) == 0 && len(announce) == 0 && !layout {
			return
		}
		sendFocus := int32(-1)
		if moved {
			sendFocus = f
		}
		var text string
		assertive := 0
		for i, an := range announce {
			if i > 0 {
				text += ". "
			}
			text += an.text
			if an.assertive {
				assertive = 1
			}
		}
		var cids *C.int32_t
		if len(ids) > 0 {
			cids = (*C.int32_t)(C.malloc(C.size_t(len(ids)) * 4))
			copy(unsafe.Slice((*int32)(unsafe.Pointer(cids)), len(ids)), ids)
		}
		var ctext *C.char
		if text != "" {
			ctext = C.CString(text)
		}
		// giodeAXChanged takes the buffers and frees them.
		C.giodeAXChanged(C.uintptr_t(view), C.int32_t(sendFocus), cids, C.int(len(ids)), ctext, C.int(assertive), C.int(boolInt(layout)))
	}
}

// a11yView gives Gio's view its accessibility methods.
func (a *App) a11yView(ev app.ViewEvent) {
	v, ok := ev.(app.AppKitViewEvent)
	if !ok || v.View == 0 || a.a11y == nil {
		return
	}
	axMu.Lock()
	axApps[v.View] = a
	axViews[a] = v.View
	axMu.Unlock()
	C.giodeAXInstall(C.uintptr_t(v.View))
}

func axTree(view C.uintptr_t) (*App, *a11yTree) {
	axMu.Lock()
	a := axApps[uintptr(view)]
	axMu.Unlock()
	if a == nil || a.a11y == nil {
		return nil, nil
	}
	return a, a.a11y
}

//export giodeAXWanted
func giodeAXWanted(view C.uintptr_t) {
	a, _ := axTree(view)
	if a != nil && !elements.SemanticsWanted() {
		elements.SetSemanticsWanted(true)
		a.invalidate()
	}
}

//export giodeAXGeneration
func giodeAXGeneration() C.int64_t { return C.int64_t(axGen.Load()) }

// giodeAXNode returns node id as JSON, for the caller to free; NULL when
// it's gone.
//
//export giodeAXNode
func giodeAXNode(view C.uintptr_t, id C.int32_t) *C.char {
	_, t := axTree(view)
	if t == nil {
		return nil
	}
	s := t.macJSON(int32(id))
	if s == "" {
		return nil
	}
	return C.CString(s)
}

//export giodeAXAt
func giodeAXAt(view C.uintptr_t, x, y C.double) C.int32_t {
	_, t := axTree(view)
	if t == nil {
		return 0
	}
	return C.int32_t(t.at(image.Pt(int(x), int(y))))
}

//export giodeAXFocused
func giodeAXFocused(view C.uintptr_t) C.int32_t {
	_, t := axTree(view)
	if t == nil {
		return 0
	}
	return C.int32_t(t.focusedNode())
}

//export giodeAXAction
func giodeAXAction(view C.uintptr_t, id C.int32_t, action *C.char, arg *C.char) C.int {
	a, t := axTree(view)
	if t == nil {
		return 0
	}
	act := macAction(C.GoString(action))
	if act == 0 {
		return 0
	}
	s := ""
	if arg != nil {
		s = C.GoString(arg)
	}
	return C.int(boolInt(t.perform(a, int32(id), act, s)))
}
