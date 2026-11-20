package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Google is Google's OpenID Connect provider. Register an "iOS" client
// for phones (its redirect is the reversed client ID scheme, set here
// when RedirectURL is empty on a phone) or a "Desktop app" client for
// computers (a loopback redirect; Google gives desktop clients a secret
// that isn't secret: set ClientSecret).
func Google(clientID string) Provider {
	return Provider{
		Name: "google", Issuer: "https://accounts.google.com", ClientID: clientID,
		AuthURL:   "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:  "https://oauth2.googleapis.com/token",
		RevokeURL: "https://oauth2.googleapis.com/revoke",
		Scopes:    []string{"openid", "email", "profile"},
		// A refresh token on every sign-in.
		Params: map[string]string{"access_type": "offline", "prompt": "consent"},
	}
}

// Microsoft is Microsoft's identity platform (Entra ID, formerly Azure
// AD): tenant is a directory ID or domain, "organizations" for any work
// account, "consumers" for personal ones, "common" for both. Register a
// "Mobile and desktop applications" platform with the redirect.
func Microsoft(tenant, clientID string) Provider {
	if tenant == "" {
		tenant = "common"
	}
	base := "https://login.microsoftonline.com/" + tenant
	return Provider{
		Name: "microsoft", Issuer: base + "/v2.0", ClientID: clientID,
		AuthURL:       base + "/oauth2/v2.0/authorize",
		TokenURL:      base + "/oauth2/v2.0/token",
		EndSessionURL: base + "/oauth2/v2.0/logout",
		Scopes:        []string{"openid", "email", "profile", "offline_access"},
	}
}

// OIDC is any OpenID Connect provider by its issuer URL (Okta, Auth0,
// Keycloak, a company's own): its endpoints are discovered on first use.
// SAML identity providers sign in through such a broker.
func OIDC(name, issuer, clientID string) Provider {
	return Provider{Name: name, Issuer: issuer, ClientID: clientID,
		Scopes: []string{"openid", "email", "profile", "offline_access"}}
}

// discovery is the part of an OpenID provider's configuration used.
type discovery struct {
	Issuer        string `json:"issuer"`
	AuthURL       string `json:"authorization_endpoint"`
	TokenURL      string `json:"token_endpoint"`
	RevokeURL     string `json:"revocation_endpoint"`
	EndSessionURL string `json:"end_session_endpoint"`
}

// Discover fills p's missing endpoints from its issuer's
// .well-known/openid-configuration.
func Discover(ctx context.Context, client *http.Client, p Provider) (Provider, error) {
	if p.AuthURL != "" && p.TokenURL != "" {
		return p, nil
	}
	if p.Issuer == "" {
		return p, fmt.Errorf("auth: provider %q has no issuer or endpoints", p.Name)
	}
	u := strings.TrimSuffix(p.Issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return p, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return p, fmt.Errorf("auth: discovering %s: %w", p.Issuer, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return p, fmt.Errorf("auth: discovering %s: %s", p.Issuer, resp.Status)
	}
	var d discovery
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return p, fmt.Errorf("auth: discovering %s: %w", p.Issuer, err)
	}
	set := func(dst *string, v string) {
		if *dst == "" {
			*dst = v
		}
	}
	set(&p.AuthURL, d.AuthURL)
	set(&p.TokenURL, d.TokenURL)
	set(&p.RevokeURL, d.RevokeURL)
	set(&p.EndSessionURL, d.EndSessionURL)
	if p.AuthURL == "" || p.TokenURL == "" {
		return p, fmt.Errorf("auth: %s has no authorization or token endpoint", p.Issuer)
	}
	return p, nil
}
