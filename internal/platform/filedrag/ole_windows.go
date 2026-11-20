//go:build windows

package filedrag

import (
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode/internal/winrt"
	"golang.org/x/sys/windows"
)

// Drag enter, over and leave on Windows: an OLE IDropTarget, a COM
// object built here by hand - a vtable of Go callbacks - registered on
// the window's thread, where OLE must be initialized.

var (
	ole32                = windows.NewLazySystemDLL("ole32.dll")
	procOleInitialize    = ole32.NewProc("OleInitialize")
	procRegisterDragDrop = ole32.NewProc("RegisterDragDrop")
	procRevokeDragDrop   = ole32.NewProc("RevokeDragDrop")
	procReleaseStgMedium = ole32.NewProc("ReleaseStgMedium")
	procScreenToClient   = user32.NewProc("ScreenToClient")
	user32               = windows.NewLazySystemDLL("user32.dll")
	shell32              = windows.NewLazySystemDLL("shell32.dll")
	procDragAcceptFiles  = shell32.NewProc("DragAcceptFiles")
	procDragQueryFileW   = shell32.NewProc("DragQueryFileW")
	procDragFinish       = shell32.NewProc("DragFinish")
	procDragQueryPoint   = shell32.NewProc("DragQueryPoint")
	procPostMessageW     = user32.NewProc("PostMessageW")
)

const (
	cfHDROP          = 15
	dvaspectContent  = 1
	tymedHGlobal     = 1
	dropEffectNone   = 0
	dropEffectCopy   = 1
	sOK              = 0
	eNoInterface     = 0x80004002
	dataObjGetData   = 3 // IDataObject vtable slots
	dataObjQueryData = 5
)

