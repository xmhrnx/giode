// Package webauthn reads WebAuthn's JSON options (what a server's
// generateRegistrationOptions or BeginLogin returns, as
// PublicKeyCredentialCreationOptionsJSON and RequestOptionsJSON) and
// writes the credentials platforms return in the JSON servers verify.
package webauthn

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Descriptor names a credential.
type Descriptor struct {
	Type       string   `json:"type"`
	ID         string   `json:"id"`
	Transports []string `json:"transports,omitempty"`
}

// Creation is a registration's options.
type Creation struct {
	Challenge string `json:"challenge"`
	RP        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"rp"`
	User struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"user"`
	PubKeyCredParams []struct {
		Type string `json:"type"`
		Alg  int    `json:"alg"`
	} `json:"pubKeyCredParams"`
	Timeout                int          `json:"timeout"`
	ExcludeCredentials     []Descriptor `json:"excludeCredentials"`
	AuthenticatorSelection struct {
		Attachment         string `json:"authenticatorAttachment"`
		ResidentKey        string `json:"residentKey"`
		RequireResidentKey bool   `json:"requireResidentKey"`
		UserVerification   string `json:"userVerification"`
	} `json:"authenticatorSelection"`
	Attestation string `json:"attestation"`
}

// Request is a sign-in's options.
type Request struct {
	Challenge        string       `json:"challenge"`
	RPID             string       `json:"rpId"`
	Timeout          int          `json:"timeout"`
	AllowCredentials []Descriptor `json:"allowCredentials"`
	UserVerification string       `json:"userVerification"`
}

// unwrap takes the options out of a {"publicKey": {...}} wrapper, the
// shape navigator.credentials takes, when they come in one.
func unwrap(b []byte) []byte {
	var w struct {
		PublicKey json.RawMessage `json:"publicKey"`
	}
	if json.Unmarshal(b, &w) == nil && len(w.PublicKey) > 0 {
		return w.PublicKey
	}
	return b
}

// ParseCreation reads registration options, returning them and the JSON
// the platform takes (unwrapped).
func ParseCreation(b []byte) (Creation, []byte, error) {
	b = unwrap(b)
	var c Creation
	if err := json.Unmarshal(b, &c); err != nil {
		return c, nil, fmt.Errorf("passkey: registration options: %w", err)
	}
	if c.Challenge == "" || c.RP.ID == "" || c.User.ID == "" {
		return c, nil, errors.New("passkey: registration options need challenge, rp.id and user.id")
	}
	if _, err := Decode(c.Challenge); err != nil {
		return c, nil, fmt.Errorf("passkey: challenge: %w", err)
	}
	if _, err := Decode(c.User.ID); err != nil {
		return c, nil, fmt.Errorf("passkey: user.id: %w", err)
	}
	return c, b, nil
}

// ParseRequest reads sign-in options.
func ParseRequest(b []byte) (Request, []byte, error) {
	b = unwrap(b)
	var r Request
	if err := json.Unmarshal(b, &r); err != nil {
		return r, nil, fmt.Errorf("passkey: sign-in options: %w", err)
	}
	if r.Challenge == "" || r.RPID == "" {
		return r, nil, errors.New("passkey: sign-in options need challenge and rpId")
	}
	if _, err := Decode(r.Challenge); err != nil {
		return r, nil, fmt.Errorf("passkey: challenge: %w", err)
	}
	return r, b, nil
}

// Encode is base64url without padding, WebAuthn's JSON encoding of bytes.
func Encode(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// Decode reads base64url, padded or not, and standard base64, which some
// servers send.
func Decode(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")
	if strings.ContainsAny(s, "+/") {
		return base64.RawStdEncoding.DecodeString(s)
	}
	return base64.RawURLEncoding.DecodeString(s)
}

// IDs decodes descriptors' IDs, skipping bad ones.
func IDs(ds []Descriptor) [][]byte {
	var out [][]byte
	for _, d := range ds {
		if b, err := Decode(d.ID); err == nil && len(b) > 0 {
			out = append(out, b)
		}
	}
	return out
}

// ClientData is the client data JSON for a ceremony where the app builds
// it (Windows): typ is "webauthn.create" or "webauthn.get", and the
// origin the relying party's site.
func ClientData(typ, challenge, rpID string) []byte {
	b, _ := json.Marshal(struct {
		Type        string `json:"type"`
		Challenge   string `json:"challenge"`
		Origin      string `json:"origin"`
		CrossOrigin bool   `json:"crossOrigin"`
	}{typ, strings.TrimRight(challenge, "="), "https://" + rpID, false})
	return b
}

// Registration is a new credential.
type Registration struct {
	CredentialID, ClientDataJSON, AttestationObject []byte
	Transports                                      []string
}

// JSON is the RegistrationResponseJSON a server verifies.
func (r Registration) JSON() []byte {
	tr := r.Transports
	if tr == nil {
		tr = []string{"internal", "hybrid"}
	}
	b, _ := json.Marshal(map[string]any{
		"id": Encode(r.CredentialID), "rawId": Encode(r.CredentialID), "type": "public-key",
		"authenticatorAttachment": "platform",
		"response": map[string]any{
			"clientDataJSON":    Encode(r.ClientDataJSON),
			"attestationObject": Encode(r.AttestationObject),
			"transports":        tr,
		},
		"clientExtensionResults": map[string]any{},
	})
	return b
}

// Assertion is a sign-in with a credential.
type Assertion struct {
	CredentialID, ClientDataJSON, AuthenticatorData, Signature, UserHandle []byte
}

// JSON is the AuthenticationResponseJSON a server verifies.
func (a Assertion) JSON() []byte {
	resp := map[string]any{
		"clientDataJSON":    Encode(a.ClientDataJSON),
		"authenticatorData": Encode(a.AuthenticatorData),
		"signature":         Encode(a.Signature),
	}
	if len(a.UserHandle) > 0 {
		resp["userHandle"] = Encode(a.UserHandle)
	}
	b, _ := json.Marshal(map[string]any{
		"id": Encode(a.CredentialID), "rawId": Encode(a.CredentialID), "type": "public-key",
		"authenticatorAttachment": "platform",
		"response":                resp,
		"clientExtensionResults":  map[string]any{},
	})
	return b
}
