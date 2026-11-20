//go:build !(darwin && !ios)

package giode

// Native menu bars are macOS's; elsewhere apps show menus in the window.
func (a *App) setNativeMenuBar([]menuBarSpec) {}

func (a *App) menuBarView() {}
