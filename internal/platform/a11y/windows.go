//go:build windows

package a11y

import (
	"image"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode/elements"
)

// Screen readers on Windows (Narrator, NVDA, JAWS) through UI
// Automation: the window answers WM_GETOBJECT with a fragment root whose
// fragments are the accessibility tree's nodes (see a11y_tree.go). Each
// node is a COM object implementing the provider interfaces and the
// control patterns its kind takes. UI Automation calls them from its
// own threads; they read the tree under its lock and run actions on the
// UI goroutine.

var (
	user32                                     = windows.NewLazySystemDLL("user32.dll")
	uiaCore                                    = windows.NewLazySystemDLL("uiautomationcore.dll")
	procUiaReturnRawElementProvider            = uiaCore.NewProc("UiaReturnRawElementProvider")
	procUiaHostProviderFromHwnd                = uiaCore.NewProc("UiaHostProviderFromHwnd")
	procUiaRaiseAutomationEvent                = uiaCore.NewProc("UiaRaiseAutomationEvent")
	procUiaRaiseAutomationPropertyChangedEvent = uiaCore.NewProc("UiaRaiseAutomationPropertyChangedEvent")
	procUiaRaiseNotificationEvent              = uiaCore.NewProc("UiaRaiseNotificationEvent")
	procUiaClientsAreListening                 = uiaCore.NewProc("UiaClientsAreListening")
	oleaut32                                   = windows.NewLazySystemDLL("oleaut32.dll")
	procSysAllocString                         = oleaut32.NewProc("SysAllocString")
	procSysFreeString                          = oleaut32.NewProc("SysFreeString")
	procSafeArrayCreateVector                  = oleaut32.NewProc("SafeArrayCreateVector")
	procSafeArrayPutElement                    = oleaut32.NewProc("SafeArrayPutElement")
	procClientToScreen                         = user32.NewProc("ClientToScreen")
)

const (
	WMGetObject     = 0x003D
	UIARootObjectID = -25

	uiaAppendRuntimeID = 3
	eNotImpl           = 0x80004001
	uiaENotAvailable   = 0x80040201 // UIA_E_ELEMENTNOTAVAILABLE
	uiaEInvalidOp      = 0x80131509 // UIA_E_INVALIDOPERATION

	providerServerSide = 0x2

	vtEmpty = 0
	vtI4    = 3
	vtR8    = 5
	vtBSTR  = 8
	vtBool  = 11

	navParent      = 0
	navNextSibling = 1
	navPrevSibling = 2
	navFirstChild  = 3
	navLastChild   = 4

	patInvoke         = 10000
	patValue          = 10002
	patRangeValue     = 10003
	patExpandCollapse = 10005
	patSelectionItem  = 10010
	patToggle         = 10015
	patText           = 10014

	propProcessID            = 30002
	propControlType          = 30003
	propLocalizedControlType = 30004
	propName                 = 30005
	propHasKeyboardFocus     = 30008
	propIsKeyboardFocusable  = 30009
	propIsEnabled            = 30010
	propAutomationID         = 30011
	propHelpText             = 30013
	propIsControlElement     = 30016
	propIsContentElement     = 30017
	propIsRequiredForForm    = 30025
	propItemStatus           = 30026
	propFrameworkID          = 30024
	propValueValue           = 30045
	propRangeValueValue      = 30047
	propExpandCollapseState  = 30070
	propSelectionIsSelected  = 30079
	propToggleState          = 30086
	propIsDataValidForForm   = 30103
	propLiveSetting          = 30135
	propPositionInSet        = 30152
	propSizeOfSet            = 30153
	propLevel                = 30154
	propLandmarkType         = 30157
	propLocalizedLandmark    = 30158
	propFullDescription      = 30159
	propHeadingLevel         = 30173
	propIsDialog             = 30174

	evFocusChanged      = 20005
	evLiveRegionChanged = 20024
	evTextSelection     = 20014
	evTextChanged       = 20015

	landmarkCustom = 80000
	headingLevel1  = 80051

	notificationKindOther           = 4
	notificationImportantMostRecent = 1
	notificationAll                 = 2
)

