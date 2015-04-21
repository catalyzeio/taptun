// +build linux

package taptun

import (
	"fmt"
	"io"
	"strings"
	"sync/atomic"
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

type wrapper struct {
	fd uintptr // atomic uintptr
}

func wrap(ifce *Interface) (*wrapper, error) {
	// grab the file descriptor
	fd := ifce.file.Fd()

	// validate that the file descriptor can be used in a select call
	if fd < 0 || fd >= syscall.FD_SETSIZE {
		return nil, fmt.Errorf("file descriptor cannot be used with select(2)")
	}

	// set the file descriptor in non-blocking mode
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		return nil, err
	}

	return &wrapper{fd}, nil
}

func (w *wrapper) Write(p []byte) (n int, err error) {
	ptr := &w.fd
	ready := true
	for {
		// grab current fd and bail if already stopped
		current := atomic.LoadUintptr(ptr)
		fd := int(current)
		if fd < 0 {
			return 0, io.EOF
		}

		// attempt write if descriptor is ready
		if ready {
			n, err := syscall.Write(fd, p)
			if err != nil && err != syscall.EAGAIN || n > 0 {
				return n, err
			}
		}

		// wait for fd to become ready
		selected, err := waitFD(fd, false)
		if err != nil {
			return 0, err
		}
		ready = selected
	}
}

func (w *wrapper) Read(p []byte) (n int, err error) {
	ptr := &w.fd
	ready := true
	for {
		// grab current fd and bail if already stopped
		current := atomic.LoadUintptr(ptr)
		fd := int(current)
		if fd < 0 {
			return 0, io.EOF
		}

		// attempt read if descriptor is ready
		if ready {
			n, err := syscall.Read(fd, p)
			if err != nil && err != syscall.EAGAIN || n > 0 {
				return n, err
			}
		}

		// wait for fd to become ready
		selected, err := waitFD(fd, true)
		if err != nil {
			return 0, err
		}
		ready = selected
	}
}

func (w *wrapper) Stop() bool {
	ptr := &w.fd
	for {
		// grab current fd and bail if already stopped
		old := atomic.LoadUintptr(ptr)
		fd := int(old)
		if fd < 0 {
			return false
		}

		// replace fd with stopped marker
		newFD := -1
		new := uintptr(newFD)
		if atomic.CompareAndSwapUintptr(ptr, old, new) {
			return true
		}
	}
}

/*
Related issue: https://golang.org/issue/500. In this case, the tap/tun
device reads and writes are definitely not mapped to select, so we have
to emulate the related macros (FD_SET, FD_ISSET, etc) directly.
*/

const (
	fdBits = syscall.FD_SETSIZE / len(syscall.FdSet{}.Bits)
)

func waitFD(fd int, read bool) (bool, error) {
	nfd := fd + 1

	var off int
	var index uint
	switch fdBits {
	case 16:
		off = fd >> 4
		index = uint(fd & 0x0F)
	case 32:
		off = fd >> 5
		index = uint(fd & 0x1F)
	case 64:
		off = fd >> 6
		index = uint(fd & 0x3F)
	case 128:
		off = fd >> 7
		index = uint(fd & 0x7F)
	default:
		return false, fmt.Errorf("unexpected FdSet element size")
	}

	set := syscall.FdSet{}
	eset := syscall.FdSet{}
	set.Bits[off] |= 1 << index
	eset.Bits[off] |= 1 << index

	tv := syscall.Timeval{Sec: 1}

	var n int
	var err error
	if read {
		n, err = syscall.Select(nfd, &set, nil, &eset, &tv)
	} else {
		n, err = syscall.Select(nfd, nil, &set, &eset, &tv)
	}

	if err != nil {
		return false, err
	}

	// We're only waiting on one file descriptor, so there is no need to
	// check which file descriptor is ready.
	return n > 0, nil
}
