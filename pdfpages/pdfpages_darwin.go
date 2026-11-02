package pdfpages

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation

#include <stdlib.h>
#include <stdint.h>
#include <CoreGraphics/CoreGraphics.h>

// giodePDFOpen opens a PDF, returning NULL on failure.
static CGPDFDocumentRef giodePDFOpen(const char *path) {
	CFURLRef url = CFURLCreateFromFileSystemRepresentation(NULL, (const UInt8 *)path, strlen(path), false);
	if (url == NULL) {
		return NULL;
	}
	CGPDFDocumentRef doc = CGPDFDocumentCreateWithURL(url);
	CFRelease(url);
	if (doc != NULL && CGPDFDocumentIsEncrypted(doc) && !CGPDFDocumentIsUnlocked(doc)) {
		CGPDFDocumentRelease(doc);
		return NULL;
	}
	return doc;
}

// giodePDFPageSize returns page i's (from 1) shown size in points.
static void giodePDFPageSize(CGPDFDocumentRef doc, int i, double *w, double *h) {
	CGPDFPageRef page = CGPDFDocumentGetPage(doc, i);
	*w = *h = 0;
	if (page == NULL) {
		return;
	}
	CGRect r = CGPDFPageGetBoxRect(page, kCGPDFCropBox);
	int rot = CGPDFPageGetRotationAngle(page);
	if (rot % 180 != 0) {
		*w = r.size.height;
		*h = r.size.width;
	} else {
		*w = r.size.width;
		*h = r.size.height;
	}
}

// giodePDFRender draws page i (from 1) into pix, w by h RGBA pixels, on
// white.
static int giodePDFRender(CGPDFDocumentRef doc, int i, void *pix, int w, int h) {
	CGPDFPageRef page = CGPDFDocumentGetPage(doc, i);
	if (page == NULL) {
		return 0;
	}
	CGColorSpaceRef cs = CGColorSpaceCreateDeviceRGB();
	CGContextRef ctx = CGBitmapContextCreate(pix, w, h, 8, w * 4, cs, kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
	CGColorSpaceRelease(cs);
	if (ctx == NULL) {
		return 0;
	}
	CGContextSetRGBFillColor(ctx, 1, 1, 1, 1);
	CGContextFillRect(ctx, CGRectMake(0, 0, w, h));
	CGContextSetInterpolationQuality(ctx, kCGInterpolationHigh);
	CGContextSetRenderingIntent(ctx, kCGRenderingIntentDefault);
	// Scale the page's crop box, rotated as shown, to the bitmap.
	CGAffineTransform t = CGPDFPageGetDrawingTransform(page, kCGPDFCropBox, CGRectMake(0, 0, w, h), 0, true);
	// GetDrawingTransform never scales up: do the rest ourselves.
	double pw, ph;
	giodePDFPageSize(doc, i, &pw, &ph);
	if (pw > 0 && ph > 0 && (w > pw || h > ph)) {
		CGContextScaleCTM(ctx, w / pw, h / ph);
		t = CGPDFPageGetDrawingTransform(page, kCGPDFCropBox, CGRectMake(0, 0, pw, ph), 0, true);
	}
	CGContextConcatCTM(ctx, t);
	CGContextDrawPDFPage(ctx, page);
	CGContextRelease(ctx);
	return 1;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

const supported = true

type platformDoc struct{ doc C.CGPDFDocumentRef }

func open(path string) (platformDoc, int, error) {
	cp := C.CString(path)
	defer C.free(unsafe.Pointer(cp))
	doc := C.giodePDFOpen(cp)
	if doc == nil {
		return platformDoc{}, 0, errors.New("pdfpages: can't open the PDF (missing, damaged or password protected)")
	}
	return platformDoc{doc: doc}, int(C.CGPDFDocumentGetNumberOfPages(doc)), nil
}

func (d platformDoc) pageSize(i int) (float64, float64, error) {
	var w, h C.double
	C.giodePDFPageSize(d.doc, C.int(i+1), &w, &h)
	return float64(w), float64(h), nil
}

func (d platformDoc) render(i, w, h int) ([]byte, error) {
	pix := make([]byte, w*h*4)
	if C.giodePDFRender(d.doc, C.int(i+1), unsafe.Pointer(&pix[0]), C.int(w), C.int(h)) == 0 {
		return nil, errors.New("pdfpages: can't render the page")
	}
	return pix, nil
}

func (d platformDoc) close() error {
	C.CGPDFDocumentRelease(d.doc)
	return nil
}