var (
	iidRawSimple       = windows.GUID{Data1: 0xd6dd68d1, Data2: 0x86fd, Data3: 0x4332, Data4: [8]byte{0x86, 0x66, 0x9a, 0xbe, 0xde, 0xa2, 0xd2, 0x4c}}
	iidRawFragment     = windows.GUID{Data1: 0xf7063da8, Data2: 0x8359, Data3: 0x439c, Data4: [8]byte{0x92, 0x97, 0xbb, 0xc5, 0x29, 0x9a, 0x7d, 0x87}}
	iidRawFragmentRoot = windows.GUID{Data1: 0x620ce2a5, Data2: 0xab8f, Data3: 0x40a9, Data4: [8]byte{0x86, 0xcb, 0xde, 0x3c, 0x75, 0x59, 0x9b, 0x58}}
	iidInvoke          = windows.GUID{Data1: 0x54fcb24b, Data2: 0xe18e, Data3: 0x47a2, Data4: [8]byte{0xb4, 0xd3, 0xec, 0xcb, 0xe7, 0x75, 0x99, 0xa2}}
	iidToggle          = windows.GUID{Data1: 0x56d00bd0, Data2: 0xc4f4, Data3: 0x433c, Data4: [8]byte{0xa8, 0x36, 0x1a, 0x52, 0xa5, 0x7e, 0x08, 0x92}}
	iidValue           = windows.GUID{Data1: 0xc7935180, Data2: 0x6fb3, Data3: 0x4201, Data4: [8]byte{0xb1, 0x74, 0x7d, 0xf7, 0x3a, 0xdb, 0xf6, 0x4a}}
	iidRangeValue      = windows.GUID{Data1: 0x36dc7aef, Data2: 0x33e6, Data3: 0x4691, Data4: [8]byte{0xaf, 0xe1, 0x2b, 0xe7, 0x27, 0x4b, 0x3d, 0x33}}
	iidExpandCollapse  = windows.GUID{Data1: 0xd847d3a5, Data2: 0xcab0, Data3: 0x4a98, Data4: [8]byte{0x8c, 0x32, 0xec, 0xb4, 0x5c, 0x59, 0xad, 0x24}}
	iidSelectionItem   = windows.GUID{Data1: 0x2acad808, Data2: 0xb2d4, Data3: 0x452d, Data4: [8]byte{0xa4, 0x07, 0x91, 0xff, 0x1a, 0xd1, 0x67, 0xb2}}
)

// variant is the start of a VARIANT: its type and its 8-byte value, all
// a provider writes. (A VARIANT is 24 bytes on 64-bit Windows; the rest
// is only used by records.)
type variant struct {
	vt  uint16
	_   [3]uint16
	val uint64
}

func (v *variant) setI4(i int32)   { v.vt, v.val = vtI4, uint64(uint32(i)) }
func (v *variant) setR8(f float64) { v.vt, v.val = vtR8, math.Float64bits(f) }
func (v *variant) setBool(b bool) {
	v.vt, v.val = vtBool, 0
	if b {
		v.val = 0xFFFF // VARIANT_TRUE
	}
}
func (v *variant) setString(s string) {
	v.vt, v.val = vtBSTR, uint64(bstr(s))
}

// bstr allocates s as a BSTR, which the caller passes on to its owner.
func bstr(s string) uintptr {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		p, _ = windows.UTF16PtrFromString("")
	}
	r, _, _ := procSysAllocString.Call(uintptr(unsafe.Pointer(p)))
	return r
}

// uiaRect is a UiaRect: screen coordinates.
type uiaRect struct{ left, top, width, height float64 }

// The interface slots of a node object, in order: each points to its
// vtable. A COM interface pointer is the address of its slot.
type uiaNode struct {
	simple   *[7]uintptr
	fragment *[9]uintptr
	root     *[5]uintptr
	invoke   *[4]uintptr
	toggle   *[5]uintptr
	value    *[6]uintptr
	rangeV   *[10]uintptr
	expand   *[6]uintptr
	selItem  *[8]uintptr
	text     *[9]uintptr

	refs atomic.Int32
	b    *uiaBridge
	k    int32
}

// The slots' offsets in a node object.
var (
	offSimple   = unsafe.Offsetof(uiaNode{}.simple)
	offFragment = unsafe.Offsetof(uiaNode{}.fragment)
	offRoot     = unsafe.Offsetof(uiaNode{}.root)
	offInvoke   = unsafe.Offsetof(uiaNode{}.invoke)
	offToggle   = unsafe.Offsetof(uiaNode{}.toggle)
	offValue    = unsafe.Offsetof(uiaNode{}.value)
	offRange    = unsafe.Offsetof(uiaNode{}.rangeV)
	offExpand   = unsafe.Offsetof(uiaNode{}.expand)
	offSelItem  = unsafe.Offsetof(uiaNode{}.selItem)
	offText     = unsafe.Offsetof(uiaNode{}.text)
)

// nodeOf returns the object whose slot at off this is.
func nodeOf(this unsafe.Pointer, off uintptr) *uiaNode {
	return (*uiaNode)(unsafe.Add(this, -int(off)))
}

func (o *uiaNode) iface(off uintptr) unsafe.Pointer {
	return unsafe.Add(unsafe.Pointer(o), int(off))
}

