//go:build linux && !android

package login

import (
	"bytes"
	"image"
	"image/png"
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
	icon := testPNG(t)
	e := Entry{Name: "Login Test - window", Display: "Login Test", Icon: icon}
	if Enabled(e.Name) {
		t.Fatal("registered before Set")
	}
	if err := Set(e, true); err != nil {
		t.Fatal(err)
	}
	if !Enabled(e.Name) {
		t.Fatal("not registered after Set(true)")
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
	if Enabled(e.Name) {
		t.Error("registered while turned off")
	}
	if err := Set(e, false); err != nil {
		t.Fatal(err)
	}
	if Enabled(e.Name) {
		t.Error("registered after Set(false)")
	}
	if _, err := os.Stat(iconFile); err == nil {
		t.Error("icon file left after Set(false)")
	}
	if err := Set(e, false); err != nil {
		t.Errorf("removing twice: %v", err)
	}
}

func TestLaunchAtLogin_Flatpak(t *testing.T) {
	t.Setenv("FLATPAK_ID", "com.example.App")
	cmd, err := Command()
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
	e := Entry{Name: "com.example.app", Display: "Example", ID: "com.example.app", Icon: testPNG(t)}
	if err := Set(e, true); err != nil {
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

// testPNG is a 1x1 PNG.
func testPNG(t *testing.T) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
