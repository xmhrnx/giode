package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Claims are the usual claims of an OpenID Connect ID token: who signed
// in.
type Claims struct {
	Issuer        string   `json:"iss"`
	Subject       string   `json:"sub"`
	Audience      audience `json:"aud"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Name          string   `json:"name"`
	Picture       string   `json:"picture"`
	Nonce         string   `json:"nonce"`
	Expiry        int64    `json:"exp"`
	// TenantID is Microsoft's directory; HostedDomain Google Workspace's.
	TenantID     string `json:"tid"`
	HostedDomain string `json:"hd"`
	// Raw has every claim.
	Raw map[string]any `json:"-"`
}

type audience []string

func (a *audience) UnmarshalJSON(b []byte) error {
	var one string
	if json.Unmarshal(b, &one) == nil {
		*a = audience{one}
		return nil
	}
	var many []string
	err := json.Unmarshal(b, &many)
	*a = many
	return err
}

// ParseClaims reads an ID token's claims. It doesn't check the token's
// signature: the token came straight from the provider's token endpoint
// over TLS, which is what OpenID Connect allows a client to rely on; a
// server the app sends it to must verify it against the provider's keys.
func ParseClaims(idToken string) (Claims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("auth: not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return Claims{}, fmt.Errorf("auth: ID token: %w", err)
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Claims{}, fmt.Errorf("auth: ID token: %w", err)
	}
	json.Unmarshal(payload, &c.Raw)
	return c, nil
}

// Claims returns the claims of the token's ID token.
func (t Token) Claims() (Claims, error) {
	if t.IDToken == "" {
		return Claims{}, errors.New("auth: no ID token (ask for the openid scope)")
	}
	return ParseClaims(t.IDToken)
}

// checkIDToken checks a fresh ID token was issued for this sign-in: the
// nonce, the audience and, where known, the issuer.
func checkIDToken(idToken string, p Provider, nonce string) error {
	c, err := ParseClaims(idToken)
	if err != nil {
		return err
	}
	if c.Nonce != nonce {
		return errors.New("auth: the ID token's nonce doesn't match this sign-in")
	}
	found := false
	for _, a := range c.Audience {
		if a == p.ClientID {
			found = true
		}
	}
	if !found {
		return errors.New("auth: the ID token isn't for this app")
	}
	// Microsoft's "common" and "organizations" issuers name the tenant
	// in the token instead.
	if p.Issuer != "" && !strings.Contains(p.Issuer, "{tenantid}") && !strings.Contains(p.Issuer, "/common/") &&
		!strings.Contains(p.Issuer, "/organizations/") && !strings.Contains(p.Issuer, "/consumers/") &&
		strings.TrimSuffix(c.Issuer, "/") != strings.TrimSuffix(p.Issuer, "/") {
		return fmt.Errorf("auth: the ID token's issuer %q isn't %q", c.Issuer, p.Issuer)
	}
	if c.Expiry != 0 && time.Unix(c.Expiry, 0).Before(time.Now().Add(-5*time.Minute)) {
		return errors.New("auth: the ID token has expired")
	}
	return nil
}
