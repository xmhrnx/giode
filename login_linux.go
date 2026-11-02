//go:build linux && !android

package giode

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func autostartFile(name string) (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", name+".desktop"), nil
}

func setLaunchAtLogin(e loginEntry, on bool) error {
	path, err := autostartFile(e.name)
	if err != nil {
		return err
	}
	iconFile := loginIconFile(e.name)
	if !on {
		if iconFile != "" {
			os.Remove(iconFile)
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	cmd, err := loginCommand()
	if err != nil {
		return err
	}
	quoted := make([]string, len(cmd))
	for i, c := range cmd {
		quoted[i] = desktopQuote(c)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Desktop Entry]\nType=Application\nName=%s\nExec=%s\n", desktopValue(e.display), strings.Join(quoted, " "))
	if icon := loginIcon(e, iconFile); icon != "" {
		fmt.Fprintf(&b, "Icon=%s\n", desktopValue(icon))
	}
	b.WriteString("X-GNOME-Autostart-enabled=true\nNoDisplay=true\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// loginIcon is the entry's Icon value: the Flatpak's icon, the icon a
// package installed under the app ID (deb, rpm), else Config.Icon
// written to iconFile. Empty when there's none.
func loginIcon(e loginEntry, iconFile string) string {
	if id := os.Getenv("FLATPAK_ID"); id != "" {
		return id
	}
	if AppID != "" && themeIconInstalled(AppID) {
		return AppID
	}
	if len(e.icon) == 0 || iconFile == "" {
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(iconFile), 0o755); err != nil {
		return ""
	}
	if err := os.WriteFile(iconFile, e.icon, 0o644); err != nil {
		return ""
	}
	return iconFile
}

// loginIconFile is where Config.Icon is written for the entry:
// $XDG_DATA_HOME/icons, which icon lookups search too.
func loginIconFile(name string) string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "icons", name+"-login.png")
}

// themeIconInstalled reports whether an icon theme has an icon called
// name: hicolor, where the CLI's packages install the app's icon.
func themeIconInstalled(name string) bool {
	var dirs []string
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		dirs = append(dirs, d)
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share"))
	}
	sys := os.Getenv("XDG_DATA_DIRS")
	if sys == "" {
		sys = "/usr/local/share:/usr/share"
	}
	dirs = append(dirs, filepath.SplitList(sys)...)
	for _, d := range dirs {
		for _, pattern := range []string{"*/apps/" + name + ".png", "*/apps/" + name + ".svg"} {
			if m, _ := filepath.Glob(filepath.Join(d, "icons", "hicolor", pattern)); len(m) > 0 {
				return true
			}
		}
	}
	return false
}

// desktopValue keeps a value on one line of a desktop entry.
func desktopValue(s string) string {
	return strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(s)
}

func launchAtLogin(name string) bool {
	path, err := autostartFile(name)
	if err != nil {
		return false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return !strings.Contains(string(b), "X-GNOME-Autostart-enabled=false") && !strings.Contains(string(b), "Hidden=true")
}

// desktopQuote quotes an Exec argument as the desktop entry spec asks.
func desktopQuote(s string) string {
	if !strings.ContainsAny(s, " \t\n\"'\\><~|&;$*?#()`%") {
		return s
	}
	r := strings.NewReplacer(`\`, `\\\\`, `"`, `\\"`, "`", "\\\\`", "$", `\\$`, "%", "%%")
	return `"` + r.Replace(s) + `"`
}

func launchedByStartupTask() bool { return false }
