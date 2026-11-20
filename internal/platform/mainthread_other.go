//go:build !darwin

package platform

// OnMainThread reports false: only Apple platforms need the hops.
func OnMainThread() bool { return false }
