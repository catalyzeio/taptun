// +build windows

package taptun

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync/atomic"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	tapWin32MaxRegSize = 256
	tunTapComponentID  = "tap0901"
	adapterKey         = "SYSTEM\\CurrentControlSet\\Control\\Class\\{4D36E972-E325-11CE-BFC1-08002BE10318}"
)

var (
	//tapIOCTLGetMTU         = tapControlCode(3, 0)
	tapIOCTLSetMediaStatus = tapControlCode(6, 0)
	tapIOCTLConfigTun      = tapControlCode(10, 0)

	errTunTapNotFound = errors.New("Tun/tap device not found")
)

func createInterface(isTAP bool) (syscall.Handle, string, error) {
	// find existing device, if it does not exist, create it
	id, err := getTuntapComponentID()
	if err != nil {
		if err != errTunTapNotFound {
			return 0, "", err
		}
		c := exec.Command("cmd", "/C", "tapinstall", "install", "C:\\Program Files\\TAP-Windows\\driver\\OemVista.inf", "tap0901")
		err = c.Run()
		if err != nil {
			return 0, "", err
		}
	}
	id, err = getTuntapComponentID()
	if err != nil {
		return 0, "", err
	}
	devicePath := fmt.Sprintf(`\\.\Global\%s.tap`, id)
	name := syscall.StringToUTF16(devicePath)
	tuntap, err := syscall.CreateFile(
		&name[0],
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_SYSTEM|syscall.FILE_FLAG_OVERLAPPED,
		0)
	if err != nil {
		return 0, "", err
	}
	var returnLen uint32
	if !isTAP {
		//var configTunParam = append(addr, network...)
		//configTunParam = append(configTunParam, mask...)
		// TODO figure out what this is doing and if the ip, network, and mask are required
		configTunParam := []byte{10, 0, 0, 1, 10, 0, 0, 0, 255, 255, 255, 0}
		err = syscall.DeviceIoControl(
			tuntap,
			tapIOCTLConfigTun,
			&configTunParam[0],
			uint32(len(configTunParam)),
			&configTunParam[0],
			uint32(len(configTunParam)),
			&returnLen,
			nil)
		if err != nil {
			return 0, "", err
		}
	}
	// set connect.
	inBuffer := []byte("\x01\x00\x00\x00")
	err = syscall.DeviceIoControl(
		tuntap,
		tapIOCTLSetMediaStatus,
		&inBuffer[0],
		uint32(len(inBuffer)),
		&inBuffer[0],
		uint32(len(inBuffer)),
		&returnLen,
		nil)
	if err != nil {
		return 0, "", err
	}
	return tuntap, devicePath, nil
}

func getTuntapComponentID() (string, error) {
	adapters, err := registry.OpenKey(registry.LOCAL_MACHINE, adapterKey, registry.READ)
	if err != nil {
		return "", err
	}
	var i uint32
	for ; i < 1000; i++ {
		var nameLength uint32 = tapWin32MaxRegSize
		buf := make([]uint16, nameLength)
		err = syscall.RegEnumKeyEx(
			syscall.Handle(adapters),
			i,
			&buf[0],
			&nameLength,
			nil,
			nil,
			nil,
			nil)
		if err != nil {
			// hit the end
			break
		}
		keyName := syscall.UTF16ToString(buf[:])
		adapter, err := registry.OpenKey(adapters, keyName, registry.READ)
		if err != nil {
			// typically access denied, just skip it
			continue
		}
		name := syscall.StringToUTF16("ComponentId")
		name2 := syscall.StringToUTF16("NetCfgInstanceId")
		var valtype uint32
		var componentID = make([]byte, tapWin32MaxRegSize)
		var componentLen = uint32(len(componentID))
		err = syscall.RegQueryValueEx(
			syscall.Handle(adapter),
			&name[0],
			nil,
			&valtype,
			&componentID[0],
			&componentLen)
		if err != nil {
			// doesn't have a component ID, skip it
			continue
		}

		if unicodeTostring(componentID) == tunTapComponentID {
			var valtype uint32
			var netCfgInstanceID = make([]byte, tapWin32MaxRegSize)
			var netCfgInstanceIDLen = uint32(len(netCfgInstanceID))
			err = syscall.RegQueryValueEx(
				syscall.Handle(adapter),
				&name2[0],
				nil,
				&valtype,
				&netCfgInstanceID[0],
				&netCfgInstanceIDLen)
			if err != nil {
				// doesnt have a netCfgInstanceID, skip it
				continue
			}
			return unicodeTostring(netCfgInstanceID), nil
		}
	}
	return "", errTunTapNotFound
}

type wrapper struct {
	fd uintptr // atomic uintptr
}

func wrap(ifce *Interface) (*wrapper, error) {
	// grab the file descriptor
	fd := ifce.file.Fd()

	// validate that the file descriptor can be used in a select call
	if fd < 0 {
		return nil, fmt.Errorf("file descriptor cannot be used")
	}

	return &wrapper{fd}, nil
}

func (w *wrapper) Write(p []byte) (int, error) {
	overlappedRx := syscall.Overlapped{}
	hevent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return 0, err
	}
	overlappedRx.HEvent = syscall.Handle(hevent)
	var l uint32
	ptr := &w.fd
	// grab current fd and bail if already stopped
	current := atomic.LoadUintptr(ptr)
	fd := int(current)
	if fd < 0 {
		return 0, io.EOF
	}

	err = syscall.WriteFile(syscall.Handle(w.fd), p, &l, &overlappedRx)
	if err != nil && err != syscall.ERROR_IO_PENDING {
		return 0, err
	}
	_, err = syscall.WaitForSingleObject(overlappedRx.HEvent, syscall.INFINITE) //syscall.WAIT_TIMEOUT)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wrapper) Read(p []byte) (int, error) {
	overlappedRx := syscall.Overlapped{}
	hevent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		fmt.Println("2")
		return 0, err
	}
	overlappedRx.HEvent = syscall.Handle(hevent)
	var l uint32
	ptr := &w.fd
	// grab current fd and bail if already stopped
	current := atomic.LoadUintptr(ptr)
	fd := int(current)
	if fd < 0 {
		fmt.Println("1")
		return 0, io.EOF
	}

	err = syscall.ReadFile(syscall.Handle(w.fd), p, &l, &overlappedRx)
	if err != nil && err != syscall.ERROR_IO_PENDING {
		return 0, err
	}
	_, err = syscall.WaitForSingleObject(overlappedRx.HEvent, syscall.INFINITE) //syscall.WAIT_TIMEOUT)
	if err != nil {
		return 0, err
	}
	// TODO I _think_ this is the correct way to get bytes read
	return int(overlappedRx.InternalHigh), nil
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

func unicodeTostring(src []byte) string {
	var dst []byte
	for _, ch := range src {
		if ch != byte(0) {
			dst = append(dst, ch)
		}
	}
	return string(dst)
}

func ctlCode(deviceType, function, method, access uint32) uint32 {
	return (deviceType << 16) | (access << 14) | (function << 2) | method
}

func tapControlCode(request, method uint32) uint32 {
	return ctlCode(34, request, method, 0)
}
