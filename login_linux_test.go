//go:build linux && !android

package giode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLaunchAtLogin_Autostart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("FLATPAK_ID", "")
	t.Setenv("APPIMAGE", "/opt/My App.AppImage")
	a := New(Config{Title: "Login Test"})
	if a.LaunchAtLogin() {
		t.Fatal("registered before SetLaunchAtLogin")
	}
	if err := a.SetLaunchAtLogin(true); err != nil {
		t.Fatal(err)
	}
	if !a.LaunchAtLogin() {
		t.Fatal("not registered after SetLaunchAtLogin(true)")
	}
	b, err := os.ReadFile(filepath.Join(dir, "autostart", "Login Test.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `Exec="/opt/My App.AppImage" --giode-login`) {
		t.Errorf("entry:\n%s", b)
	}
	// The user turning it off in the desktop's settings.
	path := filepath.Join(dir, "autostart", "Login Test.desktop")
	os.WriteFile(path, []byte(strings.Replace(string(b), "enabled=true", "enabled=false", 1)), 0o644)
	if a.LaunchAtLogin() {
		t.Error("registered while turned off")
	}
	if err := a.SetLaunchAtLogin(false); err != nil {
		t.Fatal(err)
	}
	if a.LaunchAtLogin() {
		t.Error("registered after SetLaunchAtLogin(false)")
	}
	if err := a.SetLaunchAtLogin(false); err != nil {
		t.Errorf("removing twice: %v", err)
	}
}

func TestLaunchAtLogin_Flatpak(t *testing.T) {
	t.Setenv("FLATPAK_ID", "com.example.App")
	cmd, err := loginCommand()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(cmd, " ") != "flatpak run com.example.App --giode-login" {
		t.Errorf("command %q", cmd)
	}
}
