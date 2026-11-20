//go:build windows

package a11y

import (
	"image"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/internal/winrt"
)

type uiaClickable struct{ clicked int }

func (c *uiaClickable) Click() { c.clicked++ }

// TestUIAProviders drives the providers through their COM vtables, as
// UI Automation does.
func TestUIAProviders(t *testing.T) {
	uiaOnce.Do(initUIAVtables)
	tree := NewTree()
	ok := &uiaClickable{}
	tree.Update([]elements.SemanticNode{
		{ID: "save", Kind: elements.SemanticButton, Label: "Save", Bounds: image.Rect(10, 10, 90, 40), Target: ok, Focused: true},
		{ID: "dark", Kind: elements.SemanticSwitch, Label: "Dark mode", Bounds: image.Rect(10, 50, 90, 80), Checked: elements.Checked},
		{ID: "h", Kind: elements.SemanticHeading, Label: "Settings", Level: 2, Bounds: image.Rect(10, 90, 200, 120)},
	})
	b := &uiaBridge{t: tree, objs: map[int32]*uiaNode{}}
	root := b.provider(0, offSimple)

	// Root -> first child through IRawElementProviderFragment.
	var frag unsafe.Pointer
	if hr := winrt.ComCall(root, 0, uintptr(unsafe.Pointer(&iidRawFragment)), uintptr(unsafe.Pointer(&frag))); hr != sOK || frag == nil {
		t.Fatalf("QueryInterface(fragment) = 0x%x", hr)
	}
	var first unsafe.Pointer
	if hr := winrt.ComCall(frag, 3, navFirstChild, uintptr(unsafe.Pointer(&first))); hr != sOK || first == nil {
		t.Fatalf("Navigate(first child) = 0x%x", hr)
	}
	prop := func(p unsafe.Pointer, id int) variant {
		var v [3]uint64
		if hr := winrt.ComCall(p, 5, uintptr(id), uintptr(unsafe.Pointer(&v))); hr != sOK {
			t.Fatalf("GetPropertyValue(%d) = 0x%x", id, hr)
		}
		return *(*variant)(unsafe.Pointer(&v))
	}
	str := func(v variant) string {
		if v.vt != vtBSTR {
			return ""
		}
		return windows.UTF16PtrToString(*(**uint16)(unsafe.Pointer(&v.val)))
	}
	if v := prop(first, propControlType); v.vt != vtI4 || int32(v.val) != 50000 {
		t.Errorf("control type = %v", v)
	}
	if s := str(prop(first, propName)); s != "Save" {
		t.Errorf("name = %q", s)
	}
	if v := prop(first, propHasKeyboardFocus); v.vt != vtBool || v.val == 0 {
		t.Errorf("focus = %v", v)
	}
	// Invoke clicks it, on the UI goroutine: here, Post runs inline.
	var inv unsafe.Pointer
	if hr := winrt.ComCall(first, 4, patInvoke, uintptr(unsafe.Pointer(&inv))); hr != sOK || inv == nil {
		t.Fatalf("GetPatternProvider(invoke) = 0x%x", hr)
	}
	var tog unsafe.Pointer
	winrt.ComCall(first, 4, patToggle, uintptr(unsafe.Pointer(&tog)))
	if tog != nil {
		t.Error("a button takes the toggle pattern")
	}

	// The switch: next sibling, toggle pattern, state on.
	var ff, second unsafe.Pointer
	winrt.ComCall(first, 0, uintptr(unsafe.Pointer(&iidRawFragment)), uintptr(unsafe.Pointer(&ff)))
	if hr := winrt.ComCall(ff, 3, navNextSibling, uintptr(unsafe.Pointer(&second))); hr != sOK || second == nil {
		t.Fatalf("Navigate(next sibling) = 0x%x", hr)
	}
	if v := prop(second, propControlType); int32(v.val) != 50002 {
		t.Errorf("switch control type = %d", int32(v.val))
	}
	if s := str(prop(second, propLocalizedControlType)); s != "toggle switch" {
		t.Errorf("switch localized type = %q", s)
	}
	winrt.ComCall(second, 4, patToggle, uintptr(unsafe.Pointer(&tog)))
	if tog == nil {
		t.Fatal("switch has no toggle pattern")
	}
	var state int32
	winrt.ComCall(tog, 4, uintptr(unsafe.Pointer(&state)))
	if state != 1 {
		t.Errorf("toggle state = %d", state)
	}

	// The heading's level, and the root's focus.
	var sf, third unsafe.Pointer
	winrt.ComCall(second, 0, uintptr(unsafe.Pointer(&iidRawFragment)), uintptr(unsafe.Pointer(&sf)))
	winrt.ComCall(sf, 3, navNextSibling, uintptr(unsafe.Pointer(&third)))
	if v := prop(third, propHeadingLevel); int32(v.val) != headingLevel1+1 {
		t.Errorf("heading level = %d", int32(v.val))
	}
	var rootFrag, focused unsafe.Pointer
	if hr := winrt.ComCall(root, 0, uintptr(unsafe.Pointer(&iidRawFragmentRoot)), uintptr(unsafe.Pointer(&rootFrag))); hr != sOK {
		t.Fatalf("QueryInterface(fragment root) = 0x%x", hr)
	}
	winrt.ComCall(rootFrag, 4, uintptr(unsafe.Pointer(&focused)))
	if focused != first {
		t.Error("GetFocus isn't the focused button")
	}
	var none unsafe.Pointer
	if hr := winrt.ComCall(first, 0, uintptr(unsafe.Pointer(&iidRawFragmentRoot)), uintptr(unsafe.Pointer(&none))); hr != eNoInterface {
		t.Error("a child answers as a fragment root")
	}

	// Runtime IDs are [UiaAppendRuntimeId, number].
	var sa uintptr
	winrt.ComCall(ff, 4, uintptr(unsafe.Pointer(&sa)))
	if sa == 0 {
		t.Error("no runtime ID")
	}
}

