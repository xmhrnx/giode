//go:build !android && !darwin && !windows

package printing

import (
	"context"

	"github.com/xmhrnx/giode"
)

const supported = false

func printFile(context.Context, *giode.App, string, Options) error { return giode.ErrNotSupported }
