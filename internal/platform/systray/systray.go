//go:build windows || (darwin && !ios) || (linux && !android)

// Package systray shows the app's icon and menu in the system tray (the
// notification area, the macOS menu bar, a StatusNotifierItem).
package systray

import "fyne.io/systray"

// Supported reports whether the platform has a tray.
const Supported = true

// Item is a tray menu item, as giode.TrayItem.
type Item struct {
	Label, Tooltip                          string
	Checkable, Checked, Disabled, Separator bool
	Items                                   []Item
	OnClick                                 func()
}

// Apply shows the icon (PNG), title, tooltip and menu in the tray.
// onTapped runs on a click on the icon, or nil leaves clicks to the menu.
// post runs the items' OnClick on the UI goroutine.
func Apply(icon []byte, template bool, title, tooltip string, onTapped func(), items []Item, post func(func())) {
	if len(icon) > 0 {
		setIcon(icon, template)
	}
	systray.SetTitle(title)
	systray.SetTooltip(tooltip)
	systray.SetOnTapped(onTapped)
	systray.ResetMenu()
	addItems(nil, items, post)
}

// addItems adds items to the menu, or to parent's submenu.
func addItems(parent *systray.MenuItem, items []Item, post func(func())) {
	for _, it := range items {
		if it.Separator {
			if parent == nil {
				systray.AddSeparator()
			} else {
				parent.AddSeparator()
			}
			continue
		}
		var mi *systray.MenuItem
		switch {
		case parent == nil && it.Checkable:
			mi = systray.AddMenuItemCheckbox(it.Label, it.Tooltip, it.Checked)
		case parent == nil:
			mi = systray.AddMenuItem(it.Label, it.Tooltip)
		case it.Checkable:
			mi = parent.AddSubMenuItemCheckbox(it.Label, it.Tooltip, it.Checked)
		default:
			mi = parent.AddSubMenuItem(it.Label, it.Tooltip)
		}
		if it.Disabled {
			mi.Disable()
		}
		if len(it.Items) > 0 {
			addItems(mi, it.Items, post)
		}
		if fn := it.OnClick; fn != nil {
			// The channel closes when the menu is rebuilt.
			go func() {
				for range mi.ClickedCh {
					post(fn)
				}
			}()
		}
	}
}
