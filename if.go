package taptun

import (
	"os"
)

// Interface is a TUN/TAP interface.
type Interface struct {
	isTAP bool
	file  *os.File
	name  string
}

// Create a new TAP interface whose name is ifName.
// If ifName is empty, a default name (tap0, tap1, ... ) will be assigned.
// ifName should not exceed 16 bytes.
func NewTAP(ifName string) (*Interface, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	name, err := createInterface(file.Fd(), ifName, true)
	if err != nil {
		return nil, err
	}
	return &Interface{isTAP: true, file: file, name: name}, nil
}

// Create a new TUN interface whose name is ifName.
// If ifName is empty, a default name (tap0, tap1, ... ) will be assigned.
// ifName should not exceed 16 bytes.
func NewTUN(ifName string) (*Interface, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	name, err := createInterface(file.Fd(), ifName, false)
	if err != nil {
		return nil, err
	}
	return &Interface{isTAP: false, file: file, name: name}, err
}

// Sets the TUN/TAP device in persistent mode.
func (ifce *Interface) SetPersistent(persistent bool) error {
	return setPersistent(ifce.file.Fd(), persistent)
}

// Returns true if ifce is a TUN interface; otherwise returns false.
func (ifce *Interface) IsTUN() bool {
	return !ifce.isTAP
}

// Returns true if ifce is a TAP interface; otherwise returns false.
func (ifce *Interface) IsTAP() bool {
	return ifce.isTAP
}

// Returns the interface name of ifce, e.g. tun0, tap1, etc.
func (ifce *Interface) Name() string {
	return ifce.name
}

// Closes the TUN/TAP interface.
func (ifce *Interface) Close() error {
	return ifce.file.Close()
}

// Implement io.Writer interface.
func (ifce *Interface) Write(p []byte) (n int, err error) {
	return ifce.file.Write(p)
}

// Implement io.Reader interface.
func (ifce *Interface) Read(p []byte) (n int, err error) {
	return ifce.file.Read(p)
}
