package giode

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform/ctxmenu"
)

// ContextAction is an entry the app adds to the file manager's
// right-click menu - "Ask Pocket Agent about this file", "Open in
// Notes" - on files, folders, or the empty space inside a folder.
// Choosing it starts the app (or, with a SingleInstanceLock, hands the
// paths to the running one) and runs Config.OnContextAction with the
// action's ID and the paths.
//
// Declare actions in giode.toml, where the giode CLI also writes them
// into the Windows installer and the macOS app, or in
// Config.ContextActions. They appear:
//
//   - on Windows, in Explorer's context menu ("Show more options" on
//     Windows 11): installed by the setup program, or by
//     App.InstallContextActions for the current user;
//   - on macOS, under Quick Actions / Services in Finder's context
//     menu, from the app's Info.plist (built by the giode CLI) once the
//     app is in /Applications and has run once;
//   - on Linux, in GNOME Files and Caja (under Scripts), Dolphin and
//     Nemo, once App.InstallContextActions has run.
type ContextAction struct {
	// ID identifies the action to OnContextAction. Letters, digits, -
	// and _.
	ID string `json:"id"`
	// Label is the menu text.
	Label string `json:"label"`
	// Files offers the action on files; Extensions limits it to files
	// with these extensions (".md"). Without Files, Folders or
	// FolderBackground, it's offered on files.
	Files      bool     `json:"files,omitempty"`
	Extensions []string `json:"extensions,omitempty"`
	// Folders offers the action on folders.
	Folders bool `json:"folders,omitempty"`
	// FolderBackground offers it on the empty space of an open folder;
	// the path is that folder. Windows and Dolphin only.
	FolderBackground bool `json:"background,omitempty"`
}

// onFiles reports whether the action is offered on files.
func (c ContextAction) onFiles() bool { return ctxmenu.Action(c).OnFiles() }

// contextActionFlag starts the arguments of a context action
// invocation: --giode-action=<id> <paths...>.
const contextActionFlag = ctxmenu.Flag

// buildContextActions is giode.toml's context actions, set by the giode
// CLI at build time: base64 (URL encoding) of their JSON.
var buildContextActions string

// ContextActions returns the app's context actions: giode.toml's
// (through the giode CLI) and Config.ContextActions, which win by ID.
func (a *App) ContextActions() []ContextAction {
	var out []ContextAction
	if buildContextActions != "" {
		if b, err := base64.URLEncoding.DecodeString(buildContextActions); err == nil {
			json.Unmarshal(b, &out)
		}
	}
	for _, c := range a.cfg.ContextActions {
		replaced := false
		for i := range out {
			if out[i].ID == c.ID {
				out[i], replaced = c, true
			}
		}
		if !replaced {
			out = append(out, c)
		}
	}
	return out
}

// parseContextAction reads a context action invocation from
// command-line arguments (without the program name).
func parseContextAction(args []string) (id string, paths []string, ok bool) {
	for i, arg := range args {
		if strings.HasPrefix(arg, contextActionFlag) {
			return strings.TrimPrefix(arg, contextActionFlag), args[i+1:], true
		}
	}
	return "", nil, false
}

// runContextAction runs OnContextAction for args when they invoke one,
// on the UI goroutine.
func (a *App) runContextAction(args []string) bool {
	id, paths, ok := parseContextAction(args)
	if !ok {
		return false
	}
	if a.cfg.OnContextAction == nil {
		a.logf(LogLevelWarning, "giode: context action %q chosen but Config.OnContextAction isn't set", id)
		return true
	}
	a.Post(func() { a.cfg.OnContextAction(id, paths) })
	return true
}

// InstallContextActions adds the app's context actions to the file
// manager's menu for the current user, pointing at the running
// executable; no administrator rights needed. Call it on first run (or
// from a setting) when the app isn't installed by its setup program.
// It replaces actions installed before, so it's safe to call on every
// start. On macOS the actions come from the app bundle instead, and it
// does nothing.
func (a *App) InstallContextActions() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	actions := a.ContextActions()
	for _, c := range actions {
		if c.ID == "" || c.Label == "" || sanitizeID(c.ID) != c.ID {
			return errors.New("giode: a context action needs a Label and an ID of letters, digits, - and _")
		}
	}
	ca := make([]ctxmenu.Action, len(actions))
	for i, c := range actions {
		ca[i] = ctxmenu.Action(c)
	}
	return ctxmenu.Install(a.contextActionPrefix(), exe, ca)
}

// UninstallContextActions removes what InstallContextActions added.
func (a *App) UninstallContextActions() error {
	return ctxmenu.Uninstall(a.contextActionPrefix())
}

// contextActionPrefix names the app's entries: its app ID, or the
// executable's name.
func (a *App) contextActionPrefix() string {
	id := app.ID
	if id == "" {
		exe, _ := os.Executable()
		id = strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
	}
	return sanitizeID(id)
}

func init() {
	// macOS's Services: the app provides them once it launches.
	launchHook = func(a *App) {
		if a.cfg.OnContextAction != nil {
			ctxmenu.RegisterServices()
		}
	}
	ctxmenu.OnService = func(id string, paths []string) {
		if a := currentApp.Load(); a != nil && a.cfg.OnContextAction != nil {
			a.runContextAction(append([]string{contextActionFlag + id}, paths...))
		}
	}
}
