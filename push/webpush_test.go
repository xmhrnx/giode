package push

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestWebPushRoundTrip(t *testing.T) {
	k, err := newWebPushKey()
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte(`{"title":"Hi","body":"There"}`)
	enc, err := encryptWebPush(k.public(), msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := k.decrypt(enc)
	if err != nil || !bytes.Equal(got, msg) {
		t.Fatalf("%q %v", got, err)
	}
	// Another device's keys can't read it.
	other, _ := newWebPushKey()
	if _, err := other.decrypt(enc); err == nil {
		t.Error("decrypted with the wrong key")
	}
	if _, err := k.decrypt([]byte("plain text")); err != errNotWebPush {
		t.Errorf("plain text: %v", err)
	}
}

// TestWebPushRFC8291 decrypts the example message of RFC 8291, section 5.
func TestWebPushRFC8291(t *testing.T) {
	dec := func(s string) []byte {
		b, err := base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	priv, err := ecdhP256Private(dec("q1dXpw3UpT5VOmu_cf_v6ih07Aems3njxI-JWgLcM94"))
	if err != nil {
		t.Fatal(err)
	}
	k := webPushKey{private: priv, auth: dec("BTBZMqHH6r4Tts7J_aSIgg")}
	body := dec("DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN")
	got, err := k.decrypt(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "When I grow up, I want to be a watermelon" {
		t.Errorf("%q", got)
	}
}
