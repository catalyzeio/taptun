// +build linux

package taptun

import (
	"strings"
	"syscall"
	"unsafe"
)

const (
	cIFF_TUN   = 0x0001
	cIFF_TAP   = 0x0002
	cIFF_NO_PI = 0x1000
)

type ifReq struct {
	Name  [0x10]byte
	Flags uint16
	pad   [0x28 - 0x10 - 2]byte
}

func createInterface(fd uintptr, ifName string, flags uint16) (createdIFName string, errno error) {
	var req ifReq
	req.Flags = flags
	copy(req.Name[:], ifName)
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if err != 0 {
		return "", err
	}
	return strings.Trim(string(req.Name[:]), "\x00"), nil
}

func setPersistent(fd uintptr, persistent bool) error {
	var val uintptr
	if persistent {
		val = 1
	} else {
		val = 0
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TUNSETPERSIST), val)
	if errno != 0 {
		return errno
	}
	return nil
}
