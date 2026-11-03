package giode

import (
	"context"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

func (t ShareText) share(ctx context.Context, a *App) error {
	return a.onAndroidMain(ctx, func(view uintptr) error {
		return mobile.ShareText(app.JavaVM(), app.AppContext(), view, t.Text, t.Subject, t.Title)
	})
}

func (f ShareFile) share(ctx context.Context, a *App) error {
	mime := f.mimeType()
	// The copy runs here, off the main thread; only the share sheet
	// needs it.
	uri, err := mobile.Publish(app.JavaVM(), app.AppContext(), f.Path, mime)
	if err != nil {
		return err
	}
	return a.onAndroidMain(ctx, func(view uintptr) error {
		return mobile.ShareURI(app.JavaVM(), app.AppContext(), view, uri, mime, f.Title)
	})
}

func (f ShareFiles) share(ctx context.Context, a *App) error {
	if len(f.Paths) == 0 {
		return nil
	}
	uris := make([]string, len(f.Paths))
	for i, p := range f.Paths {
		uri, err := mobile.Publish(app.JavaVM(), app.AppContext(), p, ShareFile{Path: p}.mimeType())
		if err != nil {
			return err
		}
		uris[i] = uri
	}
	mime := f.mimeType()
	return a.onAndroidMain(ctx, func(view uintptr) error {
		return mobile.ShareURIs(app.JavaVM(), app.AppContext(), view, uris, mime, f.Title)
	})
}

func (a *App) saveToPhotosIn(ctx context.Context, path, mime, album string) error {
	return mobile.SaveToGallery(app.JavaVM(), app.AppContext(), path, mime, album)
}

func (a *App) saveToDownloads(ctx context.Context, path, mime, folder string) (string, error) {
	return mobile.SaveToDownloads(app.JavaVM(), app.AppContext(), path, mime, folder)
}

func (a *App) openDownloads(ctx context.Context, folder string) error {
	return a.onAndroidMain(ctx, func(view uintptr) error {
		return mobile.OpenDownloads(app.JavaVM(), app.AppContext(), view)
	})
}

func (a *App) openFolder(ctx context.Context, path string) error { return ErrNotSupported }

func (a *App) openURL(ctx context.Context, u string) error {
	return a.onAndroidMain(ctx, func(view uintptr) error {
		return mobile.OpenURL(app.JavaVM(), app.AppContext(), view, u)
	})
}

// onAndroidMain runs fn with the GioView on the Android main thread
// and waits for it.
func (a *App) onAndroidMain(ctx context.Context, fn func(view uintptr) error) error {
	view := a.NativeHandle()
	if view == 0 {
		return ErrNotSupported
	}
	_, err := await(ctx, func() (struct{}, error) {
		var err error
		a.window.Run(func() { err = fn(view) })
		return struct{}{}, err
	}, func(struct{}) {})
	return err
}

// watchLifecycle reports activity stops and starts to OnBackground and
// OnForeground.
func (a *App) watchLifecycle() {
	if a.cfg.OnBackground == nil && a.cfg.OnForeground == nil {
		return
	}
	mobile.SetLifecycleHandlers(a.cfg.OnBackground, a.cfg.OnForeground)
	if err := mobile.WatchLifecycle(app.JavaVM(), app.AppContext()); err != nil {
		a.logf(LogLevelWarning, "giode: can't watch the app lifecycle: %v", err)
	}
}

// listenShares hands what other apps share with this one (through
// ShareActivity, which app.share_target declares) to OnShareReceived.
func (a *App) listenShares() {
	err := mobile.ListenShares(app.JavaVM(), app.AppContext(), func(s mobile.Shared) {
		c := SharedContent{Text: s.Text, Subject: s.Subject}
		for _, f := range s.Files {
			c.Files = append(c.Files, sharedFile(f.File, f.Name, f.MIME, f.Size))
		}
		a.shareReceived(c)
	})
	if err != nil {
		a.logf(LogLevelWarning, "giode: can't receive shares: %v", err)
	}
}

// Android reports going to the background through the activity
// lifecycle, not the window state.
const lifecycleFromWindowState = false
