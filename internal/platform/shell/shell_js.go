package shell

import (
	"context"
	"syscall/js"

	"github.com/xmhrnx/giode/internal/platform"
)

// OpenURL opens u in a new tab.
func OpenURL(ctx context.Context, h platform.Host, u string) error {
	js.Global().Call("open", u, "_blank", "noopener")
	return nil
}

func OpenWith(ctx context.Context, h platform.Host, path, mime string) error {
	return platform.ErrNotSupported
}

func OpenFolder(ctx context.Context, h platform.Host, path string) error {
	return platform.ErrNotSupported
}

func SaveToDownloads(ctx context.Context, path, mime, folder string) (string, error) {
	return "", platform.ErrNotSupported
}

func OpenDownloads(ctx context.Context, h platform.Host, folder string) error {
	return platform.ErrNotSupported
}

func ShareText(ctx context.Context, h platform.Host, text, subject, title string) error {
	return platform.ErrNotSupported
}

func ShareFiles(ctx context.Context, h platform.Host, files []File, mime, title string) error {
	return platform.ErrNotSupported
}
