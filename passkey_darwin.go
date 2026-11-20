//go:build darwin

package giode

/*
#include <stdint.h>
#include <stdlib.h>

void giodePasskeyStart(uintptr_t view, int id, int create, char *rpID, void *challenge, int challengeLen,
	char *userName, void *userID, int userIDLen, void *ids, int *idLens, int idCount, char *uv);
*/
import "C"

import (
	"context"
	"errors"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode/auth"
	"github.com/xmhrnx/giode/internal/webauthn"
)

// passkeyDone is how an ASAuthorizationController request ended.
type passkeyDone struct {
	kind int // 0 registered, 1 asserted, 2 canceled, 3 no passkey, 4 error
	bufs [][]byte
	msg  string
}

var passkeyReqs = struct {
	sync.Mutex
	next int
	done map[int]chan passkeyDone
}{done: map[int]chan passkeyDone{}}

//export giodePasskeyDone
func giodePasskeyDone(id, kind C.int, bufs *unsafe.Pointer, lens *C.int, n C.int, msg *C.char) {
	passkeyReqs.Lock()
	ch := passkeyReqs.done[int(id)]
	delete(passkeyReqs.done, int(id))
	passkeyReqs.Unlock()
	if ch == nil {
		return
	}
	d := passkeyDone{kind: int(kind), msg: C.GoString(msg)}
	if n > 0 {
		ps := unsafe.Slice(bufs, int(n))
		ls := unsafe.Slice(lens, int(n))
		for i := range ps {
			if ps[i] == nil || ls[i] == 0 {
				d.bufs = append(d.bufs, nil)
				continue
			}
			d.bufs = append(d.bufs, C.GoBytes(ps[i], ls[i]))
		}
	}
	ch <- d
}

func (a *App) passkeyCreate(ctx context.Context, c webauthn.Creation, _ []byte) ([]byte, error) {
	challenge, _ := webauthn.Decode(c.Challenge)
	userID, _ := webauthn.Decode(c.User.ID)
	name := c.User.Name
	if name == "" {
		name = c.User.DisplayName
	}
	d, err := a.applePasskey(ctx, true, c.RP.ID, challenge, name, userID,
		webauthn.IDs(c.ExcludeCredentials), c.AuthenticatorSelection.UserVerification)
	if err != nil {
		return nil, err
	}
	// credentialID, clientDataJSON, attestationObject
	return webauthn.Registration{CredentialID: d.bufs[0], ClientDataJSON: d.bufs[1], AttestationObject: d.bufs[2]}.JSON(), nil
}

func (a *App) passkeyGet(ctx context.Context, r webauthn.Request, _ []byte) ([]byte, error) {
	challenge, _ := webauthn.Decode(r.Challenge)
	d, err := a.applePasskey(ctx, false, r.RPID, challenge, "", nil, webauthn.IDs(r.AllowCredentials), r.UserVerification)
	if err != nil {
		return nil, err
	}
	// credentialID, clientDataJSON, authenticatorData, signature, userID
	return webauthn.Assertion{CredentialID: d.bufs[0], ClientDataJSON: d.bufs[1], AuthenticatorData: d.bufs[2],
		Signature: d.bufs[3], UserHandle: d.bufs[4]}.JSON(), nil
}

func (a *App) applePasskey(ctx context.Context, create bool, rpID string, challenge []byte, user string, userID []byte, ids [][]byte, uv string) (passkeyDone, error) {
	ch := make(chan passkeyDone, 1)
	passkeyReqs.Lock()
	passkeyReqs.next++
	id := passkeyReqs.next
	passkeyReqs.done[id] = ch
	passkeyReqs.Unlock()

	// Everything goes in C memory: the request outlives this call.
	cstrs := []*C.char{C.CString(rpID), C.CString(user), C.CString(uv)}
	defer func() {
		for _, s := range cstrs {
			C.free(unsafe.Pointer(s))
		}
	}()
	cbytes := func(b []byte) unsafe.Pointer {
		if len(b) == 0 {
			return nil
		}
		return C.CBytes(b)
	}
	ch0, uid := cbytes(challenge), cbytes(userID)
	var idBuf unsafe.Pointer
	var lens *C.int
	if len(ids) > 0 {
		var all []byte
		ls := make([]C.int, len(ids))
		for i, b := range ids {
			all = append(all, b...)
			ls[i] = C.int(len(b))
		}
		idBuf = C.CBytes(all)
		lens = (*C.int)(C.CBytes(unsafe.Slice((*byte)(unsafe.Pointer(&ls[0])), len(ls)*int(unsafe.Sizeof(ls[0])))))
	}
	defer func() {
		for _, p := range []unsafe.Pointer{ch0, uid, idBuf, unsafe.Pointer(lens)} {
			if p != nil {
				C.free(p)
			}
		}
	}()
	cr := C.int(0)
	if create {
		cr = 1
	}
	// giodePasskeyStart copies what it needs before returning.
	C.giodePasskeyStart(C.uintptr_t(a.authAnchorView()), C.int(id), cr, cstrs[0], ch0, C.int(len(challenge)),
		cstrs[1], uid, C.int(len(userID)), idBuf, lens, C.int(len(ids)), cstrs[2])
	var d passkeyDone
	select {
	case d = <-ch:
	case <-ctx.Done():
		return d, ctx.Err()
	}
	switch d.kind {
	case 0, 1:
		if want := map[int]int{0: 3, 1: 5}[d.kind]; len(d.bufs) != want {
			return d, errors.New("passkey: incomplete response")
		}
		return d, nil
	case 2:
		return d, auth.ErrCanceled
	case 3:
		return d, auth.ErrNoPasskey
	case 5:
		return d, auth.ErrUnsupported
	}
	return d, errors.New("passkey: " + d.msg)
}
