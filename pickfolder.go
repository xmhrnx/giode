package giode

import (
	"context"
	"sync"
)

// pickMu shows one folder picker at a time.
var pickMu sync.Mutex

// FolderOptions configures PickFolder.
type FolderOptions struct {
	// Title is the dialog's title or prompt: "Where should recovered
	// files go?". Empty uses the system's.
	Title string
	// Directory is the folder the dialog opens in; empty lets the
	// system choose.
	Directory string
}

// PickFolder shows the platform's folder picker and returns the path of
// the folder the user chose: IFileOpenDialog on Windows, NSOpenPanel on
// macOS, the XDG desktop portal on Linux (zenity or kdialog when no
// portal runs). Like OpenFile it blocks until the user decides: call it
// from a goroutine. It returns ErrCanceled when the user closes the
// picker, and ErrNotSupported on Android, iOS and the web, where apps
// get no plain paths to the user's folders.
//
// In the macOS App Sandbox (Mac App Store builds), access to the folder
// would end when the app quits; giode keeps a security-scoped bookmark
// of it and reopens it at every start, so the path keeps working.
func (a *App) PickFolder(ctx context.Context, opts FolderOptions) (string, error) {
	return await(ctx, func() (string, error) {
		pickMu.Lock()
		defer pickMu.Unlock()
		return pickFolder(a, opts)
	}, func(string) {})
}