// uiaBridge is a window's UI Automation provider tree.
type uiaBridge struct {
	a    Host
	t    *Tree
	hwnd uintptr

	mu   sync.Mutex
	objs map[int32]*uiaNode
	// focus is the node last reported focused.
	focus int32
	// notify is set where UiaRaiseNotificationEvent exists (Windows 10
	// 1709 on).
	notify bool
	// ranges keeps the text ranges clients hold from the collector.
	ranges map[*uiaRange]struct{}
	// caret and text are the focused field's last frame, to report
	// changes.
	caret [3]int
	text  string
}

var (
	uiaOnce    sync.Once
	vtSimple   *[7]uintptr
	vtFragment *[9]uintptr
	vtRoot     *[5]uintptr
	vtInvoke   *[4]uintptr
	vtToggle   *[5]uintptr
	vtValue    *[6]uintptr
	vtRange    *[10]uintptr
	vtExpand   *[6]uintptr
	vtSelItem  *[8]uintptr
	vtText     *[9]uintptr
	vtTextRng  *[21]uintptr
	uiaMu      sync.Mutex
	uiaBridges = map[Host]*uiaBridge{}
)

// unknown returns the IUnknown slots of an interface at off.
func unknown(off uintptr) [3]uintptr {
	return [3]uintptr{
		windows.NewCallback(func(this unsafe.Pointer, riid *windows.GUID, out *unsafe.Pointer) uintptr {
			return nodeOf(this, off).query(riid, out)
		}),
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return uintptr(nodeOf(this, off).refs.Add(1))
		}),
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return uintptr(nodeOf(this, off).release())
		}),
	}
}

func initUIAVtables() {
	u := unknown(offSimple)
	simple := &[7]uintptr{u[0], u[1], u[2],
		windows.NewCallback(func(this unsafe.Pointer, out *int32) uintptr {
			*out = providerServerSide
			return sOK
		}),
		windows.NewCallback(func(this unsafe.Pointer, pattern uintptr, out *unsafe.Pointer) uintptr {
			return nodeOf(this, offSimple).pattern(int32(pattern), out)
		}),
		windows.NewCallback(func(this unsafe.Pointer, prop uintptr, out *variant) uintptr {
			return nodeOf(this, offSimple).property(int32(prop), out)
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
			o := nodeOf(this, offSimple)
			*out = nil
			if o.k == 0 {
				hr, _, _ := procUiaHostProviderFromHwnd.Call(o.b.hwnd, uintptr(unsafe.Pointer(out)))
				return hr
			}
			return sOK
		}),
	}
	u = unknown(offFragment)
	fragment := &[9]uintptr{u[0], u[1], u[2],
		windows.NewCallback(func(this unsafe.Pointer, dir uintptr, out *unsafe.Pointer) uintptr {
			return nodeOf(this, offFragment).navigate(int32(dir), out)
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *uintptr) uintptr {
			return nodeOf(this, offFragment).runtimeID(out)
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *uiaRect) uintptr {
			return nodeOf(this, offFragment).bounds(out)
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *uintptr) uintptr {
			*out = 0
			return sOK
		}),
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return nodeOf(this, offFragment).do(a11yFocus, "")
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
			o := nodeOf(this, offFragment)
			*out = o.b.provider(0, offSimple)
			return sOK
		}),
	}
	u = unknown(offRoot)
	root := &[5]uintptr{u[0], u[1], u[2],
		elementFromPointCallback(),
		windows.NewCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
			o := nodeOf(this, offRoot)
			*out = nil
			if k := o.b.t.focusedNode(); k != 0 {
				*out = o.b.provider(k, offSimple)
			}
			return sOK
		}),
	}
	u = unknown(offInvoke)
	invoke := &[4]uintptr{u[0], u[1], u[2],
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return nodeOf(this, offInvoke).do(a11yActivate, "")
		}),
	}
	u = unknown(offToggle)
	toggle := &[5]uintptr{u[0], u[1], u[2],
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return nodeOf(this, offToggle).do(a11yActivate, "")
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *int32) uintptr {
			n, ok := nodeOf(this, offToggle).node()
			if !ok {
				return uiaENotAvailable
			}
			*out = toggleState(n)
			return sOK
		}),
	}
	u = unknown(offValue)
	value := &[6]uintptr{u[0], u[1], u[2],
		windows.NewCallback(func(this unsafe.Pointer, s *uint16) uintptr {
			return nodeOf(this, offValue).do(a11ySetText, windows.UTF16PtrToString(s))
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *uintptr) uintptr {
			n, ok := nodeOf(this, offValue).node()
			if !ok {
				return uiaENotAvailable
			}
			*out = bstr(n.Value)
			return sOK
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *int32) uintptr {
			n, ok := nodeOf(this, offValue).node()
			if !ok {
				return uiaENotAvailable
			}
			_, _, setText, _ := a11yActions(n)
			*out = boolInt(!setText)
			return sOK
		}),
	}
	u = unknown(offRange)
	rangeGet := func(f func(n elements.SemanticNode) float64) uintptr {
		return windows.NewCallback(func(this unsafe.Pointer, out *float64) uintptr {
			n, ok := nodeOf(this, offRange).node()
			if !ok {
				return uiaENotAvailable
			}
			*out = f(n)
			return sOK
		})
	}
	rangeV := &[10]uintptr{u[0], u[1], u[2],
		// SetValue takes a double, in a floating-point register a Go
		// callback can't read: the slider's keys change it instead. It
		// declares two words for it, as 32-bit Windows passes it on the
		// stack and the callee pops it.
		windows.NewCallback(func(this unsafe.Pointer, _, _ uintptr) uintptr { return eNotImpl }),
		rangeGet(func(n elements.SemanticNode) float64 { return n.Now }),
		windows.NewCallback(func(this unsafe.Pointer, out *int32) uintptr {
			n, ok := nodeOf(this, offRange).node()
			if !ok {
				return uiaENotAvailable
			}
			*out = boolInt(n.Kind != elements.SemanticSlider || n.Disabled)
			return sOK
		}),
		rangeGet(func(n elements.SemanticNode) float64 { return n.Max }),
		rangeGet(func(n elements.SemanticNode) float64 { return n.Min }),
		rangeGet(func(n elements.SemanticNode) float64 { return (n.Max - n.Min) / 10 }),
		rangeGet(func(n elements.SemanticNode) float64 { return (n.Max - n.Min) / 100 }),
	}
	u = unknown(offExpand)
	expand := &[6]uintptr{u[0], u[1], u[2],
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return nodeOf(this, offExpand).do(a11yExpand, "")
		}),
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return nodeOf(this, offExpand).do(a11yCollapse, "")
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *int32) uintptr {
			n, ok := nodeOf(this, offExpand).node()
			if !ok {
				return uiaENotAvailable
			}
			*out = expandState(n)
			return sOK
		}),
	}
	u = unknown(offSelItem)
	selItem := &[8]uintptr{u[0], u[1], u[2],
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return nodeOf(this, offSelItem).selectItem()
		}),
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return nodeOf(this, offSelItem).selectItem()
		}),
		windows.NewCallback(func(this unsafe.Pointer) uintptr { return uiaEInvalidOp }),
		windows.NewCallback(func(this unsafe.Pointer, out *int32) uintptr {
			n, ok := nodeOf(this, offSelItem).node()
			if !ok {
				return uiaENotAvailable
			}
			*out = boolInt(n.Selected)
			return sOK
		}),
		windows.NewCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
			o := nodeOf(this, offSelItem)
			*out = nil
			if _, parent, _, ok := o.b.t.node(o.k); ok && parent != 0 {
				*out = o.b.provider(parent, offSimple)
			}
			return sOK
		}),
	}
	vtSimple, vtFragment, vtRoot, vtInvoke, vtToggle = simple, fragment, root, invoke, toggle
	vtValue, vtRange, vtExpand, vtSelItem = value, rangeV, expand, selItem
	initUIATextVtables()
}

