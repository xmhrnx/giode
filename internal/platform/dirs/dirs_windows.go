//go:build windows

package dirs

import "golang.org/x/sys/windows"

// Pictures is the user's Pictures folder.
func Pictures() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Pictures, 0)
}

// Downloads is the user's Downloads folder.
func Downloads() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Downloads, 0)
}
