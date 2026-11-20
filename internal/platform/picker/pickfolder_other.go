//go:build android || ios || js || (!linux && !darwin && !windows)

package picker

import "github.com/xmhrnx/giode/internal/platform"

// PickFolder: phones and the web give apps no plain paths to folders.
func PickFolder(h platform.Host, title, dir string) (string, error) {
	return "", platform.ErrNotSupported
}
