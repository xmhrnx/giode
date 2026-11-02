// Package pdfpages renders the pages of a PDF to images, to import an
// existing PDF into a scanner or show thumbnails of it:
//
//	doc, err := pdfpages.Open("contract.pdf")
//	defer doc.Close()
//	for i := 0; i < doc.PageCount(); i++ {
//		img, err := doc.Render(i, 1600) // 1600 pixels wide
//		...
//	}
//
// Android uses PdfRenderer; iOS and macOS use Core Graphics' PDF
// support. Elsewhere Open returns giode.ErrNotSupported. Password
// protected PDFs aren't supported.
package pdfpages

import (
	"errors"
	"image"
	"sync"

	"github.com/xmhrnx/giode"
)

// ErrClosed is returned by a closed Doc's methods.
var ErrClosed = errors.New("pdfpages: closed")

// Doc is an open PDF. Its methods may be called from any goroutine;
// pages render one at a time.
type Doc struct {
	mu     sync.Mutex
	plat   platformDoc
	pages  int
	closed bool
}

// Supported reports whether this platform can render PDFs.
func Supported() bool { return supported }

// Open opens the PDF at path.
func Open(path string) (*Doc, error) {
	if !supported {
		return nil, giode.ErrNotSupported
	}
	p, n, err := open(path)
	if err != nil {
		return nil, err
	}
	return &Doc{plat: p, pages: n}, nil
}

// PageCount returns the number of pages.
func (d *Doc) PageCount() int { return d.pages }

// PageSize returns page i's size in points (1/72 inch), as it's shown:
// rotated pages report their rotated size.
func (d *Doc) PageSize(i int) (w, h float64, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return 0, 0, ErrClosed
	}
	if i < 0 || i >= d.pages {
		return 0, 0, errors.New("pdfpages: no such page")
	}
	return d.plat.pageSize(i)
}

// Render renders page i (from 0) width pixels wide, its height in
// proportion, on a white background.
func (d *Doc) Render(i, width int) (*image.RGBA, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil, ErrClosed
	}
	if i < 0 || i >= d.pages {
		return nil, errors.New("pdfpages: no such page")
	}
	if width <= 0 || width > 10000 {
		return nil, errors.New("pdfpages: width out of range")
	}
	pw, ph, err := d.plat.pageSize(i)
	if err != nil {
		return nil, err
	}
	if pw <= 0 || ph <= 0 {
		return nil, errors.New("pdfpages: empty page")
	}
	height := max(1, int(float64(width)*ph/pw+0.5))
	pix, err := d.plat.render(i, width, height)
	if err != nil {
		return nil, err
	}
	if len(pix) != width*height*4 {
		return nil, errors.New("pdfpages: renderer returned the wrong size")
	}
	return &image.RGBA{Pix: pix, Stride: width * 4, Rect: image.Rect(0, 0, width, height)}, nil
}

// Close releases the document.
func (d *Doc) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	return d.plat.close()
}
