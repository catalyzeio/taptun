// +build windows

package taptun

import (
	"os"
	"syscall"
)

// Create a new TAP interface. ifName is ignored for windows.
func NewTAP(ifName string) (*Interface, error) {
	fd, name, err := createInterface(true)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	return &Interface{isTAP: true, file: file, name: name}, nil
}

// Create a new TUN interface. ifName is ignored for windows.
func NewTUN(ifName string) (*Interface, error) {
	fd, name, err := createInterface(false)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	return &Interface{isTAP: false, file: file, name: name}, err
}

// Sets the TUN/TAP device in persistent mode.
func (ifce *Interface) SetPersistent(persistent bool) error {
	return nil
}

// Returns whether ifce is a TUN interface.
func (ifce *Interface) IsTUN() bool {
	return !ifce.isTAP
}

// Returns whether ifce is a TAP interface.
func (ifce *Interface) IsTAP() bool {
	return ifce.isTAP
}

// Returns the interface name of ifce, e.g., tun0, tap1, etc.
func (ifce *Interface) Name() string {
	return ifce.name
}

// Closes the TUN/TAP interface.
func (ifce *Interface) Close() error {
	return syscall.Close(syscall.Handle(ifce.file.Fd()))
}

// Implement io.Writer interface.
func (ifce *Interface) Write(p []byte) (n int, err error) {
	w, err := wrap(ifce)
	if err != nil {
		return 0, err
	}
	return w.Write(p)
}

// Implement io.Reader interface.
func (ifce *Interface) Read(p []byte) (int, error) {
	w, err := wrap(ifce)
	if err != nil {
		return 0, err
	}
	return w.Read(p)
}
