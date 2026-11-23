//go:build !ios

package pty

import (
	"bytes"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func ioctl(fd, req, arg uintptr) error {
	if _, _, e := unix.Syscall(unix.SYS_IOCTL, fd, req, arg); e != 0 {
		return e
	}
	return nil
}

func openMaster() (*os.File, string, error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, "", err
	}
	var name [128]byte
	err = control(m, func(fd uintptr) error {
		if err := ioctl(fd, unix.TIOCPTYGRANT, 0); err != nil {
			return err
		}
		if err := ioctl(fd, unix.TIOCPTYUNLK, 0); err != nil {
			return err
		}
		return ioctl(fd, unix.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name[0])))
	})
	if err != nil {
		m.Close()
		return nil, "", err
	}
	if i := bytes.IndexByte(name[:], 0); i >= 0 {
		return m, string(name[:i]), nil
	}
	return m, string(name[:]), nil
}
