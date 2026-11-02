package review

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework StoreKit

void giodeRequestReview(void);
*/
import "C"

import (
	"context"

	"github.com/xmhrnx/giode"
)

func request(ctx context.Context, a *giode.App) error {
	C.giodeRequestReview()
	return nil
}
