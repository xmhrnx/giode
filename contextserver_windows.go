//go:build windows

package giode

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The context actions' COM server: for each action on files or folders,
// a class factory registered with CoRegisterClassObject making
// IDropTarget objects. Explorer's DropTarget verb drops the selection on
// one: Drop runs the action with every selected path, in this process.
// The objects are built by hand, vtables of Go callbacks, as the
// window's drop target is (drop_ole_windows.go).

var (
	procCoRegisterClassObject = ole32.NewProc("CoRegisterClassObject")
	procCoRevokeClassObject   = ole32.NewProc("CoRevokeClassObject")
	procCLSIDFromString       = ole32.NewProc("CLSIDFromString")
)

const (
	clsctxLocalServer     = 0x4
	regclsMultipleUse     = 1
	classENoAggregation   = 0x80040110
	eFail                 = 0x80004005
	coinitMultithreaded   = 0x0
	coinitDisableOLE1DDE  = 0x4
	sFalse                = 1
	contextServerVtblSize = 7
)

var iidIClassFactory = windows.GUID{Data1: 0x00000001, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}

// comFactory is an IClassFactory for one action.
type comFactory struct {
	vtbl *[5]uintptr
	app  *App
	id   string
}

// comTarget is an IDropTarget for one drop.
type comTarget struct {
	vtbl *[contextServerVtblSize]uintptr
	app  *App
	id   string
	refs atomic.Int32
}

var (
	comOnce       sync.Once
	factoryVtbl   *[5]uintptr
	targetVtbl    *[contextServerVtblSize]uintptr
	comObjectsMu  sync.Mutex
	comObjects    = map[unsafe.Pointer]any{} // keeps live objects from the collector
	contextServer sync.Once
	// contextServerStop ends the server: its classes are revoked before
	// the process exits, as COM calling into Go during exit crashes.
	contextServerStop    = make(chan struct{})
	contextServerDone    = make(chan struct{})
	contextServerStarted atomic.Bool
	contextServerStopped sync.Once
)

// contextDropHook, set by tests, takes drops instead of the app.
var contextDropHook func(id string, paths []string)

func initCOMVtables() {
	factoryVtbl = &[5]uintptr{
		windows.NewCallback(factoryQueryInterface),
		windows.NewCallback(factoryAddRef),
		windows.NewCallback(factoryRelease),
		windows.NewCallback(factoryCreateInstance),
		windows.NewCallback(factoryLockServer),
	}
	targetVtbl = &[contextServerVtblSize]uintptr{
		windows.NewCallback(targetQueryInterface),
		windows.NewCallback(targetAddRef),
		windows.NewCallback(targetRelease),
		windows.NewCallback(targetDragEnter),
		windows.NewCallback(targetDragOver),
		windows.NewCallback(targetDragLeave),
		windows.NewCallback(targetDrop),
	}
}

func keepCOM(p unsafe.Pointer, obj any) {
	comObjectsMu.Lock()
	comObjects[p] = obj
	comObjectsMu.Unlock()
}

func dropCOM(p unsafe.Pointer) {
	comObjectsMu.Lock()
	delete(comObjects, p)
	comObjectsMu.Unlock()
}

