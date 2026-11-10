// Package takephoto lets the user take a photo with the system camera
// screen, for an "attach a photo" button:
//
//	photo, err := takephoto.Take(ctx, app, 2048)
//
// It is a package of its own so that only apps that take photos carry
// the camera code: the App Store asks any iOS app whose binary uses the
// camera for a camera purpose string. Importing it also makes
// giode.App.PickPhoto's Camera source work.
//
// Android (the system camera app) and iOS (UIImagePickerController; set
// ios.usage.camera in giode.toml, the Info.plist's
// NSCameraUsageDescription) are supported; elsewhere Take returns
// giode.ErrNotSupported. For a live preview inside the app, see package
// camera.
package takephoto

import (
	"context"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/optin"
)

func init() {
	optin.TakePhotoLinked = true
}

// Take shows the camera and returns the photo the user takes as a
// JPEG, upright and scaled so its longer side is at most maxSide pixels
// (0 keeps its size). It blocks until the user decides: call it from a
// goroutine. It returns giode.ErrCanceled when the user backs out. It is
// a.PickPhoto with the Camera source.
func Take(ctx context.Context, a *giode.App, maxSide int) (giode.PickedFile, error) {
	return a.PickPhoto(ctx, giode.PhotoOptions{Source: giode.Camera, MaxSide: maxSide})
}
