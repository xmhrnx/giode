package elevate

import (
	"errors"
	"strings"

	"github.com/xmhrnx/giode/internal/platform"
	"golang.org/x/sys/windows"
)

// IsElevated reports whether the process has administrator rights.
func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func Start(h platform.Host, exe string, args []string) error {
	quoted := make([]string, len(args))
	for i, s := range args {
		quoted[i] = windows.EscapeArg(s)
	}
	err := windows.ShellExecute(windows.Handle(h.NativeHandle()), windows.StringToUTF16Ptr("runas"),
		windows.StringToUTF16Ptr(exe), windows.StringToUTF16Ptr(strings.Join(quoted, " ")), nil, windows.SW_SHOWNORMAL)
	if errors.Is(err, windows.ERROR_CANCELLED) {
		return platform.ErrDeclined
	}
	return err
}
