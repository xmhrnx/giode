//go:build windows

package giode

import "golang.org/x/sys/windows"

// picturesDir is the user's Pictures folder.
func picturesDir() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Pictures, 0)
}

// downloadsDir is the user's Downloads folder.
func downloadsDir() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Downloads, 0)
}
