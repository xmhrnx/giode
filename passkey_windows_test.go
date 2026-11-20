package giode

import (
	"testing"
	"unsafe"

	"github.com/xmhrnx/giode/auth"
)

// TestWebAuthnLayout checks the structs against webauthn.h's sizes and
// offsets (from mingw-w64's header).
func TestWebAuthnLayout(t *testing.T) {
	want := map[uintptr][]uintptr{
		8: {32, 40, 24, 24, 16, 24, 16, 96, 64, 104, 72, 64, 88, 32, 64},
		4: {16, 24, 16, 12, 8, 16, 8, 64, 44, 64, 44, 32, 44, 20, 40},
	}[unsafe.Sizeof(uintptr(0))]
	var mo waMakeOptions
	var gopt waGetOptions
	var at waAttestation
	var as waAssertion
	got := []uintptr{
		unsafe.Sizeof(waRPEntity{}), unsafe.Sizeof(waUserEntity{}), unsafe.Sizeof(waClientData{}),
		unsafe.Sizeof(waCoseParam{}), unsafe.Sizeof(waCoseParams{}), unsafe.Sizeof(waCredential{}), unsafe.Sizeof(waCredentials{}),
		unsafe.Sizeof(mo), unsafe.Offsetof(mo.CancellationID),
		unsafe.Sizeof(gopt), unsafe.Offsetof(gopt.CancellationID),
		unsafe.Offsetof(at.AttestationObjectLen), unsafe.Offsetof(at.CredentialID),
		unsafe.Offsetof(as.Credential), unsafe.Offsetof(as.UserID),
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("layout %d: %d, want %d", i, got[i], want[i])
		}
	}
}

func TestWebAuthnErrors(t *testing.T) {
	if waError(0x80090036) != auth.ErrCanceled || waError(0x80090011) != auth.ErrNoPasskey || waError(0) != nil {
		t.Error("error mapping")
	}
	if err := webauthnDLL.Load(); err == nil {
		v, _, _ := webauthnDLL.NewProc("WebAuthNGetApiVersionNumber").Call()
		t.Logf("webauthn.dll API version %d", v)
	}
}
