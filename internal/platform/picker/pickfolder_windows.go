package picker

import (
	"runtime"
	"unsafe"

	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/winrt"
	"golang.org/x/sys/windows"
)

var (
	procSHCreateItemFromParsingName = windows.NewLazySystemDLL("shell32.dll").NewProc("SHCreateItemFromParsingName")
	procCoCreateInstance            = windows.NewLazySystemDLL("ole32.dll").NewProc("CoCreateInstance")

	clsidFileOpenDialog = windows.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileOpenDialog  = windows.GUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidIShellItem       = windows.GUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
)

// IFileOpenDialog and IShellItem vtable slots.
const (
	vtblRelease        = 2
	vtblShow           = 3
	vtblSetOptions     = 9
	vtblGetOptions     = 10
	vtblSetFolder      = 12
	vtblSetTitle       = 17
	vtblGetResult      = 20
	vtblGetDisplayName = 5

	fosPickFolders        = 0x20
	fosForceFileSystem    = 0x40
	sigdnFileSysPath      = 0x80058000
	errorCancelledHResult = 0x800704C7
)

// PickFolder shows IFileOpenDialog for folders.
func PickFolder(h platform.Host, title, dir string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err == nil {
		defer windows.CoUninitialize()
	}
	var dlg unsafe.Pointer
	if err := coCreateInstance(&clsidFileOpenDialog, &iidIFileOpenDialog, &dlg); err != nil {
		return "", err
	}
	defer winrt.ComCall(dlg, vtblRelease)

	var fos uint32
	winrt.ComCall(dlg, vtblGetOptions, uintptr(unsafe.Pointer(&fos)))
	winrt.ComCall(dlg, vtblSetOptions, uintptr(fos|fosPickFolders|fosForceFileSystem))
	if title != "" {
		winrt.ComCall(dlg, vtblSetTitle, uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(title))))
	}
	if dir != "" {
		var item unsafe.Pointer
		r, _, _ := procSHCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(dir))), 0,
			uintptr(unsafe.Pointer(&iidIShellItem)), uintptr(unsafe.Pointer(&item)))
		if r == 0 && item != nil {
			winrt.ComCall(dlg, vtblSetFolder, uintptr(item))
			winrt.ComCall(item, vtblRelease)
		}
	}
	switch hr := uint32(winrt.ComCall(dlg, vtblShow, h.NativeHandle())); {
	case hr == errorCancelledHResult:
		return "", platform.ErrCanceled
	case hr != 0:
		return "", windows.Errno(hr)
	}
	var item unsafe.Pointer
	if hr := uint32(winrt.ComCall(dlg, vtblGetResult, uintptr(unsafe.Pointer(&item)))); hr != 0 {
		return "", windows.Errno(hr)
	}
	defer winrt.ComCall(item, vtblRelease)
	var name *uint16
	if hr := uint32(winrt.ComCall(item, vtblGetDisplayName, sigdnFileSysPath, uintptr(unsafe.Pointer(&name)))); hr != 0 {
		return "", windows.Errno(hr)
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(name))
	return windows.UTF16PtrToString(name), nil
}

func coCreateInstance(clsid, iid *windows.GUID, out *unsafe.Pointer) error {
	r, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(clsid)), 0, windows.CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(out)))
	if r != 0 {
		return windows.Errno(r)
	}
	return nil
}
