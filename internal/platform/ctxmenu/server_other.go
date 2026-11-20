//go:build !windows

package ctxmenu

// Serve and StopServing run Windows' DropTarget verbs' COM server;
// elsewhere a context action starts the app with its arguments.
func Serve(prefix string, ids []string, run func(id string, paths []string), logf func(format string, args ...any)) {
}

func StopServing() {}
