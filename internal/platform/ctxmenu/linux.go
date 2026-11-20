//go:build linux && !android

package ctxmenu

import (
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Linux file managers each read their own extension files from the
// user's home: GNOME Files and Caja run scripts (listed under
// "Scripts"), Dolphin reads service menus, Nemo actions. Each file
// carries a marker naming the app, so uninstalling finds them.

func contextMarker(prefix string) string { return "giode-context-action:" + prefix + "." }

// contextDirs returns the directories the files go to.
func contextDirs() (nautilus, caja, dolphin, nemo string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(home, ".local", "share")
	}
	conf := os.Getenv("XDG_CONFIG_HOME")
	if conf == "" {
		conf = filepath.Join(home, ".config")
	}
	return filepath.Join(data, "nautilus", "scripts"),
		filepath.Join(conf, "caja", "scripts"),
		filepath.Join(data, "kio", "servicemenus"),
		filepath.Join(data, "nemo", "actions"), nil
}

func Install(prefix, exe string, actions []Action) error {
	Uninstall(prefix)
	nautilus, caja, dolphin, nemo, err := contextDirs()
	if err != nil {
		return err
	}
	var errs []error
	write := func(dir, name, content string) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			errs = append(errs, err)
			return
		}
		errs = append(errs, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o755))
	}
	q := strconv.Quote(exe)
	for _, c := range actions {
		marker := contextMarker(prefix) + c.ID
		id := prefix + "." + c.ID
		// GNOME Files and Caja: scripts get the selection in an
		// environment variable, a path per line. They can't filter by
		// type, so they show for everything selected.
		if c.OnFiles() || c.Folders {
			name := strings.ReplaceAll(c.Label, "/", "∕")
			for _, s := range []struct{ dir, env string }{{nautilus, "NAUTILUS_SCRIPT_SELECTED_FILE_PATHS"}, {caja, "CAJA_SCRIPT_SELECTED_FILE_PATHS"}} {
				write(s.dir, name, "#!/bin/sh\n# "+marker+"\nIFS='\n'\nexec "+q+" "+Flag+c.ID+" $"+s.env+"\n")
			}
		}
		// Dolphin.
		var types []string
		if c.OnFiles() {
			if len(c.Extensions) == 0 {
				types = append(types, "all/allfiles")
			}
			for _, ext := range c.Extensions {
				if t := mime.TypeByExtension(dotted(ext)); t != "" {
					types = append(types, strings.SplitN(t, ";", 2)[0])
				} else {
					types = append(types, "all/allfiles")
				}
			}
		}
		if c.Folders || c.FolderBackground {
			types = append(types, "inode/directory")
		}
		write(dolphin, id+".desktop", "[Desktop Entry]\n# "+marker+"\nType=Service\nX-KDE-ServiceTypes=KonqPopupMenu/Plugin\n"+
			"MimeType="+strings.Join(types, ";")+";\nActions=giode\n\n[Desktop Action giode]\nName="+c.Label+
			"\nExec="+q+" "+Flag+c.ID+" %F\n")
		// Nemo.
		exts := "nodirs"
		switch {
		case len(c.Extensions) > 0:
			var e []string
			for _, ext := range c.Extensions {
				e = append(e, strings.TrimPrefix(ext, "."))
			}
			exts = strings.Join(e, ";") + ";"
		case c.OnFiles() && c.Folders:
			exts = "any"
		case c.Folders && !c.OnFiles():
			exts = "dir"
		}
		if c.OnFiles() || c.Folders {
			write(nemo, id+".nemo_action", "[Nemo Action]\n# "+marker+"\nName="+c.Label+"\nExec="+q+" "+Flag+c.ID+
				" %F\nSelection=notnone\nExtensions="+exts+"\n")
		}
	}
	return errors.Join(errs...)
}

func Uninstall(prefix string) error {
	nautilus, caja, dolphin, nemo, err := contextDirs()
	if err != nil {
		return err
	}
	marker := []byte(contextMarker(prefix))
	var errs []error
	for _, dir := range []string{nautilus, caja, dolphin, nemo} {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if b, err := os.ReadFile(p); err == nil && strings.Contains(string(b), string(marker)) {
				errs = append(errs, os.Remove(p))
			}
		}
	}
	return errors.Join(errs...)
}

func dotted(ext string) string {
	if strings.HasPrefix(ext, ".") {
		return ext
	}
	return "." + ext
}
