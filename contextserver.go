package giode

import (
	"strings"

	"github.com/xmhrnx/giode/internal/platform/ctxmenu"
)

// Explorer starts a program once per selected file for a command-line
// verb, so on Windows the context actions on files and folders are
// DropTarget verbs instead: Explorer hands every selected item to a COM
// class the running app registers (internal/platform/ctxmenu), starting
// the app first - with -Embedding - only when it isn't running. Each
// action's class ID is derived from the app's prefix and the action's
// ID, the same way the giode CLI's installer derives it.

// ContextActionCLSID returns the COM class ID Explorer's DropTarget
// verb for a context action names, as "{XXXXXXXX-XXXX-...}": a name-based
// UUID (version 5) of "giode-context-action:" + prefix + "." + id.
func ContextActionCLSID(prefix, id string) string { return ctxmenu.CLSID(prefix, id) }

// isEmbeddingArg reports whether arg is the flag COM starts a local
// server with.
func isEmbeddingArg(arg string) bool {
	return strings.EqualFold(arg, "-Embedding") || strings.EqualFold(arg, "/Embedding")
}

// withoutEmbedding drops COM's -Embedding flag from launch arguments.
func withoutEmbedding(args []string) []string {
	out := args[:0:0]
	for _, a := range args {
		if !isEmbeddingArg(a) {
			out = append(out, a)
		}
	}
	return out
}

// serveContextActions serves the app's context actions on files and
// folders to Explorer's DropTarget verbs (Windows), for the app's
// lifetime: a whole selection reaches this instance.
func (a *App) serveContextActions() {
	var ids []string
	for _, c := range a.ContextActions() {
		if c.onFiles() || c.Folders {
			ids = append(ids, c.ID)
		}
	}
	ctxmenu.Serve(a.contextActionPrefix(), ids, func(id string, paths []string) {
		a.runContextAction(append([]string{contextActionFlag + id}, paths...))
		a.showFromLaunch()
	}, func(format string, args ...any) { a.logf(LogLevelWarning, format, args...) })
}

// stopContextActions revokes the context actions' COM classes, when
// they were registered. The app calls it as it exits.
func (a *App) stopContextActions() { ctxmenu.StopServing() }
