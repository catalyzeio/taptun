// +build linux

package taptun

import (
	"fmt"
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

func createInterface(fd uintptr, ifName string, isTAP bool) (createdIFName string, err error) {
	if len(ifName) > 0x10 {
		return "", fmt.Errorf("interface name '%s' is too long", ifName)
	}
	var req ifReq
	if isTAP {
		req.Flags = cIFF_TAP | cIFF_NO_PI
	} else {
		req.Flags = cIFF_TUN | cIFF_NO_PI
	}
	copy(req.Name[:], ifName)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		return "", errno
	}
	return strings.Trim(string(req.Name[:]), "\x00"), nil
}

func setPersistent(fd uintptr, persistent bool) error {
	var val uintptr = 0
	if persistent {
		val = 1
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TUNSETPERSIST), val)
	if errno != 0 {
		return errno
	}
	return nil
}
