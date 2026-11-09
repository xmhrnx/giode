package store

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework StoreKit -framework Foundation

#include <stdlib.h>

void giodeStoreProducts(int id, const char *ids);
void giodeStoreBuy(int id, const char *product);
void giodeStoreFinish(int id, const char *token);
void giodeStoreRestore(int id);
int giodeStoreBillingRequired(void);
*/
import "C"

import (
	"context"
	"encoding/json"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/bridge"
)

var requests bridge.Requests[result]

func wait(ctx context.Context, start func(id C.int)) (string, error) {
	id, ch := requests.New()
	start(C.int(id))
	r, err := requests.Wait(ctx, id, ch)
	if err != nil {
		return "", err
	}
	return r.json, r.err
}

func cstring(s string) (*C.char, func()) {
	c := C.CString(s)
	return c, func() { C.free(unsafe.Pointer(c)) }
}

func products(ctx context.Context, ids []string) ([]Product, error) {
	b, _ := json.Marshal(ids)
	cs, free := cstring(string(b))
	defer free()
	js, err := wait(ctx, func(id C.int) { C.giodeStoreProducts(id, cs) })
	if err != nil {
		return nil, err
	}
	return parseProducts(js)
}

func purchase(ctx context.Context, a *giode.App, p Product) (Purchase, error) {
	cs, free := cstring(p.ID)
	defer free()
	js, err := wait(ctx, func(id C.int) { C.giodeStoreBuy(id, cs) })
	if err != nil {
		return Purchase{}, err
	}
	return parsePurchase(js)
}

func finish(ctx context.Context, p Purchase, consume bool) error {
	cs, free := cstring(p.Token)
	defer free()
	_, err := wait(ctx, func(id C.int) { C.giodeStoreFinish(id, cs) })
	return err
}

func owned(ctx context.Context) ([]Purchase, error) {
	js, err := wait(ctx, func(id C.int) { C.giodeStoreRestore(id) })
	if err != nil {
		return nil, err
	}
	return parsePurchases(js)
}

// startUpdates does nothing: the transaction observer is added at
// launch, as StoreKit wants.
func startUpdates() {}

//export giodeStoreResult
func giodeStoreResult(id C.int, js, errMsg *C.char) {
	r := result{json: C.GoString(js)}
	if errMsg != nil {
		r.err = nativeError(C.GoString(errMsg))
	}
	requests.Done(int(id), r)
}

//export giodeStoreUpdate
func giodeStoreUpdate(js *C.char) {
	if p, err := parsePurchase(C.GoString(js)); err == nil {
		update(p)
	}
}

func billingRequired() bool { return C.giodeStoreBillingRequired() != 0 }
