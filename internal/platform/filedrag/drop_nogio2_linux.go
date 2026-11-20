//go:build !giode_gio2 && linux && !android

package filedrag

func installLinux() bool { return false }