// serveContextActions registers the class objects of the app's context
// actions on files and folders, for the app's lifetime, so Explorer
// hands a whole selection to this instance. Only the first instance
// serves them.
func (a *App) serveContextActions() {
	// POINTL arrives by value, in one register on 64-bit systems only.
	if unsafe.Sizeof(uintptr(0)) != 8 {
		return
	}
	var ids []string
	for _, c := range a.ContextActions() {
		if c.onFiles() || c.Folders {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	contextServer.Do(func() {
		comOnce.Do(initCOMVtables)
		prefix := a.contextActionPrefix()
		contextServerStarted.Store(true)
		go func() {
			// An apartment-free (multithreaded) server: COM calls the
			// objects on its own threads, with no message loop to run.
			runtime.LockOSThread()
			if err := windows.CoInitializeEx(0, coinitMultithreaded|coinitDisableOLE1DDE); err != nil {
				a.logf(LogLevelWarning, "giode: context actions' COM server: %v", err)
				return
			}
			defer close(contextServerDone)
			defer windows.CoUninitialize()
			var cookies []uint32
			for _, id := range ids {
				var clsid windows.GUID
				s, _ := windows.UTF16PtrFromString(ContextActionCLSID(prefix, id))
				if hr, _, _ := procCLSIDFromString.Call(uintptr(unsafe.Pointer(s)), uintptr(unsafe.Pointer(&clsid))); hr != sOK {
					continue
				}
				f := &comFactory{vtbl: factoryVtbl, app: a, id: id}
				keepCOM(unsafe.Pointer(f), f)
				var cookie uint32
				hr, _, _ := procCoRegisterClassObject.Call(uintptr(unsafe.Pointer(&clsid)), uintptr(unsafe.Pointer(f)),
					clsctxLocalServer, regclsMultipleUse, uintptr(unsafe.Pointer(&cookie)))
				if hr != sOK {
					a.logf(LogLevelWarning, "giode: registering context action %q with COM failed (0x%x)", id, hr)
					continue
				}
				cookies = append(cookies, cookie)
			}
			<-contextServerStop
			for _, c := range cookies {
				procCoRevokeClassObject.Call(uintptr(c))
			}
		}()
	})
}

// stopContextActions revokes the context actions' COM classes, when
// they were registered. The app calls it as it exits.
func (a *App) stopContextActions() {
	contextServerStopped.Do(func() {
		contextServer.Do(func() {}) // a later serveContextActions does nothing
		close(contextServerStop)
		if contextServerStarted.Load() {
			select {
			case <-contextServerDone:
			case <-time.After(2 * time.Second):
			}
		}
	})
}

func factoryQueryInterface(this *comFactory, riid *windows.GUID, out *unsafe.Pointer) uintptr {
	if *riid == iidIUnknown || *riid == iidIClassFactory {
		*out = unsafe.Pointer(this)
		return sOK
	}
	*out = nil
	return eNoInterface
}

// The factories live as long as the process.
func factoryAddRef(this *comFactory) uintptr  { return 2 }
func factoryRelease(this *comFactory) uintptr { return 1 }

func factoryCreateInstance(this *comFactory, outer unsafe.Pointer, riid *windows.GUID, out *unsafe.Pointer) uintptr {
	*out = nil
	if outer != nil {
		return classENoAggregation
	}
	if *riid != iidIUnknown && *riid != iidIDropTarget {
		return eNoInterface
	}
	t := &comTarget{vtbl: targetVtbl, app: this.app, id: this.id}
	t.refs.Store(1)
	keepCOM(unsafe.Pointer(t), t)
	*out = unsafe.Pointer(t)
	return sOK
}

func factoryLockServer(this *comFactory, lock uintptr) uintptr { return sOK }

func targetQueryInterface(this *comTarget, riid *windows.GUID, out *unsafe.Pointer) uintptr {
	if *riid == iidIUnknown || *riid == iidIDropTarget {
		this.refs.Add(1)
		*out = unsafe.Pointer(this)
		return sOK
	}
	*out = nil
	return eNoInterface
}

func targetAddRef(this *comTarget) uintptr { return uintptr(this.refs.Add(1)) }

func targetRelease(this *comTarget) uintptr {
	n := this.refs.Add(-1)
	if n <= 0 {
		dropCOM(unsafe.Pointer(this))
		return 0
	}
	return uintptr(n)
}

func targetDragEnter(this *comTarget, data unsafe.Pointer, keys, pt uintptr, effect *uint32) uintptr {
	*effect = dropEffectCopy
	return sOK
}

func targetDragOver(this *comTarget, keys, pt uintptr, effect *uint32) uintptr {
	*effect = dropEffectCopy
	return sOK
}

func targetDragLeave(this *comTarget) uintptr { return sOK }

// targetDrop runs the action with the dropped selection and brings the
// window to the front.
func targetDrop(this *comTarget, data unsafe.Pointer, keys, pt uintptr, effect *uint32) uintptr {
	var paths []string
	if data != nil {
		f := hdropFormat()
		var m stgMedium
		if comCall(data, dataObjGetData, uintptr(unsafe.Pointer(&f)), uintptr(unsafe.Pointer(&m))) == sOK {
			if m.tymed == tymedHGlobal && m.handle != 0 {
				paths = readDrop(m.handle)
			}
			procReleaseStgMedium.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
	*effect = dropEffectCopy
	if len(paths) == 0 {
		return sOK
	}
	if contextDropHook != nil {
		contextDropHook(this.id, paths)
		return sOK
	}
	this.app.runContextAction(append([]string{contextActionFlag + this.id}, paths...))
	this.app.showFromLaunch()
	return sOK
}
