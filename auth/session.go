package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Session signs in with one provider and keeps its token. Make one per
// provider (App.Auth) and keep it: it's safe to use from any goroutine.
type Session struct {
	host   Host
	p      Provider
	client *http.Client

	mu         sync.Mutex
	tok        *Token
	loaded     bool
	discovered bool
	// signIn serializes sign-ins.
	signIn sync.Mutex
}

// NewSession returns a session for p on host. Apps use App.Auth.
func NewSession(host Host, p Provider) *Session {
	if p.Name == "" {
		p.Name = "default"
	}
	return &Session{host: host, p: p, client: &http.Client{Timeout: 30 * time.Second}}
}

// WithHTTPClient sets the client token requests use. It returns s.
func (s *Session) WithHTTPClient(c *http.Client) *Session { s.client = c; return s }

// Provider returns the session's provider, endpoints discovered.
func (s *Session) Provider() Provider {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.p
}

func (s *Session) secretName() string { return "auth." + s.p.Name }

// discover fills the provider's endpoints once.
func (s *Session) discover(ctx context.Context) (Provider, error) {
	s.mu.Lock()
	p, done := s.p, s.discovered
	s.mu.Unlock()
	if done {
		return p, nil
	}
	p, err := Discover(ctx, s.client, p)
	if err != nil {
		return p, err
	}
	s.mu.Lock()
	s.p, s.discovered = p, true
	s.mu.Unlock()
	return p, nil
}

// random returns n random bytes, base64url without padding.
func random(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// challenge is PKCE's S256 code challenge of verifier.
func challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// SignIn runs the sign-in: the provider's page in the browser or the
// platform's sign-in sheet, back to the app, and the code exchanged for
// tokens, which it stores. ErrCanceled when the user backs out.
func (s *Session) SignIn(ctx context.Context) (Token, error) {
	s.signIn.Lock()
	defer s.signIn.Unlock()
	p, err := s.discover(ctx)
	if err != nil {
		return Token{}, err
	}
	verifier := random(48)
	state := random(24)
	nonce := ""
	if hasScope(p.Scopes, "openid") {
		nonce = random(24)
	}

	redirect := p.RedirectURL
	var loop *loopback
	if redirect == "" {
		loop, err = listenLoopback()
		if err != nil {
			return Token{}, err
		}
		defer loop.close()
		redirect = loop.redirect
	}
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {p.ClientID},
		"redirect_uri":          {redirect},
		"state":                 {state},
		"code_challenge":        {challenge(verifier)},
		"code_challenge_method": {"S256"},
	}
	if len(p.Scopes) > 0 {
		q.Set("scope", strings.Join(p.Scopes, " "))
	}
	if nonce != "" {
		q.Set("nonce", nonce)
	}
	for k, v := range p.Params {
		q.Set(k, v)
	}
	authURL := p.AuthURL
	if strings.Contains(authURL, "?") {
		authURL += "&" + q.Encode()
	} else {
		authURL += "?" + q.Encode()
	}

	var cb *url.URL
	if loop != nil {
		if err := s.host.OpenBrowser(ctx, authURL); err != nil {
			return Token{}, err
		}
		cb, err = loop.wait(ctx)
	} else {
		cb, err = s.callback(ctx, authURL, redirect, p.Ephemeral)
	}
	if err != nil {
		return Token{}, err
	}
	code, err := checkCallback(cb, state)
	if err != nil {
		return Token{}, err
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirect},
		"client_id":     {p.ClientID},
		"code_verifier": {verifier},
	}
	tok, err := s.tokenRequest(ctx, p, form)
	if err != nil {
		return Token{}, err
	}
	if tok.IDToken != "" && nonce != "" {
		if err := checkIDToken(tok.IDToken, p, nonce); err != nil {
			return Token{}, err
		}
	}
	if err := s.save(tok); err != nil {
		return tok, err
	}
	return tok, nil
}

// callback gets the redirect back to a custom scheme or app link: the
// platform's sign-in sheet where there is one, else the browser and the
// link opening the app.
func (s *Session) callback(ctx context.Context, authURL, redirect string, ephemeral bool) (*url.URL, error) {
	ru, err := url.Parse(redirect)
	if err != nil {
		return nil, fmt.Errorf("auth: redirect URL %q: %w", redirect, err)
	}
	cb, err := s.host.AuthSession(ctx, authURL, ru.Scheme, ephemeral)
	if !errors.Is(err, ErrUnsupported) {
		return cb, err
	}
	// Waiting starts before the browser opens, so a quick return isn't
	// missed.
	type result struct {
		u   *url.URL
		err error
	}
	got := make(chan result, 1)
	wctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		u, err := s.host.WaitForURL(wctx, func(u *url.URL) bool { return matchesRedirect(u, ru) })
		got <- result{u, err}
	}()
	if err := s.host.OpenBrowser(ctx, authURL); err != nil {
		return nil, err
	}
	r := <-got
	return r.u, r.err
}

