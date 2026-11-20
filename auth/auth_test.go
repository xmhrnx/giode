package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeIdP is an OpenID provider: it checks PKCE, issues codes and
// tokens, refreshes and revokes.
type fakeIdP struct {
	srv *httptest.Server
	mu  sync.Mutex
	// codes maps a code to its PKCE challenge, redirect and nonce.
	codes     map[string][3]string
	refresh   map[string]bool
	revoked   []string
	deny      bool
	expiresIn int
}

func newIdP(t *testing.T) *fakeIdP {
	f := &fakeIdP{codes: map[string][3]string{}, refresh: map[string]bool{}, expiresIn: 3600}
	mux := http.NewServeMux()
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"issuer": f.srv.URL, "authorization_endpoint": f.srv.URL + "/authorize",
			"token_endpoint": f.srv.URL + "/token", "revocation_endpoint": f.srv.URL + "/revoke",
		})
	})
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		redirect := q.Get("redirect_uri")
		if q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" || q.Get("client_id") != "app" {
			http.Error(w, "bad request", 400)
			return
		}
		back := url.Values{"state": {q.Get("state")}}
		if f.deny {
			back.Set("error", "access_denied")
		} else {
			code := fmt.Sprintf("code-%d", len(f.codes))
			f.mu.Lock()
			f.codes[code] = [3]string{q.Get("code_challenge"), redirect, q.Get("nonce")}
			f.mu.Unlock()
			back.Set("code", code)
		}
		http.Redirect(w, r, redirect+"?"+back.Encode(), http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		f.mu.Lock()
		defer f.mu.Unlock()
		fail := func(code string) {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": code})
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			c, ok := f.codes[r.Form.Get("code")]
			delete(f.codes, r.Form.Get("code"))
			if !ok || challenge(r.Form.Get("code_verifier")) != c[0] || r.Form.Get("redirect_uri") != c[1] {
				fail("invalid_grant")
				return
			}
			rt := "refresh-" + r.Form.Get("code")
			f.refresh[rt] = true
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "access-1", "token_type": "Bearer", "expires_in": f.expiresIn,
				"refresh_token": rt, "id_token": jwt(map[string]any{"iss": f.srv.URL, "aud": "app", "sub": "42",
					"email": "ada@example.com", "nonce": c[2], "exp": time.Now().Add(time.Hour).Unix()}),
			})
		case "refresh_token":
			if !f.refresh[r.Form.Get("refresh_token")] {
				fail("invalid_grant")
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"access_token": "access-2", "token_type": "Bearer", "expires_in": 3600})
		default:
			fail("unsupported_grant_type")
		}
	})
	mux.HandleFunc("/revoke", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		f.mu.Lock()
		f.revoked = append(f.revoked, r.Form.Get("token"))
		delete(f.refresh, r.Form.Get("token"))
		f.mu.Unlock()
	})
	return f
}

