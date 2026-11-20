//go:build windows

package ctxmenu

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/xmhrnx/giode/internal/platform/filedrag"
	"golang.org/x/sys/windows"
)

// The context actions' COM server: for each action on files or folders,
// a class factory registered with CoRegisterClassObject making
// IDropTarget objects. Explorer's DropTarget verb drops the selection on
// one: Drop runs the action with every selected path, in this process.
// The objects are built by hand, vtables of Go callbacks, as the
// window's drop target is (drop_ole_windows.go).

var (
	ole32                     = windows.NewLazySystemDLL("ole32.dll")
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
	run  func(id string, paths []string)
	id   string
}

// comTarget is an IDropTarget for one drop.
type comTarget struct {
	vtbl *[contextServerVtblSize]uintptr
	run  func(id string, paths []string)
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

// Serve registers the class objects of the context actions ids on
// files and folders, for the app's lifetime, so Explorer hands a whole
// selection to this instance: run gets each drop. Only the first
// instance serves them.
func Serve(prefix string, ids []string, run func(id string, paths []string), logf func(format string, args ...any)) {
	// POINTL arrives by value, in one register on 64-bit systems only.
	if unsafe.Sizeof(uintptr(0)) != 8 || len(ids) == 0 {
		return
	}
	contextServer.Do(func() {
		comOnce.Do(initCOMVtables)
		contextServerStarted.Store(true)
		go func() {
			// An apartment-free (multithreaded) server: COM calls the
			// objects on its own threads, with no message loop to run.
			runtime.LockOSThread()
			if err := windows.CoInitializeEx(0, coinitMultithreaded|coinitDisableOLE1DDE); err != nil {
				logf("giode: context actions' COM server: %v", err)
				return
			}
			defer close(contextServerDone)
			defer windows.CoUninitialize()
			var cookies []uint32
			for _, id := range ids {
				var clsid windows.GUID
				s, _ := windows.UTF16PtrFromString(CLSID(prefix, id))
				if hr, _, _ := procCLSIDFromString.Call(uintptr(unsafe.Pointer(s)), uintptr(unsafe.Pointer(&clsid))); hr != filedrag.SOK {
					continue
				}
				f := &comFactory{vtbl: factoryVtbl, run: run, id: id}
				keepCOM(unsafe.Pointer(f), f)
				var cookie uint32
				hr, _, _ := procCoRegisterClassObject.Call(uintptr(unsafe.Pointer(&clsid)), uintptr(unsafe.Pointer(f)),
					clsctxLocalServer, regclsMultipleUse, uintptr(unsafe.Pointer(&cookie)))
				if hr != filedrag.SOK {
					logf("giode: registering context action %q with COM failed (0x%x)", id, hr)
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

// StopServing revokes the context actions' COM classes, when
// they were registered. The app calls it as it exits.
func StopServing() {
	contextServerStopped.Do(func() {
		contextServer.Do(func() {}) // a later Serve does nothing
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
	if *riid == filedrag.IIDUnknown || *riid == iidIClassFactory {
		*out = unsafe.Pointer(this)
		return filedrag.SOK
	}
	*out = nil
	return filedrag.ENoInterface
}

// The factories live as long as the process.
func factoryAddRef(this *comFactory) uintptr  { return 2 }
func factoryRelease(this *comFactory) uintptr { return 1 }

func factoryCreateInstance(this *comFactory, outer unsafe.Pointer, riid *windows.GUID, out *unsafe.Pointer) uintptr {
	*out = nil
	if outer != nil {
		return classENoAggregation
	}
	if *riid != filedrag.IIDUnknown && *riid != filedrag.IIDDropTarget {
		return filedrag.ENoInterface
	}
	t := &comTarget{vtbl: targetVtbl, run: this.run, id: this.id}
	t.refs.Store(1)
	keepCOM(unsafe.Pointer(t), t)
	*out = unsafe.Pointer(t)
	return filedrag.SOK
}

func factoryLockServer(this *comFactory, lock uintptr) uintptr { return filedrag.SOK }

func targetQueryInterface(this *comTarget, riid *windows.GUID, out *unsafe.Pointer) uintptr {
	if *riid == filedrag.IIDUnknown || *riid == filedrag.IIDDropTarget {
		this.refs.Add(1)
		*out = unsafe.Pointer(this)
		return filedrag.SOK
	}
	*out = nil
	return filedrag.ENoInterface
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
	*effect = filedrag.DropEffectCopy
	return filedrag.SOK
}

func targetDragOver(this *comTarget, keys, pt uintptr, effect *uint32) uintptr {
	*effect = filedrag.DropEffectCopy
	return filedrag.SOK
}

func targetDragLeave(this *comTarget) uintptr { return filedrag.SOK }

// targetDrop runs the action with the dropped selection and brings the
// window to the front.
func targetDrop(this *comTarget, data unsafe.Pointer, keys, pt uintptr, effect *uint32) uintptr {
	paths := filedrag.DataObjectPaths(data)
	*effect = filedrag.DropEffectCopy
	if len(paths) > 0 {
		this.run(this.id, paths)
	}
	return filedrag.SOK
}
