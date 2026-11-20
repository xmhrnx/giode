//go:build !(linux && !android) && !windows && !(darwin && !ios)

package giode

func (a *App) watchPower(done <-chan struct{}, event func(suspend bool)) {}