// installA11y sets up the tree; the bridge starts when a screen reader
// first asks the window for it (GetObject).
// Install sets the tree up for the platform's bridge.
func Install(a Host) *Tree {
	t := NewTree()
	setTree(a, t)
	t.Notify = func() {
		uiaMu.Lock()
		b := uiaBridges[a]
		uiaMu.Unlock()
		if b == nil {
			t.takeChanges()
			return
		}
		b.events()
	}
	return t
}

// GetObject answers WM_GETOBJECT for UI Automation's root object.
func GetObject(a Host, hwnd, wparam, lparam uintptr) (uintptr, bool) {
	t := treeOf(a)
	if t == nil || procUiaReturnRawElementProvider.Find() != nil {
		return 0, false
	}
	uiaOnce.Do(initUIAVtables)
	uiaMu.Lock()
	b := uiaBridges[a]
	if b == nil {
		b = &uiaBridge{a: a, t: t, hwnd: hwnd, objs: map[int32]*uiaNode{}, notify: procUiaRaiseNotificationEvent.Find() == nil}
		uiaBridges[a] = b
	}
	uiaMu.Unlock()
	if !elements.SemanticsWanted() {
		// From now on each frame records the tree.
		elements.SetSemanticsWanted(true)
		a.Invalidate()
	}
	root := b.provider(0, offSimple)
	r, _, _ := procUiaReturnRawElementProvider.Call(hwnd, wparam, lparam, uintptr(root))
	// UiaReturnRawElementProvider took its own reference.
	b.obj(0).release()
	return r, true
}

