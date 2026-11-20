package giode

import (
	"testing"

	"gioui.org/app"
	"gioui.org/io/key"
)

func TestMenuBarClick(t *testing.T) {
	a := &App{window: new(app.Window)}
	clicked := false
	a.SetMenuBar(MenuBarMenu{Title: "File", Items: []MenuBarItem{
		{Label: "New", Shortcut: "Mod+N", OnClick: func() { clicked = true }},
		{Separator: true},
		{Label: "Recent", Items: []MenuBarItem{{Label: "a.txt"}}},
	}}, EditMenu(), WindowMenu())
	menuBarState.Lock()
	n := len(menuBarState.items)
	menuBarState.Unlock()
	if n != 4+7+4 {
		t.Errorf("%d items", n)
	}
	a.menuBarClicked(1)
	a.runUICalls()
	if !clicked {
		t.Error("New didn't run")
	}
	if k := roleKeys[RoleRedo]; k.name != "Z" || k.mods != key.ModShortcut|key.ModShift {
		t.Error(k)
	}
}