// TestUIATextPattern reads a field's text, selection and words through
// ITextProvider and ITextRangeProvider, as Narrator does.
func TestUIATextPattern(t *testing.T) {
	uiaOnce.Do(initUIAVtables)
	tree := NewTree()
	tree.Update([]elements.SemanticNode{
		{ID: "name", Kind: elements.SemanticInput, Label: "Name", Value: "Ada Lovelace", Bounds: image.Rect(0, 0, 120, 30),
			Focused: true, HasCaret: true, Caret: 12, Anchor: 4},
	})
	b := &uiaBridge{t: tree, objs: map[int32]*uiaNode{}}
	field := b.provider(tree.number("name"), offSimple)
	var text unsafe.Pointer
	if hr := winrt.ComCall(field, 4, patText, uintptr(unsafe.Pointer(&text))); hr != sOK || text == nil {
		t.Fatalf("GetPatternProvider(text) = 0x%x", hr)
	}
	getText := func(r unsafe.Pointer) string {
		var s *uint16
		if hr := winrt.ComCall(r, 12, ^uintptr(0), uintptr(unsafe.Pointer(&s))); hr != sOK {
			t.Fatalf("GetText = 0x%x", hr)
		}
		return windows.UTF16PtrToString(s)
	}
	var doc unsafe.Pointer
	winrt.ComCall(text, 7, uintptr(unsafe.Pointer(&doc)))
	if got := getText(doc); got != "Ada Lovelace" {
		t.Errorf("document = %q", got)
	}
	// The selection: one range, "Lovelace".
	var sa uintptr
	winrt.ComCall(text, 3, uintptr(unsafe.Pointer(&sa)))
	var sel unsafe.Pointer
	idx := int32(0)
	if hr, _, _ := windows.NewLazySystemDLL("oleaut32.dll").NewProc("SafeArrayGetElement").Call(sa, uintptr(unsafe.Pointer(&idx)), uintptr(unsafe.Pointer(&sel))); hr != sOK || sel == nil {
		t.Fatalf("selection array: 0x%x", hr)
	}
	if got := getText(sel); got != "Lovelace" {
		t.Errorf("selection = %q", got)
	}
	// A degenerate range at the start, expanded to its word, moved one
	// word on.
	var r unsafe.Pointer
	winrt.ComCall(doc, 3, uintptr(unsafe.Pointer(&r)))
	var moved int32
	winrt.ComCall(r, 14, 1, uintptr(unitDocument), ^uintptr(0), uintptr(unsafe.Pointer(&moved))) // end to the start
	winrt.ComCall(r, 6, uintptr(unitWord))
	if got := getText(r); got != "Ada " {
		t.Errorf("first word = %q", got)
	}
	winrt.ComCall(r, 13, uintptr(unitWord), 1, uintptr(unsafe.Pointer(&moved)))
	if got := getText(r); got != "Lovelace" || moved != 1 {
		t.Errorf("next word = %q (moved %d)", got, moved)
	}
	var cmp int32
	winrt.ComCall(r, 4, uintptr(sel), uintptr(unsafe.Pointer(&cmp)))
	if cmp == 0 {
		t.Error("the moved word doesn't compare equal to the selection")
	}
}
