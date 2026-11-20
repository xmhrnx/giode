// Package menubar installs the macOS menu bar.
package menubar

// OnClick and OnQuit receive the menu bar's clicks: an item's ID, and
// the app menu's Quit.
var (
	OnClick func(id int)
	OnQuit  func()
)
