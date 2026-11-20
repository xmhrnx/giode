package winrt

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Delegate is a Windows Runtime event handler implemented in Go: a COM
// object whose Invoke calls a Go function with the sender and the event
// arguments. It lives as long as the process.
type Delegate struct {
	vtbl   *[4]uintptr
	iid    GUID
	invoke func(sender, args *Obj)
}

var (
	iidAgileObject = ParseGUID("94ea2b94-e9cc-49e0-c0ff-ee64ca8f5b90")
	delegateOnce   sync.Once
	delegateVtbl   *[4]uintptr
	delegatesMu    sync.Mutex
	delegates      []*Delegate
)

// NewDelegate makes a handler for the delegate interface iid, such as a
// TypedEventHandlerIID.
func NewDelegate(iid GUID, invoke func(sender, args *Obj)) *Delegate {
	delegateOnce.Do(func() {
		delegateVtbl = &[4]uintptr{
			windows.NewCallback(func(this *Delegate, riid *GUID, out **Delegate) uintptr {
				if *riid == iidUnknown || *riid == iidAgileObject || *riid == this.iid {
					*out = this
					return sOK
				}
				*out = nil
				return eNoInterface
			}),
			windows.NewCallback(func(this *Delegate) uintptr { return 2 }),
			windows.NewCallback(func(this *Delegate) uintptr { return 1 }),
			windows.NewCallback(func(this *Delegate, sender, args *Obj) uintptr {
				this.invoke(sender, args)
				return sOK
			}),
		}
	})
	d := &Delegate{vtbl: delegateVtbl, iid: iid, invoke: invoke}
	delegatesMu.Lock()
	delegates = append(delegates, d) // kept for COM
	delegatesMu.Unlock()
	return d
}

// Ptr is the delegate as a call argument.
func (d *Delegate) Ptr() uintptr { return uintptr(unsafe.Pointer(d)) }
