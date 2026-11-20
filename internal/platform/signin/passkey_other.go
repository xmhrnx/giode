//go:build !windows && !darwin && !android

package signin

import (
	"context"

	"github.com/xmhrnx/giode/auth"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/webauthn"
)

func PasskeyCreate(context.Context, platform.Host, webauthn.Creation, []byte) ([]byte, error) {
	return nil, auth.ErrUnsupported
}

func PasskeyGet(context.Context, platform.Host, webauthn.Request, []byte) ([]byte, error) {
	return nil, auth.ErrUnsupported
}
