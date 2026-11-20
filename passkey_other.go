//go:build !windows && !darwin && !android

package giode

import (
	"context"

	"github.com/xmhrnx/giode/auth"
	"github.com/xmhrnx/giode/internal/webauthn"
)

func (a *App) passkeyCreate(context.Context, webauthn.Creation, []byte) ([]byte, error) {
	return nil, auth.ErrUnsupported
}

func (a *App) passkeyGet(context.Context, webauthn.Request, []byte) ([]byte, error) {
	return nil, auth.ErrUnsupported
}
