package shell

import "errors"

// File is a file to share and its MIME type.
type File struct {
	Path, MIME string
}

// ErrNoShareSheet is ShareText's answer on a desktop without a share
// sheet: the caller copies the text to the clipboard instead.
var ErrNoShareSheet = errors.New("giode: no share sheet")