// WindowGone lets go of the window's providers.
func WindowGone(hwnd uintptr) {
	if procUiaReturnRawElementProvider.Find() == nil {
		procUiaReturnRawElementProvider.Call(hwnd, 0, 0, 0)
	}
}

// obj returns node k's object, made on first use. It holds no reference
// for the caller.
func (b *uiaBridge) obj(k int32) *uiaNode {
	b.mu.Lock()
	defer b.mu.Unlock()
	o := b.objs[k]
	if o == nil {
		o = &uiaNode{b: b, k: k}
		o.simple, o.fragment, o.root, o.invoke, o.toggle = vtSimple, vtFragment, vtRoot, vtInvoke, vtToggle
		o.value, o.rangeV, o.expand, o.selItem = vtValue, vtRange, vtExpand, vtSelItem
		o.text = vtText
		b.objs[k] = o
	}
	return o
}

// provider returns node k's interface at off with a reference for the
// caller, as COM out parameters are.
func (b *uiaBridge) provider(k int32, off uintptr) unsafe.Pointer {
	o := b.obj(k)
	o.refs.Add(1)
	return o.iface(off)
}

func (o *uiaNode) release() int32 {
	n := o.refs.Add(-1)
	if n <= 0 && o.k != 0 {
		// Gone from the tree and from every client: let it be collected.
		if _, _, _, ok := o.b.t.node(o.k); !ok {
			o.b.mu.Lock()
			if o.refs.Load() <= 0 && o.b.objs[o.k] == o {
				delete(o.b.objs, o.k)
			}
			o.b.mu.Unlock()
		}
	}
	return max(n, 0)
}

func (o *uiaNode) node() (elements.SemanticNode, bool) {
	if o.k == 0 {
		return elements.SemanticNode{}, false
	}
	n, _, _, ok := o.b.t.node(o.k)
	return n, ok
}

func (o *uiaNode) query(riid *windows.GUID, out *unsafe.Pointer) uintptr {
	*out = nil
	var off uintptr
	switch *riid {
	case iidIUnknown, iidRawSimple:
		off = offSimple
	case iidRawFragment:
		off = offFragment
	case iidRawFragmentRoot:
		if o.k != 0 {
			return eNoInterface
		}
		off = offRoot
	default:
		pat, ok := patternOfIID(*riid)
		if !ok || !o.hasPattern(pat) {
			return eNoInterface
		}
		off = patternOffset(pat)
	}
	o.refs.Add(1)
	*out = o.iface(off)
	return sOK
}

func patternOfIID(iid windows.GUID) (int32, bool) {
	switch iid {
	case iidInvoke:
		return patInvoke, true
	case iidToggle:
		return patToggle, true
	case iidValue:
		return patValue, true
	case iidRangeValue:
		return patRangeValue, true
	case iidExpandCollapse:
		return patExpandCollapse, true
	case iidSelectionItem:
		return patSelectionItem, true
	case iidTextProvider:
		return patText, true
	}
	return 0, false
}

func patternOffset(pat int32) uintptr {
	switch pat {
	case patInvoke:
		return offInvoke
	case patToggle:
		return offToggle
	case patValue:
		return offValue
	case patRangeValue:
		return offRange
	case patExpandCollapse:
		return offExpand
	case patSelectionItem:
		return offSelItem
	case patText:
		return offText
	}
	return 0
}

// hasPattern reports whether the node takes control pattern pat.
func (o *uiaNode) hasPattern(pat int32) bool {
	n, ok := o.node()
	if !ok {
		return false
	}
	return uiaHasPattern(n, pat)
}

func uiaHasPattern(n elements.SemanticNode, pat int32) bool {
	activate, _, setText, _ := a11yActions(n)
	switch pat {
	case patToggle:
		return n.Checked != elements.NotCheckable
	case patText:
		return n.HasCaret || textual(n)
	case patSelectionItem:
		switch n.Kind {
		case elements.SemanticRadio, elements.SemanticTab, elements.SemanticTreeItem, elements.SemanticListItem:
			return n.Checked == elements.NotCheckable
		}
	case patExpandCollapse:
		return n.Expanded != elements.NotExpandable
	case patValue:
		return n.Kind == elements.SemanticInput || n.Kind == elements.SemanticComboBox || (setText && n.Kind != elements.SemanticSlider)
	case patRangeValue:
		return n.Kind == elements.SemanticSlider || (n.Kind == elements.SemanticProgress && !n.Busy)
	case patInvoke:
		if !activate || n.Checked != elements.NotCheckable || n.Expanded != elements.NotExpandable {
			return false
		}
		switch n.Kind {
		case elements.SemanticRadio, elements.SemanticTab, elements.SemanticInput, elements.SemanticSlider:
			return false
		case elements.SemanticTreeItem, elements.SemanticListItem:
			// Selection items, unless they act when activated.
			_, click := n.Target.(interface{ Click() })
			return click
		}
		return true
	}
	return false
}

