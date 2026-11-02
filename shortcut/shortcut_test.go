package shortcut

import (
	"testing"

	"gioui.org/io/key"
)

func TestParse(t *testing.T) {
	for spec, want := range map[string]struct {
		name key.Name
		mods key.Modifiers
	}{
		"Mod+T":        {"T", key.ModShortcut},
		"ctrl+shift+p": {"P", key.ModCtrl | key.ModShift},
		"F5":           {"F5", 0},
		"Mod+,":        {",", key.ModShortcut},
		"Mod++":        {"+", key.ModShortcut},
		"Alt+Enter":    {key.NameReturn, key.ModAlt},
		"Shift+PageUp": {key.NamePageUp, key.ModShift},
		"Mod+1":        {"1", key.ModShortcut},
		"Esc":          {key.NameEscape, 0},
	} {
		name, mods, err := Parse(spec)
		if err != nil || name != want.name || mods != want.mods {
			t.Errorf("%s: %q %v %v, want %q %v", spec, name, mods, err, want.name, want.mods)
		}
	}
	for _, bad := range []string{"", "Hyper+T", "Mod+", "Mod+Banana"} {
		if _, _, err := Parse(bad); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
}

func TestHandle(t *testing.T) {
	var ran []string
	list := []Shortcut{
		New("Mod+T", func() { ran = append(ran, "tab") }),
		New("Mod+Shift+T", func() { ran = append(ran, "reopen") }),
		New("F5", func() { ran = append(ran, "refresh") }),
	}
	press := func(n key.Name, m key.Modifiers) bool {
		return Handle(key.Event{Name: n, Modifiers: m, State: key.Press}, list)
	}
	if !press("T", key.ModShortcut) || !press("T", key.ModShortcut|key.ModShift) || !press("F5", 0) {
		t.Error("a shortcut didn't match")
	}
	if press("T", 0) || press("T", key.ModAlt) {
		t.Error("T without its modifier matched")
	}
	if Handle(key.Event{Name: "T", Modifiers: key.ModShortcut, State: key.Release}, list) {
		t.Error("a release matched")
	}
	if len(ran) != 3 || ran[0] != "tab" || ran[1] != "reopen" || ran[2] != "refresh" {
		t.Errorf("ran %v", ran)
	}
}

func TestFormat(t *testing.T) {
	for _, c := range []struct {
		spec string
		mac  bool
		want string
	}{
		{"Mod+Shift+T", false, "Ctrl+Shift+T"},
		{"Mod+Shift+T", true, "⇧⌘T"},
		{"Alt+Enter", false, "Alt+Enter"},
		{"F5", true, "F5"},
		{"not a ++ spec", false, "not a ++ spec"},
	} {
		mods := c.spec
		got := format(mods, c.mac)
		if c.mac && key.ModShortcut != key.ModCommand {
			// On a non-Apple host, Mod parses to Ctrl, shown as ⌃.
			if c.spec == "Mod+Shift+T" {
				c.want = "⌃⇧T"
			}
		}
		if got != c.want {
			t.Errorf("format(%q, mac=%v) = %q, want %q", c.spec, c.mac, got, c.want)
		}
	}
}
