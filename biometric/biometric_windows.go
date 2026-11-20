package biometric

import (
	"context"
	"errors"
	"time"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/winrt"
)

// Windows Hello: UserConsentVerifier, over the app's window on Windows 11
// (IUserConsentVerifierInterop), on its own on Windows 10. The user
// verifies with a face, a fingerprint or the device's PIN.

const supported = true

var (
	iidConsentStatics = winrt.ParseGUID("af4f3f91-564c-4ddc-b8b5-973447627c65")
	iidConsentInterop = winrt.ParseGUID("39e050c3-4e74-441a-8dc0-b81104df949c")
	// IAsyncOperation<UserConsentVerificationResult>.
	iidVerificationOp = winrt.SignatureGUID("pinterface({9fc2b0bb-e446-44e2-aa61-9cab8f636af2};enum(Windows.Security.Credentials.UI.UserConsentVerificationResult;i4))")
)

const classConsent = "Windows.Security.Credentials.UI.UserConsentVerifier"

func available() Kind {
	var k Kind
	winrt.Do(func() {
		stats, err := winrt.Factory(classConsent, iidConsentStatics)
		if err != nil {
			return
		}
		defer stats.Release()
		op, err := stats.Out(6) // CheckAvailabilityAsync
		if err != nil {
			return
		}
		if v, err := winrt.AwaitInt32(op, 10*time.Second); err == nil && v == 0 { // Available
			k = Biometric
		}
	})
	return k
}

func authenticate(ctx context.Context, a *giode.App, opts Options) error {
	msg := opts.Reason
	if msg == "" {
		msg = opts.Title
	}
	hwnd := a.NativeHandle()
	var result int32
	var err error
	derr := winrt.Do(func() {
		h := winrt.NewHString(msg)
		defer h.Delete()
		var op *winrt.Obj
		if hwnd != 0 {
			if interop, ierr := winrt.Factory(classConsent, iidConsentInterop); ierr == nil {
				var p *winrt.Obj
				if interop.Call(6, hwnd, uintptr(h), uintptr(unsafe.Pointer(&iidVerificationOp)), uintptr(unsafe.Pointer(&p))) == nil {
					op = p
				}
				interop.Release()
			}
		}
		if op == nil {
			stats, serr := winrt.Factory(classConsent, iidConsentStatics)
			if serr != nil {
				err = serr
				return
			}
			op, err = stats.Out(7, uintptr(h)) // RequestVerificationAsync
			stats.Release()
			if err != nil {
				return
			}
		}
		timeout := 5 * time.Minute
		if d, ok := ctx.Deadline(); ok {
			timeout = time.Until(d)
		}
		result, err = winrt.AwaitInt32(op, timeout)
	})
	if derr != nil {
		return derr
	}
	if errors.Is(err, winrt.ErrCanceled) {
		return ErrCanceled
	}
	if err != nil {
		return err
	}
	switch result {
	case 0:
		return nil
	case 1, 3: // DeviceNotPresent, DisabledByPolicy
		return giode.ErrNotSupported
	case 2:
		return ErrNotEnrolled
	case 4: // DeviceBusy
		return ErrFailed
	case 5:
		return ErrLockout
	case 6:
		return ErrCanceled
	}
	return ErrFailed
}
