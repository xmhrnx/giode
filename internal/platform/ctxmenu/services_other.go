//go:build !(darwin && !ios)

package ctxmenu

// RegisterServices: Services are macOS's.
func RegisterServices() {}
