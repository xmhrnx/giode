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
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "none"))
	defer func(id, name string) { AppID, AppName = id, name }(AppID, AppName)
	AppID, AppName = "", "Login Test"
	icon := testPNG(t)
	a := New(Config{Title: "Login Test - window", Icon: icon})
	if a.LaunchAtLogin() {
		t.Fatal("registered before SetLaunchAtLogin")
	}
	if err := a.SetLaunchAtLogin(true); err != nil {
		t.Fatal(err)
	}
	if !a.LaunchAtLogin() {
		t.Fatal("not registered after SetLaunchAtLogin(true)")
	}
	b, err := os.ReadFile(filepath.Join(dir, "autostart", "Login Test - window.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	iconFile := filepath.Join(dir, "data", "icons", "Login Test - window-login.png")
	for _, w := range []string{`Exec="/opt/My App.AppImage" --giode-login`, "Name=Login Test\n", "Icon=" + iconFile + "\n"} {
		if !strings.Contains(string(b), w) {
			t.Errorf("entry lacks %q:\n%s", w, b)
		}
	}
	if got, err := os.ReadFile(iconFile); err != nil || string(got) != string(icon) {
		t.Errorf("icon file not written: %v", err)
	}
	// The user turning it off in the desktop's settings.
	path := filepath.Join(dir, "autostart", "Login Test - window.desktop")
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
	if _, err := os.Stat(iconFile); err == nil {
		t.Error("icon file left after SetLaunchAtLogin(false)")
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

func TestLaunchAtLogin_ThemeIcon(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "sys"))
	t.Setenv("FLATPAK_ID", "")
	themed := filepath.Join(dir, "sys", "icons", "hicolor", "256x256", "apps", "com.example.app.png")
	os.MkdirAll(filepath.Dir(themed), 0o755)
	os.WriteFile(themed, testPNG(t), 0o644)
	defer func(id, name string) { AppID, AppName = id, name }(AppID, AppName)
	AppID, AppName = "com.example.app", "Example"
	a := New(Config{Title: "x", Icon: testPNG(t)})
	if err := a.SetLaunchAtLogin(true); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "autostart", "com.example.app.desktop"))
	if !strings.Contains(string(b), "Name=Example\n") || !strings.Contains(string(b), "Icon=com.example.app\n") {
		t.Errorf("entry:\n%s", b)
	}
}

func TestDesktopValue(t *testing.T) {
	if got := desktopValue("a\nb"); got != `a\nb` {
		t.Errorf("desktopValue = %q", got)
	}
}
