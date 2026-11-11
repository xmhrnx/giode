package giode

import "context"

// MediaOptions configures PickMedia.
type MediaOptions struct {
	// Limit caps how many items the user can pick; 0 allows as many as
	// the picker does.
	Limit int
	// Photos and Videos choose what's offered; with neither set, both
	// are.
	Photos, Videos bool
}

// PickMedia lets the user pick photos and videos, several at once, and
// returns the original files - full size, in their own format (HEIC,
// MOV, ...), not converted - in the order picked. On iOS it's the Photos
// picker (PHPicker, iOS 14 and later), on Android the system Photo
// Picker (Android 13, or 11 and 12 with the updated media module): both
// need no permission, and they're where most of what a phone user sends
// lives. Elsewhere, and on Android without the Photo Picker, it's the
// file picker for images and videos. On iOS and desktops each
// file's Path is set (a temporary copy on iOS). It blocks until the
// user decides: call it from a goroutine. It returns ErrCanceled when
// the user picks nothing.
//
//	files, err := app.PickMedia(ctx, giode.MediaOptions{})
func (a *App) PickMedia(ctx context.Context, opts MediaOptions) ([]PickedFile, error) {
	if !opts.Photos && !opts.Videos {
		opts.Photos, opts.Videos = true, true
	}
	return a.pickMedia(ctx, opts)
}

// pickMediaFiles is PickMedia through the file picker.
func (a *App) pickMediaFiles(ctx context.Context, opts MediaOptions) ([]PickedFile, error) {
	var types []string
	if opts.Photos {
		types = append(types, "image/*")
	}
	if opts.Videos {
		types = append(types, "video/*")
	}
	files, err := a.PickFiles(ctx, PickOptions{Types: types, Multiple: opts.Limit != 1})
	if err != nil {
		return nil, err
	}
	if opts.Limit > 0 && len(files) > opts.Limit {
		for _, f := range files[opts.Limit:] {
			f.Close()
		}
		files = files[:opts.Limit]
	}
	return files, nil
}
