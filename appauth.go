package giode

import (
	"context"
	"errors"
	"net/url"
	"runtime"
	"strings"
	"sync"

	"github.com/xmhrnx/giode/auth"
)

// authState holds the app's sign-in sessions and the sign-ins waiting
// for a link back to the app.
type authState struct {
	mu       sync.Mutex
	sessions map[string]*auth.Session
	waiters  []*urlWaiter
}

type urlWaiter struct {
	match func(*url.URL) bool
	got   chan *url.URL
}

// take hands u to a sign-in waiting for it, reporting whether one was.
func (s *authState) take(u *url.URL) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, w := range s.waiters {
		if w.match(u) {
			s.waiters = append(s.waiters[:i], s.waiters[i+1:]...)
			w.got <- u
			return true
		}
	}
	return false
}

// waiting reports whether a sign-in waits for a link.
func (s *authState) waiting() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.waiters) > 0
}

// Auth returns the app's sign-in session for p, the same one for every
// call with the same p.Name: sign in, keep tokens fresh and sign out
// with the platform's browser or sign-in sheet. See package auth.
//
//	google := app.Auth(auth.Google("1234.apps.googleusercontent.com"))
//	go func() {
//		tok, err := google.SignIn(ctx)
//		...
//	}()
//
// Desktops redirect to a loopback address by default. On phones, Google
// providers default to the reversed client ID scheme
// (com.googleusercontent.apps.1234:/oauth2redirect); other providers
// need a RedirectURL with the app's scheme or app link, registered with
// the provider and in giode.toml. The redirect is handled before
// Config.OnOpenURL sees it.
//
// Tokens are kept in SecureStore. Sessions' methods block on the
// network: call them off the UI goroutine.
func (a *App) Auth(p auth.Provider) *auth.Session {
	if p.Name == "" {
		p.Name = "default"
	}
	if p.RedirectURL == "" && (runtime.GOOS == "android" || runtime.GOOS == "ios") {
		if id, ok := strings.CutSuffix(p.ClientID, ".apps.googleusercontent.com"); ok && strings.Contains(p.AuthURL, "accounts.google.com") {
			p.RedirectURL = "com.googleusercontent.apps." + id + ":/oauth2redirect"
		}
	}
	a.auth.mu.Lock()
	defer a.auth.mu.Unlock()
	if s, ok := a.auth.sessions[p.Name]; ok {
		return s
	}
	if a.auth.sessions == nil {
		a.auth.sessions = map[string]*auth.Session{}
	}
	s := auth.NewSession(appAuthHost{a}, p)
	a.auth.sessions[p.Name] = s
	return s
}

// appAuthHost is the App as an auth.Host.
type appAuthHost struct{ a *App }

func (h appAuthHost) OpenBrowser(ctx context.Context, u string) error {
	return h.a.openSignIn(ctx, u)
}

func (h appAuthHost) WaitForURL(ctx context.Context, match func(*url.URL) bool) (*url.URL, error) {
	w := &urlWaiter{match: match, got: make(chan *url.URL, 1)}
	s := &h.a.auth
	s.mu.Lock()
	s.waiters = append(s.waiters, w)
	s.mu.Unlock()
	select {
	case u := <-w.got:
		return u, nil
	case <-ctx.Done():
		s.mu.Lock()
		for i, x := range s.waiters {
			if x == w {
				s.waiters = append(s.waiters[:i], s.waiters[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		select {
		case u := <-w.got: // arrived meanwhile
			return u, nil
		default:
			return nil, ctx.Err()
		}
	}
}

func (h appAuthHost) AuthSession(ctx context.Context, authURL, scheme string, ephemeral bool) (*url.URL, error) {
	return h.a.authSession(ctx, authURL, scheme, ephemeral)
}

func (h appAuthHost) SecretGet(name string) ([]byte, error) { return SecureStore.Get(name) }
func (h appAuthHost) SecretSet(name string, v []byte) error { return SecureStore.Set(name, v) }
func (h appAuthHost) SecretDelete(name string) error {
	err := SecureStore.Delete(name)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}
