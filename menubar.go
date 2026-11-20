package giode

import (
	"context"
	"encoding/json"
	"sync"

	"gioui.org/io/key"

	"github.com/xmhrnx/giode/internal/platform/menubar"
	"github.com/xmhrnx/giode/shortcut"
)

// MenuBarMenu is a menu of the application's native menu bar: on macOS
// the menus at the top of the screen, after the app menu. Windows and
// Linux apps keep their menus in the window (TitleBar's menus), where
// SetMenuBar does nothing.
type MenuBarMenu struct {
	Title string
	Items []MenuBarItem
}

// MenuBarItem is an item of a MenuBarMenu.
type MenuBarItem struct {
	// Label is the item's text.
	Label string
	// Shortcut is the item's key equivalent, as the shortcut package
	// spells it: "Mod+S", "Mod+Shift+Z", "Mod+," (Mod is Cmd on macOS).
	Shortcut string
	// Disabled greys the item out; Checked shows a check mark.
	Disabled, Checked bool
	// Separator makes the item a divider line.
	Separator bool
	// Items makes the item a submenu.
	Items []MenuBarItem
	// OnClick runs on the UI goroutine when the item is chosen.
	OnClick func()
	// Role makes the item a standard one, done by giode: the edit
	// commands reach the focused text field as their keys would.
	Role MenuRole
}

// MenuRole is a standard menu item's job.
type MenuRole int

const (
	RoleNone MenuRole = iota
	RoleUndo
	RoleRedo
	RoleCut
	RoleCopy
	RolePaste
	RoleSelectAll
	RoleMinimize
	RoleZoom
	RoleFullscreen
	RoleClose
)

// EditMenu is the standard Edit menu: Undo, Redo, Cut, Copy, Paste and
// Select All, which act on the focused text field.
func EditMenu() MenuBarMenu {
	return MenuBarMenu{Title: "Edit", Items: []MenuBarItem{
		{Label: "Undo", Shortcut: "Mod+Z", Role: RoleUndo},
		{Label: "Redo", Shortcut: "Mod+Shift+Z", Role: RoleRedo},
		{Separator: true},
		{Label: "Cut", Shortcut: "Mod+X", Role: RoleCut},
		{Label: "Copy", Shortcut: "Mod+C", Role: RoleCopy},
		{Label: "Paste", Shortcut: "Mod+V", Role: RolePaste},
		{Label: "Select All", Shortcut: "Mod+A", Role: RoleSelectAll},
	}}
}

// WindowMenu is the standard Window menu: Minimize, Zoom and Enter Full
// Screen.
func WindowMenu() MenuBarMenu {
	return MenuBarMenu{Title: "Window", Items: []MenuBarItem{
		{Label: "Minimize", Shortcut: "Mod+M", Role: RoleMinimize},
		{Label: "Zoom", Role: RoleZoom},
		{Separator: true},
		{Label: "Enter Full Screen", Shortcut: "Mod+Ctrl+F", Role: RoleFullscreen},
	}}
}

// roleKeys are the keys a role's item sends to the window.
var roleKeys = map[MenuRole]struct {
	name key.Name
	mods key.Modifiers
}{
	RoleUndo:      {"Z", key.ModShortcut},
	RoleRedo:      {"Z", key.ModShortcut | key.ModShift},
	RoleCut:       {"X", key.ModShortcut},
	RoleCopy:      {"C", key.ModShortcut},
	RolePaste:     {"V", key.ModShortcut},
	RoleSelectAll: {"A", key.ModShortcut},
}

// menuBarEntry is an item as the native side gets it.
type menuBarEntry struct {
	ID        int            `json:"id"`
	Label     string         `json:"label"`
	Key       string         `json:"key,omitempty"`
	Mods      int            `json:"mods,omitempty"`
	Disabled  bool           `json:"disabled,omitempty"`
	Checked   bool           `json:"checked,omitempty"`
	Separator bool           `json:"separator,omitempty"`
	Items     []menuBarEntry `json:"items,omitempty"`
}

type menuBarSpec struct {
	Title string         `json:"title"`
	Items []menuBarEntry `json:"items"`
}

