//go:build !darwin || nometal

package gl

// MetalOnly reports a build whose Gio can't render with OpenGL.
const MetalOnly = false
