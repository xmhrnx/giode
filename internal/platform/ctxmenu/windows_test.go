//go:build windows

package ctxmenu

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestContextActions_WindowsRegistry(t *testing.T) {
	prefix := "giode-test-ctx"
	actions := []Action{
		{ID: "ask", Label: "Ask Agent", Extensions: []string{".md"}},
		{ID: "all", Label: "Send to Agent"},
		{ID: "here", Label: "Open Agent here", Folders: true, FolderBackground: true},
	}
	if err := Install(prefix, `C:\Agent\agent.exe`, actions); err != nil {
		t.Fatal(err)
	}
	defer Uninstall(prefix)
	check := func(path, want string) {
		t.Helper()
		k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Classes\`+path+`\command`, registry.QUERY_VALUE)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return
		}
		defer k.Close()
		got, _, _ := k.GetStringValue("")
		if got != want {
			t.Errorf("%s command %q, want %q", path, got, want)
		}
	}
	check(`SystemFileAssociations\.md\shell\giode-test-ctx.ask`, `"C:\Agent\agent.exe" --giode-action=ask "%1"`)
	check(`*\shell\giode-test-ctx.all`, `"C:\Agent\agent.exe" --giode-action=all "%1"`)
	check(`Directory\shell\giode-test-ctx.here`, `"C:\Agent\agent.exe" --giode-action=here "%1"`)
	check(`Directory\Background\shell\giode-test-ctx.here`, `"C:\Agent\agent.exe" --giode-action=here "%V"`)
	k, _ := registry.OpenKey(registry.CURRENT_USER, `Software\Classes\*\shell\giode-test-ctx.all`, registry.QUERY_VALUE)
	if label, _, _ := k.GetStringValue("MUIVerb"); label != "Send to Agent" {
		t.Errorf("label %q", label)
	}
	k.Close()

	if err := Uninstall(prefix); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.OpenKey(registry.CURRENT_USER, `Software\Classes\*\shell\giode-test-ctx.all`, registry.QUERY_VALUE); err == nil {
		t.Error("key left after uninstalling")
	}
}
