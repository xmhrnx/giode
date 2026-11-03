package giode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// The share extension (iOS) leaves each share in a folder of its own in
// the inbox: the files, then share.json describing them, written last.

// inboxShare is share.json.
type inboxShare struct {
	Text    string `json:"text"`
	Subject string `json:"subject"`
	Files   []struct {
		// File is the file's name in the share's folder; Name the
		// name to show.
		File string `json:"file"`
		Name string `json:"name"`
		MIME string `json:"mime"`
	} `json:"files"`
}

var inboxMu sync.Mutex

// collectShares delivers the complete shares in inbox to
// OnShareReceived, oldest first, and removes them: their files stay
// readable through the open handles. Folders left without share.json
// for an hour (an extension that didn't finish) are removed.
func (a *App) collectShares(inbox string) {
	inboxMu.Lock()
	defer inboxMu.Unlock()
	entries, err := os.ReadDir(inbox)
	if err != nil {
		return
	}
	type found struct {
		dir string
		mod time.Time
	}
	var shares []found
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(inbox, e.Name())
		st, err := os.Stat(filepath.Join(dir, "share.json"))
		if err != nil {
			if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > time.Hour {
				os.RemoveAll(dir)
			}
			continue
		}
		shares = append(shares, found{dir, st.ModTime()})
	}
	for i := 1; i < len(shares); i++ {
		for j := i; j > 0 && shares[j].mod.Before(shares[j-1].mod); j-- {
			shares[j], shares[j-1] = shares[j-1], shares[j]
		}
	}
	for _, s := range shares {
		c, ok := readInboxShare(s.dir)
		os.RemoveAll(s.dir)
		if ok {
			a.shareReceived(c)
		}
	}
}

// readInboxShare opens a share's files.
func readInboxShare(dir string) (SharedContent, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "share.json"))
	if err != nil {
		return SharedContent{}, false
	}
	var in inboxShare
	if err := json.Unmarshal(data, &in); err != nil {
		return SharedContent{}, false
	}
	c := SharedContent{Text: in.Text, Subject: in.Subject}
	for _, f := range in.Files {
		if f.File == "" || f.File != filepath.Base(f.File) {
			continue
		}
		file, err := os.Open(filepath.Join(dir, f.File))
		if err != nil {
			continue
		}
		name := f.Name
		if name == "" {
			name = f.File
		}
		c.Files = append(c.Files, sharedFile(file, name, f.MIME, -1))
	}
	if c.Text == "" && c.Subject == "" && len(c.Files) == 0 {
		return c, false
	}
	return c, true
}
