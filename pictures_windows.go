//go:build windows

package giode

import "golang.org/x/sys/windows"

// picturesDir is the user's Pictures folder.
func picturesDir() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Pictures, 0)
}
