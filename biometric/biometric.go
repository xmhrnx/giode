// Package biometric unlocks an app with the fingerprint reader or face
// recognition, as an app lock does:
//
//	if err := biometric.Authenticate(ctx, app, biometric.Options{Reason: "Unlock your documents"}); err == nil {
//		unlocked = true
//	}
//
// Android uses BiometricPrompt (Android 9 or later); iOS and macOS use
// LocalAuthentication (Face ID, Touch ID or Optic ID). On iOS, Face ID
// needs ios.usage.faceid in giode.toml (NSFaceIDUsageDescription).
// Elsewhere Available reports None and Authenticate returns
// giode.ErrNotSupported.
//
// The check runs in the system's own sheet and tells the app only that
// it passed: it doesn't identify the user, and it doesn't protect data
// by itself. Keep secrets in giode's secure store (App.SecureSet), and
// use this to decide when to read them.
package biometric

import (
	"context"
	"errors"

	"github.com/xmhrnx/giode"
)

// Kind is the kind of biometric a device has.
type Kind int

const (
	// None means no usable biometric: no hardware, or none enrolled.
	None Kind = iota
	// Fingerprint is a fingerprint reader (Touch ID).
	Fingerprint
	// Face is face recognition (Face ID).
	Face
	// Iris is iris recognition (Optic ID).
	Iris
	// Biometric is a biometric whose kind the platform doesn't say
	// (Android).
	Biometric
)

func (k Kind) String() string {
	switch k {
	case Fingerprint:
		return "fingerprint"
	case Face:
		return "face"
	case Iris:
		return "iris"
	case Biometric:
		return "biometric"
	}
	return "none"
}

var (
	// ErrCanceled is returned when the user dismisses the prompt, or
	// the system does (the app went to the background).
	ErrCanceled = errors.New("biometric: canceled")
	// ErrNotEnrolled is returned when the device can read biometrics
	// but none are set up.
	ErrNotEnrolled = errors.New("biometric: none enrolled")
	// ErrLockout is returned after too many failed attempts; the device
	// passcode unlocks it again.
	ErrLockout = errors.New("biometric: locked out after too many attempts")
	// ErrFailed is returned when the check didn't pass.
	ErrFailed = errors.New("biometric: not recognised")
)

// Options configure the prompt.
type Options struct {
	// Reason says why the app asks, shown in the prompt: "Unlock your
	// documents". Required on iOS and macOS.
	Reason string
	// Title heads the prompt on Android; the app's name when empty.
	Title string
	// Cancel labels the button that dismisses the prompt on Android;
	// "Cancel" when empty. Unused with AllowPasscode.
	Cancel string
	// AllowPasscode lets the user enter the device PIN, pattern or
	// password instead, as a fallback.
	AllowPasscode bool
}

// Available reports the device's biometric, None when there's none
// enrolled or the platform has none Giode can use.
func Available() Kind {
	if !supported {
		return None
	}
	return available()
}

// Authenticate shows the system's biometric prompt and returns nil once
// the user passes it. It returns ErrCanceled when they dismiss it, and
// ErrNotEnrolled or ErrLockout when it can't be used. It blocks until
// the prompt closes: call it from a goroutine.
func Authenticate(ctx context.Context, a *giode.App, opts Options) error {
	if !supported {
		return giode.ErrNotSupported
	}
	if opts.Title == "" {
		opts.Title = giode.AppName
		if opts.Title == "" {
			opts.Title = a.Title()
		}
	}
	if opts.Cancel == "" {
		opts.Cancel = "Cancel"
	}
	return authenticate(ctx, a, opts)
}

// result codes from the platforms.
const (
	resOK = iota
	resCanceled
	resNotEnrolled
	resLockout
	resFailed
	resUnavailable
)

func resultError(code int, msg string) error {
	switch code {
	case resOK:
		return nil
	case resCanceled:
		return ErrCanceled
	case resNotEnrolled:
		return ErrNotEnrolled
	case resLockout:
		return ErrLockout
	case resUnavailable:
		if msg != "" {
			return errors.Join(giode.ErrNotSupported, errors.New("biometric: "+msg))
		}
		return giode.ErrNotSupported
	}
	if msg != "" {
		return errors.Join(ErrFailed, errors.New("biometric: "+msg))
	}
	return ErrFailed
}
