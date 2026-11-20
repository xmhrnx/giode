package signin

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"gioui.org/app"
	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode/auth"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/webauthn"
)

// webauthn.dll, Windows 10 1903 and later: Windows Hello passkeys,
// security keys, and phones over hybrid on Windows 11.
var (
	webauthnDLL           = windows.NewLazySystemDLL("webauthn.dll")
	procWAMakeCredential  = webauthnDLL.NewProc("WebAuthNAuthenticatorMakeCredential")
	procWAGetAssertion    = webauthnDLL.NewProc("WebAuthNAuthenticatorGetAssertion")
	procWAFreeAttestation = webauthnDLL.NewProc("WebAuthNFreeCredentialAttestation")
	procWAFreeAssertion   = webauthnDLL.NewProc("WebAuthNFreeAssertion")
	procWACancellationID  = webauthnDLL.NewProc("WebAuthNGetCancellationId")
	procWACancel          = webauthnDLL.NewProc("WebAuthNCancelCurrentOperation")
	procWAErrorName       = webauthnDLL.NewProc("WebAuthNGetErrorName")
)

// The structs of webauthn.h.
type (
	waRPEntity struct {
		Version        uint32
		ID, Name, Icon *uint16
	}
	waUserEntity struct {
		Version                 uint32
		IDLen                   uint32
		ID                      *byte
		Name, Icon, DisplayName *uint16
	}
	waClientData struct {
		Version   uint32
		JSONLen   uint32
		JSON      *byte
		HashAlgID *uint16
	}
	waCoseParam struct {
		Version uint32
		Type    *uint16
		Alg     int32
	}
	waCoseParams struct {
		Count  uint32
		Params *waCoseParam
	}
	waCredential struct {
		Version uint32
		IDLen   uint32
		ID      *byte
		Type    *uint16
	}
	waCredentials struct {
		Count uint32
		Creds *waCredential
	}
	waExtensions struct {
		Count uint32
		Exts  unsafe.Pointer
	}
	waMakeOptions struct {
		Version            uint32
		TimeoutMS          uint32
		Exclude            waCredentials
		Extensions         waExtensions
		Attachment         uint32
		RequireResidentKey int32
		UserVerification   uint32
		Attestation        uint32
		Flags              uint32
		CancellationID     *windows.GUID
		ExcludeList        unsafe.Pointer
		EnterpriseAttest   uint32
		LargeBlobSupport   uint32
		PreferResidentKey  int32
	}
	waGetOptions struct {
		Version          uint32
		TimeoutMS        uint32
		Allow            waCredentials
		Extensions       waExtensions
		Attachment       uint32
		UserVerification uint32
		Flags            uint32
		U2fAppID         *uint16
		U2fAppIDUsed     *int32
		CancellationID   *windows.GUID
		AllowList        unsafe.Pointer
		LargeBlobOp      uint32
		LargeBlobLen     uint32
		LargeBlob        *byte
	}
	waAttestation struct {
		Version              uint32
		FormatType           *uint16
		AuthDataLen          uint32
		AuthData             *byte
		AttestationLen       uint32
		Attestation          *byte
		DecodeType           uint32
		Decoded              unsafe.Pointer
		AttestationObjectLen uint32
		AttestationObject    *byte
		CredentialIDLen      uint32
		CredentialID         *byte
		// Later versions add fields, not read here.
	}
	waAssertion struct {
		Version     uint32
		AuthDataLen uint32
		AuthData    *byte
		SigLen      uint32
		Sig         *byte
		Credential  waCredential
		UserIDLen   uint32
		UserID      *byte
		// Later versions add fields, not read here.
	}
)

var (
	waPublicKey = windows.StringToUTF16Ptr("public-key")
	waSHA256    = windows.StringToUTF16Ptr("SHA-256")
)

func waUserVerification(s string) uint32 {
	switch s {
	case "required":
		return 1
	case "discouraged":
		return 3
	}
	return 2 // preferred
}

func waAttachment(s string) uint32 {
	switch s {
	case "platform":
		return 1
	case "cross-platform":
		return 2
	}
	return 0
}

