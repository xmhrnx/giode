package giode

import (
	"context"
	"os"
	"path/filepath"

	"github.com/xmhrnx/giode/internal/platform/shell"
)

// OpenWith opens the file at path in another app, the one the system
// picks for its type, as tapping or double-clicking it in a file
// manager would - rather than through Share:
//
//   - Android: an ACTION_VIEW intent with the file's type (guessed from
//     its extension), opening the user's default app or the system's
//     choice of apps. The file is handed over read-only through the
//     app's content provider, which giode build declares; an app built
//     by an older giode hands over a copy in Downloads instead (Android
//     10 or later). With no app for the type, it returns an error.
//   - iOS: the "Open in…" menu of the apps that open the file, or, when
//     none does, a Quick Look preview with the share button.
//   - Desktops: the file's default app (ShellExecute on Windows, which
//     asks with "Open with" when the type has none; NSWorkspace on
//     macOS; on Linux the desktop portal's OpenURI, which Flatpak and
//     Snap apps need, else xdg-open). OpenWith waits for the portal's
//     answer: ErrCanceled when the user dismisses its app chooser.
//   - The web returns ErrNotSupported.
//
// Call it from a goroutine: on Android it may copy the file.
func (a *App) OpenWith(ctx context.Context, path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return &os.PathError{Op: "open with", Path: abs, Err: os.ErrInvalid}
	}
	return shell.OpenWith(ctx, a.host(), abs, ShareFile{Path: abs}.mimeType())
}
