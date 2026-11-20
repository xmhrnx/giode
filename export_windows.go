//go:build windows

package giode

import (
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode/elements/dnd"
)

// Dragging out of the window on Windows (Draggable.Export): the shell's
// SHDoDragDrop, with a shell data object carrying CF_HDROP and
// CF_UNICODETEXT, run on the window's thread from the subclassed window
// procedure (see dropWndProc). SHDoDragDrop provides the drop source:
// Escape cancels, releasing the button drops.

var (
	procSHDoDragDrop       = shell32.NewProc("SHDoDragDrop")
	procSHCreateDataObject = shell32.NewProc("SHCreateDataObject")
	kernel32Export         = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalAllocExport  = kernel32Export.NewProc("GlobalAlloc")
	procGlobalLockExport   = kernel32Export.NewProc("GlobalLock")
	procGlobalUnlockExport = kernel32Export.NewProc("GlobalUnlock")
	procGlobalFreeExport   = kernel32Export.NewProc("GlobalFree")
	procGetCursorPos       = user32.NewProc("GetCursorPos")
)

const (
	wmStartDrag    = wmRegisterOLE + 1
	wmLButtonUp    = 0x0202
	cfUnicodeText  = 13
	gmemMoveable   = 0x0002
	gmemZeroInit   = 0x0040
	dragdropSDrop  = 0x00040100
	dataObjSetData = 7 // IDataObject vtable slot
	unknownRelease = 2
)

var iidDataObjectExport = windows.GUID{Data1: 0x0000010E, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}

// exportJob is a drag waiting for its window's thread.
type exportJob struct {
	data dnd.ExportData
	done func(bool)
}

var (
	exportMu   sync.Mutex
	exportJobs = map[uintptr]*exportJob{}
)

func installExporter(a *App) {
	dnd.SetExporter(func(data dnd.ExportData, done func(bool)) bool {
		hwnd := a.NativeHandle()
		dropMu.Lock()
		_, subclassed := dropWindows[hwnd]
		dropMu.Unlock()
		if hwnd == 0 || !subclassed {
			return false
		}
		exportMu.Lock()
		exportJobs[hwnd] = &exportJob{data: data, done: done}
		exportMu.Unlock()
		r, _, _ := procPostMessageW.Call(hwnd, wmStartDrag, 0, 0)
		return r != 0
	})
}

// runExport runs the drag waiting for hwnd, on its thread. It returns
// when the drag ends.
func runExport(a *App, hwnd uintptr) {
	exportMu.Lock()
	job := exportJobs[hwnd]
	delete(exportJobs, hwnd)
	exportMu.Unlock()
	if job == nil {
		return
	}
	dropped := false
	defer func() {
		// The drag took the button's release: give Gio one, or it takes
		// the button for still held.
		var pt [2]int32
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
		procPostMessageW.Call(hwnd, wmLButtonUp, 0, uintptr(uint16(pt[0]))|uintptr(uint16(pt[1]))<<16)
		a.Post(func() { job.done(dropped) })
	}()
	procOleInitialize.Call(0)
	var obj unsafe.Pointer
	if hr, _, _ := procSHCreateDataObject.Call(0, 0, 0, 0, uintptr(unsafe.Pointer(&iidDataObjectExport)), uintptr(unsafe.Pointer(&obj))); hr != sOK || obj == nil {
		a.logf(LogLevelWarning, "giode: drag out unavailable: SHCreateDataObject failed (0x%x)", hr)
		return
	}
	defer comCall(obj, unknownRelease)
	if len(job.data.Files) > 0 {
		setGlobal(obj, cfHDROP, dropFiles(job.data.Files))
	}
	if job.data.Text != "" {
		setGlobal(obj, cfUnicodeText, utf16z(job.data.Text))
	}
	var effect uint32
	hr, _, _ := procSHDoDragDrop.Call(hwnd, uintptr(obj), 0, dropEffectCopy, uintptr(unsafe.Pointer(&effect)))
	dropped = hr == dragdropSDrop && effect != dropEffectNone
}

// setGlobal gives obj data in format as an HGLOBAL, which obj then owns.
func setGlobal(obj unsafe.Pointer, format uint16, data []byte) {
	h, _, _ := procGlobalAllocExport.Call(gmemMoveable|gmemZeroInit, uintptr(len(data)))
	if h == 0 {
		return
	}
	p, _, _ := procGlobalLockExport.Call(h)
	if p == 0 {
		procGlobalFreeExport.Call(h)
		return
	}
	copy(unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), len(data)), data)
	procGlobalUnlockExport.Call(h)
	fe := formatEtc{format: format, aspect: dvaspectContent, index: -1, tymed: tymedHGlobal}
	med := stgMedium{tymed: tymedHGlobal, handle: h}
	if hr := comCall(obj, dataObjSetData, uintptr(unsafe.Pointer(&fe)), uintptr(unsafe.Pointer(&med)), 1); hr != sOK {
		procGlobalFreeExport.Call(h)
	}
}

// dropFiles is a DROPFILES structure listing paths: the header, then
// the paths in UTF-16, each ended by a NUL, and a final NUL.
func dropFiles(paths []string) []byte {
	const header = 20 // pFiles, pt (x, y), fNC, fWide
	var names []uint16
	for _, p := range paths {
		names = append(names, utf16.Encode([]rune(p))...)
		names = append(names, 0)
	}
	names = append(names, 0)
	b := make([]byte, header+2*len(names))
	b[0] = header // pFiles: where the names start
	b[16] = 1     // fWide: UTF-16 names
	for i, c := range names {
		b[header+2*i] = byte(c)
		b[header+2*i+1] = byte(c >> 8)
	}
	return b
}

// utf16z is s in UTF-16 with a final NUL, as bytes.
func utf16z(s string) []byte {
	u := append(utf16.Encode([]rune(s)), 0)
	b := make([]byte, 2*len(u))
	for i, c := range u {
		b[2*i] = byte(c)
		b[2*i+1] = byte(c >> 8)
	}
	return b
}