func waCredList(ids [][]byte) (waCredentials, []waCredential) {
	if len(ids) == 0 {
		return waCredentials{}, nil
	}
	cs := make([]waCredential, len(ids))
	for i, id := range ids {
		cs[i] = waCredential{Version: 1, IDLen: uint32(len(id)), ID: &id[0], Type: waPublicKey}
	}
	return waCredentials{Count: uint32(len(cs)), Creds: &cs[0]}, cs
}

func utf16OrNil(s string) *uint16 {
	if s == "" {
		return nil
	}
	p, _ := windows.UTF16PtrFromString(s)
	return p
}

func passkeyHWND(h platform.Host) uintptr {
	if ev, ok := h.NativeView().(app.Win32ViewEvent); ok {
		return ev.HWND
	}
	return 0
}

// waCall runs a WebAuthn ceremony, cancelling it when ctx ends.
func waCall(ctx context.Context, cancelID *windows.GUID, call func() uintptr) error {
	if err := webauthnDLL.Load(); err != nil {
		return auth.ErrUnsupported
	}
	done := make(chan uintptr, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		done <- call()
	}()
	var hr uintptr
	select {
	case hr = <-done:
	case <-ctx.Done():
		procWACancel.Call(uintptr(unsafe.Pointer(cancelID)))
		<-done
		return ctx.Err()
	}
	return waError(hr)
}

func waError(hr uintptr) error {
	if hr == 0 {
		return nil
	}
	switch uint32(hr) {
	case 0x80090036: // NTE_USER_CANCELLED: closed, or timed out
		return auth.ErrCanceled
	case 0x80090011: // NTE_NOT_FOUND
		return auth.ErrNoPasskey
	}
	name := ""
	if p, _, _ := procWAErrorName.Call(hr); p != 0 {
		name = windows.UTF16PtrToString(*(**uint16)(unsafe.Pointer(&p)))
	}
	if name == "NotAllowedError" {
		return auth.ErrCanceled
	}
	return fmt.Errorf("passkey: %s (0x%08x)", name, uint32(hr))
}

func PasskeyCreate(ctx context.Context, h platform.Host, c webauthn.Creation, _ []byte) ([]byte, error) {
	userID, _ := webauthn.Decode(c.User.ID)
	if len(userID) == 0 {
		return nil, errors.New("passkey: empty user.id")
	}
	clientJSON := webauthn.ClientData("webauthn.create", c.Challenge, c.RP.ID)
	rp := waRPEntity{Version: 1, ID: utf16OrNil(c.RP.ID), Name: utf16OrNil(c.RP.Name)}
	if rp.Name == nil {
		rp.Name = rp.ID
	}
	user := waUserEntity{Version: 1, IDLen: uint32(len(userID)), ID: &userID[0],
		Name: utf16OrNil(c.User.Name), DisplayName: utf16OrNil(c.User.DisplayName)}
	if user.DisplayName == nil {
		user.DisplayName = user.Name
	}
	var params []waCoseParam
	for _, p := range c.PubKeyCredParams {
		if p.Type == "" || p.Type == "public-key" {
			params = append(params, waCoseParam{Version: 1, Type: waPublicKey, Alg: int32(p.Alg)})
		}
	}
	if len(params) == 0 {
		params = []waCoseParam{{1, waPublicKey, -7}, {1, waPublicKey, -257}}
	}
	cose := waCoseParams{Count: uint32(len(params)), Params: &params[0]}
	cd := waClientData{Version: 1, JSONLen: uint32(len(clientJSON)), JSON: &clientJSON[0], HashAlgID: waSHA256}
	exclude, excludeCreds := waCredList(webauthn.IDs(c.ExcludeCredentials))
	var cancelID windows.GUID
	procWACancellationID.Call(uintptr(unsafe.Pointer(&cancelID)))
	sel := c.AuthenticatorSelection
	opts := waMakeOptions{
		Version: 3, TimeoutMS: uint32(c.Timeout), Exclude: exclude,
		Attachment:         waAttachment(sel.Attachment),
		RequireResidentKey: boolInt32(sel.RequireResidentKey || sel.ResidentKey == "required" || sel.ResidentKey == "preferred"),
		UserVerification:   waUserVerification(sel.UserVerification),
		Attestation:        map[string]uint32{"direct": 3, "indirect": 2, "enterprise": 3}[c.Attestation],
		CancellationID:     &cancelID,
	}
	if opts.Attestation == 0 {
		opts.Attestation = 1 // none
	}
	if opts.TimeoutMS == 0 {
		opts.TimeoutMS = 300000
	}
	var out *waAttestation
	hwnd := passkeyHWND(h)
	err := waCall(ctx, &cancelID, func() uintptr {
		hr, _, _ := procWAMakeCredential.Call(hwnd, uintptr(unsafe.Pointer(&rp)), uintptr(unsafe.Pointer(&user)),
			uintptr(unsafe.Pointer(&cose)), uintptr(unsafe.Pointer(&cd)), uintptr(unsafe.Pointer(&opts)),
			uintptr(unsafe.Pointer(&out)))
		return hr
	})
	runtime.KeepAlive(userID)
	runtime.KeepAlive(params)
	runtime.KeepAlive(clientJSON)
	runtime.KeepAlive(excludeCreds)
	if err != nil {
		return nil, err
	}
	defer procWAFreeAttestation.Call(uintptr(unsafe.Pointer(out)))
	return webauthn.Registration{
		CredentialID:      copyBytes(out.CredentialID, out.CredentialIDLen),
		ClientDataJSON:    clientJSON,
		AttestationObject: copyBytes(out.AttestationObject, out.AttestationObjectLen),
		Transports:        []string{"internal", "hybrid"},
	}.JSON(), nil
}

