package giode

import (
	"context"

	"github.com/xmhrnx/giode/internal/webauthn"
)

// CreatePasskey registers a passkey for the signed-in user with the
// platform's passkey sheet: iCloud Keychain on iOS and macOS, Google
// Password Manager or another provider on Android 14 and later, Windows
// Hello or a security key on Windows.
//
// options is the JSON of the server's registration options
// (PublicKeyCredentialCreationOptionsJSON, as SimpleWebAuthn's
// generateRegistrationOptions or go-webauthn's BeginRegistration return;
// a {"publicKey": ...} wrapper is fine). The result is the
// RegistrationResponseJSON to send back for verification.
//
// It ends with auth.ErrCanceled when the user closes the sheet, and
// auth.ErrUnsupported where the platform has no passkeys (Linux, the
// web, Android before 14). The app must be associated with the relying
// party's domain; see docs/auth.md. Call it off the UI goroutine.
func (a *App) CreatePasskey(ctx context.Context, options []byte) ([]byte, error) {
	c, raw, err := webauthn.ParseCreation(options)
	if err != nil {
		return nil, err
	}
	return a.passkeyCreate(ctx, c, raw)
}

// SignInWithPasskey signs in with a passkey: the platform shows the
// user's passkeys for the site (or those the server allows) and signs
// the server's challenge with the one picked.
//
// options is the JSON of the server's authentication options
// (PublicKeyCredentialRequestOptionsJSON); the result is the
// AuthenticationResponseJSON to send back. It ends with
// auth.ErrNoPasskey when there's no passkey to use, besides the errors
// CreatePasskey can end with.
func (a *App) SignInWithPasskey(ctx context.Context, options []byte) ([]byte, error) {
	r, raw, err := webauthn.ParseRequest(options)
	if err != nil {
		return nil, err
	}
	return a.passkeyGet(ctx, r, raw)
}
