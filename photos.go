package giode

import (
	"context"
	"os"

	"github.com/xmhrnx/giode/internal/optin"
)

// PickPhotos lets the user pick several photos at once, up to limit (0
// for as many as the picker allows), and returns them in the order
// picked, each as an upright JPEG scaled to opts.MaxSide. On Android it
// uses the system photo picker (Android 13+, or the image picker
// before that); on iOS PHPicker (iOS 14+); on desktops an image file
// picker. opts.Source must be Library. It blocks until the user decides:
// call it from a goroutine. It returns ErrCanceled when the user picks
// none.
func (a *App) PickPhotos(ctx context.Context, opts PhotoOptions, limit int) ([]PickedFile, error) {
	if opts.Source == Camera {
		return nil, ErrNotSupported
	}
	return a.pickPhotos(ctx, opts, limit)
}

// SaveToPhotos adds the picture or video at path to the device's photo
// gallery, where photo apps show it: a scanned page exported as JPEG,
// say. The type comes from the extension (.jpg, .png, .heic, .mp4,
// .mov ...).
//
// On Android it goes to Pictures/<album> (Movies for videos) through
// MediaStore, Android 10 or later, with no permission needed. On iOS
// it's added to the photo library, asking for add-only access the first
// time: set ios.usage.photos_add in giode.toml
// (NSPhotoLibraryAddUsageDescription). On desktops it's copied into the
// user's Pictures folder, under album. album is a folder name, the app's
// name when empty; iOS ignores it.
//
// It needs the github.com/xmhrnx/giode/gallery package imported, so that
// only apps that save to the photo library carry that code; without it
// it returns an error wrapping ErrNotSupported.
//
// It copies the whole file: call it from a goroutine.
func (a *App) SaveToPhotos(ctx context.Context, path, album string) error {
	if !optin.SavePhotoLinked {
		return notLinked("gallery")
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	if album == "" {
		album = AppName
		if album == "" {
			album = a.Title()
		}
	}
	return a.saveToPhotosIn(ctx, path, ShareFile{Path: path}.mimeType(), album)
}
