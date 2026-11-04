//go:build !(windows || (darwin && !ios) || (linux && !android))

package giode

import "gioui.org/app"

const traySupported = false

func trayStart(*Tray)            {}
func trayApply(*Tray, trayState) {}
func trayRemove()                {}

func (a *App) hideNativeWindow() bool { return false }
func (a *App) showNativeWindow()      {}
func (a *App) warnTrayLimits()        {}
func (a *App) platformInit()          {}
func (a *App) trayView(app.ViewEvent) {}
func (a *App) applyTrayOptions()      {}

// raiseNative reports false: Raise goes through Gio here.
func (a *App) raiseNative() bool { return false }
