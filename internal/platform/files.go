package platform

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// UniquePath adds " (2)", " (3)" ... before the extension until path is
// free.
func UniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		p := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
	}
}

// CopyFile copies src to a new file dst.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}

// SanitizeFileName replaces the characters file systems reject.
func SanitizeFileName(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) || r < ' ' {
			return '-'
		}
		return r
	}, s)
}

// KnownTypes covers common files Go's built-in MIME table lacks; the
// system table that fills the gap on Linux isn't there on Windows or
// the phones.
var KnownTypes = map[string]string{
	".md":       "text/markdown; charset=utf-8",
	".markdown": "text/markdown; charset=utf-8",
	".txt":      "text/plain; charset=utf-8",
	".csv":      "text/csv; charset=utf-8",
	".heic":     "image/heic",
	".heif":     "image/heif",
	".mp3":      "audio/mpeg",
	".wav":      "audio/wav",
	".m4a":      "audio/mp4",
	".ogg":      "audio/ogg",
	".flac":     "audio/flac",
	".mp4":      "video/mp4",
	".mov":      "video/quicktime",
	".zip":      "application/zip",
	".docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xlsx":     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".pptx":     "application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

// MIMEType returns the MIME type for a file extension, or "".
func MIMEType(ext string) string {
	ext = strings.ToLower(ext)
	if t, ok := KnownTypes[ext]; ok {
		return t
	}
	return mime.TypeByExtension(ext)
}
