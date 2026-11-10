//go:build !darwin || ios

package giode

// onMainThread reports false: only macOS needs invalidate's hop.
func onMainThread() bool { return false }
