package shell

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/xmhrnx/giode/internal/platform"
	"golang.org/x/sys/windows"
)

// OpenWith opens the file in its default app with ShellExecute.
func OpenWith(ctx context.Context, h platform.Host, path, mime string) error {
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	dir, _ := windows.UTF16PtrFromString(filepath.Dir(path))
	hwnd := windows.Handle(h.NativeHandle())
	err = windows.ShellExecute(hwnd, windows.StringToUTF16Ptr("open"), file, nil, dir, windows.SW_SHOWNORMAL)
	// No app for the type: let the user pick one.
	if errors.Is(err, windows.ERROR_NO_ASSOCIATION) {
		err = windows.ShellExecute(hwnd, windows.StringToUTF16Ptr("openas"), file, nil, dir, windows.SW_SHOWNORMAL)
	}
	return err
}
