package signin

import (
	"context"
	"errors"
	"strings"

	"gioui.org/app"

	"github.com/xmhrnx/giode/auth"
	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/webauthn"
)

func PasskeyCreate(ctx context.Context, h platform.Host, c webauthn.Creation, raw []byte) ([]byte, error) {
	return androidPasskey(ctx, h, true, raw, c.User.Name, c.User.DisplayName)
}

func PasskeyGet(ctx context.Context, h platform.Host, r webauthn.Request, raw []byte) ([]byte, error) {
	return androidPasskey(ctx, h, false, raw, "", "")
}

func androidPasskey(ctx context.Context, h platform.Host, create bool, raw []byte, user, display string) ([]byte, error) {
	var ch <-chan mobile.PasskeyResult
	err := platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		var err error
		ch, err = mobile.Passkey(app.JavaVM(), app.AppContext(), view, create, string(raw), user, display)
		return err
	})
	if err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		return passkeyResult(r)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// passkeyResult maps the Credential Manager's outcome.
func passkeyResult(r mobile.PasskeyResult) ([]byte, error) {
	t := r.ErrType
	switch {
	case t == "" && r.JSON != "":
		return []byte(r.JSON), nil
	case t == "unsupported":
		return nil, auth.ErrUnsupported
	case strings.HasSuffix(t, "USER_CANCELED"), strings.HasSuffix(t, "TYPE_NOT_ALLOWED_ERROR"):
		return nil, auth.ErrCanceled
	case strings.HasSuffix(t, "NO_CREDENTIAL"), strings.HasSuffix(t, "NO_CREATE_OPTIONS"):
		return nil, auth.ErrNoPasskey
	case t == "":
		return nil, errors.New("passkey: no response")
	}
	return nil, errors.New("passkey: " + t + ": " + r.Err)
}
