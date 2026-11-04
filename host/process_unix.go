//go:build !windows

package host

import "syscall"

// processAlive reports whether the process with the given pid is
// running.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// allowForeground does nothing: only Windows restricts which process
// may take the foreground.
func allowForeground(pid int) {}
