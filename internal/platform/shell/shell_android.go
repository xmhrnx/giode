package shell

import (
	"context"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
)

// OpenURL sends an ACTION_VIEW intent.
func OpenURL(ctx context.Context, h platform.Host, u string) error {
	return platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		return mobile.OpenURL(app.JavaVM(), app.AppContext(), view, u)
	})
}

// OpenWith sends an ACTION_VIEW intent for the file.
func OpenWith(ctx context.Context, h platform.Host, path, mime string) error {
	// Resolving the URI may copy the file: off the main thread.
	uri, err := mobile.FileURI(app.JavaVM(), app.AppContext(), path, mime)
	if err != nil {
		return err
	}
	return platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		return mobile.ViewURI(app.JavaVM(), app.AppContext(), view, uri, mime)
	})
}

// OpenFolder: Android has no folders to open; OpenDownloads shows the
// Downloads app.
func OpenFolder(ctx context.Context, h platform.Host, path string) error {
	return platform.ErrNotSupported
}

// SaveToDownloads copies the file into Download/<folder> through
// MediaStore.
func SaveToDownloads(ctx context.Context, path, mime, folder string) (string, error) {
	return mobile.SaveToDownloads(app.JavaVM(), app.AppContext(), path, mime, folder)
}

// OpenDownloads opens the Downloads app.
func OpenDownloads(ctx context.Context, h platform.Host, folder string) error {
	return platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		return mobile.OpenDownloads(app.JavaVM(), app.AppContext(), view)
	})
}

// ShareText opens the system share sheet.
func ShareText(ctx context.Context, h platform.Host, text, subject, title string) error {
	return platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		return mobile.ShareText(app.JavaVM(), app.AppContext(), view, text, subject, title)
	})
}

// ShareFiles publishes the files through the app's provider and opens
// the share sheet.
func ShareFiles(ctx context.Context, h platform.Host, files []File, mime, title string) error {
	if len(files) == 0 {
		return nil
	}
	// The copies run here, off the main thread; only the share sheet
	// needs it.
	uris := make([]string, len(files))
	for i, f := range files {
		uri, err := mobile.Publish(app.JavaVM(), app.AppContext(), f.Path, f.MIME)
		if err != nil {
			return err
		}
		uris[i] = uri
	}
	if len(files) == 1 {
		return platform.OnAndroidMain(ctx, h, func(view uintptr) error {
			return mobile.ShareURI(app.JavaVM(), app.AppContext(), view, uris[0], files[0].MIME, title)
		})
	}
	return platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		return mobile.ShareURIs(app.JavaVM(), app.AppContext(), view, uris, mime, title)
	})
}
