//go:build android || ios || js || (!linux && !darwin && !windows)

package giode

func setLaunchAtLogin(loginEntry, bool) error { return ErrNotSupported }

func launchAtLogin(name string) bool { return false }

func launchedByStartupTask() bool { return false }
