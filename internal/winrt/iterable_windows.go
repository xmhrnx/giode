package winrt

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Iterable is an IIterable<T> of Windows Runtime objects implemented in
// Go, for APIs that take a collection (DataPackage.SetStorageItems). It
// holds a reference to each item and lives as long as the process.
type Iterable struct {
	vtbl     *[7]uintptr
	iid, itv GUID
	items    []*Obj
}

type iterator struct {
	vtbl *[10]uintptr
	it   *Iterable
	pos  int
}

var (
	iidInspectable = ParseGUID("af86e2e0-b12d-4c6a-9c5a-d7aa65101e90")
	iterOnce       sync.Once
	iterableVtbl   *[7]uintptr
	iteratorVtbl   *[10]uintptr
	iterKeep       sync.Map
)

// IIterableIID and IIteratorIID are the IIDs of IIterable<T> and
// IIterator<T> for an interface T.
func IIterableIID(t GUID) GUID {
	return SignatureGUID("pinterface({faa585ea-6214-4217-afda-7f46de5869b3};{" + t.String() + "})")
}

func IIteratorIID(t GUID) GUID {
	return SignatureGUID("pinterface({6a79e863-4300-459a-9966-cbb660963ee1};{" + t.String() + "})")
}

// NewIterable makes an IIterable<T> of items, which implement T.
func NewIterable(t GUID, items []*Obj) *Iterable {
	iterOnce.Do(func() {
		inspectable := func() [6]uintptr {
			return [6]uintptr{
				0, // QueryInterface, set by the caller
				windows.NewCallback(func(this uintptr) uintptr { return 2 }),
				windows.NewCallback(func(this uintptr) uintptr { return 1 }),
				windows.NewCallback(func(this uintptr, n *uint32, iids *uintptr) uintptr {
					*n, *iids = 0, 0
					return sOK
				}),
				windows.NewCallback(func(this uintptr, name *uintptr) uintptr { *name = 0; return sOK }),
				windows.NewCallback(func(this uintptr, level *int32) uintptr { *level = 0; return sOK }),
			}
		}
		base := inspectable()
		iterableVtbl = &[7]uintptr{
			windows.NewCallback(func(this *Iterable, riid *GUID, out **Iterable) uintptr {
				if *riid == iidUnknown || *riid == iidInspectable || *riid == iidAgileObject || *riid == this.iid {
					*out = this
					return sOK
				}
				*out = nil
				return eNoInterface
			}),
			base[1], base[2], base[3], base[4], base[5],
			windows.NewCallback(func(this *Iterable, out **iterator) uintptr { // First
				i := &iterator{vtbl: iteratorVtbl, it: this}
				iterKeep.Store(i, true)
				*out = i
				return sOK
			}),
		}
		iteratorVtbl = &[10]uintptr{
			windows.NewCallback(func(this *iterator, riid *GUID, out **iterator) uintptr {
				if *riid == iidUnknown || *riid == iidInspectable || *riid == iidAgileObject || *riid == this.it.itv {
					*out = this
					return sOK
				}
				*out = nil
				return eNoInterface
			}),
			base[1], base[2], base[3], base[4], base[5],
			windows.NewCallback(func(this *iterator, out **Obj) uintptr { // Current
				if this.pos >= len(this.it.items) {
					*out = nil
					return 0x8000000B // E_BOUNDS
				}
				o := this.it.items[this.pos]
				syscallAddRef(o)
				*out = o
				return sOK
			}),
			windows.NewCallback(func(this *iterator, has *bool) uintptr { // HasCurrent
				*has = this.pos < len(this.it.items)
				return sOK
			}),
			windows.NewCallback(func(this *iterator, has *bool) uintptr { // MoveNext
				if this.pos < len(this.it.items) {
					this.pos++
				}
				*has = this.pos < len(this.it.items)
				return sOK
			}),
			windows.NewCallback(func(this *iterator, capacity uint32, out **Obj, n *uint32) uintptr { // GetMany
				dst := unsafe.Slice(out, capacity)
				k := uint32(0)
				for k < capacity && this.pos < len(this.it.items) {
					o := this.it.items[this.pos]
					syscallAddRef(o)
					dst[k] = o
					k++
					this.pos++
				}
				*n = k
				return sOK
			}),
		}
	})
	it := &Iterable{vtbl: iterableVtbl, iid: IIterableIID(t), itv: IIteratorIID(t), items: items}
	iterKeep.Store(it, true)
	return it
}

// Ptr is the iterable as a call argument.
func (it *Iterable) Ptr() uintptr { return uintptr(unsafe.Pointer(it)) }

func syscallAddRef(o *Obj) {
	if o != nil {
		o.Call(1)
	}
}
