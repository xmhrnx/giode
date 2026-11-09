package giode

import (
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// IFileOpenDialog and IShellItemArray vtable slots beyond those in
// pickfolder_windows.go.
const (
	vtblSetFileTypes     = 4
	vtblDialogGetResults = 27
	vtblArrayCount       = 7
	vtblArrayItemAt      = 8

	fosNoChangeDir      = 0x8
	fosAllowMultiselect = 0x200
	fosPathMustExist    = 0x800
	fosFileMustExist    = 0x1000
)

// filterSpec is COMDLG_FILTERSPEC.
type filterSpec struct {
	name, spec *uint16
}

// openFileDialog shows the system open dialog (IFileOpenDialog), owned
// by the app's window so it stays in front of it and is modal to it.
// Each file comes back as its own shell item, so one file and many go
// the same way.
func (a *App) openFileDialog(exts []string, multiple bool) ([]string, error) {
	fileDialogMu.Lock()
	defer fileDialogMu.Unlock()
	// COM objects belong to the thread (apartment) that made them.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err == nil {
		defer windows.CoUninitialize()
	}
	var dlg unsafe.Pointer
	if err := coCreateInstance(&clsidFileOpenDialog, &iidIFileOpenDialog, &dlg); err != nil {
		return nil, err
	}
	defer comCall(dlg, vtblRelease)

	var fos uint32
	comCall(dlg, vtblGetOptions, uintptr(unsafe.Pointer(&fos)))
	fos |= fosForceFileSystem | fosFileMustExist | fosPathMustExist | fosNoChangeDir
	if multiple {
		fos |= fosAllowMultiselect
	}
	comCall(dlg, vtblSetOptions, uintptr(fos))
	if len(exts) > 0 {
		pattern := filePattern(exts)
		p := windows.StringToUTF16Ptr(pattern)
		specs := []filterSpec{{name: p, spec: p}}
		comCall(dlg, vtblSetFileTypes, uintptr(len(specs)), uintptr(unsafe.Pointer(&specs[0])))
		runtime.KeepAlive(specs)
	}
	switch hr := uint32(comCall(dlg, vtblShow, a.NativeHandle())); {
	case hr == errorCancelledHResult:
		return nil, ErrCanceled
	case hr != 0:
		return nil, windows.Errno(hr)
	}
	var items unsafe.Pointer
	if hr := uint32(comCall(dlg, vtblDialogGetResults, uintptr(unsafe.Pointer(&items)))); hr != 0 {
		return nil, windows.Errno(hr)
	}
	defer comCall(items, vtblRelease)
	var n uint32
	if hr := uint32(comCall(items, vtblArrayCount, uintptr(unsafe.Pointer(&n)))); hr != 0 {
		return nil, windows.Errno(hr)
	}
	var paths []string
	for i := uint32(0); i < n; i++ {
		var item unsafe.Pointer
		if hr := uint32(comCall(items, vtblArrayItemAt, uintptr(i), uintptr(unsafe.Pointer(&item)))); hr != 0 {
			return nil, windows.Errno(hr)
		}
		var name *uint16
		hr := uint32(comCall(item, vtblGetDisplayName, sigdnFileSysPath, uintptr(unsafe.Pointer(&name))))
		comCall(item, vtblRelease)
		if hr != 0 {
			return nil, windows.Errno(hr)
		}
		paths = append(paths, windows.UTF16PtrToString(name))
		windows.CoTaskMemFree(unsafe.Pointer(name))
	}
	if len(paths) == 0 {
		return nil, ErrCanceled
	}
	return paths, nil
}

// filePattern turns extensions (".wav") into the dialog's filter,
// "*.wav;*.mp3".
func filePattern(exts []string) string {
	pats := make([]string, len(exts))
	for i, e := range exts {
		if !strings.HasPrefix(e, "*") {
			e = "*" + e
		}
		pats[i] = e
	}
	return strings.Join(pats, ";")
}