func (o *uiaNode) pattern(pat int32, out *unsafe.Pointer) uintptr {
	*out = nil
	if off := patternOffset(pat); off != 0 && o.hasPattern(pat) {
		o.refs.Add(1)
		*out = o.iface(off)
	}
	return sOK
}

func (o *uiaNode) do(action a11yAction, arg string) uintptr {
	if o.k == 0 {
		return sOK
	}
	if _, ok := o.node(); !ok {
		return uiaENotAvailable
	}
	if !o.b.t.perform(o.b.a, o.k, action, arg) {
		return uiaEInvalidOp
	}
	return sOK
}

func (o *uiaNode) selectItem() uintptr {
	n, ok := o.node()
	if !ok {
		return uiaENotAvailable
	}
	if n.Selected {
		return sOK
	}
	return o.do(a11yActivate, "")
}

func toggleState(n elements.SemanticNode) int32 {
	switch n.Checked {
	case elements.Checked:
		return 1
	case elements.Mixed:
		return 2
	}
	return 0
}

func expandState(n elements.SemanticNode) int32 {
	switch n.Expanded {
	case elements.Expanded:
		return 1
	case elements.Collapsed:
		return 0
	}
	return 3 // a leaf
}

func (o *uiaNode) navigate(dir int32, out *unsafe.Pointer) uintptr {
	*out = nil
	_, parent, children, ok := o.b.t.node(o.k)
	if !ok {
		return uiaENotAvailable
	}
	var to int32 = -1
	switch dir {
	case navParent:
		if o.k != 0 {
			to = parent
		}
	case navFirstChild:
		if len(children) > 0 {
			to = children[0]
		}
	case navLastChild:
		if len(children) > 0 {
			to = children[len(children)-1]
		}
	case navNextSibling, navPrevSibling:
		if o.k == 0 {
			break
		}
		_, _, siblings, _ := o.b.t.node(parent)
		for i, s := range siblings {
			if s != o.k {
				continue
			}
			if dir == navNextSibling && i+1 < len(siblings) {
				to = siblings[i+1]
			}
			if dir == navPrevSibling && i > 0 {
				to = siblings[i-1]
			}
			break
		}
	}
	if to >= 0 {
		*out = o.b.provider(to, offSimple)
	}
	return sOK
}

func (o *uiaNode) runtimeID(out *uintptr) uintptr {
	*out = 0
	if o.k == 0 {
		// The window's own provider gives the root's.
		return sOK
	}
	sa, _, _ := procSafeArrayCreateVector.Call(vtI4, 0, 2)
	if sa == 0 {
		return eFail
	}
	for i, v := range [2]int32{uiaAppendRuntimeID, o.k} {
		idx := int32(i)
		procSafeArrayPutElement.Call(sa, uintptr(unsafe.Pointer(&idx)), uintptr(unsafe.Pointer(&v)))
	}
	*out = sa
	return sOK
}

func (o *uiaNode) bounds(out *uiaRect) uintptr {
	*out = uiaRect{}
	if o.k == 0 {
		return sOK
	}
	n, ok := o.node()
	if !ok {
		return uiaENotAvailable
	}
	pt := [2]int32{int32(n.Bounds.Min.X), int32(n.Bounds.Min.Y)}
	procClientToScreen.Call(o.b.hwnd, uintptr(unsafe.Pointer(&pt)))
	*out = uiaRect{float64(pt[0]), float64(pt[1]), float64(n.Bounds.Dx()), float64(n.Bounds.Dy())}
	return sOK
}

// fromScreen returns the provider of the node at screen point (x, y),
// or the root's when there is none.
func (b *uiaBridge) fromScreen(x, y int32) unsafe.Pointer {
	pt := [2]int32{x, y}
	procScreenToClient.Call(b.hwnd, uintptr(unsafe.Pointer(&pt)))
	if k := b.t.at(image.Pt(int(pt[0]), int(pt[1]))); k != 0 {
		return b.provider(k, offSimple)
	}
	return nil
}

