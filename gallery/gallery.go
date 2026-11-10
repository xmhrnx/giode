// Package gallery saves pictures and videos to the device's photo
// gallery, where photo apps show them: a scanned page exported as JPEG,
// say.
//
//	err := gallery.Save(ctx, app, "/path/to/page.jpg", "Scans")
//
// It is a package of its own so that only apps that save to the photo
// library carry that code: the App Store asks any iOS app whose binary
// uses the photo library for a purpose string. Importing it also makes
// giode.App.SaveToPhotos work.
//
// On Android it saves through MediaStore, Android 10 or later, with no
// permission needed. On iOS it adds to the photo library, asking for
// add-only access the first time: set ios.usage.photos_add in
// giode.toml (NSPhotoLibraryAddUsageDescription). On desktops it copies
// into the user's Pictures folder.
package gallery

import (
	"context"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/optin"
)

func init() {
	optin.SavePhotoLinked = true
}

// Save adds the picture or video at path to the photo gallery, under
// album (a folder name; the app's name when empty; iOS ignores it). The
// type comes from the extension (.jpg, .png, .heic, .mp4, .mov ...). It
// copies the whole file: call it from a goroutine. It is
// a.SaveToPhotos.
func Save(ctx context.Context, a *giode.App, path, album string) error {
	return a.SaveToPhotos(ctx, path, album)
}
