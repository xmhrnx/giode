package winrt

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The toast activator: a COM class implementing
// INotificationActivationCallback, which Windows calls when the user
// clicks a toast or one of its buttons, starting the app first (with
// -Embedding) when it isn't running. Its class is the app's CLSID, named
// in the registry (unpackaged apps) or the MSIX manifest.

var (
	iidUnknown              = ParseGUID("00000000-0000-0000-c000-000000000046")
	iidClassFactory         = ParseGUID("00000001-0000-0000-c000-000000000046")
	iidActivationCallback   = ParseGUID("53e31837-6600-4a81-9395-75cffe746f94")
	activatorMu             sync.Mutex
	activationHandlers      []func(args string, inputs map[string]string) bool
	activatorServed         bool
	activatorFactory        *comClass
	activatorCallback       *comClass
	factoryVtbl, activeVtbl *[5]uintptr
)

const (
	sOK          = 0
	eNoInterface = 0x80004002
)

type comClass struct{ vtbl *[5]uintptr }

type userInput struct {
	Key, Value *uint16
}

// OnActivate adds a handler for toast activations; the first that
// reports true takes one. Handlers run on a COM thread.
func OnActivate(f func(args string, inputs map[string]string) bool) {
	activatorMu.Lock()
	activationHandlers = append(activationHandlers, f)
	activatorMu.Unlock()
}

// ServeActivator registers the app's activator class with COM, once, so
// Windows can hand this process its toasts' activations.
func ServeActivator(clsid GUID) error {
	activatorMu.Lock()
	defer activatorMu.Unlock()
	if activatorServed {
		return nil
	}
	factoryVtbl = &[5]uintptr{
		windows.NewCallback(func(this *comClass, riid *GUID, out **comClass) uintptr {
			if *riid == iidUnknown || *riid == iidClassFactory {
				*out = this
				return sOK
			}
			*out = nil
			return eNoInterface
		}),
		windows.NewCallback(func(this *comClass) uintptr { return 2 }),
		windows.NewCallback(func(this *comClass) uintptr { return 1 }),
		windows.NewCallback(func(this *comClass, outer uintptr, riid *GUID, out **comClass) uintptr {
			*out = nil
			if outer != 0 {
				return 0x80040110 // CLASS_E_NOAGGREGATION
			}
			if *riid != iidUnknown && *riid != iidActivationCallback {
				return eNoInterface
			}
			*out = activatorCallback
			return sOK
		}),
		windows.NewCallback(func(this *comClass, lock uintptr) uintptr { return sOK }),
	}
	activeVtbl = &[5]uintptr{
		windows.NewCallback(func(this *comClass, riid *GUID, out **comClass) uintptr {
			if *riid == iidUnknown || *riid == iidActivationCallback {
				*out = this
				return sOK
			}
			*out = nil
			return eNoInterface
		}),
		windows.NewCallback(func(this *comClass) uintptr { return 2 }),
		windows.NewCallback(func(this *comClass) uintptr { return 1 }),
		windows.NewCallback(func(this *comClass, aumid, args *uint16, data *userInput, count uintptr) uintptr {
			inputs := map[string]string{}
			if data != nil && count > 0 {
				for _, in := range unsafe.Slice(data, int(uint32(count))) {
					inputs[windows.UTF16PtrToString(in.Key)] = windows.UTF16PtrToString(in.Value)
				}
			}
			a := windows.UTF16PtrToString(args)
			activatorMu.Lock()
			hs := append([]func(string, map[string]string) bool(nil), activationHandlers...)
			activatorMu.Unlock()
			for _, h := range hs {
				if h(a, inputs) {
					break
				}
			}
			return sOK
		}),
		0,
	}
	activatorFactory = &comClass{vtbl: factoryVtbl}
	activatorCallback = &comClass{vtbl: activeVtbl}
	var err error
	Do(func() {
		var cookie uint32
		r, _, _ := procCoRegisterClassObject.Call(uintptr(unsafe.Pointer(&clsid)), uintptr(unsafe.Pointer(activatorFactory)),
			4 /* CLSCTX_LOCAL_SERVER */, 1 /* REGCLS_MULTIPLEUSE */, uintptr(unsafe.Pointer(&cookie)))
		err = check(r)
	})
	if err == nil {
		activatorServed = true
	}
	return err
}
