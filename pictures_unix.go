//go:build !windows && !android && !ios && !js

package giode

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// picturesDir is the user's Pictures folder: XDG_PICTURES_DIR on Linux.
func picturesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS != "darwin" {
		if d := xdgUserDir(home, "XDG_PICTURES_DIR"); d != "" {
			return d, nil
		}
	}
	return filepath.Join(home, "Pictures"), nil
}

// downloadsDir is the user's Downloads folder: XDG_DOWNLOAD_DIR on
// Linux.
func downloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS != "darwin" {
		if d := xdgUserDir(home, "XDG_DOWNLOAD_DIR"); d != "" {
			return d, nil
		}
	}
	return filepath.Join(home, "Downloads"), nil
}

// xdgUserDir reads a folder from ~/.config/user-dirs.dirs.
func xdgUserDir(home, key string) string {
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = filepath.Join(home, ".config")
	}
	f, err := os.Open(filepath.Join(cfg, "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if !ok || k != key {
			continue
		}
		v = strings.Trim(v, `"`)
		v = strings.ReplaceAll(v, "$HOME", home)
		if v == home || v == "" {
			return ""
		}
		return v
	}
	return ""
}
