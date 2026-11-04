//go:build windows

package giode

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procCoCreateInstance2 = ole32.NewProc("CoCreateInstance")
	iidIDataObject        = windows.GUID{Data1: 0x0000010e, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
)

// TestContextActionServer drops two files on a context action's COM
// class as Explorer's DropTarget verb does: both arrive in one call.
func TestContextActionServer(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("64-bit only")
	}
	dir := t.TempDir()
	var paths []string
	for _, n := range []string{"a.txt", "b.txt"} {
		p := filepath.Join(dir, n)
		os.WriteFile(p, []byte(n), 0o644)
		paths = append(paths, p)
	}
	type drop struct {
		id    string
		paths []string
	}
	got := make(chan drop, 2)
	contextDropHook = func(id string, p []string) { got <- drop{id, p} }
	defer func() { contextDropHook = nil }()

	a := &App{cfg: Config{ContextActions: []ContextAction{{ID: "send", Label: "Send"}}}}
	a.serveContextActions()
	defer a.stopContextActions()
	time.Sleep(300 * time.Millisecond) // the registrations run on their own thread

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED)
	defer windows.CoUninitialize()

	var clsid windows.GUID
	s, _ := windows.UTF16PtrFromString(ContextActionCLSID(a.contextActionPrefix(), "send"))
	procCLSIDFromString.Call(uintptr(unsafe.Pointer(s)), uintptr(unsafe.Pointer(&clsid)))
	var target unsafe.Pointer
	if hr, _, _ := procCoCreateInstance2.Call(uintptr(unsafe.Pointer(&clsid)), 0, clsctxLocalServer,
		uintptr(unsafe.Pointer(&iidIDropTarget)), uintptr(unsafe.Pointer(&target))); hr != sOK {
		t.Fatalf("CoCreateInstance: 0x%x", hr)
	}
	defer comCall(target, 2)

	// A data object carrying the two files as CF_HDROP, as Explorer's
	// does (Wine has no SHCreateDataObject to build one).
	data := newTestDataObject(paths)

	effect := uint32(dropEffectCopy)
	const mkLButton = 1
	comCall(target, 3, uintptr(data), mkLButton, 0, uintptr(unsafe.Pointer(&effect))) // DragEnter
	if hr := comCall(target, 6, uintptr(data), mkLButton, 0, uintptr(unsafe.Pointer(&effect))); hr != sOK {
		t.Fatalf("Drop: 0x%x", hr)
	}
	select {
	case d := <-got:
		if d.id != "send" || !reflect.DeepEqual(d.paths, paths) {
			t.Errorf("dropped %q %q, want send %q", d.id, d.paths, paths)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the drop didn't reach the app")
	}
}

// testDataObject is a minimal IDataObject offering CF_HDROP.
type testDataObject struct {
	vtbl  *[12]uintptr
	paths []string
}

var (
	kernel32t        = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalAlloc  = kernel32t.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32t.NewProc("GlobalLock")
	procGlobalUnlock = kernel32t.NewProc("GlobalUnlock")
	testDataVtbl     *[12]uintptr
	testDataObjects  []*testDataObject
)

func newTestDataObject(paths []string) unsafe.Pointer {
	if testDataVtbl == nil {
		notImpl := windows.NewCallback(func(this uintptr) uintptr { return 0x80004001 })
		notImpl2 := windows.NewCallback(func(this, a uintptr) uintptr { return 0x80004001 })
		notImpl3 := windows.NewCallback(func(this, a, b uintptr) uintptr { return 0x80004001 })
		notImpl4 := windows.NewCallback(func(this, a, b, c uintptr) uintptr { return 0x80004001 })
		notImpl5 := windows.NewCallback(func(this, a, b, c, d uintptr) uintptr { return 0x80004001 })
		testDataVtbl = &[12]uintptr{
			windows.NewCallback(func(this *testDataObject, riid *windows.GUID, out *unsafe.Pointer) uintptr {
				if *riid == iidIUnknown || *riid == iidIDataObject {
					*out = unsafe.Pointer(this)
					return sOK
				}
				*out = nil
				return eNoInterface
			}),
			windows.NewCallback(func(this uintptr) uintptr { return 2 }),
			windows.NewCallback(func(this uintptr) uintptr { return 1 }),
			windows.NewCallback(testDataGetData),
			notImpl3, // GetDataHere
			windows.NewCallback(func(this uintptr, f *formatEtc) uintptr {
				if f.format == cfHDROP {
					return sOK
				}
				return 0x80040064 // DV_E_FORMATETC
			}),
			notImpl3, // GetCanonicalFormatEtc
			notImpl4, // SetData
			notImpl3, // EnumFormatEtc
			notImpl5, // DAdvise
			notImpl2, // DUnadvise
			notImpl2, // EnumDAdvise
		}
		_ = notImpl
	}
	o := &testDataObject{vtbl: testDataVtbl, paths: paths}
	testDataObjects = append(testDataObjects, o)
	return unsafe.Pointer(o)
}

func testDataGetData(this *testDataObject, f *formatEtc, m *stgMedium) uintptr {
	if f.format != cfHDROP || f.tymed&tymedHGlobal == 0 {
		return 0x80040064
	}
	var list []uint16
	for _, p := range this.paths {
		list = append(list, utf16.Encode([]rune(p))...)
		list = append(list, 0)
	}
	list = append(list, 0)
	const header = 20 // DROPFILES
	const gmemMoveable = 0x2
	size := header + 2*len(list)
	h, _, _ := procGlobalAlloc.Call(gmemMoveable|0x40, uintptr(size))
	p, _, _ := procGlobalLock.Call(h)
	buf := unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), size)
	*(*uint32)(unsafe.Pointer(&buf[0])) = header // pFiles
	*(*uint32)(unsafe.Pointer(&buf[16])) = 1     // fWide
	copy(buf[header:], unsafe.Slice((*byte)(unsafe.Pointer(&list[0])), 2*len(list)))
	procGlobalUnlock.Call(h)
	m.tymed, m.handle, m.release = tymedHGlobal, h, 0
	return sOK
}
