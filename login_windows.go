//go:build windows

package giode

import (
	"errors"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Login registration on Windows. An unpackaged app writes a value
// under HKCU\...\Run; Task Manager's Startup apps page can turn it off,
// which it records under Explorer\StartupApproved\Run. An MSIX package
// can't use the Run key: it declares a startup task in its manifest
// (the giode CLI adds one with windows.msix.startup_task) and turns it on
// and off through the WinRT StartupTask class.

const (
	runKey             = `Software\Microsoft\Windows\CurrentVersion\Run`
	startupApprovedKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`

	// startupTaskID is the TaskId the CLI gives the manifest's
	// startup task.
	startupTaskID = "GiodeLaunchAtLogin"
)

var (
	kernel32Login                  = windows.NewLazySystemDLL("kernel32.dll")
	procGetCurrentPackageFullName  = kernel32Login.NewProc("GetCurrentPackageFullName")
	combase                        = windows.NewLazySystemDLL("combase.dll")
	procRoInitialize               = combase.NewProc("RoInitialize")
	procRoUninitialize             = combase.NewProc("RoUninitialize")
	procRoGetActivationFactory     = combase.NewProc("RoGetActivationFactory")
	procWindowsCreateString        = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString        = combase.NewProc("WindowsDeleteString")
	iidIStartupTaskStatics         = windows.GUID{Data1: 0xEE5B60BD, Data2: 0xA148, Data3: 0x41A7, Data4: [8]byte{0xB2, 0x6E, 0xE8, 0xB8, 0x8A, 0x1E, 0x62, 0xF8}}
	iidIAppInstanceStatics         = windows.GUID{Data1: 0x9D11E77F, Data2: 0x9EA6, Data3: 0x47AF, Data4: [8]byte{0xA6, 0xEC, 0x46, 0x78, 0x4C, 0x5B, 0xA2, 0x54}}
	iidIAsyncInfo                  = windows.GUID{Data1: 0x00000036, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	errStartupTaskMissing          = errors.New("giode: the package declares no startup task: set windows.msix.startup_task in giode.toml")
	errStartupTaskDisabledByUser   = errors.New("giode: the user turned off starting at login in Settings; only they can turn it back on")
	errStartupTaskDisabledByPolicy = errors.New("giode: starting at login is turned off by policy")
	errStartupTaskTimeout          = errors.New("giode: the startup task didn't answer")
	startupTaskOnce                sync.Once
	startedByStartupTask           bool
)

// WinRT vtable slots, after IInspectable's six methods.
const (
	vtblQueryInterface         = 0
	vtblGetForCurrentPackage   = 6 // IStartupTaskStatics
	vtblGetAsync               = 7 // IStartupTaskStatics
	vtblRequestEnableAsync     = 6 // IStartupTask
	vtblDisable                = 7 // IStartupTask
	vtblGetState               = 8 // IStartupTask
	vtblGetResults             = 8 // IAsyncOperation<T>
	vtblAsyncStatus            = 7 // IAsyncInfo
	vtblGetActivatedEventArgs  = 7 // IAppInstanceStatics
	vtblGetKind                = 6 // IActivatedEventArgs
	activationKindStartupTask  = 1020
	appModelErrorNoPackage     = 15700
	errorInsufficientBuffer    = 122
	roInitMultithreaded        = 1
	asyncStarted               = 0
	asyncCompleted             = 1
	startupTaskDisabled        = 0
	startupTaskDisabledByUser  = 1
	startupTaskEnabled         = 2
	startupTaskDisabledPolicy  = 3
	startupTaskEnabledByPolicy = 4
)

// packaged reports whether the app runs from an MSIX package.
func packaged() bool {
	var n uint32
	r, _, _ := procGetCurrentPackageFullName.Call(uintptr(unsafe.Pointer(&n)), 0)
	return r == errorInsufficientBuffer
}

func setLaunchAtLogin(name string, on bool) error {
	if packaged() {
		return setStartupTask(on)
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		if err := k.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	cmd, err := loginCommand()
	if err != nil {
		return err
	}
	line := syscall.EscapeArg(cmd[0])
	for _, a := range cmd[1:] {
		line += " " + syscall.EscapeArg(a)
	}
	if err := k.SetStringValue(name, line); err != nil {
		return err
	}
	// Turning it on in the app undoes the user turning it off in Task
	// Manager, which would otherwise keep the entry from running.
	if ak, err := registry.OpenKey(registry.CURRENT_USER, startupApprovedKey, registry.SET_VALUE); err == nil {
		ak.DeleteValue(name)
		ak.Close()
	}
	return nil
}

func launchAtLogin(name string) bool {
	if packaged() {
		state, err := startupTaskState()
		return err == nil && (state == startupTaskEnabled || state == startupTaskEnabledByPolicy)
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	if _, _, err := k.GetStringValue(name); err != nil {
		return false
	}
	// Task Manager stores an odd first byte (3) when the user turned
	// the entry off, an even one (2) when on.
	if ak, err := registry.OpenKey(registry.CURRENT_USER, startupApprovedKey, registry.QUERY_VALUE); err == nil {
		defer ak.Close()
		if v, _, err := ak.GetBinaryValue(name); err == nil && len(v) > 0 && v[0]&1 == 1 {
			return false
		}
	}
	return true
}

// launchedByStartupTask reports whether the package's startup task
// started the app, read once from its activation arguments.
func launchedByStartupTask() bool {
	startupTaskOnce.Do(func() {
		if !packaged() {
			return
		}
		withWinRT(func() error {
			statics, err := activationFactory("Windows.ApplicationModel.AppInstance", &iidIAppInstanceStatics)
			if err != nil {
				return err
			}
			defer comCall(statics, vtblRelease)
			var args unsafe.Pointer
			if hr := comCall(statics, vtblGetActivatedEventArgs, uintptr(unsafe.Pointer(&args))); hr != 0 || args == nil {
				return windows.Errno(hr)
			}
			defer comCall(args, vtblRelease)
			var kind int32
			if comCall(args, vtblGetKind, uintptr(unsafe.Pointer(&kind))) == 0 {
				startedByStartupTask = kind == activationKindStartupTask
			}
			return nil
		})
	})
	return startedByStartupTask
}

func setStartupTask(on bool) error {
	return withWinRT(func() error {
		task, err := startupTask()
		if err != nil {
			return err
		}
		defer comCall(task, vtblRelease)
		if !on {
			if hr := comCall(task, vtblDisable); hr != 0 {
				return windows.Errno(hr)
			}
			return nil
		}
		var op unsafe.Pointer
		if hr := comCall(task, vtblRequestEnableAsync, uintptr(unsafe.Pointer(&op))); hr != 0 {
			return windows.Errno(hr)
		}
		defer comCall(op, vtblRelease)
		var state int32
		if err := awaitResult(op, uintptr(unsafe.Pointer(&state))); err != nil {
			return err
		}
		switch state {
		case startupTaskEnabled, startupTaskEnabledByPolicy:
			return nil
		case startupTaskDisabledByUser:
			return errStartupTaskDisabledByUser
		case startupTaskDisabledPolicy:
			return errStartupTaskDisabledByPolicy
		}
		return errors.New("giode: the startup task stayed off")
	})
}

func startupTaskState() (int32, error) {
	var state int32
	err := withWinRT(func() error {
		task, err := startupTask()
		if err != nil {
			return err
		}
		defer comCall(task, vtblRelease)
		if hr := comCall(task, vtblGetState, uintptr(unsafe.Pointer(&state))); hr != 0 {
			return windows.Errno(hr)
		}
		return nil
	})
	return state, err
}

// startupTask returns the package's IStartupTask, which the caller
// releases.
func startupTask() (unsafe.Pointer, error) {
	statics, err := activationFactory("Windows.ApplicationModel.StartupTask", &iidIStartupTaskStatics)
	if err != nil {
		return nil, err
	}
	defer comCall(statics, vtblRelease)
	id, err := hstring(startupTaskID)
	if err != nil {
		return nil, err
	}
	defer procWindowsDeleteString.Call(id)
	var op unsafe.Pointer
	if hr := comCall(statics, vtblGetAsync, id, uintptr(unsafe.Pointer(&op))); hr != 0 {
		return nil, windows.Errno(hr)
	}
	defer comCall(op, vtblRelease)
	var task unsafe.Pointer
	if err := awaitResult(op, uintptr(unsafe.Pointer(&task))); err != nil {
		// GetAsync fails when the manifest has no task by that ID.
		return nil, errStartupTaskMissing
	}
	if task == nil {
		return nil, errStartupTaskMissing
	}
	return task, nil
}

// withWinRT runs f on a thread with the Windows Runtime initialized.
func withWinRT(f func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if r, _, _ := procRoInitialize.Call(roInitMultithreaded); int32(r) >= 0 {
		defer procRoUninitialize.Call()
	}
	return f()
}

func activationFactory(class string, iid *windows.GUID) (unsafe.Pointer, error) {
	h, err := hstring(class)
	if err != nil {
		return nil, err
	}
	defer procWindowsDeleteString.Call(h)
	var f unsafe.Pointer
	if hr, _, _ := procRoGetActivationFactory.Call(h, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&f))); hr != 0 {
		return nil, windows.Errno(hr)
	}
	return f, nil
}

func hstring(s string) (uintptr, error) {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return 0, err
	}
	var h uintptr
	if hr, _, _ := procWindowsCreateString.Call(uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&h))); hr != 0 {
		return 0, windows.Errno(hr)
	}
	return h, nil
}

// awaitResult waits for a WinRT async operation and stores its result
// at out. Polling keeps it free of a completion handler's COM object;
// these operations finish in milliseconds.
func awaitResult(op unsafe.Pointer, out uintptr) error {
	var info unsafe.Pointer
	if hr := comCall(op, vtblQueryInterface, uintptr(unsafe.Pointer(&iidIAsyncInfo)), uintptr(unsafe.Pointer(&info))); hr != 0 {
		return windows.Errno(hr)
	}
	defer comCall(info, vtblRelease)
	deadline := time.Now().Add(30 * time.Second)
	for {
		var status int32
		if hr := comCall(info, vtblAsyncStatus, uintptr(unsafe.Pointer(&status))); hr != 0 {
			return windows.Errno(hr)
		}
		if status != asyncStarted {
			if status != asyncCompleted {
				if hr := comCall(op, vtblGetResults, out); hr != 0 {
					return windows.Errno(hr)
				}
				return errors.New("giode: the operation was canceled or failed")
			}
			break
		}
		if time.Now().After(deadline) {
			return errStartupTaskTimeout
		}
		time.Sleep(5 * time.Millisecond)
	}
	if hr := comCall(op, vtblGetResults, out); hr != 0 {
		return windows.Errno(hr)
	}
	return nil
}