func PasskeyGet(ctx context.Context, h platform.Host, r webauthn.Request, _ []byte) ([]byte, error) {
	clientJSON := webauthn.ClientData("webauthn.get", r.Challenge, r.RPID)
	cd := waClientData{Version: 1, JSONLen: uint32(len(clientJSON)), JSON: &clientJSON[0], HashAlgID: waSHA256}
	allow, allowCreds := waCredList(webauthn.IDs(r.AllowCredentials))
	var cancelID windows.GUID
	procWACancellationID.Call(uintptr(unsafe.Pointer(&cancelID)))
	opts := waGetOptions{
		Version: 3, TimeoutMS: uint32(r.Timeout), Allow: allow,
		UserVerification: waUserVerification(r.UserVerification),
		CancellationID:   &cancelID,
	}
	if opts.TimeoutMS == 0 {
		opts.TimeoutMS = 300000
	}
	rpID := utf16OrNil(r.RPID)
	var out *waAssertion
	hwnd := passkeyHWND(h)
	err := waCall(ctx, &cancelID, func() uintptr {
		hr, _, _ := procWAGetAssertion.Call(hwnd, uintptr(unsafe.Pointer(rpID)), uintptr(unsafe.Pointer(&cd)),
			uintptr(unsafe.Pointer(&opts)), uintptr(unsafe.Pointer(&out)))
		return hr
	})
	runtime.KeepAlive(clientJSON)
	runtime.KeepAlive(allowCreds)
	if err != nil {
		return nil, err
	}
	defer procWAFreeAssertion.Call(uintptr(unsafe.Pointer(out)))
	return webauthn.Assertion{
		CredentialID:      copyBytes(out.Credential.ID, out.Credential.IDLen),
		ClientDataJSON:    clientJSON,
		AuthenticatorData: copyBytes(out.AuthData, out.AuthDataLen),
		Signature:         copyBytes(out.Sig, out.SigLen),
		UserHandle:        copyBytes(out.UserID, out.UserIDLen),
	}.JSON(), nil
}

func copyBytes(p *byte, n uint32) []byte {
	if p == nil || n == 0 {
		return nil
	}
	return append([]byte(nil), unsafe.Slice(p, n)...)
}

func boolInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
