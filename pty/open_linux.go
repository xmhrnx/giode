package pty

import (
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

func openMaster() (*os.File, string, error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, "", err
	}
	var n uint32
	err = control(m, func(fd uintptr) error {
		if err := unix.IoctlSetPointerInt(int(fd), unix.TIOCSPTLCK, 0); err != nil {
			return err
		}
		var err error
		n, err = unix.IoctlGetUint32(int(fd), unix.TIOCGPTN)
		return err
	})
	if err != nil {
		m.Close()
		return nil, "", err
	}
	return m, "/dev/pts/" + strconv.Itoa(int(n)), nil
}
