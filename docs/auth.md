# Sign-in and SSO

The `auth` package signs people in with OAuth 2.0 and OpenID Connect: Google, Microsoft
(Entra ID), Okta, Auth0, Keycloak or any other identity provider, and through them company
single sign-on, SAML included. Apps also register and use [passkeys](#passkeys) for their
own accounts. It runs the Authorization Code flow with PKCE the way native
apps should (RFC 8252): in the system's browser or the platform's sign-in sheet, where the
user's saved passwords, existing sessions, passkeys and their company's device checks work,
never in a web view the app controls.

```go
import "github.com/xmhrnx/giode/auth"

google := app.Auth(auth.Google("1234-abc.apps.googleusercontent.com"))

go func() {
	tok, err := google.SignIn(ctx) // the browser or sign-in sheet, then back
	if errors.Is(err, auth.ErrCanceled) {
		return
	}
	who, _ := tok.Claims() // who.Email, who.Name, who.Subject
	...
}()
```

`App.Auth` returns the same session for every call with the same provider name. Its methods
block on the network, so call them off the UI goroutine.

| Method | |
|---|---|
| `SignIn(ctx)` | Opens the sign-in page and returns the token once the provider redirects back |
| `Token(ctx)` | The current token, refreshed first when it expires within a minute; `ErrSignedOut` when there's none |
| `Client(ctx)` | An `http.Client` that sends the access token, refreshing it as needed |
| `SignedIn()` | Whether a token is stored |
| `Refresh(ctx)` | Refreshes now; a refresh token the provider has revoked signs the user out |
| `SignOut(ctx)` | Revokes the refresh token where the provider supports it, and forgets the tokens |
| `EndBrowserSession(ctx, idToken)` | Also signs out of the provider's web session (its end-session page) |

Tokens are stored in `giode.SecureStore` (the Keychain, the Android Keystore, the Secret
Service or DPAPI) under `auth.<name>`, so a signed-in user stays signed in across restarts.

## Providers

```go
auth.Google(clientID)                        // Google accounts and Workspace
auth.Microsoft("contoso.onmicrosoft.com", id) // a tenant, "organizations", "consumers" or "common"
auth.OIDC("okta", "https://example.okta.com", clientID)
auth.OIDC("work", "https://sso.example.com/realms/staff", clientID) // Keycloak
```

`OIDC` takes any OpenID Connect issuer and discovers its endpoints from
`/.well-known/openid-configuration` on first use. A plain OAuth 2.0 provider (GitHub, for
example) is a `Provider` with `AuthURL`, `TokenURL` and `Scopes` set. SAML identity providers
are reached through a broker that speaks OpenID Connect: Okta, Auth0, Entra ID or Keycloak.

A `Provider`'s fields can be changed before passing it to `App.Auth`:

| Field | |
|---|---|
| `Name` | Keys the stored tokens: two providers need two names |
| `Scopes` | `openid email profile` by default, with `offline_access` where a provider needs it for a refresh token |
| `RedirectURL` | Where the provider sends the user back; see below |
| `Params` | Added to the sign-in request: `login_hint`, `prompt`, `hd` (a Google Workspace domain), `domain_hint` |
| `ClientSecret` | Only for providers that give desktop apps one (Google does; it isn't secret there) |
| `Ephemeral` | On iOS and macOS, doesn't share the browser's cookies: no account carried over, nothing left behind |

## Redirects

Register the app with the provider as a **native**, **mobile and desktop** or **public**
client (no client secret), with the redirect it will use.

**Desktops** need nothing more: with `RedirectURL` empty, the app listens on a free loopback
port and the provider redirects to `http://127.0.0.1:PORT/callback`. Register
`http://127.0.0.1` (or `http://localhost`, as the provider asks) as a loopback redirect; the
port can change between sign-ins. Google calls this a "Desktop app" client.

**Phones**, and desktops that prefer it, redirect to the app itself:

- a custom scheme, such as `com.example.app:/oauth`: list `com.example.app` under
  `app.schemes` in `giode.toml`;
- or an app link, such as `https://example.com/oauth`, listed under `app.links`.

```go
p := auth.OIDC("work", "https://example.okta.com", clientID)
p.RedirectURL = "com.example.app:/oauth"
work := app.Auth(p)
```

Google clients default to the reversed client ID scheme on phones
(`com.googleusercontent.apps.1234-abc:/oauth2redirect`, Google's "iOS" client type): add
that scheme to `app.schemes`.

The redirect reaches the waiting sign-in before `Config.OnOpenURL`, which never sees it.

## On each platform

| | Sign-in page |
|---|---|
| iOS, macOS | `ASWebAuthenticationSession`: a sheet over the app sharing Safari's cookies and passkeys. https redirects open the browser and come back as a universal link |
| Android | A Custom Tab over the app, with the browser's cookies and passwords; it closes when the redirect brings the app back |
| Windows, Linux | The default browser; the loopback redirect, or the app's scheme passed to the running instance (keep `SingleInstance` on) |
| Web | Not supported: a page signs in through its own server, or a JavaScript library |

## Passkeys

Passkeys replace passwords with a key the device keeps and unlocks with Face ID, a
fingerprint, Windows Hello or the screen lock. The server runs WebAuthn as it would for its
website (with SimpleWebAuthn, go-webauthn, or its identity provider); the app passes the
server's options to the platform and the result back:

```go
// Registering, for a signed-in user:
opts := fetch("/passkeys/register/options")     // the server's JSON options
resp, err := app.CreatePasskey(ctx, opts)
post("/passkeys/register", resp)                // the server verifies and stores it

// Signing in:
opts = fetch("/passkeys/login/options")
resp, err = app.SignInWithPasskey(ctx, opts)
switch {
case errors.Is(err, auth.ErrNoPasskey):         // offer another way in
case errors.Is(err, auth.ErrCanceled):
case err == nil:
	post("/passkeys/login", resp)
}
```

Options and responses are WebAuthn's JSON (`PublicKeyCredentialCreationOptionsJSON` and
`RegistrationResponseJSON`, and their sign-in counterparts), with byte fields in base64url,
so a server's web endpoints serve the app unchanged. A `{"publicKey": ...}` wrapper is fine.

| | Passkeys from |
|---|---|
| iOS 15+, macOS 12+ | iCloud Keychain and other passkey apps; a phone nearby; security keys |
| Android 14+ | Google Password Manager and other credential providers (Credential Manager) |
| Windows 10 1903+ | Windows Hello; security keys; a phone nearby on Windows 11 |
| Linux, the web, Android before 14 | `auth.ErrUnsupported` |

The platforms only let an app use passkeys for domains it's associated with. List the
relying party's domain (its `rp.id`) under `app.passkeys` in `giode.toml`:

```toml
[app]
passkeys = ["example.com"]
```

and serve the files `giode links` writes for it: `apple-app-site-association` gets a
`webcredentials` entry (the CLI adds the `webcredentials:` associated domain to iOS and macOS
signing), and `assetlinks.json` the `get_login_creds` relation for Android. Windows needs no
association.

The server must accept the origin each platform reports in the client data:
`https://example.com` from iOS, macOS and Windows, and `android:apk-key-hash:...` (the signing
certificate's SHA-256, base64url) from Android. Most WebAuthn libraries take a list of
allowed origins.

On iOS and macOS, having no passkey for the site ends as `ErrCanceled`: the system doesn't
say which.

## Security

- Every sign-in uses PKCE (S256), a random `state` checked on the way back and, with
  `openid`, a `nonce` checked in the ID token, together with its audience, issuer and
  expiry.
- `tok.Claims()` reads the ID token without checking its signature: it came straight from
  the provider over TLS, which OpenID Connect lets a client rely on. A server the app sends
  it to must verify it against the provider's keys.
- Send the access token to APIs; send the ID token to your own server only to identify
  the user, and verify it there.
