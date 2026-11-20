//go:build linux

package device

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// Details returns the total RAM, OS version and model.
func Details() (ram uint64, version, model string) {
	ram = memTotal()
	if runtime.GOOS == "android" {
		return ram, getprop("ro.build.version.release"), getprop("ro.product.model")
	}
	var u unix.Utsname
	if unix.Uname(&u) == nil {
		version = unix.ByteSliceToString(u.Release[:])
	}
	if b, err := os.ReadFile("/sys/devices/virtual/dmi/id/product_name"); err == nil {
		model = strings.TrimSpace(string(b))
	}
	return ram, version, model
}

// memTotal reads MemTotal from /proc/meminfo.
func memTotal() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kb, _ := strconv.ParseUint(fields[1], 10, 64)
			return kb * 1024
		}
	}
	return 0
}

// getprop reads an Android system property.
func getprop(name string) string {
	out, err := exec.Command("/system/bin/getprop", name).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