// matchesRedirect reports whether u comes back to redirect: the same
// scheme, host and path.
func matchesRedirect(u, redirect *url.URL) bool {
	return strings.EqualFold(u.Scheme, redirect.Scheme) && strings.EqualFold(u.Host, redirect.Host) &&
		strings.TrimSuffix(u.Path, "/") == strings.TrimSuffix(redirect.Path, "/")
}

// checkCallback returns the code of a redirect back, after checking its
// state.
func checkCallback(u *url.URL, state string) (string, error) {
	q := u.Query()
	if q.Get("code") == "" && u.Fragment != "" {
		// Some providers answer in the fragment.
		if fq, err := url.ParseQuery(u.Fragment); err == nil {
			q = fq
		}
	}
	if e := q.Get("error"); e != "" {
		if e == "access_denied" {
			return "", ErrCanceled
		}
		return "", &Error{Code: e, Description: q.Get("error_description")}
	}
	if q.Get("state") != state {
		return "", errors.New("auth: the redirect's state doesn't match: not this sign-in's")
	}
	code := q.Get("code")
	if code == "" {
		return "", errors.New("auth: the redirect has no code")
	}
	return code, nil
}

// Error is an OAuth error from the provider.
type Error struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e *Error) Error() string {
	if e.Description != "" {
		return "auth: " + e.Code + ": " + e.Description
	}
	return "auth: " + e.Code
}

// tokenRequest posts form to the token endpoint.
func (s *Session) tokenRequest(ctx context.Context, p Provider, form url.Values) (Token, error) {
	if p.ClientSecret != "" {
		form.Set("client_secret", p.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("auth: token request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Token{}, err
	}
	var r struct {
		Token
		ExpiresIn json.Number `json:"expires_in"`
		Error
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return Token{}, fmt.Errorf("auth: token response (%s): %w", resp.Status, err)
	}
	if r.Error.Code != "" {
		e := r.Error
		return Token{}, &e
	}
	if resp.StatusCode != http.StatusOK || r.AccessToken == "" {
		return Token{}, fmt.Errorf("auth: token response: %s", resp.Status)
	}
	tok := r.Token
	if secs, err := r.ExpiresIn.Int64(); err == nil && secs > 0 {
		tok.Expiry = time.Now().Add(time.Duration(secs) * time.Second)
	}
	return tok, nil
}

// load reads the stored token once.
func (s *Session) load() *Token {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loaded {
		s.loaded = true
		if b, err := s.host.SecretGet(s.secretName()); err == nil {
			var t Token
			if json.Unmarshal(b, &t) == nil && t.AccessToken != "" {
				s.tok = &t
			}
		}
	}
	return s.tok
}

func (s *Session) save(t Token) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.tok, s.loaded = &t, true
	s.mu.Unlock()
	return s.host.SecretSet(s.secretName(), b)
}

// SignedIn reports whether there's a token, stored or fresh (it may need
// a refresh: Token gets it).
func (s *Session) SignedIn() bool { return s.load() != nil }

// Token returns a valid token: the stored one, refreshed when it's
// about to expire. ErrSignedOut when there's none, or the provider
// refused the refresh (the user signed out elsewhere, or revoked the
// app): sign in again.
func (s *Session) Token(ctx context.Context) (Token, error) {
	t := s.load()
	if t == nil {
		return Token{}, ErrSignedOut
	}
	if t.Valid() {
		return *t, nil
	}
	return s.Refresh(ctx)
}

// Refresh gets a new access token with the refresh token.
func (s *Session) Refresh(ctx context.Context) (Token, error) {
	s.signIn.Lock()
	defer s.signIn.Unlock()
	t := s.load()
	if t == nil {
		return Token{}, ErrSignedOut
	}
	if t.Valid() {
		// Another call refreshed it meanwhile.
		return *t, nil
	}
	if t.RefreshToken == "" {
		return Token{}, ErrSignedOut
	}
	p, err := s.discover(ctx)
	if err != nil {
		return Token{}, err
	}
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {t.RefreshToken}, "client_id": {p.ClientID}}
	nt, err := s.tokenRequest(ctx, p, form)
	var oe *Error
	if errors.As(err, &oe) && oe.Code == "invalid_grant" {
		s.forget()
		return Token{}, ErrSignedOut
	}
	if err != nil {
		return Token{}, err
	}
	// Providers may keep the refresh token and ID token, not resending
	// them.
	if nt.RefreshToken == "" {
		nt.RefreshToken = t.RefreshToken
	}
	if nt.IDToken == "" {
		nt.IDToken = t.IDToken
	}
	return nt, s.save(nt)
}