func (o *uiaNode) property(prop int32, out *variant) uintptr {
	out.vt, out.val = vtEmpty, 0
	switch prop {
	case propFrameworkID:
		out.setString("Giode")
		return sOK
	case propProcessID:
		out.setI4(int32(os.Getpid()))
		return sOK
	}
	if o.k == 0 {
		// The window's own provider names it and gives its role.
		return sOK
	}
	n, ok := o.node()
	if !ok {
		return uiaENotAvailable
	}
	focusable := o.b.t.isFocusable(n, o.k)
	switch prop {
	case propControlType:
		out.setI4(uiaControlType(n))
	case propLocalizedControlType:
		if s := uiaLocalizedType(n); s != "" {
			out.setString(s)
		}
	case propName:
		if s := uiaName(n); s != "" {
			out.setString(s)
		}
	case propHasKeyboardFocus:
		out.setBool(n.Focused)
	case propIsKeyboardFocusable:
		_, _, _, focus := a11yActions(n)
		out.setBool(focus || (focusable && actionable(n) && !n.Disabled))
	case propIsEnabled:
		out.setBool(!n.Disabled)
	case propAutomationID:
		if n.ID != "" {
			out.setString(n.ID)
		}
	case propHelpText, propFullDescription:
		if n.Description != "" {
			out.setString(n.Description)
		}
	case propIsControlElement, propIsContentElement:
		out.setBool(focusable || !textual(n))
	case propIsRequiredForForm:
		out.setBool(n.Required)
	case propIsDataValidForForm:
		out.setBool(!n.Invalid)
	case propItemStatus:
		if n.Invalid {
			out.setString(nonEmpty(n.Description, "Invalid"))
		} else if n.Busy {
			out.setString("Busy")
		}
	case propLiveSetting:
		switch {
		case n.Live == elements.LiveAssertive || n.Kind == elements.SemanticAlert:
			out.setI4(2)
		case n.Live == elements.LivePolite || n.Kind == elements.SemanticStatus:
			out.setI4(1)
		}
	case propPositionInSet:
		if n.Pos > 0 {
			out.setI4(int32(n.Pos))
		}
	case propSizeOfSet:
		if n.Count > 0 {
			out.setI4(int32(n.Count))
		}
	case propLevel:
		if n.Level > 0 && n.Kind != elements.SemanticHeading {
			out.setI4(int32(n.Level))
		}
	case propHeadingLevel:
		if n.Kind == elements.SemanticHeading && n.Level > 0 {
			out.setI4(headingLevel1 + int32(min(n.Level, 9)) - 1)
		}
	case propLandmarkType:
		if n.Kind == elements.SemanticRegion {
			out.setI4(landmarkCustom)
		}
	case propLocalizedLandmark:
		if n.Kind == elements.SemanticRegion {
			out.setString("region")
		}
	case propIsDialog:
		out.setBool(n.Kind == elements.SemanticDialog)
	case propValueValue:
		if uiaHasPattern(n, patValue) {
			out.setString(n.Value)
		}
	case propRangeValueValue:
		if uiaHasPattern(n, patRangeValue) {
			out.setR8(n.Now)
		}
	case propToggleState:
		if uiaHasPattern(n, patToggle) {
			out.setI4(toggleState(n))
		}
	case propExpandCollapseState:
		if uiaHasPattern(n, patExpandCollapse) {
			out.setI4(expandState(n))
		}
	case propSelectionIsSelected:
		if uiaHasPattern(n, patSelectionItem) {
			out.setBool(n.Selected)
		}
	}
	return sOK
}

func nonEmpty(s, or string) string {
	if s != "" {
		return s
	}
	return or
}

// uiaName is what a node is called: its label; a text's or an image's
// text when it has none.
func uiaName(n elements.SemanticNode) string {
	if n.Label != "" {
		return n.Label
	}
	switch n.Kind {
	case elements.SemanticInput, elements.SemanticSlider, elements.SemanticProgress, elements.SemanticComboBox, elements.SemanticSelect:
		// Their value is read as their value.
		return ""
	}
	return n.Value
}

// uiaControlType is the UI Automation control type a kind is.
func uiaControlType(n elements.SemanticNode) int32 {
	switch n.Kind {
	case elements.SemanticButton:
		return 50000
	case elements.SemanticCheckbox, elements.SemanticSwitch:
		return 50002
	case elements.SemanticSelect, elements.SemanticComboBox:
		return 50003
	case elements.SemanticInput:
		return 50004
	case elements.SemanticLink:
		return 50005
	case elements.SemanticImage:
		return 50006
	case elements.SemanticListItem:
		return 50007
	case elements.SemanticList:
		return 50008
	case elements.SemanticMenu:
		return 50009
	case elements.SemanticMenuBar:
		return 50010
	case elements.SemanticMenuItem:
		return 50011
	case elements.SemanticProgress:
		return 50012
	case elements.SemanticRadio:
		return 50013
	case elements.SemanticSlider:
		return 50015
	case elements.SemanticTabList:
		return 50018
	case elements.SemanticTab:
		return 50019
	case elements.SemanticText, elements.SemanticHeading, elements.SemanticStatus, elements.SemanticAlert:
		return 50020
	case elements.SemanticToolbar:
		return 50021
	case elements.SemanticTree:
		return 50023
	case elements.SemanticTreeItem:
		return 50024
	case elements.SemanticTable:
		return 50036
	case elements.SemanticRow:
		return 50029 // a data item
	case elements.SemanticColumnHeader:
		return 50035
	case elements.SemanticDialog:
		return 50032
	case elements.SemanticScroll, elements.SemanticTabPanel:
		return 50033
	}
	return 50026 // a group: regions, cells, groups
}

