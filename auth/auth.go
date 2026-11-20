// Package auth signs people in with OAuth 2.0 and OpenID Connect:
// Google, Microsoft (Entra ID), Okta, Auth0, Keycloak, any identity
// provider, and through them SAML-backed company SSO. It runs the
// Authorization Code flow with PKCE (RFC 7636) the way native apps
// should (RFC 8252): in the system's browser or its in-app sign-in sheet,
// where the user's existing sessions and passkeys work, never in a web
// view the app controls.
//
//	sess := app.Auth(auth.Google("1234.apps.googleusercontent.com"))
//	tok, err := sess.SignIn(ctx)        // the browser, then back
//	...
//	tok, err = sess.Token(ctx)          // valid, refreshed when needed
//	client := sess.Client(ctx)          // an http.Client that sends it
//	sess.SignOut(ctx)
//
// Tokens are kept in the platform's secure storage and survive restarts.
package auth

import (
	"context"
	"errors"
	"net/url"
	"time"
)

// Errors a sign-in can end with.
var (
	// ErrCanceled: the user closed the browser or sign-in sheet.
	ErrCanceled = errors.New("auth: sign-in canceled")
	// ErrSignedOut: there's no token; sign in first.
	ErrSignedOut = errors.New("auth: signed out")
	// ErrUnsupported: the platform has no native sign-in sheet, or no
	// passkeys.
	ErrUnsupported = errors.New("auth: not supported on this platform")
	// ErrNoPasskey: the device has no passkey for the site, or none of
	// the ones the server allows.
	ErrNoPasskey = errors.New("auth: no passkey for this account")
)

// Host is what a session needs of the app: opening the browser, the
// links that come back to it, secure storage, and the platform's sign-in
// sheet. giode's App provides it (App.Auth).
type Host interface {
	// OpenBrowser opens u in the system's browser (an in-app browser tab
	// where the platform has one, as Custom Tabs on Android).
	OpenBrowser(ctx context.Context, u string) error
	// WaitForURL returns the next link opening the app that match takes,
	// keeping it from the app's own handler.
	WaitForURL(ctx context.Context, match func(*url.URL) bool) (*url.URL, error)
	// AuthSession runs the platform's sign-in sheet (ASWebAuthenticationSession
	// on iOS and macOS) on authURL until it's redirected to a URL of
	// callbackScheme, and returns that URL; ErrUnsupported elsewhere.
	AuthSession(ctx context.Context, authURL, callbackScheme string, ephemeral bool) (*url.URL, error)
	// Secrets are kept in secure storage under these.
	SecretGet(name string) ([]byte, error)
	SecretSet(name string, value []byte) error
	SecretDelete(name string) error
}

// Provider is an identity provider and the app's registration with it.
type Provider struct {
	// Name keys its stored tokens: "google", "work".
	Name string
	// The endpoints. Issuer alone is enough for OpenID Connect providers:
	// the rest are discovered from its .well-known/openid-configuration.
	Issuer        string
	AuthURL       string
	TokenURL      string
	RevokeURL     string
	EndSessionURL string
	// ClientID is the app's; ClientSecret only where the provider asks a
	// desktop app for one (Google does, though it isn't secret there).
	ClientID     string
	ClientSecret string
	// Scopes asked for; "openid email profile" by default for OpenID
	// Connect, with offline_access where the provider needs it to give a
	// refresh token.
	Scopes []string
	// RedirectURL is where the provider sends the user back:
	//   - empty on desktops: a loopback address on a free port
	//     (http://127.0.0.1:PORT/callback), registered with the provider
	//     as a loopback or "desktop" redirect;
	//   - a custom scheme ("com.example.app:/oauth") or an app link
	//     ("https://example.com/oauth") on phones, and on desktops that
	//     register one.
	RedirectURL string
	// Params are added to the authorization request: prompt,
	// login_hint, hd (Google Workspace domain), domain_hint.
	Params map[string]string
	// Ephemeral keeps iOS and macOS sign-in from sharing the browser's
	// cookies: no "signed in as" carried over, and nothing left behind.
	Ephemeral bool
}

// Token is what a sign-in gives.
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

// Valid reports whether the access token is there and not about to
// expire (within a minute).
func (t Token) Valid() bool {
	return t.AccessToken != "" && (t.Expiry.IsZero() || time.Until(t.Expiry) > time.Minute)
}
