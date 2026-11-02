//go:build !android && !darwin

package biometric

import (
	"context"

	"github.com/xmhrnx/giode"
)

const supported = false

func available() Kind { return None }

func authenticate(context.Context, *giode.App, Options) error { return giode.ErrNotSupported }
