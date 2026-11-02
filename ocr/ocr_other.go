//go:build !android && !darwin

package ocr

import (
	"context"

	"github.com/xmhrnx/giode"
)

const supported = false

func recognize(context.Context, []byte, Options) ([]byte, error) {
	return nil, giode.ErrNotSupported
}
