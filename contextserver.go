package giode

import (
	"crypto/sha1"
	"fmt"
	"strings"
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
func ContextActionCLSID(prefix, id string) string {
	h := sha1.Sum([]byte("giode-context-action:" + prefix + "." + id))
	b := h[:16]
	b[6] = b[6]&0x0f | 0x50
	b[8] = b[8]&0x3f | 0x80
	return strings.ToUpper(fmt.Sprintf("{%x-%x-%x-%x-%x}", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}

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
