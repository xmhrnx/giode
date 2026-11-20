//go:build darwin

package device

import (
	"runtime"

	"golang.org/x/sys/unix"
)

// Details returns the total RAM, OS version and model.
func Details() (ram uint64, version, model string) {
	ram, _ = unix.SysctlUint64("hw.memsize")
	version, _ = unix.Sysctl("kern.osproductversion")
	if runtime.GOOS == "ios" {
		// iPhones and iPads report their model identifier ("iPhone15,2")
		// as the machine.
		model, _ = unix.Sysctl("hw.machine")
	} else {
		model, _ = unix.Sysctl("hw.model")
	}
	return ram, version, model
}