func (s *Session) forget() {
	s.mu.Lock()
	s.tok, s.loaded = nil, true
	s.mu.Unlock()
	s.host.SecretDelete(s.secretName())
}

// SignOut revokes the tokens where the provider can, and forgets them.
// The browser may still be signed in to the provider (unless the
// session was Ephemeral); EndBrowserSession signs it out too.
func (s *Session) SignOut(ctx context.Context) error {
	t := s.load()
	s.forget()
	if t == nil {
		return nil
	}
	p, err := s.discover(ctx)
	if err != nil || p.RevokeURL == "" {
		return nil
	}
	tok, hint := t.RefreshToken, "refresh_token"
	if tok == "" {
		tok, hint = t.AccessToken, "access_token"
	}
	form := url.Values{"token": {tok}, "token_type_hint": {hint}, "client_id": {p.ClientID}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.RevokeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth: revoking: %w", err)
	}
	resp.Body.Close()
	return nil
}

// EndBrowserSession opens the provider's sign-out page (OpenID Connect
// RP-initiated logout) in the browser, so the next sign-in asks for the
// account again. Call it after SignOut where the provider has one.
func (s *Session) EndBrowserSession(ctx context.Context, idTokenHint string) error {
	p, err := s.discover(ctx)
	if err != nil {
		return err
	}
	if p.EndSessionURL == "" {
		return ErrUnsupported
	}
	u := p.EndSessionURL
	q := url.Values{"client_id": {p.ClientID}}
	if idTokenHint != "" {
		q.Set("id_token_hint", idTokenHint)
	}
	if strings.Contains(u, "?") {
		u += "&" + q.Encode()
	} else {
		u += "?" + q.Encode()
	}
	return s.host.OpenBrowser(ctx, u)
}

// Client returns an http.Client that sends the session's access token,
// refreshing it as it expires.
func (s *Session) Client(ctx context.Context) *http.Client {
	return &http.Client{Transport: &bearer{s: s, ctx: ctx, base: http.DefaultTransport}}
}

type bearer struct {
	s    *Session
	ctx  context.Context
	base http.RoundTripper
}

func (b *bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	t, err := b.s.Token(b.ctx)
	if err != nil {
		return nil, err
	}
	r = r.Clone(r.Context())
	typ := t.TokenType
	if typ == "" || strings.EqualFold(typ, "bearer") {
		typ = "Bearer"
	}
	r.Header.Set("Authorization", typ+" "+t.AccessToken)
	return b.base.RoundTrip(r)
}

func hasScope(scopes []string, s string) bool {
	for _, x := range scopes {
		if x == s {
			return true
		}
	}
	return false
}

// loopback is a listener on 127.0.0.1 the browser is sent back to.
type loopback struct {
	ln       net.Listener
	srv      *http.Server
	redirect string
	got      chan *url.URL
}

func listenLoopback() (*loopback, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("auth: loopback redirect: %w", err)
	}
	l := &loopback{ln: ln, got: make(chan *url.URL, 1),
		redirect: fmt.Sprintf("http://127.0.0.1:%d/callback", ln.Addr().(*net.TCPAddr).Port)}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		u := *r.URL
		select {
		case l.got <- &u:
		default:
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		io.WriteString(w, callbackPage)
	})
	l.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go l.srv.Serve(ln)
	return l, nil
}

func (l *loopback) wait(ctx context.Context) (*url.URL, error) {
	select {
	case u := <-l.got:
		return u, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *loopback) close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	l.srv.Shutdown(ctx)
}

// callbackPage is what the browser shows after the redirect back.
const callbackPage = `<!doctype html><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>Signed in</title><body style="font:16px system-ui,sans-serif;text-align:center;padding:4em 1em">
<h1 style="font-size:1.4em">You're signed in</h1><p>You can close this tab and go back to the app.</p>
<script>setTimeout(function(){window.close()},500)</script>`
