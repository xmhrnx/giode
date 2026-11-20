package winrt

import "testing"

func TestSignatureGUID(t *testing.T) {
	// IVector<HSTRING>, a known parameterized IID.
	g := SignatureGUID("pinterface({913337e9-11a1-4345-a3a2-4e7f956e222d};string)")
	if got := guidString(g); got != "98b9acc1-4b56-532e-ac73-03d5291cca90" {
		t.Errorf("IVector<string> = %s", got)
	}
	// IAsyncOperation<bool>.
	g = SignatureGUID("pinterface({9fc2b0bb-e446-44e2-aa61-9cab8f636af2};b1)")
	if got := guidString(g); got != "cdb5efb3-5788-509d-9be1-71ccb8a3362a" {
		t.Errorf("IAsyncOperation<bool> = %s", got)
	}
}

func TestParseGUID(t *testing.T) {
	g := ParseGUID("{50ac103f-d235-4598-bbef-98fe4d1a3ad4}")
	if guidString(g) != "50ac103f-d235-4598-bbef-98fe4d1a3ad4" {
		t.Error(guidString(g))
	}
	if NameGUID("a") == NameGUID("b") || NameGUID("a") != NameGUID("a") {
		t.Error("NameGUID")
	}
}
