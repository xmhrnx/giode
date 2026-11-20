//go:build darwin && !ios

package giode

/*
void giodeRegisterServices(void);
*/
import "C"

import "strings"

// On macOS, context actions are Services: the giode CLI declares them
// in the app's Info.plist (NSServices), Finder lists them under Quick
// Actions, and macOS calls the provider registered here with the
// selected files.

func installContextActions(string, string, []ContextAction) error { return nil }
func uninstallContextActions(string) error                        { return nil }

// registerServices makes the app the provider of its Services.
func init() { launchHook = (*App).registerServices }

func (a *App) registerServices() {
	if a.cfg.OnContextAction != nil {
		C.giodeRegisterServices()
	}
}

//export giodeServiceInvoked
func giodeServiceInvoked(id, paths *C.char) {
	a := currentApp.Load()
	if a == nil || a.cfg.OnContextAction == nil {
		return
	}
	args := []string{contextActionFlag + C.GoString(id)}
	if p := C.GoString(paths); p != "" {
		args = append(args, strings.Split(p, "\n")...)
	}
	a.runContextAction(args)
}
