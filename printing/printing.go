// Package printing prints documents through the system's print dialog:
//
//	err := printing.Print(ctx, app, "scan.pdf", printing.Options{JobName: "Receipts"})
//
// Android uses its print service (PDF only); iOS the print panel (PDF
// or an image); macOS the print panel (PDF or an image); Windows hands
// the file to its default app's Print command. On Linux and the web it
// returns giode.ErrNotSupported.
package printing

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmhrnx/giode"
)

// Options configure a print job.
type Options struct {
	// JobName names the job in the print queue; the file's name when
	// empty.
	JobName string
}

// Supported reports whether this platform can print.
func Supported() bool { return supported }

// Print opens the print dialog for the PDF (or, where supported, image)
// at path. It returns once the dialog closes, nil also when the user
// canceled: the platforms don't all say. Call it from a goroutine.
func Print(ctx context.Context, a *giode.App, path string, opts Options) error {
	if !supported {
		return giode.ErrNotSupported
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return err
	}
	if opts.JobName == "" {
		opts.JobName = strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	}
	return printFile(ctx, a, abs, opts)
}

// isPDF reports whether path names a PDF.
func isPDF(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".pdf")
}
