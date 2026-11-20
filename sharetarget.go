package giode

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/sharetarget"
)

// SharedContent is what another app shared with this one through the
// system share sheet: files, text (a note, a link) or both. See
// Config.OnShareReceived.
type SharedContent struct {
	// Files are the shared files, already open: Giode opens them as
	// they arrive, while Android's permission to read them lasts. Their
	// Path is empty. Close each one.
	Files []PickedFile
	// Text is shared text or a link; Subject a title some apps add
	// (an email's subject).
	Text, Subject string
}

// listenShares hands what other apps share with this one to
// OnShareReceived.
func (a *App) listenShares() {
	err := sharetarget.Listen(func(s mobile.Shared) {
		c := SharedContent{Text: s.Text, Subject: s.Subject}
		for _, f := range s.Files {
			c.Files = append(c.Files, sharedFile(f.File, f.Name, f.MIME, f.Size))
		}
		a.shareReceived(c)
	})
	if err != nil {
		a.logf(LogLevelWarning, "giode: can't receive shares: %v", err)
	}
}

// shareReceived hands a share to OnShareReceived on the UI goroutine.
func (a *App) shareReceived(s SharedContent) {
	if a.cfg.OnShareReceived == nil {
		a.logf(LogLevelWarning, "giode: received a share but Config.OnShareReceived isn't set")
		for _, f := range s.Files {
			f.Close()
		}
		return
	}
	a.Post(func() { a.cfg.OnShareReceived(s) })
}

// sharedFile describes a shared file opened by the platform. Its
// name's extension, then the platform's type, then the content decide
// MIME.
func sharedFile(f *os.File, name, mimeType string, size int64) PickedFile {
	p := PickedFile{ReadCloser: f, Name: filepath.Base(name), MIME: mimeType, Size: size}
	if p.Name == "." || p.Name == "/" {
		p.Name = ""
	}
	if t := platform.MIMEType(filepath.Ext(p.Name)); t != "" {
		p.MIME = t
	}
	if p.MIME == "" || p.MIME == "application/octet-stream" {
		br := bufio.NewReaderSize(f, 512)
		head, _ := br.Peek(512)
		p.MIME = http.DetectContentType(head)
		p.ReadCloser = struct {
			io.Reader
			io.Closer
		}{br, f}
	}
	if p.Size < 0 {
		if st, err := f.Stat(); err == nil && st.Mode().IsRegular() {
			p.Size = st.Size()
		}
	}
	return p
}
