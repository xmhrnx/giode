//go:build android || js

package dirs

import "github.com/xmhrnx/giode/internal/platform"

// Android's folders are MediaStore collections, reached through Java;
// the web has none.

func Pictures() (string, error)  { return "", platform.ErrNotSupported }
func Downloads() (string, error) { return "", platform.ErrNotSupported }