var menuBarState struct {
	sync.Mutex
	items map[int]MenuBarItem
}

// SetMenuBar sets the application's menus (macOS); see MenuBarMenu.
// Calling it again replaces them, to update checks and disabled items.
func (a *App) SetMenuBar(menus ...MenuBarMenu) {
	items := map[int]MenuBarItem{}
	next := 0
	var conv func([]MenuBarItem) []menuBarEntry
	conv = func(in []MenuBarItem) []menuBarEntry {
		out := make([]menuBarEntry, 0, len(in))
		for _, it := range in {
			next++
			e := menuBarEntry{ID: next, Label: it.Label, Disabled: it.Disabled, Checked: it.Checked, Separator: it.Separator}
			items[next] = it
			if it.Shortcut != "" && (it.Role == RoleNone || roleKeys[it.Role].name == "" || injectsKeys) {
				if name, mods, err := shortcut.Parse(it.Shortcut); err == nil {
					e.Key, e.Mods = string(name), int(mods)
				}
			}
			if len(it.Items) > 0 {
				e.Items = conv(it.Items)
			}
			out = append(out, e)
		}
		return out
	}
	specs := make([]menuBarSpec, len(menus))
	for i, m := range menus {
		specs[i] = menuBarSpec{Title: m.Title, Items: conv(m.Items)}
	}
	menuBarState.Lock()
	menuBarState.items = items
	menuBarState.Unlock()
	a.setNativeMenuBar(specs)
}

// menuBarClicked runs the item with the given id.
func (a *App) menuBarClicked(id int) {
	menuBarState.Lock()
	it, ok := menuBarState.items[id]
	menuBarState.Unlock()
	if !ok {
		return
	}
	switch it.Role {
	case RoleNone:
		if it.OnClick != nil {
			a.Post(it.OnClick)
		}
	case RoleMinimize:
		a.Post(a.Minimize)
	case RoleZoom:
		a.Post(func() {
			if a.Maximized() {
				a.Unmaximize()
			} else {
				a.Maximize()
			}
		})
	case RoleFullscreen:
		a.Post(func() { a.SetFullscreen(!a.Fullscreen()) })
	case RoleClose:
		a.RequestClose()
	default:
		k := roleKeys[it.Role]
		a.sendKey(k.name, k.mods)
	}
}

// sendKey types a key into the window, as if pressed: a role's keys
// reach the focused field. It needs giode's Gio patches.
func (a *App) sendKey(name key.Name, mods key.Modifiers) {
	go func() {
		ctx := context.Background()
		for _, st := range []key.State{key.Press, key.Release} {
			if injectEvent(ctx, a, key.Event{Name: name, Modifiers: mods, State: st}) != nil {
				return
			}
		}
	}()
}

// menuQuit quits from the app menu, asking OnCloseRequest first.
func (a *App) menuQuit() {
	if a.cfg.OnCloseRequest == nil {
		a.Quit()
		return
	}
	a.Post(func() {
		if a.cfg.OnCloseRequest() {
			a.Quit()
		}
	})
}

func init() {
	menubar.OnClick = func(id int) {
		if a := currentApp.Load(); a != nil {
			a.menuBarClicked(id)
		}
	}
	menubar.OnQuit = func() {
		if a := currentApp.Load(); a != nil {
			a.menuQuit()
		}
	}
}

// setNativeMenuBar hands the menu bar to the system's (macOS's); other
// platforms show menus in the window.
func (a *App) setNativeMenuBar(specs []menuBarSpec) {
	if a.cfg.MenuBarApp {
		return // no app menu in the menu bar
	}
	js, _ := json.Marshal(specs)
	a.menuBarJSON.Store(string(js))
	a.applyNativeMenuBar()
}

// applyNativeMenuBar installs the menu bar; again once the window
// exists, after Gio's launch installed its own.
func (a *App) applyNativeMenuBar() {
	if js, _ := a.menuBarJSON.Load().(string); js != "" {
		menubar.Set(js, a.Title())
	}
}

// menuBarView puts the menu bar back once the window exists: Gio's
// launch installs its own menu.
func (a *App) menuBarView() { a.applyNativeMenuBar() }
