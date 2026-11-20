//go:build !windows && !(darwin && !ios) && !(linux && !android && !nox11)

package giode

func (a *App) windowButtonsView() {}
