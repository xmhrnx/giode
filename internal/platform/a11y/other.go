//go:build !(android && giode_gio6) && !windows && !darwin && !(linux && !android)

package a11y

// Install: no screen reader bridge here yet; the tree still serves the
// MCP tools.
func Install(a Host) *Tree {
	t := NewTree()
	setTree(a, t)
	return t
}
