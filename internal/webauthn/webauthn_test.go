package webauthn

import (
	"encoding/json"
	"testing"
)

func TestParse(t *testing.T) {
	c, raw, err := ParseCreation([]byte(`{"publicKey":{"challenge":"AQID","rp":{"id":"example.com","name":"Ex"},
		"user":{"id":"dXNlcg","name":"ada","displayName":"Ada"},"pubKeyCredParams":[{"type":"public-key","alg":-7}],
		"excludeCredentials":[{"type":"public-key","id":"BAU"},{"type":"public-key","id":"!"}],
		"authenticatorSelection":{"residentKey":"required","userVerification":"preferred"}}}`))
	if err != nil || c.RP.ID != "example.com" || c.User.Name != "ada" || len(IDs(c.ExcludeCredentials)) != 1 {
		t.Fatalf("%+v, %v", c, err)
	}
	if json.Valid(raw) == false || raw[0] != '{' || string(raw[:12]) == `{"publicKey"` {
		t.Errorf("not unwrapped: %s", raw)
	}
	if _, _, err := ParseCreation([]byte(`{"challenge":"AQID","rp":{"id":"x"}}`)); err == nil {
		t.Error("missing user taken")
	}
	r, _, err := ParseRequest([]byte(`{"challenge":"AQID==","rpId":"example.com","allowCredentials":[]}`))
	if err != nil || r.RPID != "example.com" {
		t.Fatalf("%+v, %v", r, err)
	}
	if b, err := Decode("+/8="); err != nil || len(b) != 2 {
		t.Errorf("standard base64: %v %v", b, err)
	}
}

func TestJSON(t *testing.T) {
	var reg struct {
		ID       string `json:"id"`
		Response struct {
			ClientDataJSON    string `json:"clientDataJSON"`
			AttestationObject string `json:"attestationObject"`
		} `json:"response"`
	}
	json.Unmarshal(Registration{CredentialID: []byte{1, 2}, ClientDataJSON: []byte("{}"), AttestationObject: []byte{9}}.JSON(), &reg)
	if reg.ID != "AQI" || reg.Response.ClientDataJSON != "e30" || reg.Response.AttestationObject != "CQ" {
		t.Errorf("%+v", reg)
	}
	var as map[string]any
	json.Unmarshal(Assertion{CredentialID: []byte{1}, Signature: []byte{2}}.JSON(), &as)
	if r := as["response"].(map[string]any); r["signature"] != "Ag" || r["userHandle"] != nil {
		t.Errorf("%v", as)
	}
	var cd map[string]any
	json.Unmarshal(ClientData("webauthn.get", "AQID==", "example.com"), &cd)
	if cd["origin"] != "https://example.com" || cd["challenge"] != "AQID" || cd["type"] != "webauthn.get" {
		t.Errorf("%v", cd)
	}
}
