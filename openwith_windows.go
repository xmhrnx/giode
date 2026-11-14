package giode

import (
	"context"
	"errors"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func (a *App) openWith(ctx context.Context, path, mime string) error {
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	dir, _ := windows.UTF16PtrFromString(filepath.Dir(path))
	hwnd := windows.Handle(a.NativeHandle())
	err = windows.ShellExecute(hwnd, windows.StringToUTF16Ptr("open"), file, nil, dir, windows.SW_SHOWNORMAL)
	// No app for the type: let the user pick one.
	if errors.Is(err, windows.ERROR_NO_ASSOCIATION) {
		err = windows.ShellExecute(hwnd, windows.StringToUTF16Ptr("openas"), file, nil, dir, windows.SW_SHOWNORMAL)
	}
	return err
}
