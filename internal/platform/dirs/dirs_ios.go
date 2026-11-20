package dirs

import (
	"os"
	"path/filepath"

	"github.com/xmhrnx/giode/internal/platform"
)

// Documents is the app's Documents folder, which the Files app shows
// with UIFileSharingEnabled and LSSupportsOpeningDocumentsInPlace.
func Documents() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents"), nil
}

// Downloads is Documents: an iOS app's files live there.
func Downloads() (string, error) { return Documents() }

// Pictures has no folder on iOS: photos go to the library.
func Pictures() (string, error) { return "", platform.ErrNotSupported }
