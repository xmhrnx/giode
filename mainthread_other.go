//go:build !darwin

package giode

// onMainThread reports false: only Apple platforms need the hops.
func onMainThread() bool { return false }
