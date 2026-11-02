package printing

import (
	"context"

	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode"
)

const supported = true

// printFile runs the file's Print verb: its default app (a PDF reader, the
// Photos app) prints it.
func printFile(ctx context.Context, a *giode.App, path string, opts Options) error {
	verb, _ := windows.UTF16PtrFromString("print")
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
}
