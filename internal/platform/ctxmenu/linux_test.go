//go:build linux && !android

package ctxmenu

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextActions_LinuxFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	actions := []Action{
		{ID: "ask", Label: "Ask Agent", Extensions: []string{".md", "txt"}},
		{ID: "here", Label: "Open Agent here", Folders: true, FolderBackground: true},
	}
	if err := Install("com-example-agent", "/opt/agent/agent", actions); err != nil {
		t.Fatal(err)
	}
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Join(home, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if s := read(".local/share/nautilus/scripts/Ask Agent"); !strings.Contains(s, `exec "/opt/agent/agent" --giode-action=ask $NAUTILUS_SCRIPT_SELECTED_FILE_PATHS`) {
		t.Errorf("nautilus script:\n%s", s)
	}
	if info, _ := os.Stat(filepath.Join(home, ".local/share/nautilus/scripts/Ask Agent")); info.Mode()&0o100 == 0 {
		t.Error("nautilus script isn't executable")
	}
	if s := read(".local/share/kio/servicemenus/com-example-agent.ask.desktop"); !strings.Contains(s, "X-KDE-ServiceTypes=KonqPopupMenu/Plugin") || !strings.Contains(s, "Exec=\"/opt/agent/agent\" --giode-action=ask %F") {
		t.Errorf("dolphin service menu:\n%s", s)
	}
	if s := read(".local/share/kio/servicemenus/com-example-agent.here.desktop"); !strings.Contains(s, "MimeType=inode/directory;") {
		t.Errorf("dolphin folder menu:\n%s", s)
	}
	if s := read(".local/share/nemo/actions/com-example-agent.ask.nemo_action"); !strings.Contains(s, "Extensions=md;txt;") {
		t.Errorf("nemo action:\n%s", s)
	}
	if s := read(".local/share/nemo/actions/com-example-agent.here.nemo_action"); !strings.Contains(s, "Extensions=dir") {
		t.Errorf("nemo folder action:\n%s", s)
	}
	read(".config/caja/scripts/Open Agent here")

	// Another app's files survive uninstalling.
	other := filepath.Join(home, ".local/share/nemo/actions/other.nemo_action")
	os.WriteFile(other, []byte("[Nemo Action]\n"), 0o644)
	if err := Uninstall("com-example-agent"); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{".local/share/nautilus/scripts", ".config/caja/scripts", ".local/share/kio/servicemenus"} {
		if e, _ := os.ReadDir(filepath.Join(home, dir)); len(e) != 0 {
			t.Errorf("%s still has %d files", dir, len(e))
		}
	}
	if _, err := os.Stat(other); err != nil {
		t.Error("uninstall removed another app's action")
	}
}
