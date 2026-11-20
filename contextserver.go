package giode

import (
	"strings"

	"github.com/xmhrnx/giode/internal/platform/ctxmenu"
)

// Explorer starts a program once per selected file for a command-line
// verb, so on Windows the context actions on files and folders are
// DropTarget verbs instead: Explorer hands every selected item to a COM
// class the running app registers (contextserver_windows.go), starting
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