// uiaLocalizedType names the kinds whose control type doesn't say what
// they are.
func uiaLocalizedType(n elements.SemanticNode) string {
	switch n.Kind {
	case elements.SemanticSwitch:
		return "toggle switch"
	case elements.SemanticHeading:
		return "heading"
	case elements.SemanticAlert:
		return "alert"
	case elements.SemanticStatus:
		return "status"
	case elements.SemanticDialog:
		return "dialog"
	case elements.SemanticRegion:
		return "region"
	case elements.SemanticCell:
		return "cell"
	case elements.SemanticTabPanel:
		return "tab panel"
	}
	return ""
}

func uiaListening() bool {
	if procUiaClientsAreListening.Find() != nil {
		return false
	}
	r, _, _ := procUiaClientsAreListening.Call()
	return r != 0
}

// events tells listening clients what changed in the frame just laid
// out: the focus, the focused node's state, and what to announce. It
// runs on the UI goroutine after each tree update.
func (b *uiaBridge) events() {
	changed, announce := b.t.takeChanges()
	if !uiaListening() {
		return
	}
	focus := b.t.focusedNode()
	b.mu.Lock()
	moved := focus != b.focus
	b.focus = focus
	b.mu.Unlock()
	if moved && focus != 0 {
		b.raise(focus, evFocusChanged)
	}
	for _, k := range changed {
		if k == 0 || (k == focus && moved) {
			continue
		}
		n, _, _, ok := b.t.node(k)
		if !ok {
			continue
		}
		if n.Focused {
			b.raiseState(k, n)
			b.raiseText(k, n)
		}
		if !b.notify && (n.Live != elements.LiveOff || n.Kind == elements.SemanticAlert) {
			b.raise(k, evLiveRegionChanged)
		}
	}
	if b.notify {
		for _, a := range announce {
			processing := uintptr(notificationAll)
			if a.assertive {
				processing = notificationImportantMostRecent
			}
			root := b.provider(0, offSimple)
			text, id := bstr(a.text), bstr("giode.announce")
			procUiaRaiseNotificationEvent.Call(uintptr(root), notificationKindOther, processing, text, id)
			procSysFreeString.Call(text)
			procSysFreeString.Call(id)
			b.obj(0).release()
		}
	}
}

func (b *uiaBridge) raise(k int32, event uintptr) {
	p := b.provider(k, offSimple)
	procUiaRaiseAutomationEvent.Call(uintptr(p), event)
	b.obj(k).release()
}

// raiseState reports the state of focused node k as changed - its
// toggle, expansion, value - so a screen reader reads the new one.
func (b *uiaBridge) raiseState(k int32, n elements.SemanticNode) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		// VARIANTs go by value on 32-bit Windows; not passed here.
		return
	}
	prop := int32(0)
	var nv [3]uint64
	v := (*variant)(unsafe.Pointer(&nv))
	switch {
	case uiaHasPattern(n, patToggle):
		prop = propToggleState
		v.setI4(toggleState(n))
	case uiaHasPattern(n, patExpandCollapse):
		prop = propExpandCollapseState
		v.setI4(expandState(n))
	case uiaHasPattern(n, patRangeValue):
		prop = propRangeValueValue
		v.setR8(n.Now)
	case uiaHasPattern(n, patValue):
		prop = propValueValue
		v.setString(n.Value)
	case uiaHasPattern(n, patSelectionItem):
		prop = propSelectionIsSelected
		v.setBool(n.Selected)
	default:
		prop = propName
		v.setString(uiaName(n))
	}
	var old [3]uint64
	p := b.provider(k, offSimple)
	procUiaRaiseAutomationPropertyChangedEvent.Call(uintptr(p), uintptr(prop), uintptr(unsafe.Pointer(&old)), uintptr(unsafe.Pointer(&nv)))
	b.obj(k).release()
	if v.vt == vtBSTR {
		procSysFreeString.Call(uintptr(v.val))
	}
}

// fromCursor answers ElementProviderFromPoint for the cursor position.
func fromCursor(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
	o := nodeOf(this, offRoot)
	var pt [2]int32
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	*out = o.b.fromScreen(pt[0], pt[1])
	return sOK
}

// COM results and interfaces the UI Automation providers use.
const (
	sOK          = 0
	eNoInterface = 0x80004002
	eFail        = 0x80004005
)

var iidIUnknown = windows.GUID{Data1: 0x00000000, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}

var (
	procGetCursorPos   = user32.NewProc("GetCursorPos")
	procScreenToClient = user32.NewProc("ScreenToClient")
)