func jwt(claims map[string]any) string {
	b, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"none"}`)) + "." + enc(b) + ".sig"
}

// fakeHost plays the browser: it fetches the authorization URL and
// follows its redirect, to the loopback listener or as a link opening
// the app.
type fakeHost struct {
	mu      sync.Mutex
	secrets map[string][]byte
	links   chan *url.URL
}

func newHost() *fakeHost {
	return &fakeHost{secrets: map[string][]byte{}, links: make(chan *url.URL, 1)}
}

func (h *fakeHost) OpenBrowser(ctx context.Context, u string) error {
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			// A custom scheme: the OS hands it to the app.
			h.links <- req.URL
			return http.ErrUseLastResponse
		}
		return nil
	}}
	go func() {
		resp, err := client.Get(u)
		if err == nil {
			resp.Body.Close()
		}
	}()
	return nil
}

func (h *fakeHost) WaitForURL(ctx context.Context, match func(*url.URL) bool) (*url.URL, error) {
	for {
		select {
		case u := <-h.links:
			if match(u) {
				return u, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (h *fakeHost) AuthSession(context.Context, string, string, bool) (*url.URL, error) {
	return nil, ErrUnsupported
}

func (h *fakeHost) SecretGet(name string) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	b, ok := h.secrets[name]
	if !ok {
		return nil, errors.New("not found")
	}
	return b, nil
}

func (h *fakeHost) SecretSet(name string, v []byte) error {
	h.mu.Lock()
	h.secrets[name] = v
	h.mu.Unlock()
	return nil
}

func (h *fakeHost) SecretDelete(name string) error {
	h.mu.Lock()
	delete(h.secrets, name)
	h.mu.Unlock()
	return nil
}

func ctx(t *testing.T) context.Context {
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return c
}

func TestLoopbackSignIn(t *testing.T) {
	idp := newIdP(t)
	host := newHost()
	s := NewSession(host, OIDC("test", idp.srv.URL, "app"))
	tok, err := s.SignIn(ctx(t))
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "access-1" || !tok.Valid() || tok.RefreshToken == "" {
		t.Errorf("token = %+v", tok)
	}
	c, err := tok.Claims()
	if err != nil || c.Email != "ada@example.com" || c.Subject != "42" {
		t.Errorf("claims = %+v, %v", c, err)
	}
	if _, ok := host.secrets["auth.test"]; !ok {
		t.Error("token not stored")
	}
	// A new session (the app restarted) finds it.
	s2 := NewSession(host, OIDC("test", idp.srv.URL, "app"))
	if got, err := s2.Token(ctx(t)); err != nil || got.AccessToken != "access-1" {
		t.Errorf("stored token = %+v, %v", got, err)
	}
}

func TestCustomSchemeSignInAndRefresh(t *testing.T) {
	idp := newIdP(t)
	idp.expiresIn = 30 // under a minute: Token refreshes at once
	host := newHost()
	p := OIDC("phone", idp.srv.URL, "app")
	p.RedirectURL = "com.example.app:/oauth"
	s := NewSession(host, p)
	tok, err := s.SignIn(ctx(t))
	if err != nil {
		t.Fatal(err)
	}
	if tok.Valid() {
		t.Fatal("a 30-second token counts as valid")
	}
	fresh, err := s.Token(ctx(t))
	if err != nil || fresh.AccessToken != "access-2" || fresh.RefreshToken != tok.RefreshToken || fresh.IDToken == "" {
		t.Fatalf("refreshed = %+v, %v", fresh, err)
	}
	// The client sends it.
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, r.Header.Get("Authorization"))
	}))
	defer api.Close()
	resp, err := s.Client(ctx(t)).Get(api.URL)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	fmt.Fscan(resp.Body, &b)
	resp.Body.Close()
	// Sign out revokes the refresh token and forgets.
	if err := s.SignOut(ctx(t)); err != nil {
		t.Fatal(err)
	}
	if len(idp.revoked) != 1 || idp.revoked[0] != tok.RefreshToken || s.SignedIn() {
		t.Errorf("revoked %v, signed in %v", idp.revoked, s.SignedIn())
	}
	if _, err := s.Token(ctx(t)); !errors.Is(err, ErrSignedOut) {
		t.Errorf("after sign-out: %v", err)
	}
}

func TestDeniedAndForgedRedirects(t *testing.T) {
	idp := newIdP(t)
	idp.deny = true
	s := NewSession(newHost(), OIDC("test", idp.srv.URL, "app"))
	if _, err := s.SignIn(ctx(t)); !errors.Is(err, ErrCanceled) {
		t.Errorf("denied: %v", err)
	}
	if _, err := checkCallback(&url.URL{RawQuery: "code=x&state=other"}, "mine"); err == nil {
		t.Error("a redirect with another state was taken")
	}
	if err := checkIDToken(jwt(map[string]any{"aud": "app", "nonce": "n2", "iss": "x"}), Provider{ClientID: "app"}, "n1"); err == nil {
		t.Error("an ID token with another nonce was taken")
	}
	if err := checkIDToken(jwt(map[string]any{"aud": "other", "nonce": "n1"}), Provider{ClientID: "app"}, "n1"); err == nil {
		t.Error("an ID token for another app was taken")
	}
}

func TestRevokedRefreshSignsOut(t *testing.T) {
	idp := newIdP(t)
	idp.expiresIn = 1
	host := newHost()
	s := NewSession(host, OIDC("test", idp.srv.URL, "app"))
	tok, err := s.SignIn(ctx(t))
	if err != nil {
		t.Fatal(err)
	}
	idp.mu.Lock()
	delete(idp.refresh, tok.RefreshToken) // revoked elsewhere
	idp.mu.Unlock()
	if _, err := s.Token(ctx(t)); !errors.Is(err, ErrSignedOut) {
		t.Errorf("refresh refused: %v", err)
	}
	if s.SignedIn() {
		t.Error("still signed in")
	}
}
