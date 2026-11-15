package properties

import (
	"image"
	"sync"

	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/internal/linear"
)

// imageOps caches the paint.ImageOp of each image. A fresh ImageOp per
// frame makes Gio treat the image as new and upload its pixels to the
// GPU again; reusing one keeps the upload.
var (
	imageOpsMu sync.Mutex
	imageOps   = map[image.Image]paint.ImageOp{}
)

const maxCachedImages = 256

// ImageOp returns a cached paint.ImageOp for img. Images are keyed by
// identity, so an image whose pixels are modified in place keeps
// showing the old upload; call ForgetImage after mutating one.
func ImageOp(img image.Image) paint.ImageOp {
	imageOpsMu.Lock()
	defer imageOpsMu.Unlock()
	if op, ok := imageOps[img]; ok {
		return op
	}
	if len(imageOps) >= maxCachedImages {
		imageOps = map[image.Image]paint.ImageOp{}
	}
	// Partly transparent pixels premultiplied as the GPU blends.
	op := paint.NewImageOp(linear.Image(img))
	imageOps[img] = op
	return op
}

// ForgetImage drops img from the cache, so its pixels are uploaded
// again the next time it's painted.
func ForgetImage(img image.Image) {
	imageOpsMu.Lock()
	delete(imageOps, img)
	imageOpsMu.Unlock()
}
