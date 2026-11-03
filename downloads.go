package giode

import (
	"context"
	"os"
)

// SaveToDownloads copies the file at path into the user's Downloads
// folder, under folder (the app's name when empty), and returns where
// it went. Use it for files the user receives or exports and expects
// to find with their other downloads.
//
//   - Android: Download/<folder> through MediaStore, where the Files and
//     Downloads apps show it; Android 10 or later, no permission. It
//     returns the content:// URI.
//   - iOS: the app's Documents/<folder>, which the Files app shows under
//     the app when Info.plist has UIFileSharingEnabled and
//     LSSupportsOpeningDocumentsInPlace (set both to true under
//     [ios.plist] in giode.toml). It returns the file's path.
//   - Desktops: the Downloads folder (Windows' known folder,
//     XDG_DOWNLOAD_DIR on Linux, ~/Downloads on macOS). It returns the
//     file's path.
//
// A file of the same name isn't overwritten: the copy gets " (2)" and
// so on. It copies the whole file: call it from a goroutine.
func (a *App) SaveToDownloads(ctx context.Context, path, folder string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return a.saveToDownloads(ctx, path, ShareFile{Path: path}.mimeType(), a.folderName(folder))
}

// OpenDownloads shows where SaveToDownloads puts files: the Downloads
// app on Android, the app's folder in the Files app on iOS, and
// Downloads/<folder> in the file manager on desktops (folder is the
// app's name when empty).
func (a *App) OpenDownloads(ctx context.Context, folder string) error {
	return a.openDownloads(ctx, a.folderName(folder))
}

// OpenFolder opens the folder at path in the file manager: Explorer,
// Finder or the desktop's file manager; on iOS the Files app, for
// folders inside the app's Documents. Android returns ErrNotSupported:
// use OpenDownloads. To point at one file inside its folder, Share a
// ShareFile instead.
func (a *App) OpenFolder(ctx context.Context, path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return &os.PathError{Op: "open folder", Path: path, Err: os.ErrInvalid}
	}
	return a.openFolder(ctx, path)
}

// folderName is folder, else the app's name.
func (a *App) folderName(folder string) string {
	if folder == "" {
		folder = AppName
		if folder == "" {
			folder = a.Title()
		}
	}
	return sanitizeFileName(folder)
}
