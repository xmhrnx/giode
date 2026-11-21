//go:build darwin && !nometal

package gl

// MetalOnly reports a build whose Gio can't render with OpenGL: macOS
// and iOS without the nometal tag.
const MetalOnly = true
