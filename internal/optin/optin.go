// Package optin connects the core package to the opt-in packages that
// hold the platform code of features touching privacy-sensitive APIs:
// dictate, takephoto and gallery. The App Store asks any iOS app whose
// binary uses the microphone, speech recognition, the camera or the
// photo library for purpose strings, so only apps that import those
// packages carry the code. Each package sets its Linked flag (and on
// iOS its function) in init; the core methods return an error naming
// the package otherwise.
package optin

import "context"

// DictateLinked is set by package dictate.
var DictateLinked bool

// Dictate, set by package dictate on iOS, starts dictating in lang (a
// BCP 47 tag, or "" for the device's language). emit receives each
// result; the final one comes last, once. stop ends listening early;
// the final result still follows.
var Dictate func(lang string, emit func(text string, final bool, err error)) (stop func(), err error)

// TakePhotoLinked is set by package takephoto.
var TakePhotoLinked bool

// TakePhoto, set by package takephoto on iOS, shows the camera over the
// view controller vc and returns the photo as an upright JPEG scaled to
// maxSide (0 for full size).
var TakePhoto func(ctx context.Context, vc uintptr, maxSide int) ([]byte, error)

// SavePhotoLinked is set by package gallery.
var SavePhotoLinked bool

// SavePhoto, set by package gallery on iOS, adds the picture or video at
// path to the photo library.
var SavePhoto func(ctx context.Context, path string, video bool) error
