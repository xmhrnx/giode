//go:build !(windows || (darwin && !ios) || (linux && !android))

package giode

import "gioui.org/app"

const traySupported = false

func trayStart(*Tray)            {}
func trayApply(*Tray, trayState) {}
func trayRemove()                {}

func (a *App) hideNativeWindow()      {}
func (a *App) showNativeWindow()      {}
func (a *App) warnTrayLimits()        {}
func (a *App) platformInit()          {}
func (a *App) trayView(app.ViewEvent) {}
func (a *App) applyTrayOptions()      {}
