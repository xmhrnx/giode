package giode

import (
	"errors"
	"sync"
	"time"
	"unsafe"

	"github.com/xmhrnx/giode/internal/winrt"
)

// Sharing on Windows: the system's share sheet (Windows 10 and later),
// through DataTransferManager for the app's window. It runs on the
// window's thread, reached with a window message as drag and drop is;
// the files are loaded as StorageFiles first, so the share sheet's data
// request is answered at once.

const wmShare = 0x8000 + 0x4746 // WM_APP + "GF"

var (
	iidDTMInterop       = winrt.ParseGUID("3a3dcd6c-3eab-43dc-bcde-45671ce800c8")
	iidDTM              = winrt.ParseGUID("a5caee9b-8708-49d1-8d36-67d25a8da00c")
	iidDataRequestedArg = winrt.ParseGUID("cb8ba807-6ac5-43c9-8ac5-9ba232163182")
	iidDataRequest      = winrt.ParseGUID("4341ae3b-fc12-4e53-8c02-ac714c415a27")
	iidDataPackage      = winrt.ParseGUID("61ebf5c7-efea-4346-9554-981d7e198ffe")
	iidDataPackageProps = winrt.ParseGUID("cd1c93eb-4c4c-443a-a8d3-f5c241e91689")
	iidStorageFileStats = winrt.ParseGUID("5984c710-daf2-43c8-8bb4-a4d3eacfd03f")
	iidStorageItem      = winrt.ParseGUID("4207a996-ca2f-42f7-bde8-8b10457a7f30")
	dataRequestedIID    = winrt.TypedEventHandlerIID(
		winrt.RuntimeClassSignature("Windows.ApplicationModel.DataTransfer.DataTransferManager", iidDTM),
		winrt.RuntimeClassSignature("Windows.ApplicationModel.DataTransfer.DataRequestedEventArgs", iidDataRequestedArg))
)

var shareReq struct {
	sync.Mutex
	title, text string
	paths       []string
	done        chan error
	// The share sheet's data, ready for its request.
	files   *winrt.Iterable
	handler *winrt.Delegate
	dtm     *winrt.Obj
	token   int64
}

// shareNative shows the share sheet for text or files.
func (a *App) shareNative(title, text string, paths []string) error {
	hwnd := a.NativeHandle()
	if hwnd == 0 {
		return ErrNotSupported
	}
	if title == "" {
		title = a.Title()
	}
	done := make(chan error, 1)
	shareReq.Lock()
	shareReq.title, shareReq.text, shareReq.paths, shareReq.done = title, text, paths, done
	shareReq.Unlock()
	procPostMessageW.Call(hwnd, wmShare, 0, 0)
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		return errors.New("giode: the share sheet didn't open")
	}
}

// runShare opens the share sheet; on the window's thread.
func runShare(a *App, hwnd uintptr) {
	shareReq.Lock()
	done := shareReq.done
	paths := shareReq.paths
	shareReq.Unlock()
	if done == nil {
		return
	}
	err := openShareSheet(hwnd, paths)
	done <- err
}

func openShareSheet(hwnd uintptr, paths []string) error {
	// The files, as StorageFiles.
	var files []*winrt.Obj
	if len(paths) > 0 {
		stats, err := winrt.Factory("Windows.Storage.StorageFile", iidStorageFileStats)
		if err != nil {
			return err
		}
		defer stats.Release()
		for _, p := range paths {
			h := winrt.NewHString(p)
			op, err := stats.Out(6, uintptr(h)) // GetFileFromPathAsync
			h.Delete()
			if err != nil {
				return err
			}
			f, err := winrt.Await(op, 5*time.Second)
			if err != nil {
				return err
			}
			item, err := f.Query(iidStorageItem)
			f.Release()
			if err != nil {
				return err
			}
			files = append(files, item)
		}
	}
	interop, err := winrt.Factory("Windows.ApplicationModel.DataTransfer.DataTransferManager", iidDTMInterop)
	if err != nil {
		return err
	}
	defer interop.Release()
	shareReq.Lock()
	defer shareReq.Unlock()
	if len(files) > 0 {
		shareReq.files = winrt.NewIterable(iidStorageItem, files)
	} else {
		shareReq.files = nil
	}
	if shareReq.dtm == nil {
		var dtm *winrt.Obj
		if err := interop.Call(3, hwnd, uintptr(unsafe.Pointer(&iidDTM)), uintptr(unsafe.Pointer(&dtm))); err != nil { // GetForWindow
			return err
		}
		shareReq.handler = winrt.NewDelegate(dataRequestedIID, onDataRequested)
		if err := dtm.Call(6, shareReq.handler.Ptr(), uintptr(unsafe.Pointer(&shareReq.token))); err != nil { // add_DataRequested
			dtm.Release()
			return err
		}
		shareReq.dtm = dtm
	}
	return interop.Call(4, hwnd) // ShowShareUIForWindow
}

// onDataRequested fills the share sheet's data package.
func onDataRequested(_, args *winrt.Obj) {
	shareReq.Lock()
	title, text, files := shareReq.title, shareReq.text, shareReq.files
	shareReq.Unlock()
	a, err := args.Query(iidDataRequestedArg)
	if err != nil {
		return
	}
	defer a.Release()
	r, err := a.Out(6) // Request
	if err != nil {
		return
	}
	defer r.Release()
	req, err := r.Query(iidDataRequest)
	if err != nil {
		return
	}
	defer req.Release()
	d, err := req.Out(6) // Data
	if err != nil {
		return
	}
	defer d.Release()
	pkg, err := d.Query(iidDataPackage)
	if err != nil {
		return
	}
	defer pkg.Release()
	if p, err := pkg.Out(7); err == nil { // Properties
		if props, err := p.Query(iidDataPackageProps); err == nil {
			h := winrt.NewHString(title)
			props.Call(7, uintptr(h)) // SetTitle: the sheet needs one
			h.Delete()
			props.Release()
		}
		p.Release()
	}
	if text != "" {
		h := winrt.NewHString(text)
		pkg.Call(16, uintptr(h)) // SetText
		h.Delete()
	}
	if files != nil {
		pkg.Call(23, files.Ptr()) // SetStorageItems
	}
}
