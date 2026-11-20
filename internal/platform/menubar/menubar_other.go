//go:build !(darwin && !ios)

package menubar

// Native menu bars are macOS's; elsewhere apps show menus in the window.
func Set(specJSON, appName string) {}