var (
	iidIUnknown    = windows.GUID{Data1: 0x00000000, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIDropTarget = windows.GUID{Data1: 0x00000122, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
)

type formatEtc struct {
	format uint16
	ptd    uintptr
	aspect uint32
	index  int32
	tymed  uint32
}

type stgMedium struct {
	tymed   uint32
	handle  uintptr
	release uintptr
}

// oleDropTarget is the COM object: its first word points to the
// vtable.
type oleDropTarget struct {
	vtbl  *[7]uintptr
	hwnd  uintptr
	files bool // the drag carries files
}

var (
	oleVtbl    *[7]uintptr
	oleVtblMu  sync.Mutex
	oleTargets = map[uintptr]*oleDropTarget{} // keeps them alive
)

// RegisterOLE registers the window's IDropTarget, for drag enter, over
// and leave; on the window's thread (WMRegisterOLE).
func RegisterOLE(hwnd uintptr, logf func(format string, args ...any)) {
	// POINTL arrives by value: one register on 64-bit systems, which
	// the callbacks rely on.
	if unsafe.Sizeof(uintptr(0)) != 8 {
		return
	}
	procOleInitialize.Call(0)
	oleVtblMu.Lock()
	if oleVtbl == nil {
		oleVtbl = &[7]uintptr{
			windows.NewCallback(oleQueryInterface),
			windows.NewCallback(oleAddRef),
			windows.NewCallback(oleRelease),
			windows.NewCallback(oleDragEnter),
			windows.NewCallback(oleDragOver),
			windows.NewCallback(oleDragLeave),
			windows.NewCallback(oleDrop),
		}
	}
	t := &oleDropTarget{vtbl: oleVtbl, hwnd: hwnd}
	oleTargets[hwnd] = t
	oleVtblMu.Unlock()
	if hr, _, _ := procRegisterDragDrop.Call(hwnd, uintptr(unsafe.Pointer(t))); hr != sOK {
		logf("giode: drag events unavailable: RegisterDragDrop failed (0x%x)", hr)
		oleVtblMu.Lock()
		delete(oleTargets, hwnd)
		oleVtblMu.Unlock()
	}
}

// RevokeOLE revokes the window's IDropTarget as it goes away.
func RevokeOLE(hwnd uintptr) {
	oleVtblMu.Lock()
	_, ok := oleTargets[hwnd]
	delete(oleTargets, hwnd)
	oleVtblMu.Unlock()
	if ok {
		procRevokeDragDrop.Call(hwnd)
	}
}

func oleQueryInterface(this *oleDropTarget, riid *windows.GUID, out **oleDropTarget) uintptr {
	if *riid == iidIUnknown || *riid == iidIDropTarget {
		*out = this
		return sOK
	}
	*out = nil
	return eNoInterface
}

// The targets live as long as their windows (oleTargets), so the
// reference count isn't kept.
func oleAddRef(this *oleDropTarget) uintptr  { return 1 }
func oleRelease(this *oleDropTarget) uintptr { return 1 }

// clientPos converts a POINTL in screen pixels to window pixels.
func (t *oleDropTarget) clientPos(pt uintptr) (x, y float32) {
	p := [2]int32{int32(uint32(pt)), int32(uint32(pt >> 32))}
	procScreenToClient.Call(t.hwnd, uintptr(unsafe.Pointer(&p)))
	return float32(p[0]), float32(p[1])
}

func hdropFormat() formatEtc {
	return formatEtc{format: cfHDROP, aspect: dvaspectContent, index: -1, tymed: tymedHGlobal}
}

func (t *oleDropTarget) effect(out *uint32) uintptr {
	*out = dropEffectNone
	if t.files {
		*out = dropEffectCopy
	}
	return sOK
}

func oleDragEnter(t *oleDropTarget, data unsafe.Pointer, keys, pt uintptr, effect *uint32) uintptr {
	f := hdropFormat()
	t.files = data != nil && winrt.ComCall(data, dataObjQueryData, uintptr(unsafe.Pointer(&f))) == sOK
	if t.files {
		x, y := t.clientPos(pt)
		drag(Enter, x, y)
	}
	return t.effect(effect)
}

func oleDragOver(t *oleDropTarget, keys, pt uintptr, effect *uint32) uintptr {
	if t.files {
		x, y := t.clientPos(pt)
		drag(Over, x, y)
	}
	return t.effect(effect)
}

func oleDragLeave(t *oleDropTarget) uintptr {
	if t.files {
		t.files = false
		drag(Leave, 0, 0)
	}
	return sOK
}

func oleDrop(t *oleDropTarget, data unsafe.Pointer, keys, pt uintptr, effect *uint32) uintptr {
	if !t.files || data == nil {
		return t.effect(effect)
	}
	t.files = false
	x, y := t.clientPos(pt)
	drop(DataObjectPaths(data), x, y)
	*effect = dropEffectCopy
	return sOK
}

// DataObjectPaths reads the files an IDataObject carries (CF_HDROP).
func DataObjectPaths(data unsafe.Pointer) []string {
	if data == nil {
		return nil
	}
	f := hdropFormat()
	var m stgMedium
	var paths []string
	if winrt.ComCall(data, dataObjGetData, uintptr(unsafe.Pointer(&f)), uintptr(unsafe.Pointer(&m))) == sOK {
		if m.tymed == tymedHGlobal && m.handle != 0 {
			paths = ReadDrop(m.handle)
		}
		procReleaseStgMedium.Call(uintptr(unsafe.Pointer(&m)))
	}
	return paths
}

const (
	maxDropFiles  = 10000
	maxPathLength = 32768
)

// ReadDrop reads the paths of a drop handle (an HDROP).
func ReadDrop(hdrop uintptr) []string {
	n, _, _ := procDragQueryFileW.Call(hdrop, 0xFFFFFFFF, 0, 0)
	n = min(n, maxDropFiles)
	paths := make([]string, 0, n)
	for i := uintptr(0); i < n; i++ {
		l, _, _ := procDragQueryFileW.Call(hdrop, i, 0, 0)
		if l == 0 || l > maxPathLength {
			continue
		}
		buf := make([]uint16, l+1)
		procDragQueryFileW.Call(hdrop, i, uintptr(unsafe.Pointer(&buf[0])), l+1)
		paths = append(paths, windows.UTF16ToString(buf))
	}
	return paths
}

// AcceptFiles makes Explorer send the window WM_DROPFILES.
func AcceptFiles(hwnd uintptr) { procDragAcceptFiles.Call(hwnd, 1) }

// DropFiles handles WM_DROPFILES: it reads the drop and hands it on.
func DropFiles(hdrop uintptr) {
	var pt [2]int32
	procDragQueryPoint.Call(hdrop, uintptr(unsafe.Pointer(&pt)))
	paths := ReadDrop(hdrop)
	procDragFinish.Call(hdrop)
	drop(paths, float32(pt[0]), float32(pt[1]))
}

// Window messages the app's window procedure hands here, on the
// window's thread.
const (
	WMRegisterOLE = 0x8000 + 0x4744 // WM_APP + "GD": register the IDropTarget
	WMStartDrag   = WMRegisterOLE + 1
)

// The IDataObject and COM constants ctxmenu's DropTarget server shares.
const (
	SOK            = sOK
	ENoInterface   = eNoInterface
	DropEffectCopy = dropEffectCopy
)

var (
	IIDUnknown    = iidIUnknown
	IIDDropTarget = iidIDropTarget
)
