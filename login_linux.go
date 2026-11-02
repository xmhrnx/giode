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

func setLaunchAtLogin(name string, on bool) error {
	path, err := autostartFile(name)
	if err != nil {
		return err
	}
	if !on {
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
	entry := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=%s\nExec=%s\nX-GNOME-Autostart-enabled=true\nNoDisplay=true\n",
		name, strings.Join(quoted, " "))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(entry), 0o644)
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
