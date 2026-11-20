//go:build !linux && !darwin && !windows

package device

// Details returns the total RAM, OS version and model.
func Details() (ram uint64, version, model string) { return 0, "", "" }
