package winrt

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	combase                    = windows.NewLazySystemDLL("combase.dll")
	procRoInitialize           = combase.NewProc("RoInitialize")
	procRoGetActivationFactory = combase.NewProc("RoGetActivationFactory")
	procRoActivateInstance     = combase.NewProc("RoActivateInstance")
	procWindowsCreateString    = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString    = combase.NewProc("WindowsDeleteString")
	procWindowsGetStringRaw    = combase.NewProc("WindowsGetStringRawBuffer")
	ole32                      = windows.NewLazySystemDLL("ole32.dll")
	procCoRegisterClassObject  = ole32.NewProc("CoRegisterClassObject")
	procCoCreateInstance       = ole32.NewProc("CoCreateInstance")
)

// HRESULT is a failed COM call's result.
type HRESULT uint32

func (h HRESULT) Error() string { return fmt.Sprintf("winrt: HRESULT 0x%08x", uint32(h)) }

func check(r uintptr) error {
	if int32(r) < 0 {
		return HRESULT(r)
	}
	return nil
}

// Obj is a COM interface pointer.
type Obj struct{ vtbl *[64]uintptr }

// Call calls the method in vtable slot m and returns its HRESULT as an
// error.
func (o *Obj) Call(m int, args ...uintptr) error {
	r, _, _ := syscall.SyscallN(o.vtbl[m], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return check(r)
}

// Release releases o; nil is fine.
func (o *Obj) Release() {
	if o != nil {
		syscall.SyscallN(o.vtbl[2], uintptr(unsafe.Pointer(o)))
	}
}

// Query asks o for another interface.
func (o *Obj) Query(iid GUID) (*Obj, error) {
	var out *Obj
	if err := o.Call(0, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

// Out calls method m with the arguments followed by a pointer to an
// interface pointer, and returns that.
func (o *Obj) Out(m int, args ...uintptr) (*Obj, error) {
	var out *Obj
	if err := o.Call(m, append(args, uintptr(unsafe.Pointer(&out)))...); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, errors.New("winrt: no object")
	}
	return out, nil
}

// HString is a Windows Runtime string.
type HString uintptr

// NewHString makes an HString; free it with Delete.
func NewHString(s string) HString {
	if s == "" {
		return 0
	}
	u, _ := windows.UTF16FromString(s)
	var h HString
	procWindowsCreateString.Call(uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&h)))
	return h
}

// Delete frees h.
func (h HString) Delete() {
	if h != 0 {
		procWindowsDeleteString.Call(uintptr(h))
	}
}

// String returns h's text.
func (h HString) String() string {
	if h == 0 {
		return ""
	}
	var n uint32
	r, _, _ := procWindowsGetStringRaw.Call(uintptr(h), uintptr(unsafe.Pointer(&n)))
	p := *(**uint16)(unsafe.Pointer(&r))
	if p == nil {
		return ""
	}
	return windows.UTF16ToString(unsafe.Slice(p, n))
}

// Factory returns a runtime class's activation factory, as iid.
func Factory(class string, iid GUID) (*Obj, error) {
	h := NewHString(class)
	defer h.Delete()
	var out *Obj
	r, _, _ := procRoGetActivationFactory.Call(uintptr(h), uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&out)))
	if err := check(r); err != nil {
		return nil, fmt.Errorf("%s: %w", class, err)
	}
	return out, nil
}

// Activate makes an instance of a runtime class with a default
// constructor.
func Activate(class string) (*Obj, error) {
	h := NewHString(class)
	defer h.Delete()
	var out *Obj
	r, _, _ := procRoActivateInstance.Call(uintptr(h), uintptr(unsafe.Pointer(&out)))
	if err := check(r); err != nil {
		return nil, fmt.Errorf("%s: %w", class, err)
	}
	return out, nil
}

// CreateInstance makes a classic COM object.
func CreateInstance(clsid, iid GUID) (*Obj, error) {
	var out *Obj
	r, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsid)), 0, 1|4, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&out)))
	if err := check(r); err != nil {
		return nil, err
	}
	return out, nil
}

var iidAsyncInfo = ParseGUID("00000036-0000-0000-c000-000000000046")

// Await waits for an IAsyncOperation to finish, polling its status, and
// returns its result (GetResults, slot 8). It releases op.
func Await(op *Obj, timeout time.Duration) (*Obj, error) {
	defer op.Release()
	if err := wait(op, timeout); err != nil {
		return nil, err
	}
	return op.Out(8)
}

// AwaitInt32 waits for an IAsyncOperation of an enum or 32-bit number
// and returns its result. It releases op.
func AwaitInt32(op *Obj, timeout time.Duration) (int32, error) {
	defer op.Release()
	if err := wait(op, timeout); err != nil {
		return 0, err
	}
	var v int32
	err := op.Call(8, uintptr(unsafe.Pointer(&v)))
	return v, err
}

// wait polls op's status until it completes.
func wait(op *Obj, timeout time.Duration) error {
	info, err := op.Query(iidAsyncInfo)
	if err != nil {
		return err
	}
	defer info.Release()
	deadline := time.Now().Add(timeout)
	for {
		var status int32
		if err := info.Call(7, uintptr(unsafe.Pointer(&status))); err != nil {
			return err
		}
		switch status {
		case 1: // Completed
			return nil
		case 2:
			return ErrCanceled
		case 3:
			var code uint32
			info.Call(8, uintptr(unsafe.Pointer(&code)))
			return HRESULT(code)
		}
		if time.Now().After(deadline) {
			info.Call(9) // Cancel
			return errors.New("winrt: operation timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// ErrCanceled is returned for a canceled operation.
var ErrCanceled = errors.New("winrt: operation canceled")

var (
	threadOnce sync.Once
	calls      chan func()
	threadErr  error
)

// Do runs fn on giode's Windows Runtime thread, which stays in the
// multithreaded apartment for the process's life.
func Do(fn func()) error {
	threadOnce.Do(func() {
		calls = make(chan func())
		ready := make(chan error)
		go func() {
			runtime.LockOSThread()
			r, _, _ := procRoInitialize.Call(1) // RO_INIT_MULTITHREADED
			// S_FALSE and RPC_E_CHANGED_MODE leave the thread usable.
			if err := check(r); err != nil && r != 0x80010106 {
				ready <- err
				return
			}
			ready <- nil
			for f := range calls {
				f()
			}
		}()
		threadErr = <-ready
	})
	if threadErr != nil {
		return threadErr
	}
	done := make(chan struct{})
	calls <- func() {
		defer close(done)
		fn()
	}
	<-done
	return nil
}
